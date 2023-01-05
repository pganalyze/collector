package config

import (
	"testing"
)

type aivenTestItem struct {
	input        string
	expectedSvc  string
	expectedProj string
}

var aivenTests = []aivenTestItem{
	{"my-service-myproject.aivencloud.com", "my-service", "myproject"},
	{"myservice-myproject.aivencloud.com", "myservice", "myproject"},
	// probably not what's expected, but this can be overridden manually, and if
	// project names with dashes do exist, there's not much we can do to
	// disambiguate
	{"my-service-my-project.aivencloud.com", "my-service-my", "project"},
}

func TestPreprocessConfigAiven(t *testing.T) {
	for idx, item := range aivenTests {
		var config ServerConfig
		config.DbHost = item.input
		processed, err := preprocessConfig(&config)
		if err != nil {
			t.Errorf("%d: want nil; got %v", idx, err)
		}

		if processed.AivenServiceID != item.expectedSvc {
			t.Errorf("%d: want %v; got %v", idx, item.expectedSvc, processed.AivenServiceID)
		}
		if processed.AivenProjectID != item.expectedProj {
			t.Errorf("%d: want %v; got %v", idx, item.expectedProj, processed.AivenProjectID)
		}
	}

	{
		// ensure we avoid overwriting explicitly-specified config values
		var config ServerConfig
		config.DbHost = "my-service-my-project.aivencloud.com"
		config.AivenServiceID = "my-service"
		config.AivenProjectID = "my-project"
		processed, err := preprocessConfig(&config)
		if err != nil {
			t.Errorf("want nil; got %v", err)
		}
		if processed.AivenServiceID != "my-service" {
			t.Errorf("want %v; got %v", "my-service", processed.AivenServiceID)
		}
		if processed.AivenProjectID != "my-project" {
			t.Errorf("want %v; got %v", "my-project", processed.AivenProjectID)
		}
	}

}
