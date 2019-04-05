package models

import (
	"time"

	"fmt"

	"encoding/json"

	"sync"

	"github.com/jinzhu/gorm"
	"github.com/opacity/storage-node/utils"
)

var indexMutex = &sync.Mutex{}

/*File defines a model for managing a user subscription for uploads*/
type File struct {
	/*FileID will either be the file handle, or a hash of the file handle.  We should add an appropriate length
	restriction and can change the name to FileHandle if it is appropriate*/
	FileID           string    `gorm:"primary_key" json:"fileID" binding:"required"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
	AwsUploadID      string    `json:"awsUploadID" binding:"required"`
	AwsObjectKey     string    `json:"awsObjectKey" binding:"required"`
	EndIndex         int       `json:"endIndex" binding:"required,gte=0"`
	CompletedIndexes *string   `json:"completedIndexes"`
}

type IndexMap map[int]bool

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

/*UploadStatusMap is for pretty printing the UploadStatus*/
var UploadStatusMap = make(map[UploadStatusType]string)

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

	fmt.Print("AwsUploadID:                      ")
	fmt.Println(file.AwsUploadID)

	fmt.Print("AwsObjectKey:                      ")
	fmt.Println(file.AwsObjectKey)

	fmt.Print("EndIndex:                      ")
	fmt.Println(file.EndIndex)

	fmt.Print("CompletedIndexes:                      ")
	fmt.Println(file.CompletedIndexes)
}

/*UpdateCompletedIndexes - update the completed indexes*/
func (file *File) UpdateCompletedIndexes(index int) error {
	// TODO:  QA and see if we even need this?
	indexMutex.Lock()
	defer indexMutex.Unlock()
	completedIndexes := file.GetCompletedIndexesAsMap()
	completedIndexes[index] = true
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

/*VerifyAllChunksUploaded gets the file's CompletedIndexes as a map, then checks that map to verify
that all expected chunk indexes have been added to the map.  If any are not found it returns false.
If none are discovered missing it returns true.  */
func (file *File) VerifyAllChunksUploaded() bool {
	completedIndexMap := file.GetCompletedIndexesAsMap()

	for index := 0; index <= file.EndIndex; index++ {
		if complete, ok := completedIndexMap[index]; !ok && !complete {
			return false
		}
	}

	return true
}
