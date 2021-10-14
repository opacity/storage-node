package utils

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	dto "github.com/prometheus/client_model/go"
)

// List of all Metrics throughout the application
var (
	TotalLbl = "total"

	Metrics_PingStdOut_Counter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "storagenode_pingstdout_counter",
		Help: "The total number of ping std out",
	})

	// Statis for Ok, BadRequest, InternalError, ServiceUnavailable, Forbidden, NotFound
	Metrics_Http_Response_Counter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "storagenode_http_response_counter",
		Help: "The total number of Http Response code",
	}, []string{"response_code"})
	Metrics_200_Response_Counter = Metrics_Http_Response_Counter.With(prometheus.Labels{"response_code": "200"})
	Metrics_400_Response_Counter = Metrics_Http_Response_Counter.With(prometheus.Labels{"response_code": "400"})
	Metrics_403_Response_Counter = Metrics_Http_Response_Counter.With(prometheus.Labels{"response_code": "403"})
	Metrics_404_Response_Counter = Metrics_Http_Response_Counter.With(prometheus.Labels{"response_code": "404"})
	Metrics_500_Response_Counter = Metrics_Http_Response_Counter.With(prometheus.Labels{"response_code": "500"})
	Metrics_503_Response_Counter = Metrics_Http_Response_Counter.With(prometheus.Labels{"response_code": "503"})

	Metrics_Percent_Of_Space_Used = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "storagenode_percent_of_space_used_gauge",
		Help: "Space used as a percentage of space purchased, for all paid accounts",
	}, []string{"plan_type"})
	Metrics_Percent_Of_Space_Used_Map = make(map[string]prometheus.Gauge)

	Metrics_Total_Accounts = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "storagenode_total_accounts",
		Help: "Total number of accounts",
	})

	Metrics_Total_Paid_Accounts = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "storagenode_total_paid_accounts",
		Help: "Total number of paid accounts",
	}, []string{"plan_type"})
	Metrics_Total_Paid_Accounts_Map = make(map[string]prometheus.Gauge)

	Metrics_Total_Stripe_Paid_Accounts = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "storagenode_total_stripe_paid_accounts",
		Help: "Total number of paid accounts via Stripe",
	}, []string{"plan_type"})
	Metrics_Total_Stripe_Paid_Accounts_Map = make(map[string]prometheus.Gauge)

	Metrics_Total_Unpaid_Accounts = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "storagenode_total_unpaid_accounts",
		Help: "Total number of unpaid accounts",
	})

	Metrics_Total_Collected_Accounts = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "storagenode_total_collected_accounts",
		Help: "Total number of paid accounts which we have finished collecting the tokens for",
	})

	Metrics_Total_ExpiredArchived_Accounts = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "storagenode_total_expiredarchived_accounts",
		Help: "Total number of expired accounts which were archived",
	})

	Metrics_Total_Expired_Accounts = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "storagenode_total_expired_accounts",
		Help: "Total number of expired accounts which have time to renew",
	})

	Metrics_Total_Renewed_Accounts = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "storagenode_total_renewed_accounts",
		Help: "Total number of renewed accounts",
	})

	Metrics_Percent_Of_Accounts_Paid = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "storagenode_percent_of_accounts_paid",
		Help: "Accounts that are paid, as a percentage of all accounts",
	})

	Metrics_Percent_Of_Accounts_Unpaid = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "storagenode_percent_of_accounts_unpaid",
		Help: "Accounts that are unpaid, as a percentage of all accounts",
	})

	Metrics_Percent_Of_Paid_Accounts_Collected = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "storagenode_percent_of_paid_accounts_collected",
		Help: "Accounts that we've finished collecting the tokens from, as a percentage of all paid accounts",
	})

	Metrics_Completed_Files_Count_SQL = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "storagenode_completed_files_count_sql",
		Help: "Total number of completed files in SQL database",
	})

	Metrics_Uploaded_File_Size_MB_SQL = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "storagenode_uploaded_file_size_mb_sql",
		Help: "Totals all the file sizes of rows in completed_files table in SQL, as MB",
	})

	// TODO:  use AWS cloudwatch to get these last two metrics
	// https://docs.aws.amazon.com/sdk-for-go/api/service/cloudwatch/#CloudWatch.GetMetricStatistics
	//Metrics_Files_Count_S3 = promauto.NewGauge(prometheus.GaugeOpts{
	//	Name: "storagenode_files_count_s3",
	//	Help: "Total number of objects in the bucket",
	//})
	//
	//Metrics_Uploaded_File_Size_MB_S3 = promauto.NewGauge(prometheus.GaugeOpts{
	//	Name: "storagenode_uploaded_file_size_mb_s3",
	//	Help: "Total data uploaded to S3 bucket, as MB",
	//})
)

func CreatePlanMetrics() {
	Metrics_Percent_Of_Space_Used_Map[TotalLbl] = Metrics_Percent_Of_Space_Used.With(prometheus.Labels{"plan_type": TotalLbl})
	Metrics_Total_Paid_Accounts_Map[TotalLbl] = Metrics_Total_Paid_Accounts.With(prometheus.Labels{"plan_type": TotalLbl})
	Metrics_Total_Stripe_Paid_Accounts_Map[TotalLbl] = Metrics_Total_Stripe_Paid_Accounts.With(prometheus.Labels{"plan_type": TotalLbl})
	for _, plan := range Env.Plans {
		name := plan.Name
		Metrics_Percent_Of_Space_Used_Map[name] = Metrics_Percent_Of_Space_Used.With(prometheus.Labels{"plan_type": name})
		Metrics_Total_Paid_Accounts_Map[name] = Metrics_Total_Paid_Accounts.With(prometheus.Labels{"plan_type": name})
		Metrics_Total_Stripe_Paid_Accounts_Map[name] = Metrics_Total_Stripe_Paid_Accounts.With(prometheus.Labels{"plan_type": name})
	}
}

func GetMetricCounter(m prometheus.Counter) float64 {
	pb := &dto.Metric{}
	m.Write(pb)
	return pb.GetCounter().GetValue()
}
