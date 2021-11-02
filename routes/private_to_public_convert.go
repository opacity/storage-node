package routes

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"sync"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gabriel-vasile/mimetype"
	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
	"golang.org/x/sync/errgroup"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

const (
	NonceByteLength  = 16
	TagByteLength    = 16
	DefaultBlockSize = 64 * 1024
	BlockOverhead    = TagByteLength + NonceByteLength
	DefaultPartSize  = 80 * (DefaultBlockSize + BlockOverhead)
)

type FileMetadata struct {
	Size     int    `json:"size"`
	FileName string `json:"name"`
}

type DownloadProgress struct {
	RawProgress        int
	SizeWithEncryption int
	PartSize           int
	ActivePart         int
	Parts              [][]byte
	ReadIndex          []int
	ReadPartIndex      int
	ReadTotalIndex     int
	mux                sync.Mutex
}

func (dl *DownloadProgress) Write(part []byte) (int, error) {
	dl.mux.Lock()
	defer dl.mux.Unlock()

	length := len(part)
	dl.RawProgress += length

	if dl.ActivePart >= cap(dl.Parts) {
		dl.Parts = append(dl.Parts, []byte{})
	}

	dl.Parts[dl.ActivePart] = append(dl.Parts[dl.ActivePart], part...)

	return length, nil
}

func (dl *DownloadProgress) Read(part []byte) (int, error) {
	dl.mux.Lock()
	defer dl.mux.Unlock()

	if dl.ReadPartIndex >= cap(dl.ReadIndex) {
		dl.ReadIndex = append(dl.ReadIndex, make([]int, dl.ReadPartIndex-cap(dl.ReadIndex)+1)...)
	}

	if dl.ActivePart >= cap(dl.Parts) {
		dl.Parts = append(dl.Parts, []byte{})
	}

	if dl.RawProgress == dl.SizeWithEncryption && dl.RawProgress == dl.ReadTotalIndex {
		return 0, io.EOF
	}

	lenToRead := len(part)
	if lenToRead > len(dl.Parts[dl.ActivePart]) {
		lenToRead = len(dl.Parts[dl.ActivePart])
	}

	for i := 0; i < lenToRead; i++ {
		part[i] = dl.Parts[dl.ActivePart][i]
	}

	dl.Parts[dl.ActivePart] = dl.Parts[dl.ActivePart][lenToRead:]
	dl.ReadIndex[dl.ReadPartIndex] += lenToRead
	dl.ReadTotalIndex += lenToRead

	if dl.ReadIndex[dl.ReadPartIndex] == dl.PartSize {
		dl.ReadPartIndex++

		if dl.ReadPartIndex >= cap(dl.ReadIndex) {
			dl.ReadIndex = append(dl.ReadIndex, make([]int, dl.ReadPartIndex-cap(dl.ReadIndex)+1)...)
		}
		dl.ReadIndex = dl.ReadIndex[:dl.ReadPartIndex+1]

		dl.ReadIndex[dl.ReadPartIndex] = 0
	}

	return lenToRead, nil
}

type DecryptProgress struct {
	Key                []byte
	RawProgress        int
	SizeWithEncryption int
	FileSize           int
	ChunkSize          int
	Part               []byte
	Data               []byte
	ReadIndex          int

	mux sync.Mutex
}

func (dc *DecryptProgress) Write(part []byte) (int, error) {
	dc.mux.Lock()
	defer dc.mux.Unlock()

	length := len(part)
	dc.RawProgress += length

	dc.Part = append(dc.Part, part...)

	for {
		if dc.RawProgress == dc.SizeWithEncryption && len(dc.Part) == 0 {
			return length, io.EOF
		}

		if dc.RawProgress != dc.SizeWithEncryption && len(dc.Part) < dc.ChunkSize {
			return length, nil
		}

		if len(dc.Part) < dc.ChunkSize {
			dc.ChunkSize = len(dc.Part)
		}

		chunk := dc.Part[:dc.ChunkSize]
		dc.mux.Unlock()
		data, err := DecryptWithNonceSize(dc.Key, chunk)
		if err != nil {
			return -1, err
		}
		dc.mux.Lock()

		dc.Part = dc.Part[dc.ChunkSize:]
		dc.Data = append(dc.Data, data...)
	}
}

