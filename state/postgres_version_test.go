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
