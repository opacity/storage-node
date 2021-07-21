package routes

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"strconv"
	"sync"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/disintegration/imaging"
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
	"golang.org/x/sync/errgroup"
)

const (
	NonceByteLength  = 16
	TagByteLength    = 16
	DefaultBlockSize = 64 * 1024
	BlockOverhead    = TagByteLength + NonceByteLength
	DefaultPartSize  = 80 * (DefaultBlockSize + BlockOverhead)
)

func getAcceptedMimeTypesThumbnail() []string {
	return []string{
		"image/jpeg",
		"image/png",
		"image/gif",
		"image/tiff",
		"image/bmp",
	}
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

type FileMetadata struct {
	Size     int    `json:"size"`
	FileName string `json:"name"`
}

type PrivateToPublicObj struct {
	FileHandle string `json:"fileHandle" binding:"required,len=128" minLength:"128" maxLength:"128" example:"a deterministically created file handle"`
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

	fileMetadata, err := utils.GetDefaultBucketObject(models.GetFileMetadataKey(hash), false)
	if err != nil {
		return NotFoundResponse(c, errors.New("the data does not exist"))
	}

	if !utils.DoesDefaultBucketObjectExist(models.GetFileDataKey(hash)) {
		return NotFoundResponse(c, errors.New("the data does not exist"))
	}

	decryptedMetadata, err := DecryptMetadata(encryptionKey, []byte(fileMetadata))
	if err != nil {
		return InternalErrorResponse(c, err)
	}
	realSize, err := getFileContentLength(hash)
	fileSize := decryptedMetadata.Size

	if err != nil {
		return InternalErrorResponse(c, err)
	}
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
	g.Go(func() error {
		return UploadPublicFileAndGenerateThumb(decryptProgress, hash)
	})
	g.Go(func() error {
		return ReadAndDecryptPrivateFile(downloadProgress, decryptProgress)
	})

	g.Go(func() error {
		return DownloadPrivateFile(hash, encryptionKey, numberOfParts, realSize, downloadProgress)
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

func DownloadPrivateFile(fileID string, key []byte, numberOfParts, sizeWithEncryption int, downloadProgress *DownloadProgress) error {
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

func UploadPublicFileAndGenerateThumb(decryptProgress *DecryptProgress, hash string) (err error) {
	awsKey := models.GetFileDataPublicKey(hash)
	uploadID := new(string)
	var completedParts []*s3.CompletedPart
	uploadPartNumber := 1
	uploadPart := make([]byte, 0)

	generateThumbnail, firstRun := true, true
	partForThumbnailBuf := make([]byte, 0)
	fileContentType := ""
	aceptedMimeTypesThumbnail := getAcceptedMimeTypesThumbnail()

	for {
		b := make([]byte, bytes.MinRead)
		n := 0
		n, err = decryptProgress.Read(b)
		if err != nil && err != io.EOF {
			return
		}

		if n != 0 {
			b = b[:n]
			if generateThumbnail && firstRun {
				fileContentType = http.DetectContentType(b)
				if !mimeTypeContains(aceptedMimeTypesThumbnail, fileContentType) {
					generateThumbnail = false
				}
				_, uploadID, err = utils.CreateMultiPartUpload(awsKey, fileContentType)
				if err != nil {
					return
				}
				firstRun = false
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
			completedPart, uploadError := utils.UploadMultiPartPart(awsKey, *uploadID, uploadPart, uploadPartNumber)
			if uploadError != nil {
				utils.AbortMultiPartUpload(awsKey, *uploadID)
			}

			if generateThumbnail {
				partForThumbnailBuf = append(partForThumbnailBuf, uploadPart...)
				generatePublicShareThumbnail(hash, partForThumbnailBuf, fileContentType)
			}
			completedParts = append(completedParts, completedPart)
			break
		}
	}

	if _, err = utils.CompleteMultiPartUpload(awsKey, *uploadID, completedParts); err != nil {
		return
	}

	return utils.SetDefaultObjectCannedAcl(awsKey, utils.CannedAcl_PublicRead)

}

func ReadAndDecryptPrivateFile(downloadProgress *DownloadProgress, decryptProgress *DecryptProgress) error {
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

func generatePublicShareThumbnail(fileID string, imageBytes []byte, fileContentType string) error {
	thumbnailKey := models.GetPublicThumbnailKey(fileID)
	buf := bytes.NewBuffer(imageBytes)
	image, err := imaging.Decode(buf)
	if err != nil {
		return err
	}
	newH := math.Round((float64(image.Bounds().Max.Y) / float64(image.Bounds().Max.X)) * 1024)

	thumbnailImage := imaging.Thumbnail(image, 1024, int(newH), imaging.MitchellNetravali)
	distThumbnailWriter := new(bytes.Buffer)
	if err = imaging.Encode(distThumbnailWriter, thumbnailImage, imaging.JPEG, imaging.JPEGQuality(70)); err != nil {
		return err
	}

	distThumbnailString := distThumbnailWriter.String()
	if err = utils.SetDefaultBucketObject(thumbnailKey, distThumbnailString, fileContentType); err != nil {
		return err
	}

	return utils.SetDefaultObjectCannedAcl(thumbnailKey, utils.CannedAcl_PublicRead)
}

func getFileContentLength(fileID string) (int, error) {
	resp, err := http.Head(models.GetBucketUrl() + fileID + "/file")

	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return int(resp.ContentLength), nil
	}

	return 0, err
}

func mimeTypeContains(mimeTypes []string, mimeType string) bool {
	for _, t := range mimeTypes {
		if t == mimeType {
			return true
		}
	}
	return false
}
