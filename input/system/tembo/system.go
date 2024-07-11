package tembo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

type Response struct {
	Status string `json:"status"`
	Data   Data   `json:"data"`
}

type Data struct {
	ResultType string         `json:"resultType"`
	Result     []MetricResult `json:"result"`
}

type MetricResult struct {
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
func GetSystemState(ctx context.Context, server *state.Server, logger *util.Logger) (system state.SystemState) {
	system.Info.Type = state.TemboSystem
	config := server.Config
	headers := map[string]string{
		"Authorization": "Bearer " + config.TemboAPIToken,
		"accept":        "application/json",
	}

	client := http.Client{}
	metricsUrl := "https://" + config.TemboMetricsAPIURL + "/" + config.TemboNamespace + "/metrics/query?query="

	// Get CPU usage percentage
	query := "sum(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{ namespace=\"" + config.TemboNamespace + "\"}) / sum(kube_pod_container_resource_requests{job=\"kube-state-metrics\",  namespace=\"" + config.TemboNamespace + "\", resource=\"cpu\"})"
	cpuUsage, err := getFloat64(ctx, query, metricsUrl, client, headers)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectSystemStats, "error getting CPU info: %v", err)
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
	query = "sum(max by(pod) (kube_pod_container_resource_requests{job=\"kube-state-metrics\", namespace=\"" + config.TemboNamespace + "\", resource=\"memory\"}))"
	memoryTotalBytes, err := getUint64(ctx, query, metricsUrl, client, headers)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectSystemStats, "error getting memory info: %v", err)
		logger.PrintError("Tembo/System: Encountered error when getting memory info %v\n", err)
		return
	}

	// Get available memory
	query = "sum(max by(pod) (kube_pod_container_resource_requests{job=\"kube-state-metrics\", namespace=\"" + config.TemboNamespace + "\", resource=\"memory\"})) - sum(container_memory_working_set_bytes{job=\"kubelet\", metrics_path=\"/metrics/cadvisor\", namespace=\"" + config.TemboNamespace + "\",container!=\"\", image!=\"\"})"
	memoryAvailableBytes, err := getUint64(ctx, query, metricsUrl, client, headers)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectSystemStats, "error getting memory info: %v", err)
		logger.PrintError("Tembo/System: Encountered error when getting memory info %v\n", err)
		return
	}

	system.Memory.TotalBytes = memoryTotalBytes
	system.Memory.AvailableBytes = memoryAvailableBytes
	system.Memory.FreeBytes = memoryAvailableBytes

	// Get disk capacity
	// Note this does not yet handle multiple volume claims in cases like HA
	query = "kubelet_volume_stats_capacity_bytes{namespace=\"" + config.TemboNamespace + "\", persistentvolumeclaim=~\"" + config.TemboNamespace + "-1" + "\"}"
	diskCapacity, err := getUint64(ctx, query, metricsUrl, client, headers)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectSystemStats, "error getting disk info: %v", err)
		logger.PrintError("Tembo/System: Encountered error when getting disk info %v\n", err)
		return
	}

	// Get disk available
	// Note this does not yet handle multiple volume claims in cases like HA
	query = "kubelet_volume_stats_available_bytes{namespace=\"" + config.TemboNamespace + "\", persistentvolumeclaim=~\"" + config.TemboNamespace + "-1" + "\"}"
	diskAvailable, err := getUint64(ctx, query, metricsUrl, client, headers)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectSystemStats, "error getting disk info: %v", err)
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

	server.SelfTest.MarkCollectionAspectOk(state.CollectionAspectSystemStats)

	return
}

func getFloat64(ctx context.Context, query string, metricsUrl string, client http.Client, headers map[string]string) (float64, error) {
	res, err := getSystemInfo(ctx, metricsUrl, query, client, headers)
	if err != nil {
		return 0, err
	}

	// Check if res.Data.Result is empty
	if len(res.Data.Result) == 0 {
		return 0, nil
	}

	strValue := res.Data.Result[0].Value[1].(string)
	value, err := strconv.ParseFloat(strValue, 64)

	if err != nil {
		return 0, err
	}

	return value, nil
}

func getUint64(ctx context.Context, query string, metricsUrl string, client http.Client, headers map[string]string) (uint64, error) {
	res, err := getSystemInfo(ctx, metricsUrl, query, client, headers)
	if err != nil {
		return 0, err
	}

	// Check if res.Data.Result is empty
	if len(res.Data.Result) == 0 {
		return 0, nil
	}

	strValue := res.Data.Result[0].Value[1].(string)
	value, err := strconv.ParseUint(strValue, 10, 64)

	if err != nil {
		return 0, err
	}

	return value, nil
}

func getSystemInfo(ctx context.Context, metricsUrl string, query string, client http.Client, headers map[string]string) (Response, error) {
	encodedQuery := url.QueryEscape(query)

	metricsUrl = metricsUrl + encodedQuery

	req, err := http.NewRequestWithContext(ctx, "GET", metricsUrl, nil)
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
