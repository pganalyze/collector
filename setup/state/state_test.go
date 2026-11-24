package state

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-ini/ini"
	"github.com/guregu/null"
)

type reportedStep struct {
	apiKey   string
	lastStep string
	success  string
}

func runReportStep(t *testing.T, configText string, inputs SetupInputs) (reports []reportedStep, serverURL string) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v2/setup/guided_setup" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		reports = append(reports, reportedStep{
			apiKey:   r.Header.Get("Pganalyze-Api-Key"),
			lastStep: r.FormValue("last_step"),
			success:  r.FormValue("success"),
		})
	}))
	defer server.Close()

	setupState := SetupState{Inputs: &inputs}
	if configText != "" {
		conf, err := ini.Load([]byte(configText + "\napi_base_url = " + server.URL + "\n"))
		if err != nil {
			t.Fatal(err)
		}
		setupState.Config = conf
		setupState.PGAnalyzeSection = conf.Section("pganalyze")
	} else if inputs.Settings.APIKey.Valid {
		setupState.Inputs.Settings.APIBaseURL = null.StringFrom(server.URL)
	}
	setupState.ReportStep("test_step", nil)
	return reports, server.URL
}

func TestReportStepUsesAPIKeyFromConfig(t *testing.T) {
	// This is what the install script produces: the API key is written to the
	// config file, and not passed to the setup command
	reports, _ := runReportStep(t, "[pganalyze]\napi_key = config_key", SetupInputs{})
	if len(reports) != 1 {
		t.Fatalf("expected 1 report, got %d", len(reports))
	}
	expected := reportedStep{apiKey: "config_key", lastStep: "test_step", success: "true"}
	if reports[0] != expected {
		t.Errorf("expected %+v, got %+v", expected, reports[0])
	}
}

func TestReportStepPrefersAPIKeyFromInputs(t *testing.T) {
	inputs := SetupInputs{}
	inputs.Settings.APIKey = null.StringFrom("input_key")
	reports, _ := runReportStep(t, "[pganalyze]\napi_key = config_key", inputs)
	if len(reports) != 1 {
		t.Fatalf("expected 1 report, got %d", len(reports))
	}
	if reports[0].apiKey != "input_key" {
		t.Errorf("expected input_key, got %s", reports[0].apiKey)
	}
}

func TestReportStepWithoutConfigLoaded(t *testing.T) {
	inputs := SetupInputs{}
	inputs.Settings.APIKey = null.StringFrom("input_key")
	reports, _ := runReportStep(t, "", inputs)
	if len(reports) != 1 || reports[0].apiKey != "input_key" {
		t.Errorf("expected 1 report with input_key, got %+v", reports)
	}
}

func TestReportStepSkippedWithoutAPIKey(t *testing.T) {
	reports, _ := runReportStep(t, "[pganalyze]\n", SetupInputs{})
	if len(reports) != 0 {
		t.Errorf("expected no reports, got %+v", reports)
	}
	// No config loaded yet, and no API key passed
	reports, _ = runReportStep(t, "", SetupInputs{})
	if len(reports) != 0 {
		t.Errorf("expected no reports, got %+v", reports)
	}
}
