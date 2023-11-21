package crunchy_bridge

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pganalyze/collector/util"
)

const apiBaseURL = "https://api.crunchybridge.com"

type Client struct {
	http.Client

	BaseURL     string
	BearerToken string
	ClusterID   string
}

type ClusterInfo struct {
	CPU        int32   `json:"cpu"`
	CreatedAt  string  `json:"created_at"`
	Memory     float32 `json:"memory"`
	Name       string  `json:"name"`
	PlanID     string  `json:"plan_id"`
	ProviderID string  `json:"provider_id"`
	RegionID   string  `json:"region_id"`
	Storage    int32   `json:"storage"`
}

type MetricViews struct {
	Name   string         `json:"name"`
	Series []MetricSeries `json:"series"`
}

type MetricSeries struct {
	IsEmpty bool          `json:"is_empty"`
	Name    string        `json:"name"`
	Points  []MetricPoint `json:"points"`
	Title   string        `json:"title"`
}

type MetricPoint struct {
	Time  string  `json:"time"`
	Value float64 `json:"value"`
}

type CPUMetrics struct {
	Iowait float64
	System float64
	User   float64
	Steal  float64
}

type MemoryMetrics struct {
	MemoryUsed float64
	SwapUsed   float64
}

type IOPSMetrics struct {
	Writes float64
	Reads  float64
}

type LoadAverageMetrics struct {
	One float64
}

type DiskUsageMetrics struct {
	DatabaseSize uint64
}

func (c *Client) NewRequest(method string, path string) (*http.Request, error) {
	req, err := http.NewRequest(method, c.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.BearerToken)
	req.Header.Set("User-Agent", util.CollectorNameAndVersion)
	req.Header.Add("Accept", "application/json")
	return req, nil
}

func (c *Client) GetClusterInfo() (*ClusterInfo, error) {
	req, err := c.NewRequest("GET", "/clusters/"+c.ClusterID)
	if err != nil {
		return nil, err
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK || len(body) == 0 {
		return nil, err
	}

	clusterInfo := ClusterInfo{}
	err = json.Unmarshal(body, &clusterInfo)
	if err != nil {
		return nil, err
	}
	return &clusterInfo, err
}

func (c *Client) getMetrics(name string) (*MetricViews, error) {
	req, err := c.NewRequest("GET", fmt.Sprintf("/metric-views/%s?cluster_id=%s&period=15m", name, c.ClusterID))
	if err != nil {
		return nil, err
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK || len(body) == 0 {
		return nil, err
	}

	metricViews := MetricViews{}
	err = json.Unmarshal(body, &metricViews)
	if err != nil {
		return nil, err
	}

	return &metricViews, nil
}

func (c *Client) GetCPUMetrics() (*CPUMetrics, error) {
	metricViews, err := c.getMetrics("cpu")
	if err != nil {
		return nil, err
	}

	metrics := CPUMetrics{}
	for _, series := range metricViews.Series {
		switch series.Name {
		case "cpu_load_iowait":
			metrics.Iowait = average(series.Points)
		case "cpu_load_system":
			metrics.System = average(series.Points)
		case "cpu_load_user":
			metrics.User = average(series.Points)
		case "cpu_load_steal":
			metrics.Steal = average(series.Points)
		}
	}

	return &metrics, err
}

func (c *Client) GetMemoryMetrics() (*MemoryMetrics, error) {
	metricViews, err := c.getMetrics("memory")
	if err != nil {
		return nil, err
	}

	// Currently, it's returned as percentage which doesn't work well with the current structure
	// Potentially we can calculate bytes based on the total memory bytes
	metrics := MemoryMetrics{}
	for _, series := range metricViews.Series {
		switch series.Name {
		case "memory_used":
			metrics.MemoryUsed = average(series.Points)
		case "swap_used":
			metrics.SwapUsed = average(series.Points)
		}
	}

	return &metrics, err
}

func (c *Client) GetIOPSMetrics() (*IOPSMetrics, error) {
	metricViews, err := c.getMetrics("iops")
	if err != nil {
		return nil, err
	}

	metrics := IOPSMetrics{}
	for _, series := range metricViews.Series {
		switch series.Name {
		case "io_wtps":
			metrics.Writes = average(series.Points)
		case "io_rtps":
			metrics.Reads = average(series.Points)
		}
	}

	return &metrics, err
}

func (c *Client) GetLoadAverageMetrics() (*LoadAverageMetrics, error) {
	metricViews, err := c.getMetrics("load-average")
	if err != nil {
		return nil, err
	}

	metrics := LoadAverageMetrics{}
	for _, series := range metricViews.Series {
		switch series.Name {
		case "load_average_1":
			metrics.One = average(series.Points)
		}
	}

	return &metrics, err
}

func (c *Client) GetDiskUsageMetrics() (*DiskUsageMetrics, error) {
	metricViews, err := c.getMetrics("load-average")
	if err != nil {
		return nil, err
	}

	metrics := DiskUsageMetrics{}
	for _, series := range metricViews.Series {
		switch series.Name {
		case "postgres_databases_size_bytes":
			metrics.DatabaseSize = uint64(average(series.Points))
		}
	}

	return &metrics, err
}

func average(points []MetricPoint) float64 {
	sum := 0.0
	for _, point := range points {
		sum += point.Value
	}
	return sum / float64(len(points))
}
