package state

import "testing"

func TestHasNov2026SecurityFixes(t *testing.T) {
	tests := []struct {
		numeric  int
		expected bool
	}{
		{180006, false},
		{180007, true},
		{180010, true},
		{170011, false},
		{170012, true},
		{160015, false},
		{160016, true},
		{150019, false},
		{150020, true},
		{140024, false},
		{140025, true},
		{130022, false},
		{90624, false},
		{180000, false}, // 18 pre-releases
		{190000, true},  // newer major versions, including pre-releases
		{200001, true},
	}

	for _, test := range tests {
		v := PostgresVersion{Numeric: test.numeric}
		if actual := v.HasNov2026SecurityFixes(); actual != test.expected {
			t.Errorf("HasNov2026SecurityFixes() for %d = %v, want %v", test.numeric, actual, test.expected)
		}
	}
}

func TestHasNov2026SecurityFixesAurora(t *testing.T) {
	tests := []struct {
		numeric       int
		isAwsAurora   bool
		auroraVersion string
		expected      bool
	}{
		{180006, true, "18.6.2", true},
		{180006, true, "18.6.3", true},
		{180006, true, "18.6.1", false},
		{180006, true, "18.6", false},
		{180006, true, "", false},
		{180006, true, "garbage", false},
		{180006, false, "18.6.2", false}, // only trusted when Aurora is detected
		{180007, true, "18.7.0", true},   // regular minor version check still applies
		{170011, true, "17.11.2", true},
		{170011, true, "17.11.1", false},
		{160015, true, "16.15.2", true},
		{160015, true, "16.15.1", false},
		{150019, true, "15.19.2", true},
		{150019, true, "15.19.1", false},
		{140024, true, "14.24.2", true},
		{140024, true, "14.24.1", false},
		{160013, true, "16.13.5", false}, // older minor releases did not receive these fixes
		{130022, true, "13.22.9", false},
	}

	for _, test := range tests {
		v := PostgresVersion{Numeric: test.numeric, IsAwsAurora: test.isAwsAurora, AuroraVersion: test.auroraVersion}
		if actual := v.HasNov2026SecurityFixes(); actual != test.expected {
			t.Errorf("HasNov2026SecurityFixes() for %d (aurora=%v, %q) = %v, want %v", test.numeric, test.isAwsAurora, test.auroraVersion, actual, test.expected)
		}
	}
}
