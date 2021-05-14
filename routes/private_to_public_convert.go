package routes

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

const (
	blockSize     = 64 * 1024
	blockOverhead = 32
	blockSizeOnFS = blockSize + blockOverhead
)

func numberOfBlocks(size float64) float64 {
	return math.Ceil(size / blockSize)
}

func numberOfBlocksOnFS(sizeOnFS float64) float64 {
	return math.Ceil(sizeOnFS / blockSizeOnFS)
}

func sizeOnFS(size float64) float64 {
	return size + blockOverhead*numberOfBlocks(size)
}

func getBlockSize(metadata FileMetadata) float64 {
	if metadata.P.BlockSize == 0 {
		return blockSize
	}

	return float64(metadata.P.BlockSize)
}

func getUploadSize(size int, metadata FileMetadata) float64 {
	size64 := float64(size)
	blockCount := numberOfBlocks(size64)

	return size64 + blockCount*blockOverhead
}

type PrivateToPublicReq struct {
	verification
	requestBody
	privateToPublicObj PrivateToPublicObj
}

type PrivateToPublicObj struct {
	FileHandle string `form:"fileHandle" binding:"required,len=128" minLength:"128" maxLength:"128" example:"a deterministically created file handle"`
}

type FileMetadata struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Size int    `json:"size"`
	P    struct {
		BlockSize int `json:"blockSize"`
		PartSize  int `json:"partSize"`
	} `json:"p"`
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
// @Success 200 {object} routes.shortlinkFileResp
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

	fileMetadata, err := utils.GetDefaultBucketObject(models.GetFileMetadataKey(hash), false)
	if err != nil {
		return err
	}
	decryptedMetadata, _ := DecryptMetadata(encryptionKey, []byte(fileMetadata))

	fmt.Println(decryptedMetadata.P.PartSize)
	fmt.Println(decryptedMetadata.P.BlockSize)

	PublicShareDownloadFile(hash, encryptionKey, decryptedMetadata)
	// ----------------------
	// err = utils.SetDefaultBucketObject(models.GetFileDataPublicKey(fileID), string(decryptedFile))
	// if err != nil {
	// 	return err
	// }

	return nil
}

func PublicShareDecrypt(key []byte, data []byte) (decryptedData []byte, err error) {
	nonceSize := 16

	block, err := aes.NewCipher(key)
	if err != nil {
		return
	}

	aesgcm, err := cipher.NewGCMWithNonceSize(block, nonceSize)
	if err != nil {
		return
	}

	byteData := []byte(data)
	splitter := len(byteData) - nonceSize
	nonce := byteData[splitter:]
	cipherText := byteData[:splitter]
	if err != nil {
		return
	}

	decryptedData, err = aesgcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return
	}

	return
}

func DecryptMetadata(key []byte, data []byte) (fileMetadata FileMetadata, err error) {
	decryptedByteData, err := PublicShareDecrypt(key, data)
	if err != nil {
		return
	}
	fmt.Println(string(decryptedByteData))
	err = json.Unmarshal(decryptedByteData, &fileMetadata)
	if err != nil {
		return
	}

	return
}

func PublicShareDownloadFile(hash string, key []byte, metadata FileMetadata) error {
	file, err := utils.GetBucketObject(models.GetFileDataKey(hash), false)

	blockSize := getBlockSize(metadata)
	partSize := 80 * (blockSize + blockOverhead)
	blockCount := partSize / (blockSize + blockOverhead)
	if blockCount != math.Floor(blockCount) {
		return fmt.Errorf("partSize must be a multiple of blockSize + blockOverhead")
	}

	decryptedFile := *new([]byte)
	if err != nil {
		return err
	}
	defer file.Body.Close()

	sizeBytes := 65568

	nBytes, nChunks := int64(0), int64(0)
	buf := make([]byte, 0, sizeBytes)

	for {
		chunk, err := file.Body.Read(buf[:cap(buf)])
		buf = buf[:chunk]
		if chunk == 0 {
			if err == nil {
				continue
			}
			if err == io.EOF {
				break
			}
			log.Fatal(err)
		}
		nChunks++
		nBytes += int64(len(buf))
		if err != nil && err != io.EOF {
			log.Fatal(err)
		}
		decryptedChunk, err := PublicShareDecrypt(key, buf)
		decryptedFile = append(decryptedFile, decryptedChunk...)
		if err != nil {
			log.Fatal(err)
		}
	}
	log.Println("Bytes:", nBytes, "Chunks:", nChunks)

	// upload public file back
	utils.SetDefaultBucketObject(models.GetFileDataPublicKey(hash), string(decryptedFile))

	return nil
}