func (dc *DecryptProgress) Read(part []byte) (int, error) {
	dc.mux.Lock()
	defer dc.mux.Unlock()

	if dc.ReadIndex == dc.FileSize {
		return 0, io.EOF
	}

	lenToRead := len(part)
	if lenToRead > len(dc.Data) {
		lenToRead = len(dc.Data)
	}

	for i := 0; i < lenToRead; i++ {
		part[i] = dc.Data[i]
	}

	dc.Data = dc.Data[lenToRead:]
	dc.ReadIndex += lenToRead

	return lenToRead, nil
}

type PrivateToPublicObj struct {
	FileHandle string `json:"fileHandle" binding:"required,len=128" minLength:"128" maxLength:"128" example:"a deterministically created file handle"`
	FileSize   int    `json:"fileSize" binding:"required" example:"543534"`
}

func (v *PrivateToPublicReq) getObjectRef() interface{} {
	return &v.privateToPublicObj
}

// PrivateToPublicConvertHandler godoc
// @Summary convert private file to a public shared one
// @Description convert private file to a public shared one
// @Accept json
// @Produce json
// @Param PrivateToPublicReq body routes.PrivateToPublicReq true "an object to do the conversion of a private file to a public one"
// @description requestBody should be a stringified version of:
// @description {
// @description 	"fileHandle": "a deterministically created file handle",
// @description 	"fileSize": 543534,
// @description }
// @Success 200 {object} routes.StatusRes
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 403 {string} string "signature did not match"
// @Failure 404 {string} string "the data does not exist"
// @Router /api/v2/public-share/convert [post]
/*PrivateToPublicConvertHandler is a handler for the user to convert an existing private file to a public share on*/
func PrivateToPublicConvertHandler() gin.HandlerFunc {
	return ginHandlerFunc(privateToPublicConvertWithContext)
}

func privateToPublicConvertWithContext(c *gin.Context) error {
	request := PrivateToPublicReq{}

	if err := verifyAndParseBodyRequest(&request, c); err != nil {
		return err
	}

	hash := request.privateToPublicObj.FileHandle[:64]
	key := request.privateToPublicObj.FileHandle[64:]
	encryptionKey, _ := hex.DecodeString(key)

	if !utils.DoesDefaultBucketObjectExist(models.GetFileDataKey(hash)) {
		return NotFoundResponse(c, errors.New("the data does not exist"))
	}

	realSize, err := getFileContentLength(hash)
	if err != nil || realSize == 0 {
		return InternalErrorResponse(c, err)
	}

	fileSize := request.privateToPublicObj.FileSize
	numberOfParts := ((fileSize-1)/DefaultPartSize + 1)

	downloadProgress := &DownloadProgress{
		SizeWithEncryption: realSize,
		PartSize:           DefaultPartSize,
	}

	decryptProgress := &DecryptProgress{
		Key:                encryptionKey,
		FileSize:           fileSize,
		SizeWithEncryption: realSize,
		ChunkSize:          DefaultBlockSize + BlockOverhead,
	}

	var g errgroup.Group
	sentryMainSpan := sentry.TransactionFromContext(c.Request.Context())

	g.Go(func() error {
		return UploadPublicFileAndGenerateThumb(decryptProgress, hash, sentryMainSpan)
	})
	g.Go(func() error {
		return ReadAndDecryptPrivateFile(downloadProgress, decryptProgress, sentryMainSpan)
	})

	g.Go(func() error {
		return DownloadPrivateFile(hash, numberOfParts, realSize, downloadProgress, sentryMainSpan)
	})

	if err := g.Wait(); err != nil {
		return InternalErrorResponse(c, err)
	}

	return OkResponse(c, StatusRes{
		Status: "private file converted to public",
	})
}

func DecryptWithNonceSize(key []byte, encryptedData []byte) (decryptedData []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return
	}

	aesgcm, err := cipher.NewGCMWithNonceSize(block, NonceByteLength)
	if err != nil {
		return
	}

	encryptedByteData := []byte(encryptedData)
	rawData := encryptedByteData[0 : len(encryptedByteData)-BlockOverhead]
	tag := encryptedByteData[len(encryptedByteData)-BlockOverhead : len(encryptedByteData)-BlockOverhead+NonceByteLength]
	nonce := encryptedByteData[len(encryptedByteData)-BlockOverhead+TagByteLength:]

	cipherText := append(rawData, tag...)
	decryptedData, err = aesgcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return
	}

	return
}

