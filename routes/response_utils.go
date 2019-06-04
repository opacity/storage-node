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
	"github.com/opacity/storage-node/models"
)

const REQUEST_UUID = "request_uuid"

type handlerFunc func(*gin.Context) error

func InternalErrorResponse(c *gin.Context, err error) error {
	c.AbortWithStatusJSON(http.StatusInternalServerError, err.Error())
	utils.Metrics_500_Response_Counter.Inc()

	getLogger(c).LogIfError(err, nil)
	return err
}

func BadRequestResponse(c *gin.Context, err error) error {
	c.AbortWithStatusJSON(http.StatusBadRequest, err.Error())
	utils.Metrics_400_Response_Counter.Inc()

	return err
}

func ServiceUnavailableResponse(c *gin.Context, err error) error {
	c.AbortWithStatusJSON(http.StatusServiceUnavailable, err.Error())
	utils.Metrics_503_Response_Counter.Inc()

	return err
}

func AccountNotFoundResponse(c *gin.Context, id string) error {
	return NotFoundResponse(c, fmt.Errorf("no account with that id: %s", id))
}

func FileNotFoundResponse(c *gin.Context, fileId string) error {
	return NotFoundResponse(c, fmt.Errorf("no file with that id: %s", fileId))
}

func ForbiddenResponse(c *gin.Context, err error) error {
	c.AbortWithStatusJSON(http.StatusForbidden, err.Error())
	utils.Metrics_403_Response_Counter.Inc()
	return err
}

func AccountNotPaidResponse(c *gin.Context, response interface{}) error {
	if err := utils.Validator.Struct(response); err != nil {
		err = fmt.Errorf("could not create a valid response:  %v", err)
		return BadRequestResponse(c, err)
	}
	c.AbortWithStatusJSON(http.StatusForbidden, response)
	utils.Metrics_403_Response_Counter.Inc()

	return errors.New("Account not paid and forbidden to access the resource")
}

func AccountNotEnoughSpaceResponse(c *gin.Context) error {
	return BadRequestResponse(c, errors.New("Account does not have enough space to upload more object."))
}

func NotFoundResponse(c *gin.Context, err error) error {
	c.AbortWithStatusJSON(http.StatusNotFound, err.Error())
	utils.Metrics_404_Response_Counter.Inc()
	return err
}

func OkResponse(c *gin.Context, response interface{}) error {
	if err := utils.Validator.Struct(response); err != nil {
		err = fmt.Errorf("could not create a valid response:  %v", err)
		return BadRequestResponse(c, err)
	}
	c.JSON(http.StatusOK, response)
	utils.Metrics_200_Response_Counter.Inc()
	return nil
}

func ginHandlerFunc(f handlerFunc) gin.HandlerFunc {
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

func verifyIfPaid(account models.Account, c *gin.Context) error {
	// Check if paid
	paid, err := account.CheckIfPaid()

	if err == nil && !paid {
		cost, _ := account.Cost()
		response := accountCreateRes{
			Invoice: models.Invoice{
				Cost:       cost,
				EthAddress: account.EthAddress,
			},
			ExpirationDate: account.ExpirationDate(),
		}
		return AccountNotPaidResponse(c, response)
	}

	return nil
}