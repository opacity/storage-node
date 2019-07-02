package routes

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
	"time"
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
		// TODO: more granularity with errors
		// https://stripe.com/docs/api/errors/handling
		return ForbiddenResponse(c, err)
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

	costInWei := account.GetTotalCostInWei()

	success, _, _ := EthWrapper.TransferToken(
		services.MainWalletAddress,
		services.MainWalletPrivateKey,
		services.StringToAddress(account.EthAddress),
		*costInWei,
		services.FastGasPrice)

	if !success {
		return InternalErrorResponse(c, errors.New("OPQ transaction failed, but will try again"))
	}

	if err := models.DB.Model(&stripePayment).Update("opq_tx_status", models.OpqTxInProgress).Error; err != nil {
		return BadRequestResponse(c, err)
	}

	retries := 0
	maxRetries := 200

	for {
		paid, _ := account.CheckIfPaid()

		if paid {
			if err := models.DB.Model(&stripePayment).Update("opq_tx_status", models.OpqTxSuccess).Error; err != nil {
				return BadRequestResponse(c, err)
			}
			break
		}
		retries++
		if retries >= maxRetries {
			return InternalErrorResponse(c, errors.New("reached max number of retries, "+
				"but opq transfer still in progress"))
		}
		time.Sleep(2 * time.Second)
	}

	return OkResponse(c, StatusRes{
		Status: "successfully charged card and sent OPQ to payment address",
	})
}
