package runner

import (
	"testing"
	"time"

	"github.com/pganalyze/collector/config"
)

func TestLogTestTimeout(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.ServerConfig
		want time.Duration
	}{
		{name: "default when unset", cfg: config.ServerConfig{}, want: 10 * time.Second},
		{name: "self_hosted uses default", cfg: config.ServerConfig{SystemType: "self_hosted"}, want: 10 * time.Second},
		{name: "supabase gets a longer default", cfg: config.ServerConfig{SystemType: "supabase"}, want: 30 * time.Second},
		{name: "explicit override wins", cfg: config.ServerConfig{LogTestTimeoutSecs: 45}, want: 45 * time.Second},
		{name: "explicit override wins over supabase default", cfg: config.ServerConfig{SystemType: "supabase", LogTestTimeoutSecs: 11}, want: 11 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := logTestTimeout(tt.cfg); got != tt.want {
				t.Errorf("logTestTimeout() = %v, want %v", got, tt.want)
			}
		})
	}
}
