package routes

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httputil"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

// DownloadFileSiaHandler godoc
// @Summary download a Sia file
// @Description download a Sia file
// @Param routes.GenericSiaFileReq body routes.GenericSiaFileReq true "file info object"
// @Accept json
// @Produce */*
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"fileHandle": "a deterministically created file handle",
// @description }
// @Success 200 {string} string ""
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 404 {string} string "such data does not exist"
// @Failure 500 {string} string "some information about the internal error"
// @Router /api/v2/sia/download [post]
/*DownloadFileSiaHandler returns the file location on the storage platform*/
func DownloadFileSiaHandler() gin.HandlerFunc {
	return ginHandlerFunc(downloadFileSia)
}

func downloadFileSia(c *gin.Context) error {
	request := GenericSiaFileReq{}

	if err := verifyAndParseBodyRequest(&request, c); err != nil {
		return err
	}

	account, err := request.getAccount(c)
	if err != nil {
		return err
	}

	if err := verifyAccountPlan(account, utils.Sia, c); err != nil {
		return err
	}

	c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(nil)) // don't care about the body for the rest of the req
	c.Request.ContentLength = 0

	fileID := request.genericFileActionObj.FileHandle
	completedFile, completedErr := models.GetCompletedFileByFileID(fileID)
	if completedErr != nil {
		return NotFoundResponse(c, completedErr)
	}

	if err := verifyPermissions(request.PublicKey, fileID, completedFile.ModifierHash, c); err != nil {
		return err
	}

	director := func(req *http.Request) {
		req.Method = http.MethodGet
		req.Host = utils.GetSiaAddress()
		req.URL.Scheme = "http"
		req.URL.Host = utils.GetSiaAddress()
		req.URL.Path = "/renter/download/" + fileID

		q := req.URL.Query()
		q.Set("httpresp", "true")
		req.URL.RawQuery = q.Encode()

		req.Header.Set("User-Agent", "Sia-Agent")
		req.SetBasicAuth("", utils.Env.SiaApiPassword)
	}

	proxy := &httputil.ReverseProxy{Director: director}
	proxy.ServeHTTP(c.Writer, c.Request)

	return nil
}
