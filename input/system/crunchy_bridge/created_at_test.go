package crunchy_bridge

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

type mockTransport func(*http.Request) (*http.Response, error)

func (f mockTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGetSystemStateClusterCreatedAt(t *testing.T) {
	const created = "2026-11-01T12:34:56Z"
	client := &http.Client{Transport: mockTransport(func(r *http.Request) (*http.Response, error) {
		body := `{"series":[]}`
		if strings.HasPrefix(r.URL.Path, "/clusters/") {
			body = `{"created_at":"` + created + `","name":"example","cpu":2,"memory":4,"storage":100}`
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
	server := state.MakeServer(config.ServerConfig{CrunchyBridgeAPIKey: "fake-test-key", CrunchyBridgeClusterID: "example", HTTPClientWithRetry: client}, false)
	got := GetSystemState(context.Background(), server, &util.Logger{})
	want, _ := time.Parse(time.RFC3339, created)
	if got.Info.CrunchyBridge == nil || !got.Info.CrunchyBridge.CreatedAt.Equal(want) {
		t.Fatalf("API returned created_at=%s; collected state=%+v", created, got.Info.CrunchyBridge)
	}
}
