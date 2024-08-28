package crunchy_bridge

import (
	"context"
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
	MemoryUsedPct float64
	SwapUsedPct   float64
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
	LogSize      uint64
	WalSize      uint64
}

func (c *Client) NewRequest(ctx context.Context, method string, path string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.BearerToken)
	req.Header.Set("User-Agent", util.CollectorNameAndVersion)
	req.Header.Add("Accept", "application/json")
	return req, nil
}

func (c *Client) GetClusterInfo(ctx context.Context) (*ClusterInfo, error) {
	req, err := c.NewRequest(ctx, "GET", "/clusters/"+c.ClusterID)
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
		return nil, fmt.Errorf("unexpected status code: %d, response body: %s", resp.StatusCode, string(body))
	}

	clusterInfo := ClusterInfo{}
	err = json.Unmarshal(body, &clusterInfo)
	if err != nil {
		return nil, err
	}
	return &clusterInfo, err
}

func (c *Client) getMetrics(ctx context.Context, name string) (*MetricViews, error) {
	req, err := c.NewRequest(ctx, "GET", fmt.Sprintf("/metric-views/%s?cluster_id=%s&period=15m", name, c.ClusterID))
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

func (c *Client) GetCPUMetrics(ctx context.Context) (*CPUMetrics, error) {
	metricViews, err := c.getMetrics(ctx, "cpu")
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

func (c *Client) GetMemoryMetrics(ctx context.Context) (*MemoryMetrics, error) {
	metricViews, err := c.getMetrics(ctx, "memory")
	if err != nil {
		return nil, err
	}

	metrics := MemoryMetrics{}
	for _, series := range metricViews.Series {
		switch series.Name {
		case "memory_used":
			metrics.MemoryUsedPct = average(series.Points)
		case "swap_used":
			metrics.SwapUsedPct = average(series.Points)
		}
	}

	return &metrics, err
}

func (c *Client) GetIOPSMetrics(ctx context.Context) (*IOPSMetrics, error) {
	metricViews, err := c.getMetrics(ctx, "iops")
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

func (c *Client) GetLoadAverageMetrics(ctx context.Context) (*LoadAverageMetrics, error) {
	metricViews, err := c.getMetrics(ctx, "load-average")
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

func (c *Client) GetDiskUsageMetrics(ctx context.Context) (*DiskUsageMetrics, error) {
	metricViews, err := c.getMetrics(ctx, "disk-usage")
	if err != nil {
		return nil, err
	}

	metrics := DiskUsageMetrics{}
	for _, series := range metricViews.Series {
		switch series.Name {
		case "postgres_databases_size_bytes":
			metrics.DatabaseSize = uint64(average(series.Points))
		case "postgres_log_size_bytes":
			metrics.LogSize = uint64(average(series.Points))
		case "postgres_wal_size_bytes":
			metrics.WalSize = uint64(average(series.Points))
		}
	}

	return &metrics, err
}

func average(points []MetricPoint) float64 {
	// With metric-views endpoint, it returns metrics for the last 15 minutes
	// The latest data point(s) often returns value 0 as there is some lag within the metrics collection on Crunchy side
	// When calculating average, ignore value 0
	// Note that this will also ignore the actual 0 value too (e.g. load average),
	// though average of such points should be close to zero anyways so ignore them to simplify
	var sum float64
	var count float64
	for _, point := range points {
		if point.Value != 0 {
			sum += point.Value
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / count
}
