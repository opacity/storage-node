package routes

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"

	"time"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
)

type GenericRequest struct {
	requestBody
	verification
}

const noAccountWithThatID = "no account with that id"

const REQUEST_UUID = "request_uuid"

type handlerFunc func(*gin.Context) error

func sentryCaptureException(c *gin.Context, err error) {
	if hub := sentrygin.GetHubFromContext(c); hub != nil {
		hub.WithScope(func(scope *sentry.Scope) {
			hub.CaptureException(err)
		})
	}
}

func InternalErrorResponse(c *gin.Context, err error) error {
	c.AbortWithStatusJSON(http.StatusInternalServerError, err.Error())
	utils.Metrics_500_Response_Counter.Inc()

	sentryCaptureException(c, err)

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

	sentryCaptureException(c, err)

	return err
}

func AccountNotFoundResponse(c *gin.Context, id string) error {
	return NotFoundResponse(c, fmt.Errorf(noAccountWithThatID+": %s", id))
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

func AccountExpiredResponse(c *gin.Context, response interface{}) error {
	if err := utils.Validator.Struct(response); err != nil {
		err = fmt.Errorf("could not create a valid response:  %v", err)
		return BadRequestResponse(c, err)
	}
	c.AbortWithStatusJSON(http.StatusForbidden, response)
	utils.Metrics_403_Response_Counter.Inc()

	return errors.New("Account expired and forbidden to access the resource, must renew account.")
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
		sentryOptions := sentry.ContinueFromRequest(c.Request)

		span := sentry.StartSpan(c.Request.Context(), c.Request.Method,
			sentry.TransactionName(c.Request.URL.String()),
			sentryOptions,
		)
		span.Status = sentry.SpanStatusOK

		setUpSession(c)
		c.Request = c.Request.Clone(span.Context())
		sentry.ContinueFromRequest(c.Request)

		defer func() {
			// Capture the error
			if r := recover(); r != nil {
				utils.SlackLogError(fmt.Sprintf("recover from err %v", r))

				buff := bytes.NewBufferString("")
				buff.Write(debug.Stack())
				stacks := strings.Split(buff.String(), "\n")

				threadId := stacks[0]
				if len(stacks) > 5 {
					stacks = stacks[5:] // skip the Stack() and Defer method.
				}
				getLogger(c).Error(fmt.Sprintf("[StorageNode]recover from err %v\nRunning on thread: %s,\nStack: \n%v\n", r, threadId, strings.Join(stacks, "\n")))

				if err, ok := r.(error); ok {
					InternalErrorResponse(c, err)
				} else {
					InternalErrorResponse(c, fmt.Errorf("unknown error: %v", r))
				}
			}
		}()

		f(c) // we don't care about this returned error as it can be a response body
		defer span.Finish()
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

func verifyIfPaid(account models.Account) bool {
	// Check if paid
	paid := false
	for networkID := range services.EthWrappers {
		paid, _ = account.CheckIfPaid(networkID)
	}
	paidWithCreditCard, _ := models.CheckForPaidStripePayment(account.AccountID)

	return paid || paidWithCreditCard
}

func verifyAccountStillActive(account models.Account) bool {
	return account.ExpirationDate().After(time.Now()) || utils.Env.Plans[int(account.StorageLimit)].Name == "Free"
}

func verifyIfPaidWithContext(account models.Account, c *gin.Context) error {
	paid := verifyIfPaid(account)

	cost, _ := account.Cost()
	response := accountCreateRes{
		Invoice: models.Invoice{
			Cost:       cost,
			EthAddress: account.EthAddress,
		},
		ExpirationDate: account.ExpirationDate(),
	}

	if !paid {
		return AccountNotPaidResponse(c, response)
	}

	if !verifyAccountStillActive(account) {
		return AccountExpiredResponse(c, response)
	}

	return nil
}

func verifyValidStorageLimit(storageLimit int, c *gin.Context) error {
	_, ok := utils.Env.Plans[storageLimit]
	if !ok {
		return BadRequestResponse(c, models.InvalidStorageLimitError)
	}
	return nil
}

func verifyUpgradeEligible(account models.Account, newStorageLimit int, c *gin.Context) error {
	err := verifyValidStorageLimit(newStorageLimit, c)
	if err != nil {
		return err
	}
	if newStorageLimit <= int(account.StorageLimit) {
		return BadRequestResponse(c, errors.New("cannot upgrade to storage limit lower than current limit"))
	}
	return nil
}

func verifyRenewEligible(account models.Account, c *gin.Context) error {
	renewalCutoffTimestamp := time.Now().Add(time.Hour * 24 * 180)
	if account.ExpirationDate().After(renewalCutoffTimestamp) {
		return ForbiddenResponse(c, errors.New("account has too much time left to renew"))
	}
	return nil
}
