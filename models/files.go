package models

import (
	"errors"
	"time"

	"fmt"

	"encoding/json"

	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/jinzhu/gorm"
	"github.com/opacity/storage-node/utils"
)

var getFileMutex = &sync.Mutex{}

/*File defines a model for managing a user subscription for uploads*/
type File struct {
	/*FileID will either be the file handle, or a hash of the file handle.  We should add an appropriate length
	restriction and can change the name to FileHandle if it is appropriate*/
	FileID           string    `gorm:"primary_key" json:"fileID" binding:"required"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
	ExpiredAt        time.Time `json:"expiredAt"`
	AwsUploadID      *string   `json:"awsUploadID"`
	AwsObjectKey     *string   `json:"awsObjectKey"`
	EndIndex         int       `json:"endIndex" binding:"required,gte=1"`
	CompletedIndexes *string   `json:"completedIndexes" gorm:"type:mediumtext"`
	sync.Mutex
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

/*GetOrCreateFile - Get or create the file. */
func GetOrCreateFile(file File) (*File, error) {
	getFileMutex.Lock()
	defer getFileMutex.Unlock()
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
	// TODO:  QA and see if we need this mutex?
	file.Lock()
	defer file.Unlock()
	completedIndexes := file.GetCompletedIndexesAsMap()
	completedIndexes[*completedPart.PartNumber] = completedPart
	err := file.SaveCompletedIndexesAsString(completedIndexes)

	return err
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

/*SaveCompletedIndexesAsString accepts a map of chunk indexes, converts them to a string,
and saves it to the file's CompletedIndexes.*/
func (file *File) SaveCompletedIndexesAsString(completedIndexes IndexMap) error {
	indexAsBytes, err := json.Marshal(completedIndexes)
	utils.PanicOnError(err)
	indexAsString := string(indexAsBytes)
	if err = DB.Model(&file).Update("completed_indexes", &indexAsString).Error; err != nil {
		return err
	}

	return nil
}

/*UploadCompleted gets the file's CompletedIndexes as a map, then checks that map to verify
that all expected chunk indexes have been added to the map.  If any are not found it returns false.
If none are discovered missing it returns true.  */
func (file *File) UploadCompleted() bool {
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

	completedParts := file.GetCompletedPartsAsArray()

	objectKey := aws.StringValue(file.AwsObjectKey)
	if _, err := utils.CompleteMultiPartUpload(objectKey, aws.StringValue(file.AwsUploadID), completedParts); err != nil {
		return CompletedFile{}, err
	}

	objectSize := utils.GetDefaultBucketObjectSize(objectKey)
	compeletedFile := CompletedFile{
		FileID:         file.FileID,
		ExpiredAt:      file.ExpiredAt,
		FileSizeInByte: objectSize,
	}
	if err := DB.Save(&compeletedFile).Error; err != nil {
		return CompletedFile{}, err
	}
	return compeletedFile, DB.Delete(file).Error
}

/*Return File object(first one) if there is not any error. */
func GetFileById(fileID string) (File, error) {
	file := File{}
	err := DB.Where("file_id = ?", fileID).First(&file).Error
	return file, err
}

/*CompleteUploadsNewerThan will attempt to finish the uploads of files created after the time provided*/
func CompleteUploadsNewerThan(createdAtTime time.Time) error {
	files := []File{}
	err := DB.Where("created_at > ?",
		createdAtTime).Find(&files).Error

	if err != nil {
		return err
	}

	for _, file := range files {
		file.FinishUpload()
	}

	return nil
}

/*DeleteUploadsOlderThan will delete files older than the time provided.  If a file still isn't complete by the
time passed in, the assumption is it error'd or will never be finished. */
func DeleteUploadsOlderThan(createdAtTime time.Time) error {
	files := []File{}
	err := DB.Where("created_at < ?",
		createdAtTime).Find(&files).Error

	if err != nil {
		return err
	}

	for _, file := range files {
		DB.Delete(file)
	}

	return nil
}
