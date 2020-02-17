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
	StripeToken      string `json:"stripeToken" binding:"required" example:"tok_KPte7942xySKBKyrBu11yEpf"`
	Timestamp        int64  `json:"timestamp" binding:"required"`
	UpgradeAccount   bool   `json:"upgradeAccount"`
	StorageLimit     int    `json:"storageLimit" binding:"omitempty,gte=128" minimum:"128" example:"128"`
	DurationInMonths int    `json:"durationInMonths" binding:"omitempty,gte=1" minimum:"1" example:"12"`
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

	if !utils.Env.EnableCreditCards {
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

	var costInDollars float64
	if request.createStripePaymentObject.UpgradeAccount {
		if err := verifyValidStorageLimit(request.createStripePaymentObject.StorageLimit, c); err != nil {
			return err
		}
		costInDollars, _ = account.UpgradeCostInUSD(request.createStripePaymentObject.StorageLimit,
			request.createStripePaymentObject.DurationInMonths)
	} else {
		costInDollars = utils.Env.Plans[int(account.StorageLimit)].CostInUSD
	}

	if costInDollars <= float64(0.50) {
		return ForbiddenResponse(c, errors.New("cannot create stripe charge for less than $0.50"))
	}

	if paid := verifyIfPaid(account); paid && !utils.FreeModeEnabled() &&
		!request.createStripePaymentObject.UpgradeAccount {
		return ForbiddenResponse(c, errors.New("account is already paid for"))
	}

	charge, stripePayment, err := createChargeAndStripePayment(c, costInDollars, account, request.createStripePaymentObject)
	if err != nil {
		return err
	}

	if !request.createStripePaymentObject.UpgradeAccount {
		if err := stripePayment.SendAccountOPQ(); err != nil {
			return InternalErrorResponse(c, err)
		}
	} else {

		if err := payUpgradeCostWithStripe(c, stripePayment, account, request.createStripePaymentObject); err != nil {
			return err
		}
	}

	account.UpdatePaymentViaStripe()

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
			Amount:              float64(charge.Amount) / 100.00,
		},
	})
}

func createChargeAndStripePayment(c *gin.Context, costInDollars float64, account models.Account,
	reqBody createStripePaymentObject) (*stripe.Charge, models.StripePayment, error) {
	var charge *stripe.Charge
	var err error
	for i := 0; i < stripeRetryCount; i++ {
		charge, err = services.CreateCharge(costInDollars, reqBody.StripeToken, account.AccountID)
		if !waitOnRetryableStripeError(err) {
			break
		}
	}

	if err != nil {
		return charge, models.StripePayment{}, handleStripeError(err, c)
	}

	stripePayment := models.StripePayment{
		StripeToken:    reqBody.StripeToken,
		AccountID:      account.AccountID,
		ChargeID:       charge.ID,
		UpgradePayment: reqBody.UpgradeAccount,
	}

	// Add stripe payment to DB
	if err := models.DB.Create(&stripePayment).Error; err != nil {
		return charge, stripePayment, BadRequestResponse(c, err)
	}
	return charge, stripePayment, nil
}

func payUpgradeCostWithStripe(c *gin.Context, stripePayment models.StripePayment, account models.Account, createStripePaymentObject createStripePaymentObject) error {
	if err := stripePayment.SendUpgradeOPQ(account.AccountID, createStripePaymentObject.StorageLimit); err != nil {
		return InternalErrorResponse(c, err)
	}
	var paid bool
	var err error
	if paid, err = checkChargePaid(c, stripePayment); err != nil {
		return InternalErrorResponse(c, err)
	} else if paid {
		err = models.DB.Model(&account).Update("payment_status", models.InitialPaymentInProgress).Error
	}
	if err != nil {
		return InternalErrorResponse(c, err)
	}
	return nil
}

func checkChargePaid(c *gin.Context, stripePayment models.StripePayment) (bool, error) {
	if len(stripePayment.ChargeID) == 0 {
		return false, InternalErrorResponse(c, errors.New("no charge ID"))
	}

	var paid bool
	var err error

	for i := 0; i < stripeRetryCount; i++ {
		paid, err = stripePayment.CheckChargePaid()
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
