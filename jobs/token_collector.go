package jobs

import (
	"time"

	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type tokenCollector struct{}

func (t tokenCollector) Name() string {
	return "tokenCollector"
}

func (t tokenCollector) ScheduleInterval() string {
	return "@every 30m"
}

const hoursToWaitForReset = 48

func (t tokenCollector) Run() {
	utils.SlackLog("running " + t.Name())
	for paymentStatus := models.InitialPaymentInProgress; paymentStatus < models.PaymentRetrievalComplete; paymentStatus++ {
		accounts := models.GetAccountsByPaymentStatus(paymentStatus)
		runAccountsCollectionSequence(accounts)

		upgrades := models.GetUpgradesByPaymentStatus(paymentStatus)
		runUpgradesCollectionSequence(upgrades)

		renewals := models.GetRenewalsByPaymentStatus(paymentStatus)
		runRenewalsCollectionSequence(renewals)
	}

	for paymentStatus := models.GasTransferInProgress; paymentStatus < models.PaymentRetrievalComplete; paymentStatus++ {
		err := models.SetAccountsToLowerPaymentStatusByUpdateTime(paymentStatus, time.Now().Add(-1*hoursToWaitForReset*time.Hour))
		utils.LogIfError(err, nil)

		err = models.SetUpgradesToLowerPaymentStatusByUpdateTime(paymentStatus, time.Now().Add(-1*hoursToWaitForReset*time.Hour))
		utils.LogIfError(err, nil)

		err = models.SetRenewalsToLowerPaymentStatusByUpdateTime(paymentStatus, time.Now().Add(-1*hoursToWaitForReset*time.Hour))
		utils.LogIfError(err, nil)
	}
}

func (t tokenCollector) Runnable() bool {
	// @TODO: investigate this
	// err := services.SetWallets()
	// utils.LogIfError(err, nil)
	// return models.DB != nil && err == nil
	return models.DB != nil
}

func runAccountsCollectionSequence(accounts []models.Account) {
	for _, account := range accounts {
		cost, _ := account.Cost()
		err := models.AccountCollectionFunctions[account.PaymentStatus](account)
		utils.LogIfError(err, map[string]interface{}{
			"message":        "error running token collection functions on account",
			"eth_address":    account.EthAddress,
			"account_id":     account.AccountID,
			"payment_status": models.PaymentStatusMap[account.PaymentStatus],
			"cost":           cost,
		})

	}
}

func runUpgradesCollectionSequence(upgrades []models.Upgrade) {
	for _, upgrade := range upgrades {
		err := models.UpgradeCollectionFunctions[upgrade.PaymentStatus](upgrade)
		utils.LogIfError(err, map[string]interface{}{
			"message":        "error running token collection functions on upgrade",
			"eth_address":    upgrade.EthAddress,
			"account_id":     upgrade.AccountID,
			"payment_status": models.PaymentStatusMap[upgrade.PaymentStatus],
			"cost":           upgrade.OpctCost,
		})
	}
}

func runRenewalsCollectionSequence(renewals []models.Renewal) {
	for _, renewal := range renewals {
		err := models.RenewalCollectionFunctions[renewal.PaymentStatus](renewal)
		utils.LogIfError(err, map[string]interface{}{
			"message":        "error running token collection functions on renewal",
			"eth_address":    renewal.EthAddress,
			"account_id":     renewal.AccountID,
			"payment_status": models.PaymentStatusMap[renewal.PaymentStatus],
			"cost":           renewal.OpctCost,
		})
	}
}
