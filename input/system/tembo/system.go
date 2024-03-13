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
	Name                  string `json:"__name__"`
	Endpoint              string `json:"endpoint"`
	Instance              string `json:"instance"`
	Job                   string `json:"job"`
	MetricsPath           string `json:"metrics_path"`
	Namespace             string `json:"namespace"`
	Node                  string `json:"node"`
	PersistentVolumeClaim string `json:"persistentvolumeclaim"`
	Service               string `json:"service"`
}

// GetSystemState - Gets system information for a Tembo Cloud instance
func GetSystemState(config config.ServerConfig, logger *util.Logger) (system state.SystemState) {
	headers := map[string]string{
		"Authorization": "Bearer " + config.TemboAPIToken,
		"accept":        "application/json",
	}

	client := http.Client{}

	// Get CPU usage percentage
	query := "sum(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{ namespace=\"" + config.TemboMetricsNamespace + "\"}) / sum(kube_pod_container_resource_requests{job=\"kube-state-metrics\",  namespace=\"" + config.TemboMetricsNamespace + "\", resource=\"cpu\"})"
	cpuUsage, err := getFloat64(query, "https://api.data-1.use1.tembo.io/"+config.TemboMetricsNamespace+"/metrics/query?query=", client, headers)
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
	query = "sum(max by(pod) (kube_pod_container_resource_requests{job=\"kube-state-metrics\", namespace=\"" + config.TemboMetricsNamespace + "\", resource=\"memory\"}))"
	memoryTotalBytes, err := getUint64(query, "https://api.data-1.use1.tembo.io/"+config.TemboMetricsNamespace+"/metrics/query?query=", client, headers)
	if err != nil {
		logger.PrintError("Tembo/System: Encountered error when getting memory info %v\n", err)
		return
	}

	// Get available memory
	query = "sum(max by(pod) (kube_pod_container_resource_requests{job=\"kube-state-metrics\", namespace=\"" + config.TemboMetricsNamespace + "\", resource=\"memory\"})) - sum(container_memory_working_set_bytes{job=\"kubelet\", metrics_path=\"/metrics/cadvisor\", namespace=\"" + config.TemboMetricsNamespace + "\",container!=\"\", image!=\"\"})"
	memoryAvailableBytes, err := getUint64(query, "https://api.data-1.use1.tembo.io/"+config.TemboMetricsNamespace+"/metrics/query?query=", client, headers)
	if err != nil {
		logger.PrintError("Tembo/System: Encountered error when getting memory info %v\n", err)
		return
	}

	system.Memory.TotalBytes = memoryTotalBytes
	system.Memory.AvailableBytes = memoryAvailableBytes
	system.Memory.FreeBytes = memoryAvailableBytes

	// Get disk capacity
	//TODO(ianstanton) Check if volume claim names differ in cases like HA
	query = "kubelet_volume_stats_capacity_bytes{namespace=\"" + config.TemboMetricsNamespace + "\", persistentvolumeclaim=~\"" + config.TemboMetricsNamespace + "-1" + "\"}"
	diskCapacity, err := getUint64(query, "https://api.data-1.use1.tembo.io/"+config.TemboMetricsNamespace+"/metrics/query?query=", client, headers)
	if err != nil {
		logger.PrintError("Tembo/System: Encountered error when getting disk info %v\n", err)
		return
	}

	// Get disk available
	//TODO(ianstanton) Check if volume claim names differ in cases like HA
	query = "kubelet_volume_stats_available_bytes{namespace=\"" + config.TemboMetricsNamespace + "\", persistentvolumeclaim=~\"" + config.TemboMetricsNamespace + "-1" + "\"}"
	diskAvailable, err := getUint64(query, "https://api.data-1.use1.tembo.io/"+config.TemboMetricsNamespace+"/metrics/query?query=", client, headers)
	if err != nil {
		logger.PrintError("Tembo/System: Encountered error when getting disk info %v\n", err)
		return
	}

	diskUsed := diskCapacity - diskAvailable
	system.DiskPartitions = make(state.DiskPartitionMap)
	system.DiskPartitions["/"] = state.DiskPartition{
		DiskName:   "default",
		UsedBytes:  diskUsed,
		TotalBytes: diskAvailable,
	}

	return
}

func getFloat64(query string, metricsUrl string, client http.Client, headers map[string]string) (float64, error) {
	res, err := getSystemInfo(metricsUrl, query, client, headers)
	if err != nil {
		return 0, err
	}

	strValue := res.Data.Result[0].Value[1].(string)
	value, err := strconv.ParseFloat(strValue, 64)

	return value, nil
}

func getUint64(query string, metricsUrl string, client http.Client, headers map[string]string) (uint64, error) {
	res, err := getSystemInfo(metricsUrl, query, client, headers)
	if err != nil {
		return 0, err
	}

	strValue := res.Data.Result[0].Value[1].(string)
	value, err := strconv.ParseUint(strValue, 10, 64)

	return value, nil
}

func getSystemInfo(metricsUrl string, query string, client http.Client, headers map[string]string) (Response, error) {
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
