package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/utils"
)

type deleteFilesRes struct {
	UnsuccessfulDeletions map[string]string `json:"unsuccessfulDeletions"`
}

type deleteFilesObj struct {
	FileIDs []string `json:"fileIDs" validate:"required" example:"the handle of the files"`
}

type deleteFilesReq struct {
	verification
	requestBody
	deleteFilesObj deleteFilesObj
}

func (v *deleteFilesReq) getObjectRef() interface{} {
	return &v.deleteFilesObj
}

// DeleteFilesHandler godoc
// @Summary deletes files
// @Description deletes files
// @Accept json
// @Produce json
// @Param deleteFilesReq body routes.deleteFilesReq true "file(s) deletion object"
// @description requestBody should be a stringified version of:
// @description {
// @description 	"fileIDs": [],
// @description }
// @Success 200 {object} routes.deleteFilesRes
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 500 {string} string "some information about the internal error"
// @Router /api/v2/delete [post]
/*DeleteFilesHandler is a handler for the user to delete files*/
func DeleteFilesHandler() gin.HandlerFunc {
	return ginHandlerFunc(deleteFiles)
}

func deleteFiles(c *gin.Context) error {
	if !utils.WritesEnabled() {
		return ServiceUnavailableResponse(c, errMaintenance)
	}

	request := deleteFilesReq{}

	if err := verifyAndParseBodyRequest(&request, c); err != nil {
		return err
	}

	account, err := request.getAccount(c)
	if err != nil {
		return err
	}

	deleteFilesRes := deleteFilesRes{
		UnsuccessfulDeletions: make(map[string]string),
	}
	for _, fileID := range request.deleteFilesObj.FileIDs {
		if err := DeleteFileByID(fileID, request.PublicKey, account, c); err != nil {
			deleteFilesRes.UnsuccessfulDeletions[fileID] = err.Error()
		}
	}

	return OkResponse(c, deleteFilesRes)
}
