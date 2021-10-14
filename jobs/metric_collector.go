package jobs

import (
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
	"github.com/prometheus/client_golang/prometheus"
)

type metricCollector struct{}

func (m metricCollector) Name() string {
	return "metricCollector"
}

func (m metricCollector) ScheduleInterval() string {
	return "@every 1h"
}

func (m metricCollector) Run() {
	utils.SlackLog("running " + m.Name())

	m.spaceUsageMetrics()
	m.accountsMetrics()
	m.fileMetrics()
}

func (m metricCollector) Runnable() bool {
	return models.DB != nil
}

func (m metricCollector) spaceUsageMetrics() {
	spaceReport := models.CreateSpaceUsedReport()

	utils.Metrics_Percent_Of_Space_Used_Map[utils.TotalLbl].Set(models.CalculatePercentSpaceUsed(spaceReport))

	plans, _ := models.GetAllPlans()

	for _, plan := range plans {
		spaceReport := models.CreateSpaceUsedReportForPlanType(plan)

		utils.Metrics_Percent_Of_Space_Used_Map[plan.Name].Set(models.CalculatePercentSpaceUsed(spaceReport))
	}
}

func (m metricCollector) accountsMetrics() {
	accountsCount := 0
	totalAccountErr := models.DB.Model(&models.Account{}).Count(&accountsCount).Error
	if totalAccountErr == nil {
		utils.Metrics_Total_Accounts.Set(float64(accountsCount))
	}

	accountsPaidWithStripe := 0
	totalAccountPaidWithStripeErr := models.DB.Model(&models.Account{}).Where("payment_method = ?", models.PaymentMethodWithCreditCard).Count(&accountsPaidWithStripe).Error
	if totalAccountPaidWithStripeErr == nil {
		utils.Metrics_Total_Stripe_Paid_Accounts_Map[utils.TotalLbl].Set(float64(accountsPaidWithStripe))
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
		utils.Metrics_Total_Paid_Accounts_Map[utils.TotalLbl].Set(float64(paidAccountsCount))

		percentOfAccountsPaid := (float64(paidAccountsCount) / float64(accountsCount)) * float64(100)

		utils.Metrics_Percent_Of_Accounts_Paid.Set(float64(percentOfAccountsPaid))

		percentOfAccountsUnpaid := (float64(unpaidAccountsCount) / float64(accountsCount)) * float64(100)

		utils.Metrics_Percent_Of_Accounts_Unpaid.Set(float64(percentOfAccountsUnpaid))

		percentOfPaidAccountsCollected := (float64(collectedAccountsCount) / float64(paidAccountsCount)) * float64(100)

		utils.Metrics_Percent_Of_Paid_Accounts_Collected.Set(float64(percentOfPaidAccountsCollected))
	}

	plans, _ := models.GetAllPlans()

	for _, plan := range plans {
		name := plan.Name
		accountCount, err := models.CountPaidAccountsByPlanType(plan)
		if err == nil {
			utils.Metrics_Total_Paid_Accounts_Map[name].Set(float64(accountCount))
		}
		utils.LogIfError(err, nil)

		stripeCount, err := models.CountPaidAccountsByPaymentMethodAndPlanType(plan, models.PaymentMethodWithCreditCard)
		if err == nil {
			utils.Metrics_Total_Stripe_Paid_Accounts_Map[name].Set(float64(stripeCount))
		}
		utils.LogIfError(err, nil)
	}
}

func CreatePlanMetrics() {
	utils.Metrics_Percent_Of_Space_Used_Map[utils.TotalLbl] = utils.Metrics_Percent_Of_Space_Used.With(prometheus.Labels{"plan_type": utils.TotalLbl})
	utils.Metrics_Total_Paid_Accounts_Map[utils.TotalLbl] = utils.Metrics_Total_Paid_Accounts.With(prometheus.Labels{"plan_type": utils.TotalLbl})

	utils.Metrics_Total_Stripe_Paid_Accounts_Map[utils.TotalLbl] = utils.Metrics_Total_Stripe_Paid_Accounts.With(prometheus.Labels{"plan_type": utils.TotalLbl})

	plans, _ := models.GetAllPlans()

	for _, plan := range plans {
		name := plan.Name
		utils.Metrics_Percent_Of_Space_Used_Map[name] = utils.Metrics_Percent_Of_Space_Used.With(prometheus.Labels{"plan_type": name})
		utils.Metrics_Total_Paid_Accounts_Map[name] = utils.Metrics_Total_Paid_Accounts.With(prometheus.Labels{"plan_type": name})
		utils.Metrics_Total_Stripe_Paid_Accounts_Map[name] = utils.Metrics_Total_Stripe_Paid_Accounts.With(prometheus.Labels{"plan_type": name})
	}
}

func (m metricCollector) fileMetrics() {
	completedFileInSQLCountS3 := 0
	err := models.DB.Where("storage_type = ?", models.S3).Model(&models.CompletedFile{}).Count(&completedFileInSQLCountS3).Error
	utils.LogIfError(err, nil)
	if err == nil {
		utils.Metrics_Completed_Files_Count_SQL.Set(float64(completedFileInSQLCountS3))
	}

	completedFileInSQLCountSia := 0
	err = models.DB.Where("storage_type = ?", models.Sia).Model(&models.CompletedFile{}).Count(&completedFileInSQLCountSia).Error
	utils.LogIfError(err, nil)
	if err == nil {
		utils.Metrics_Completed_Files_Count_SQL_Sia.Set(float64(completedFileInSQLCountSia))
	}

	fileSizeInByteInSQLS3, err := models.GetTotalFileSizeInByteByStorageType(models.S3)
	utils.LogIfError(err, nil)
	if err == nil {
		utils.Metrics_Uploaded_File_Size_MB_SQL.Set(float64(fileSizeInByteInSQLS3) / 1000000.0)
	}

	fileSizeInByteInSQLSia, err := models.GetTotalFileSizeInByteByStorageType(models.Sia)
	utils.LogIfError(err, nil)
	if err == nil {
		utils.Metrics_Uploaded_File_Size_MB_SQL_Sia.Set(float64(fileSizeInByteInSQLSia) / 1000000.0)
	}
}
