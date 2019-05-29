package jobs

import (
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type metricCollector struct{}

func (m metricCollector) ScheduleInterval() string {
	return "@every 24h"
}

func (m metricCollector) Run() {
	m.spaceUsageMetrics()
	m.accountsMetrics()
	m.fileMetrics()
}

func (m metricCollector) Runnable() bool {
	return models.DB != nil
}

func (m metricCollector) spaceUsageMetrics() {
	spaceReport := models.CreateSpaceUsedReport()
	spaceUsed := (spaceReport.SpaceUsedSum / float64(spaceReport.SpaceAllotedSum)) * float64(100)

	utils.Metrics_Percent_Of_Space_Used.Set(spaceUsed)
}

func (m metricCollector) accountsMetrics() {
	accountsCount := 0
	totalAccountErr := models.DB.Model(&models.Account{}).Count(&accountsCount).Error
	if totalAccountErr == nil {
		utils.Metrics_Total_Accounts.Set(float64(accountsCount))
	}

	unpaidAccountsCount, unpaidCountErr := models.CountAccountsByPaymentStatus(models.InitialPaymentInProgress)
	if unpaidCountErr == nil {
		utils.Metrics_Total_Unpaid_Accounts.Set(float64(unpaidAccountsCount))
	}

	collectedAccountsCount, collectedAccountErr := models.CountAccountsByPaymentStatus(models.PaymentRetrievalComplete)
	if collectedAccountErr == nil {
		utils.Metrics_Total_Collected_Accounts.Set(float64(collectedAccountsCount))
	}

	if utils.ReturnFirstError([]error{totalAccountErr, unpaidCountErr, collectedAccountErr}) == nil {
		paidAccountsCount := accountsCount - unpaidAccountsCount
		utils.Metrics_Total_Paid_Accounts.Set(float64(paidAccountsCount))

		percentOfAccountsPaid := (float64(paidAccountsCount) / float64(accountsCount)) * float64(100)

		utils.Metrics_Percent_Of_Accounts_Paid.Set(float64(percentOfAccountsPaid))

		percentOfAccountsUnpaid := (float64(unpaidAccountsCount) / float64(accountsCount)) * float64(100)

		utils.Metrics_Percent_Of_Accounts_Unpaid.Set(float64(percentOfAccountsUnpaid))

		percentOfPaidAccountsCollected := (float64(collectedAccountsCount) / float64(paidAccountsCount)) * float64(100)

		utils.Metrics_Percent_Of_Paid_Accounts_Collected.Set(float64(percentOfPaidAccountsCollected))
	}
}

func (m metricCollector) fileMetrics() {
	completedFileInSQLCount := 0
	err := models.DB.Model(&models.CompletedFile{}).Count(&completedFileInSQLCount).Error
	utils.LogIfError(err, nil)
	if err == nil {
		utils.Metrics_Completed_Files_Count_SQL.Set(float64(completedFileInSQLCount))
	}

	fileSizeInByteInSQL, err := models.GetTotalFileSizeInByte()
	utils.LogIfError(err, nil)
	if err == nil {
		utils.Metrics_Uploaded_File_Size_MB_SQL.Set(float64(fileSizeInByteInSQL) / 1000000.0)
	}
}