package routes

import (
	"bytes"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/utils"
)

func InternalErrorResponse(c *gin.Context, err error) {
	c.AbortWithStatusJSON(http.StatusInternalServerError, err.Error())
	utils.Metrics_500_Response_Counter.Inc()
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
	c.AbortWithStatusJSON(http.StatusNotFound, fmt.Sprintf("no account with that id: %s", id))
	utils.Metrics_404_Response_Counter.Inc()
}

func ForbiddenResponse(c *gin.Context, err error) {
	c.AbortWithStatusJSON(http.StatusForbidden, err.Error())
	utils.Metrics_403_Response_Counter.Inc()
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
				fmt.Printf("[StorageNode]Recover from err %v\nRunning on thread: %s,\nStack: \n%v\n", r, threadId, strings.Join(stacks, "\n"))

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
	return nil
}
