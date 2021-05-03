package main

import (
	"fmt"
	"testing"

	"net/http"

	"os"

	"time"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/routes"
	"github.com/opacity/storage-node/utils"
	"github.com/stretchr/testify/assert"
)

const maxAllowedSecondsPerUpload = 10

func deleteEverything(t *testing.T) {
	// TODO:  Have all the s3 uploads go to one directory on s3 so we can make sure we delete all of them
	models.DeleteFilesForTest()
	models.DeleteAccountsForTest()
	models.DeleteCompletedFilesForTest()
}

func init() {
	utils.SetTesting(".env")
	models.Connect(utils.Env.TestDatabaseURL)
	gin.SetMode(gin.TestMode)
}

func ReturnChunkDataForTestBigFile(t *testing.T) [][]byte {
	workingDir, _ := os.Getwd()
	localFilePath := workingDir + string(os.PathSeparator) + "test_files" + string(os.PathSeparator) + "loremBig.txt"

	file, err := os.Open(localFilePath)
	assert.Nil(t, err)
	defer file.Close()
	fileInfo, _ := file.Stat()
	size := fileInfo.Size()
	buffer := make([]byte, size)
	file.Read(buffer)

	arrayOfBuffers := make([][]byte, 0)
	var curr, partLength int64
	var remaining = size
	for curr = 0; remaining != 0; curr += partLength {
		if remaining < utils.MinMultiPartSize {
			partLength = remaining
		} else {
			partLength = utils.MinMultiPartSize
		}
		arrayOfBuffers = append(arrayOfBuffers, buffer[curr:curr+partLength])
		remaining -= partLength
	}

	return arrayOfBuffers
}

func Test_Performance_Testing_10(t *testing.T) {
	t.Skip()
	numUploadsToDo := 10
	logPerformanceResults(performanceTest(numUploadsToDo, t))
}

func Test_Performance_Testing_100(t *testing.T) {
	t.Skip()
	numUploadsToDo := 100
	logPerformanceResults(performanceTest(numUploadsToDo, t))
}

func Test_Performance_Testing_1000(t *testing.T) {
	t.Skip()
	numUploadsToDo := 1000
	logPerformanceResults(performanceTest(numUploadsToDo, t))
}

func performanceTest(numUploadsToDo int, t *testing.T) (numUploadsAttempted int,
	startTime time.Time, abortTime time.Time, finishTime time.Time, success bool) {
	deleteEverything(t)

	numUploadsAttempted = numUploadsToDo
	success = false
	startTime = time.Now()
	abortTime = getAbortTime(numUploadsToDo)
	for i := 0; i < numUploadsToDo; i++ {
		go func() {
			uploadBody := routes.ReturnValidUploadFileBodyForTest(t)
			uploadBody.PartIndex = models.FirstChunkIndex

			privateKey, err := utils.GenerateKey()
			assert.Nil(t, err)

			// create an array of arrays of bytes
			arrayOfChunkDataBuffers := ReturnChunkDataForTestBigFile(t)

			// start upload of first chunk
			// set PartIndex to 1
			uploadBody.PartIndex = 1

			// remove first byte array from array of byte arrays
			arrayOfChunkDataBuffers = arrayOfChunkDataBuffers[1:]

			// create a valid request
			request := routes.ReturnValidUploadFileReqForTest(t, uploadBody, privateKey)
			// first the first byte array
			request.ChunkData = string(arrayOfChunkDataBuffers[0])

			// get the file handle
			fileHandle := uploadBody.FileHandle

			// create a paid account
			accountID, err := utils.HashString(request.PublicKey)
			assert.Nil(t, err)
			routes.CreatePaidAccountForTest(t, accountID)

			// init upload
			routes.InitUploadFileForTest(t, request.PublicKey, uploadBody.FileHandle, len(arrayOfChunkDataBuffers))

			// perform the first request and verify the expected status
			w := routes.UploadFileHelperForTest(t, request, routes.UploadPath, "v1")
			assert.Equal(t, http.StatusOK, w.Code)

			// perform the requests for subsequent chunks
			for index, buffer := range arrayOfChunkDataBuffers {
				//TODO:  having issues when wrapping this in a go-routine.  Figure out why and uncomment out go-routine.
				//go func() {
				uploadBody.PartIndex = index + 2
				request := routes.ReturnValidUploadFileReqForTest(t, uploadBody, privateKey)
				request.ChunkData = string(buffer)
				w := routes.UploadFileHelperForTest(t, request, routes.UploadPath, "v1")
				assert.Equal(t, http.StatusOK, w.Code)
				//}()
			}

			err = utils.DeleteDefaultBucketObject(fileHandle)
			assert.Nil(t, err)
		}()
	}

	for {
		filesInDB := []models.File{}
		models.DB.Find(&filesInDB)

		completedFilesInDB := []models.CompletedFile{}
		models.DB.Find(&completedFilesInDB)
		if len(filesInDB) == 0 && len(completedFilesInDB) == numUploadsToDo {
			success = true
			finishTime = time.Now()
			break
		}
		if time.Now().After(abortTime) {
			deleteEverything(t)
			t.Fatalf("Could not complete %d uploads within the allotted time\n", numUploadsToDo)
		}
	}
	deleteEverything(t)
	return numUploadsAttempted, startTime, abortTime, finishTime, success
}

func getAbortTime(numUploads int) time.Time {
	return time.Now().Add(time.Second * time.Duration(numUploads) * maxAllowedSecondsPerUpload)
}

func logPerformanceResults(numUploadsAttempted int, startTime time.Time, abortTime time.Time, finishTime time.Time, success bool) {
	utils.SlackLog(fmt.Sprintf("      ~~~~PERFORMANCE TESTING RESULTS FOR %d UPLOADS~~~~\n", numUploadsAttempted))
	if success {
		utils.SlackLog(fmt.Sprintf("SUCCESSFULLY completed %d uploads\n", numUploadsAttempted))
		utils.SlackLog(fmt.Sprintf("start time was: %s\n", startTime.Format("3:04:05.000")))
		utils.SlackLog(fmt.Sprintf("finish time was: %s\n", finishTime.Format("3:04:05.000")))
		utils.SlackLog(fmt.Sprintf("abort time would have been: %s\n", abortTime.Format("3:04:05.000")))
	} else {
		utils.SlackLogError(fmt.Sprintf("FAILED to complete %d uploads\n", numUploadsAttempted))
		utils.SlackLogError(fmt.Sprintf("start time was: %s\n", startTime.Format("3:04:05.000")))
		utils.SlackLogError(fmt.Sprintf("abort time was: %s\n", abortTime.Format("3:04:05.000")))
	}
	utils.SlackLog("________________________________________________")
}