func DownloadPrivateFile(fileID string, numberOfParts, sizeWithEncryption int, downloadProgress *DownloadProgress, sentryMainSpan *sentry.Span) error {
	sentrySpanDownload := sentryMainSpan.StartChild("download")
	defer sentrySpanDownload.Finish()
	for i := 0; i < numberOfParts; i++ {
		offset := i * DefaultPartSize
		limit := offset + DefaultPartSize

		if limit > sizeWithEncryption {
			limit = sizeWithEncryption
		}

		downloadRange := "bytes=" + strconv.Itoa(offset) + "-" + strconv.Itoa(limit-1)
		fileChunkObjOutput, err := utils.GetBucketObject(models.GetFileDataKey(fileID), downloadRange, false)
		if err != nil {
			return err
		}

		for {
			b := make([]byte, bytes.MinRead)
			fileChunk, err := fileChunkObjOutput.Body.Read(b)

			if err != nil && err != io.EOF {
				return err
			}

			if fileChunk != 0 {
				b = b[:fileChunk]
				_, err := downloadProgress.Write(b)
				if err != nil && err != io.EOF {
					return err
				}
			}

			if err == io.EOF {
				break
			}
		}
	}

	return nil
}

func UploadPublicFileAndGenerateThumb(decryptProgress *DecryptProgress, hash string, sentryMainSpan *sentry.Span) (err error) {
	awsKey := models.GetFileDataPublicKey(hash)
	uploadID := new(string)
	var completedParts []*s3.CompletedPart
	uploadPartNumber := 1
	uploadPart := make([]byte, 0)
	sentrySpanUpload := sentryMainSpan.StartChild("upload")
	sentrySpanUpload.SetTag("upload-file-size", strconv.Itoa(decryptProgress.FileSize))

	generateThumbnail, firstRun := false, true
	partForThumbnailBuf := make([]byte, 0)
	fileContentType := ""

	for {
		b := make([]byte, bytes.MinRead)
		n := 0
		n, err = decryptProgress.Read(b)
		if err != nil && err != io.EOF {
			return
		}

		if n != 0 {
			b = b[:n]
			if firstRun {
				fileContentType = mimetype.Detect(b).String()
				if ct, _ := SplitMime(fileContentType); ct == "image" || ct == "video" {
					generateThumbnail = true
				}
				firstRun = false
				_, uploadID, err = utils.CreateMultiPartUpload(awsKey, fileContentType)
				if err != nil {
					return
				}
			}

			uploadPart = append(uploadPart, b...)

			if int64(len(uploadPart)) == utils.MinMultiPartSize {
				completedPart, uploadError := utils.UploadMultiPartPart(awsKey, *uploadID, uploadPart, uploadPartNumber)
				if uploadError != nil {
					utils.AbortMultiPartUpload(awsKey, *uploadID)
				}
				completedParts = append(completedParts, completedPart)

				if generateThumbnail {
					partForThumbnailBuf = append(partForThumbnailBuf, uploadPart...)
				}

				uploadPartNumber++
				uploadPart = make([]byte, 0)
			}
		}

		if err == io.EOF {
			completedPart, err := utils.UploadMultiPartPart(awsKey, *uploadID, uploadPart, uploadPartNumber)
			if err != nil {
				utils.AbortMultiPartUpload(awsKey, *uploadID)
				return err
			}

			if generateThumbnail {
				partForThumbnailBuf = append(partForThumbnailBuf, uploadPart...)
				generatePublicShareThumbnail(hash, partForThumbnailBuf, fileContentType, sentrySpanUpload)
			}

			sentrySpanUpload.Finish()
			completedParts = append(completedParts, completedPart)
			break
		}
	}

	if _, err = utils.CompleteMultiPartUpload(awsKey, *uploadID, completedParts); err != nil {
		return
	}

	return utils.SetDefaultObjectCannedAcl(awsKey, utils.CannedAcl_PublicRead)

}

