package routes

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
	"github.com/stripe/stripe-go"
)

const (
	stripeRetryCount        = 3
	retrySleepIntervalInSec = 1
)

type createStripePaymentObject struct {
	StripeToken string `json:"stripeToken" binding:"required" example:"tok_KPte7942xySKBKyrBu11yEpf"`
	Timestamp   int64  `json:"timestamp" binding:"required"`
}

type createStripePaymentReq struct {
	verification
	requestBody
	createStripePaymentObject createStripePaymentObject
}

type stripeDataRes struct {
	stripeDataObj
	StatusRes
}

type stripeDataObj struct {
	StripePaymentExists bool    `json:"stripePaymentExists"`
	ChargePaid          bool    `json:"chargePaid"`
	StripeToken         string  `json:"stripeToken"`
	OpqTxStatus         string  `json:"opqTxStatus"`
	ChargeID            string  `json:"chargeID"`
	Amount              float64 `json:"amount" binding:"omitempty,gte=0"`
}

func (v *createStripePaymentReq) getObjectRef() interface{} {
	return &v.createStripePaymentObject
}

// CreateStripePaymentHandler godoc
// @Summary create a stripe payment
// @Description create a stripe payment
// @Accept  json
// @Produce  json
// @Param createStripePaymentReq body routes.createStripePaymentReq true "stripe payment creation object"
// @description requestBody should be a stringified version of (values are just examples):
// @description {
// @description 	"stripeToken": "tok_KPte7942xySKBKyrBu11yEpf",
// @description }
// @Success 200 {object} routes.stripeDataRes
// @Failure 400 {string} string "bad request, unable to parse request body: (with the error)"
// @Failure 404 {string} string "no account with that id: (with your accountID)"
// @Failure 403 {string} string "account is already paid for"
// @Failure 500 {string} string "some information about the internal error"
// @Router /api/v1/stripe/create [post]
/*CreateStripePaymentHandler is a handler for post requests to create stripe payments*/
func CreateStripePaymentHandler() gin.HandlerFunc {
	return ginHandlerFunc(createStripePayment)
}

func createStripePayment(c *gin.Context) error {

	if !utils.Env.EnableCreditCards && !utils.IsTestEnv() {
		return ForbiddenResponse(c, errors.New("not accepting credit cards yet"))
	}

	request := createStripePaymentReq{}

	if err := verifyAndParseBodyRequest(&request, c); err != nil {
		return err
	}

	account, err := request.getAccount(c)
	if err != nil {
		return err
	}

	if account.PaymentStatus > models.InitialPaymentInProgress && !utils.FreeModeEnabled() {
		return ForbiddenResponse(c, errors.New("account is already paid for"))
	}

	costInDollars := utils.Env.Plans[int(account.StorageLimit)].CostInUSD

	var charge *stripe.Charge
	for i := 0; i < stripeRetryCount; i++ {
		charge, err = services.CreateCharge(costInDollars, request.createStripePaymentObject.StripeToken, account.AccountID)
		if !waitOnRetryableStripeError(err) {
			break
		}
	}

	if err != nil {
		return handleStripeError(err, c)
	}

	stripePayment := models.StripePayment{
		StripeToken: request.createStripePaymentObject.StripeToken,
		AccountID:   account.AccountID,
		ChargeID:    charge.ID,
	}

	// Add stripe payment to DB
	if err := models.DB.Create(&stripePayment).Error; err != nil {
		return BadRequestResponse(c, err)
	}

	if err := stripePayment.SendAccountOPQ(); err != nil {
		return InternalErrorResponse(c, err)
	}

	amount, err := checkChargeAmount(c, stripePayment.ChargeID)
	if err != nil {
		return err
	}

	return OkResponse(c, stripeDataRes{
		StatusRes: StatusRes{
			Status: "successfully charged card, now sending OPQ to payment address",
		},
		stripeDataObj: stripeDataObj{
			StripePaymentExists: true,
			ChargePaid:          charge.Paid,
			StripeToken:         stripePayment.StripeToken,
			OpqTxStatus:         models.OpqTxStatusMap[stripePayment.OpqTxStatus],
			ChargeID:            charge.ID,
			Amount:              amount,
		},
	})

	return OkResponse(c, StatusRes{
		Status: "successfully charged card, now sending OPQ to payment address",
	})
}

func checkChargePaid(c *gin.Context, stripePayment models.StripePayment) (bool, error) {
	if len(stripePayment.ChargeID) == 0 {
		return false, InternalErrorResponse(c, errors.New("no charge ID"))
	}

	var paid bool
	var err error

	for i := 0; i < stripeRetryCount; i++ {
		paid, err := stripePayment.CheckChargePaid()
		if !waitOnRetryableStripeError(err) {
			break
		}
	}
	return paid, handleStripeError(err, c)
}

func checkChargeAmount(c *gin.Context, chargeID string) (float64, error) {
	if len(chargeID) == 0 {
		return 0, InternalErrorResponse(c, errors.New("no charge ID"))
	}

	var amount float64
	var err error
	for i := 0; i < stripeRetryCount; i++ {
		amount, err = services.CheckChargeAmount(chargeID)
		if !waitOnRetryableStripeError(err) {
			break
		}
	}

	return amount, handleStripeError(err, c)
}

func waitOnRetryableStripeError(err error) bool {
	if err == nil {
		return false
	}

	if stripeErr, ok := err.(*stripe.Error); ok && stripeErr.Code == stripe.ErrorCodeRateLimit {
		time.Sleep(retrySleepIntervalInSec * time.Second)
		return true
	}

	return false
}

func handleStripeError(err error, c *gin.Context) error {
	if err == nil {
		return nil
	}

	if stripeErr, ok := err.(*stripe.Error); ok {
		switch stripeErr.Code {
		case stripe.ErrorCodeRateLimit,
			stripe.ErrorCodeProcessingError:
			return ServiceUnavailableResponse(c, err)
		case stripe.ErrorCodeCardDeclined,
			stripe.ErrorCodeExpiredCard,
			stripe.ErrorCodeIncorrectCVC,
			stripe.ErrorCodeIncorrectZip,
			stripe.ErrorCodeIncorrectNumber,
			stripe.ErrorCodeInvalidExpiryMonth,
			stripe.ErrorCodeInvalidExpiryYear,
			stripe.ErrorCodeInvalidNumber,
			stripe.ErrorCodeInvalidSwipeData,
			stripe.ErrorCodeResourceMissing:
			return BadRequestResponse(c, err)
		case stripe.ErrorCodeMissing:
			return InternalErrorResponse(c, err)
		}
	}
	return InternalErrorResponse(c, err)
}
