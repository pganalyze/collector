package tembo

import (
	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"net/http"
	"net/url"
)

// GetSystemState - Gets system information for a Tembo Cloud instance
func GetSystemState(config config.ServerConfig, logger *util.Logger) (system state.SystemState) {
	headers := map[string]string{
		"Authorization": "Bearer " + config.TemboAPIToken,
		"accept":        "application/json",
	}

	query := "sum(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{ namespace=\"" + config.TemboNamespace + "\"}) / sum(kube_pod_container_resource_requests{job=\"kube-state-metrics\",  namespace=\"" + config.TemboNamespace + "\", resource=\"cpu\"})"
	encodedQuery := url.QueryEscape(query)

	url := "https://api.data-1.use1.cdb-dev.com/org-demo-inst-pganalyze-test/metrics/query?query=" + encodedQuery

	client := http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.PrintError("Tembo/System: Encountered error when getting cluster info %v\n", err)
		return
	}

	for key, value := range headers {
		req.Header.Add(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.PrintError("Tembo/System: Encountered error when getting cluster info %v\n", err)
		return
	}
	logger.PrintInfo("Tembo/System: Response: %v\n", resp)
	defer resp.Body.Close()

	return
}
