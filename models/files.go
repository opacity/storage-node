package models

import (
	"time"

	"fmt"

	"github.com/jinzhu/gorm"
)

/*File defines a model for managing a user subscription for uploads*/
type File struct {
	/*FileID will either be the file handle, or a hash of the file handle.  We should add an appropriate length
	restriction and can change the name to FileHandle if it is appropriate*/
	FileID string `gorm:"primary_key" json:"fileID" binding:"required"`
	/*AccountID associates an entry in the files table with an entry in the accounts table*/
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	FileSize  int       `json:"fileSize" binding:"required"`
	/*FileStorageLocation is information to find this specific file in the storage*/
	FileStorageLocation string           `json:"fileStorageLocation" binding:"required"`
	UploadStatus        UploadStatusType `json:"uploadStatus" gorm:"default:1" binding:"required"`
	ChunkIndex          int              `json:"chunkIndex" gorm:"default:0"`
}

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
	if file.UploadStatus < FileUploadNotStarted {
		file.UploadStatus = FileUploadNotStarted
	}
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

	fmt.Print("FileSize:                      ")
	fmt.Println(file.FileSize)

	fmt.Print("FileStorageLocation:                      ")
	fmt.Println(file.FileStorageLocation)

	fmt.Print("UploadStatus:                      ")
	fmt.Println(UploadStatusMap[file.UploadStatus])

	fmt.Print("ChunkIndex:                      ")
	fmt.Println(file.ChunkIndex)
}