func ReadAndDecryptPrivateFile(downloadProgress *DownloadProgress, decryptProgress *DecryptProgress, sentryMainSpan *sentry.Span) error {
	sentrySpanDecrypt := sentryMainSpan.StartChild("decryption")
	sentrySpanDecrypt.SetTag("encrypted-file-size", strconv.Itoa(decryptProgress.SizeWithEncryption))
	defer sentrySpanDecrypt.Finish()
	for {
		b := make([]byte, bytes.MinRead)

		n, err := downloadProgress.Read(b)
		if err != nil && err != io.EOF {
			return err
		}

		if n != 0 {
			b = b[:n]
			_, err := decryptProgress.Write(b)
			if err != nil && err != io.EOF {
				return err
			}
		}

		if err == io.EOF {
			break
		}
	}
	return nil
}

func generatePublicShareThumbnail(fileID string, fileBytes []byte, fileContentType string, sentryMainSpan *sentry.Span) error {
	sentrySpanGenerateThumbnail := sentryMainSpan.StartChild("thumbnail-generation")
	sentrySpanGenerateThumbnail.SetTag("content-type", fileContentType)
	defer sentrySpanGenerateThumbnail.Finish()

	thumbnailKey := models.GetPublicThumbnailKey(fileID)
	ct, _ := SplitMime(fileContentType)
	ffprobeVideoDurationCmd := exec.Command("ffprobe",
		"-f", "image2pipe",
		"-v", "quiet",
		"-show_format",
		"-show_streams",
		"-of", "json",
		"-",
	)

	if ct == "video" {
		ffprobeVideoDurationCmd.Args = RemoveIndexFromSliceString(ffprobeVideoDurationCmd.Args, 1)
		ffprobeVideoDurationCmd.Args = RemoveIndexFromSliceString(ffprobeVideoDurationCmd.Args, 1)
	}

	ffprobeVideoDurationCmd.Stdin = bytes.NewBuffer(fileBytes)
	videoDurationOutput, _ := ffprobeVideoDurationCmd.Output()
	type inputProbeInfo struct {
		Streams []struct {
			CodecType string `json:"codec_type"`
			Width     int    `json:"width"`
			Duration  string `json:"duration"`
		} `json:"streams"`
	}
	iProbeInfo := &inputProbeInfo{}
	json.Unmarshal(videoDurationOutput, iProbeInfo)
	videoDurationString := ""
	videoWidth := 0

	for _, s := range iProbeInfo.Streams {
		if s.CodecType == "video" {
			videoDurationString = s.Duration
			videoWidth = s.Width
		}
	}

	videoDurationFloat32, _ := strconv.ParseFloat(videoDurationString, 32)

	buf := bytes.NewBuffer(nil)

	ffmpegOutputArgs := ffmpeg.KwArgs{
		"frames:v": 1,
		"q:v":      2,
		"f":        "image2pipe",
		"ss":       fmt.Sprintf("%.1f", videoDurationFloat32/5),
	}
	if videoWidth >= 1024 {
		ffmpegOutputArgs["filter:v"] = "scale='1024:-1'" // don't upscale
	}

	ffmpegInputArgs := ffmpeg.KwArgs{
		"loglevel": "error",
	}

	if ct == "image" {
		ffmpegInputArgs["f"] = "image2pipe"
	}

	err := ffmpeg.Input("pipe:0", ffmpegInputArgs).
		Output("pipe:1", ffmpegOutputArgs).
		OverWriteOutput().
		WithInput(bytes.NewBuffer(fileBytes)).
		WithOutput(buf, os.Stdout).
		Run()

	if err != nil {
		return err
	}

	if buf.Len() != 0 {
		// thumbnail is always image/jpeg
		if err := utils.SetDefaultBucketObject(thumbnailKey, buf.String(), "image/jpeg"); err != nil {
			return err
		}
		return utils.SetDefaultObjectCannedAcl(thumbnailKey, utils.CannedAcl_PublicRead)
	}

	return nil
}

func RemoveIndexFromSliceString(s []string, index int) []string {
	return append(s[:index], s[index+1:]...)
}

func getFileContentLength(fileID string) (int, error) {
	resp, err := http.Head(utils.GetS3BucketUrl() + fileID + "/file")

	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return int(resp.ContentLength), nil
	}

	return 0, err
}

func DecryptMetadata(key []byte, data []byte) (fileMetadata FileMetadata, err error) {
	decryptedByteData, err := DecryptWithNonceSize(key, data)
	if err != nil {
		return
	}

	err = json.Unmarshal(decryptedByteData, &fileMetadata)
	if err != nil {
		return
	}

	return
}
