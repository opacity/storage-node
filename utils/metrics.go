package utils

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// List of all Metrics throughout the application
var (
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
)
