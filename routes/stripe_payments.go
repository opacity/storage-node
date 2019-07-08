package routes

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
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
// @Success 200 {object} routes.StatusRes
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

	charge, err := services.CreateCharge(costInDollars, request.createStripePaymentObject.StripeToken)

	if err != nil {
		err = handleStripeError(err, c)
		return err
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

	err = stripePayment.SendAccountOPQ()

	if err != nil {
		return InternalErrorResponse(c, err)
	}

	return OkResponse(c, StatusRes{
		Status: "successfully charged card, now sending OPQ to payment address",
	})
}

func checkChargePaid(c *gin.Context, chargeID string) (bool, error) {
	if len(chargeID) == 0 {
		return false, InternalErrorResponse(c, errors.New("no charge ID"))
	}
	paid, err := services.CheckChargePaid(chargeID)
	err = handleStripeError(err, c)
	return paid, err
}

func handleStripeError(err error, c *gin.Context) error {
	if err != nil {
		// TODO: more granularity with errors
		// https://stripe.com/docs/api/errors/handling
		err = InternalErrorResponse(c, err)
	}
	return err
}
