package routes

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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

func getBlockSize(metadata FileMetadata) int {
	if metadata.P.BlockSize == 0 {
		return blockSize
	}

	return metadata.P.BlockSize
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

	// file, err := utils.GetDefaultBucketObject(models.GetFileDataKey(hash), false)
	// if err != nil {
	// 	return err
	// }
	// decryptedFile, err := PublicShareDecrypt(encryptionKey, file)
	// if err != nil {
	// 	return err
	// }

	fileMetadata, err := utils.GetDefaultBucketObject(models.GetFileMetadataKey(hash), false)
	if err != nil {
		return err
	}
	decryptedMetadata, _ := DecryptMetadata(encryptionKey, fileMetadata)
	fmt.Println(string(decryptedMetadata.Name))

	// ----------------------
	// err = utils.SetDefaultBucketObject(models.GetFileDataPublicKey(fileID), string(decryptedFile))
	// if err != nil {
	// 	return err
	// }

	return nil
}

func PublicShareDecrypt(key []byte, data string) (decryptedData []byte, err error) {
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

func DecryptMetadata(key []byte, data string) (fileMetadata FileMetadata, err error) {
	decryptedByteData, err := PublicShareDecrypt(key, data)
	if err != nil {
		return
	}

	err = json.Unmarshal(decryptedByteData, &fileMetadata)
	if err != nil {
		return
	}

	return
}

// func PublicShareDownloadFile(fileID string, metadata FileMetadata) error {
// 	downloadFileURL, err := GetFileDownloadURL(fileID)
// 	if err != nil {
// 		return err
// 	}
// 	req, err := http.NewRequest(http.MethodGet, downloadFileURL+"/file", nil)
// 	if err != nil {
// 		return err
// 	}
// 	blockSize := getBlockSize(metadata)
// 	blockCount := metadata.P.PartSize / blockSizeOnFS
// 	if (blockCount != int(math.Floor(float64(blockCount)))) {
// 		return errors.New("metadata partSize must be a multiple of blockSize + blockOverhead")
// 	}
// 	headerRange := partInex
// 	// req.Header.Add()

// 	res, err := http.DefaultClient.Do(req)
// 	if err != nil {
// 		return err
// 	}
// 	defer res.Body.Close()

// 	return nil
// }
