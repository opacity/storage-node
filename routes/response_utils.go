package routes

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/opacity/storage-node/utils"
)

const REQUEST_UUID = "request_uuid"

func InternalErrorResponse(c *gin.Context, err error) {
	c.AbortWithStatusJSON(http.StatusInternalServerError, err.Error())
	utils.Metrics_500_Response_Counter.Inc()

	getLogger(c).LogIfError(err, nil)
}

func BadRequestResponse(c *gin.Context, err error) {
	c.AbortWithStatusJSON(http.StatusBadRequest, err.Error())
	utils.Metrics_400_Response_Counter.Inc()
}

func ServiceUnavailableResponse(c *gin.Context, err error) {
	c.AbortWithStatusJSON(http.StatusServiceUnavailable, err.Error())
	utils.Metrics_503_Response_Counter.Inc()
}

func AccountNotFoundResponse(c *gin.Context, id string) {
	NotFoundResponse(c, fmt.Errorf("no account with that id: %s", id))
}

func FileNotFoundResponse(c *gin.Context, fileId string) {
	NotFoundResponse(c, fmt.Errorf("no file with that id: %s", fileId))
}

func ForbiddenResponse(c *gin.Context, err error) {
	c.AbortWithStatusJSON(http.StatusForbidden, err.Error())
	utils.Metrics_403_Response_Counter.Inc()
}

func AccountNotPaidResponse(c *gin.Context, response interface{}) {
	if err := utils.Validator.Struct(response); err != nil {
		err = fmt.Errorf("could not create a valid response:  %v", err)
		BadRequestResponse(c, err)
		return
	}
	c.AbortWithStatusJSON(http.StatusForbidden, response)
	utils.Metrics_403_Response_Counter.Inc()
}

func AccountNotEnoughSpaceResponse(c *gin.Context) {
	BadRequestResponse(c, errors.New("Account does not have enough space to upload more object."))
}

func NotFoundResponse(c *gin.Context, err error) {
	c.AbortWithStatusJSON(http.StatusNotFound, err.Error())
	utils.Metrics_404_Response_Counter.Inc()
}

func OkResponse(c *gin.Context, response interface{}) {
	if err := utils.Validator.Struct(response); err != nil {
		err = fmt.Errorf("could not create a valid response:  %v", err)
		BadRequestResponse(c, err)
		return
	}
	c.JSON(http.StatusOK, response)
	utils.Metrics_200_Response_Counter.Inc()
}

func ginHandlerFunc(f gin.HandlerFunc) gin.HandlerFunc {
	injectToRecoverFromPanic := func(c *gin.Context) {
		setUpSession(c)

		defer func() {
			// Capture the error
			if r := recover(); r != nil {
				utils.SlackLogError(fmt.Sprintf("Recover from err %v", r))

				buff := bytes.NewBufferString("")
				buff.Write(debug.Stack())
				stacks := strings.Split(buff.String(), "\n")

				threadId := stacks[0]
				if len(stacks) > 5 {
					stacks = stacks[5:] // skip the Stack() and Defer method.
				}
				getLogger(c).Error(fmt.Sprintf("[StorageNode]Recover from err %v\nRunning on thread: %s,\nStack: \n%v\n", r, threadId, strings.Join(stacks, "\n")))

				if err, ok := r.(error); ok {
					InternalErrorResponse(c, err)
				} else {
					InternalErrorResponse(c, fmt.Errorf("Unknown error: %v", r))
				}
			}
		}()

		f(c)
	}
	return gin.HandlerFunc(injectToRecoverFromPanic)
}

func getLogger(c *gin.Context) utils.Logger {
	requestUuid := c.Writer.Header().Get(REQUEST_UUID)
	return utils.GetLogger(requestUuid)
}

func setUpSession(c *gin.Context) {
	v := c.GetHeader(REQUEST_UUID)
	if v == "" {
		v = uuid.New().String()
	}
	c.Writer.Header().Set(REQUEST_UUID, v)
}
