package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/jinzhu/gorm"
	"github.com/opacity/storage-node/utils"
)

/*File defines a model for managing a user subscription for uploads*/
type File struct {
	/*FileID will either be the file handle, or a hash of the file handle.  We should add an appropriate length
	restriction and can change the name to FileHandle if it is appropriate*/
	FileID           string    `gorm:"primary_key" json:"fileID" binding:"required,len=64" minLength:"64" maxLength:"64"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
	ExpiredAt        time.Time `json:"expiredAt"`
	AwsUploadID      *string   `json:"awsUploadID"`
	AwsObjectKey     *string   `json:"awsObjectKey"`
	EndIndex         int       `json:"endIndex" binding:"required,gte=1"`
	CompletedIndexes *string   `json:"completedIndexes" gorm:"type:mediumtext"`
	ModifierHash     string    `json:"modifierHash" binding:"required,len=64" minLength:"64" maxLength:"64"`
}

type IndexMap map[int64]*s3.CompletedPart

/*UploadStatusType defines a type for the upload statuses*/
type UploadStatusType int

const (
	/*FileUploadNotStarted is for files we haven't started uploading yet*/
	FileUploadNotStarted UploadStatusType = iota + 1

	/*FileUploadStarted is for files we have started uploading*/
	FileUploadStarted

	/*FileUploadComplete is for files we have finished uploading*/
	FileUploadComplete

	/*FileUploadError is for files we experienced an error uploading*/
	FileUploadError = -1
)

const FirstChunkIndex = 1

/*UploadStatusMap is for pretty printing the UploadStatus*/
var UploadStatusMap = make(map[UploadStatusType]string)

/*IncompleteUploadErr is what we will get if we call FinishUpload on an upload that is not done.
It's not really an error.*/
var IncompleteUploadErr = errors.New("missing some chunks, cannot finish upload")

func init() {
	UploadStatusMap[FileUploadNotStarted] = "FileUploadNotStarted"
	UploadStatusMap[FileUploadStarted] = "FileUploadStarted"
	UploadStatusMap[FileUploadComplete] = "FileUploadComplete"
	UploadStatusMap[FileUploadError] = "FileUploadError"
}

/*BeforeCreate - callback called before the row is created*/
func (file *File) BeforeCreate(scope *gorm.Scope) error {
	return nil
}

/*PrettyString - print the file in a friendly way.  Not used for external logging, just for watching in the
terminal*/
func (file *File) PrettyString() {
	fmt.Print("FileID:                         ")
	fmt.Println(file.FileID)

	fmt.Print("CreatedAt:                      ")
	fmt.Println(file.CreatedAt)

	fmt.Print("UpdatedAt:                      ")
	fmt.Println(file.UpdatedAt)

	fmt.Print("AwsUploadID:                    ")
	fmt.Println(*file.AwsUploadID)

	fmt.Print("AwsObjectKey:                   ")
	fmt.Println(*file.AwsObjectKey)

	fmt.Print("EndIndex:                       ")
	fmt.Println(file.EndIndex)

	fmt.Print("CompletedIndexes:               ")
	fmt.Println(file.CompletedIndexes)
}

func GetFileMetadataKey(fileID string) string {
	return fileID + "/metadata"
}

func GetFileDataKey(fileID string) string {
	return fileID + "/file"
}

/*GetOrCreateFile - Get or create the file. */
func GetOrCreateFile(file File) (*File, error) {
	var fileFromDB File
	err := DB.Where(File{FileID: file.FileID}).Attrs(file).FirstOrCreate(&fileFromDB).Error

	return &fileFromDB, err
}

/*UpdateKeyAndUploadID - update the key and uploadID*/
func (file *File) UpdateKeyAndUploadID(key, uploadID *string) error {
	if err := DB.Model(&file).Updates(File{AwsObjectKey: key, AwsUploadID: uploadID}).Error; err != nil {
		return err
	}
	return nil
}

/*UpdateCompletedIndexes - update the completed indexes*/
func (file *File) UpdateCompletedIndexes(completedPart *s3.CompletedPart) error {
	completedIndexes := file.GetCompletedIndexesAsMap()
	completedIndexes[*completedPart.PartNumber] = completedPart

	if err := file.SaveCompletedIndexesAsString(completedIndexes); err != nil {
		return err
	}

	completedIndex := int(aws.Int64Value(completedPart.PartNumber))
	etag := aws.StringValue(completedPart.ETag)
	return CreateCompletedUploadIndex(file.FileID, completedIndex, etag)
}

/*GetCompletedIndexesAsMap takes the file's CompletedIndexes, converts them to a map,
and returns the map*/
func (file *File) GetCompletedIndexesAsMap() IndexMap {
	var completedIndexes IndexMap
	if file.CompletedIndexes == nil {
		completedIndexes = make(IndexMap)
	} else {
		err := json.Unmarshal([]byte(*(file.CompletedIndexes)), &completedIndexes)
		utils.PanicOnError(err)
	}
	return completedIndexes
}

/*GetCompletedPartsAsArray takes the map of completed parts and makes them into an array*/
func (file *File) GetCompletedPartsAsArray() []*s3.CompletedPart {
	completedIndexes := file.GetCompletedIndexesAsMap()
	var completedParts []*s3.CompletedPart

	for index := FirstChunkIndex; index <= file.EndIndex; index++ {
		if _, ok := completedIndexes[int64(index)]; ok {
			completedParts = append(completedParts, completedIndexes[int64(index)])
		}
	}

	return completedParts
}

/*GetIncompleteIndexesAsArray returns an array of the missing indexes */
func (file *File) GetIncompleteIndexesAsArray() []int64 {
	completedIndexMap := file.GetCompletedIndexesAsMap()
	var incompleteIndexes []int64

	for index := FirstChunkIndex; index <= file.EndIndex; index++ {
		if _, ok := completedIndexMap[int64(index)]; !ok {
			incompleteIndexes = append(incompleteIndexes, int64(index))
		}
	}

	return incompleteIndexes
}

/*SaveCompletedIndexesAsString accepts a map of chunk indexes, converts them to a string,
and saves it to the file's CompletedIndexes.*/
func (file *File) SaveCompletedIndexesAsString(completedIndexes IndexMap) error {
	indexAsBytes, err := json.Marshal(completedIndexes)
	if err != nil {
		return err
	}
	indexAsString := string(indexAsBytes)
	if err = DB.Model(&file).UpdateColumn("completed_indexes", &indexAsString).Error; err != nil {
		return err
	}

	return nil
}

/*UploadCompleted gets the file's CompletedIndexes as a map, then checks that map to verify
that all expected chunk indexes have been added to the map.  If any are not found it returns false.
If none are discovered missing it returns true.  */
func (file *File) UploadCompleted() bool {
	// Use completed_upload_indexes table to see whether we finished or not
	// Fallback to indexs map. If count is 0, then fall back to old way.
	count, err := GetCompletedUploadProgress(file.FileID)
	if err == nil && count > 0 {
		return count == ((file.EndIndex - FirstChunkIndex) + 1)
	}

	completedIndexMap := file.GetCompletedIndexesAsMap()

	for index := FirstChunkIndex; index <= file.EndIndex; index++ {
		if _, ok := completedIndexMap[int64(index)]; !ok {
			return false
		}
	}

	return true
}

/*FinishUpload - finishes the upload*/
func (file *File) FinishUpload() (CompletedFile, error) {
	allChunksUploaded := file.UploadCompleted()
	if !allChunksUploaded {
		return CompletedFile{}, IncompleteUploadErr
	}

	completedParts, err := GetCompletedPartsAsArray(file.FileID)
	if err != nil {
		return CompletedFile{}, err
	}

	objectKey := aws.StringValue(file.AwsObjectKey)
	if _, err := utils.CompleteMultiPartUpload(objectKey, aws.StringValue(file.AwsUploadID), completedParts); err != nil {
		return CompletedFile{}, err
	}

	objectSize := utils.GetDefaultBucketObjectSize(objectKey)
	compeletedFile := CompletedFile{
		FileID:         file.FileID,
		ExpiredAt:      file.ExpiredAt,
		FileSizeInByte: objectSize,
		ModifierHash:   file.ModifierHash,
	}
	if err := DB.Save(&compeletedFile).Error; err != nil {
		return CompletedFile{}, err
	}

	if err := DeleteCompletedUploadIndexes(file.FileID); err != nil {
		return compeletedFile, err
	}

	return compeletedFile, DB.Delete(file).Error
}

/*Return File object(first one) if there is not any error. If not found, return nil without error. */
func GetFileById(fileID string) (File, error) {
	file := File{}
	err := DB.Where("file_id = ?", fileID).First(&file).Error
	return file, err
}

/*DeleteUploadsOlderThan will delete files older than the time provided.  If a file still isn't complete by the
time passed in, the assumption is it error'd or will never be finished. */
func DeleteUploadsOlderThan(createdAtTime time.Time) ([]File, error) {
	files := []File{}

	if err := DB.Where("created_at < ?",
		createdAtTime).Find(&files).Error; err != nil {
		return files, err
	}

	for _, file := range files {
		DB.Delete(file)
	}

	return files, nil
}
