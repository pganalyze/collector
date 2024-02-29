package tembo

import (
	"encoding/json"
	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"net/http"
	"net/url"
	"strconv"
)

type Response struct {
	Status string `json:"status"`
	Data   Data   `json:"data"`
}

type Data struct {
	ResultType string   `json:"resultType"`
	Result     []Result `json:"result"`
}

type Result struct {
	Metric Metric        `json:"metric"`
	Value  []interface{} `json:"value"`
}

type Metric struct {
}

// GetSystemState - Gets system information for a Tembo Cloud instance
func GetSystemState(config config.ServerConfig, logger *util.Logger) (system state.SystemState) {
	headers := map[string]string{
		"Authorization": "Bearer " + config.TemboAPIToken,
		"accept":        "application/json",
	}

	client := http.Client{}

	// Get CPU usage percentage
	cpuUsage, err := getCpuUsage("https://api.data-1.use1.cdb-dev.com/"+config.TemboNamespace+"/metrics/query?query=", client, headers, config.TemboNamespace, logger)
	if err != nil {
		logger.PrintError("Tembo/System: Encountered error when getting CPU info %v\n", err)
		return
	}

	system.CPUStats = make(state.CPUStatisticMap)
	system.CPUStats["all"] = state.CPUStatistic{
		DiffedOnInput: true,
		DiffedValues: &state.DiffedSystemCPUStats{
			UserPercent: cpuUsage,
		},
	}

	// Get total memory
	memoryTotalBytes, err := getTotalMemory("https://api.data-1.use1.cdb-dev.com/"+config.TemboNamespace+"/metrics/query?query=", client, headers, config.TemboNamespace, logger)
	if err != nil {
		logger.PrintError("Tembo/System: Encountered error when getting memory info %v\n", err)
		return
	}

	// Get available memory
	memoryAvailableBytes, err := getAvailableMemory("https://api.data-1.use1.cdb-dev.com/"+config.TemboNamespace+"/metrics/query?query=", client, headers, config.TemboNamespace, logger)
	if err != nil {
		logger.PrintError("Tembo/System: Encountered error when getting memory info %v\n", err)
		return
	}

	system.Memory.TotalBytes = memoryTotalBytes
	system.Memory.AvailableBytes = memoryAvailableBytes
	system.Memory.FreeBytes = memoryAvailableBytes

	return
}

func getCpuUsage(metricsUrl string, client http.Client, headers map[string]string, namespace string, logger *util.Logger) (float64, error) {
	query := "sum(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{ namespace=\"" + namespace + "\"}) / sum(kube_pod_container_resource_requests{job=\"kube-state-metrics\",  namespace=\"" + namespace + "\", resource=\"cpu\"})"
	res, err := getSystemInfo(metricsUrl, query, client, headers, logger)
	if err != nil {
		return 0, err
	}

	// Get cpuUsage from response
	cpuUsageStr := res.Data.Result[0].Value[1].(string)

	// Convert cpuUsage to float64
	cpuUsage, err := strconv.ParseFloat(cpuUsageStr, 64)

	return cpuUsage, nil
}

func getTotalMemory(metricsUrl string, client http.Client, headers map[string]string, namespace string, logger *util.Logger) (uint64, error) {
	query := "sum(max by(pod) (kube_pod_container_resource_requests{job=\"kube-state-metrics\", namespace=\"" + namespace + "\", resource=\"memory\"}))"

	res, err := getSystemInfo(metricsUrl, query, client, headers, logger)
	if err != nil {
		return 0, err
	}

	// Get totalMemory from response
	totalMemoryStr := res.Data.Result[0].Value[1].(string)

	// Convert totalMemory to uint64
	totalMemory, err := strconv.ParseUint(totalMemoryStr, 10, 64)

	return totalMemory, nil
}

func getAvailableMemory(metricsUrl string, client http.Client, headers map[string]string, namespace string, logger *util.Logger) (uint64, error) {
	query := "sum(max by(pod) (kube_pod_container_resource_requests{job=\"kube-state-metrics\", namespace=\"" + namespace + "\", resource=\"memory\"})) - sum(container_memory_working_set_bytes{job=\"kubelet\", metrics_path=\"/metrics/cadvisor\", namespace=\"" + namespace + "\",container!=\"\", image!=\"\"})"

	res, err := getSystemInfo(metricsUrl, query, client, headers, logger)
	if err != nil {
		return 0, err
	}

	// Get availableMemory from response
	availableMemoryStr := res.Data.Result[0].Value[1].(string)

	// Convert availableMemory to uint64
	availableMemory, err := strconv.ParseUint(availableMemoryStr, 10, 64)

	return availableMemory, nil
}

func getSystemInfo(metricsUrl string, query string, client http.Client, headers map[string]string, logger *util.Logger) (Response, error) {
	encodedQuery := url.QueryEscape(query)

	metricsUrl = metricsUrl + encodedQuery

	req, err := http.NewRequest("GET", metricsUrl, nil)
	if err != nil {
		return Response{}, err
	}

	for key, value := range headers {
		req.Header.Add(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return Response{}, err
	}

	var result Response
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return Response{}, err
	}

	return result, nil
}
