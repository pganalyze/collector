package runner

import (
	"testing"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/state"
)

func TestMemberServerConfig(t *testing.T) {
	sectionCfg := config.ServerConfig{
		APIKey:              "key1",
		APIBaseURL:          "https://api.pganalyze.com",
		SectionName:         "prod",
		DbHost:              "mycluster.cluster-abc123.us-east-1.rds.amazonaws.com",
		DbUsername:          "pganalyze",
		DbPassword:          "secret",
		DbName:              "mydb",
		AwsRegion:           "us-east-1",
		AwsAccountID:        "abc123",
		AwsDbClusterID:      "mycluster",
		AwsDbClusterMembers: "all",
		SystemID:            "mycluster",
		SystemType:          "amazon_rds",
		SystemScope:         "us-east-1/cluster-abc123",
	}

	writer := memberServerConfig(sectionCfg, "mycluster", "mycluster-instance-1", "mycluster-instance-1.abc123.us-east-1.rds.amazonaws.com", 5432, true)
	if writer.SectionName != "prod/mycluster-instance-1" {
		t.Errorf("unexpected section name: %s", writer.SectionName)
	}
	if writer.DbHost != "mycluster-instance-1.abc123.us-east-1.rds.amazonaws.com" || writer.DbPort != 5432 {
		t.Errorf("unexpected connection endpoint: %s:%d", writer.DbHost, writer.DbPort)
	}
	if writer.DbUsername != "pganalyze" || writer.DbPassword != "secret" || writer.DbName != "mydb" {
		t.Errorf("unexpected connection settings: %s/%s", writer.DbUsername, writer.DbName)
	}
	if writer.AwsDbInstanceID != "mycluster-instance-1" {
		t.Errorf("unexpected instance ID: %s", writer.AwsDbInstanceID)
	}
	if writer.SystemID != "mycluster-instance-1" || writer.SystemType != "amazon_rds" || writer.SystemScope != "us-east-1/abc123" {
		t.Errorf("unexpected identity: %s / %s / %s", writer.SystemID, writer.SystemType, writer.SystemScope)
	}
	// The writer takes over the identity previously used by the cluster-level config as fallback
	if writer.SystemIDFallback != "mycluster" || writer.SystemTypeFallback != "amazon_rds" || writer.SystemScopeFallback != "us-east-1/cluster-abc123" {
		t.Errorf("unexpected fallback identity: %s / %s / %s", writer.SystemIDFallback, writer.SystemTypeFallback, writer.SystemScopeFallback)
	}
	if writer.Identifier.SystemID != "mycluster-instance-1" || writer.Identifier.SystemScope != "us-east-1/abc123" || writer.Identifier.APIKey != "key1" {
		t.Errorf("unexpected identifier: %+v", writer.Identifier)
	}

	reader := memberServerConfig(sectionCfg, "mycluster", "mycluster-instance-2", "mycluster-instance-2.abc123.us-east-1.rds.amazonaws.com", 5432, false)
	if reader.SystemIDFallback != "" || reader.SystemTypeFallback != "" || reader.SystemScopeFallback != "" {
		t.Errorf("expected no fallback identity for reader, got: %s / %s / %s", reader.SystemIDFallback, reader.SystemTypeFallback, reader.SystemScopeFallback)
	}
	if reader.Identifier == writer.Identifier {
		t.Errorf("expected distinct identifiers for writer and reader")
	}
}

func TestMemberServerConfigDbURL(t *testing.T) {
	sectionCfg := config.ServerConfig{
		SectionName:         "prod",
		DbURL:               "postgres://urluser:urlpass@mycluster.cluster-abc123.us-east-1.rds.amazonaws.com:6432/urldb?sslmode=verify-full",
		AwsRegion:           "us-east-1",
		AwsAccountID:        "abc123",
		AwsDbClusterID:      "mycluster",
		AwsDbClusterMembers: "all",
	}

	member := memberServerConfig(sectionCfg, "mycluster", "instance-1", "instance-1.abc123.us-east-1.rds.amazonaws.com", 5432, true)
	if member.DbURL != "" {
		t.Errorf("expected db_url to be cleared, got: %s", member.DbURL)
	}
	if member.DbHost != "instance-1.abc123.us-east-1.rds.amazonaws.com" || member.DbPort != 5432 {
		t.Errorf("unexpected connection endpoint: %s:%d", member.DbHost, member.DbPort)
	}
	if member.DbUsername != "urluser" || member.DbPassword != "urlpass" || member.DbName != "urldb" {
		t.Errorf("unexpected connection settings from db_url: %s/%s", member.DbUsername, member.DbName)
	}
	if member.DbSslMode != "verify-full" {
		t.Errorf("unexpected sslmode from db_url: %s", member.DbSslMode)
	}

	// Explicitly configured settings take precedence over db_url parts
	sectionCfg.DbUsername = "explicituser"
	sectionCfg.DbSslMode = "require"
	member = memberServerConfig(sectionCfg, "mycluster", "instance-1", "instance-1.abc123.us-east-1.rds.amazonaws.com", 5432, true)
	if member.DbUsername != "explicituser" || member.DbSslMode != "require" {
		t.Errorf("expected explicit settings to take precedence, got: %s / %s", member.DbUsername, member.DbSslMode)
	}
}

func TestApplyRemovalHysteresis(t *testing.T) {
	cfgA := config.ServerConfig{SectionName: "prod/a", Identifier: config.ServerIdentifier{SystemID: "a"}}
	cfgB := config.ServerConfig{SectionName: "prod/b", Identifier: config.ServerIdentifier{SystemID: "b"}}

	serverA := state.MakeServer(cfgA, false)
	serverB := state.MakeServer(cfgB, false)
	current := []*state.Server{serverA, serverB}

	missCounts := make(map[config.ServerIdentifier]int)

	// First run without server B: it must be kept
	desired := applyRemovalHysteresis([]config.ServerConfig{cfgA}, current, missCounts)
	if len(desired) != 2 {
		t.Errorf("expected server B to be kept after one absence, got %d configs", len(desired))
	}

	// Server B comes back: the miss count must reset
	desired = applyRemovalHysteresis([]config.ServerConfig{cfgA, cfgB}, current, missCounts)
	if len(desired) != 2 {
		t.Errorf("expected both servers, got %d configs", len(desired))
	}
	if missCounts[cfgB.Identifier] != 0 {
		t.Errorf("expected miss count to reset, got %d", missCounts[cfgB.Identifier])
	}

	// Absent on two consecutive runs: removed
	desired = applyRemovalHysteresis([]config.ServerConfig{cfgA}, current, missCounts)
	if len(desired) != 2 {
		t.Errorf("expected server B to be kept after one absence, got %d configs", len(desired))
	}
	desired = applyRemovalHysteresis([]config.ServerConfig{cfgA}, current, missCounts)
	if len(desired) != 1 {
		t.Errorf("expected server B to be removed after two absences, got %d configs", len(desired))
	}
}
