package util_test

import (
	"testing"

	"github.com/pganalyze/collector/util"
)

var labels = map[string]string{
	"app":       "myapp",
	"component": "server",
	"tier":      "backend",
}

var checkLabelSelectorMismatchTests = []struct {
	selectors []string
	expected  bool
}{
	{
		[]string{"app=value1"},
		true,
	},
	{
		[]string{"app!=value1"},
		false,
	},
	{
		[]string{"app=myapp"},
		false,
	},
	{
		[]string{"app = myapp"},
		false,
	},
	{
		[]string{"app==myapp"},
		false,
	},
	{
		[]string{"app!=myapp"},
		true,
	},
	{
		// false due to the wrong format
		[]string{"app!==myapp"},
		false,
	},
	{
		[]string{"app=myapp", "component=server"},
		false,
	},
	{
		[]string{"app=myapp", "component=server", "tier=backend"},
		false,
	},
	{
		[]string{"app=myapp", "component=server", "tier=frontend"},
		true,
	},
	{
		[]string{"app=myapp", "component=server", "tier!=frontend"},
		false,
	},
	// Equality match not present in labels - should mismatch
	{
		[]string{"nonexistent=value"},
		true,
	},
	// Inequality match not present - should match
	{
		[]string{"nonexistent!=value"},
		false,
	},
	// Multiple selectors where first matches but second key is missing - should mismatch
	{
		[]string{"app=myapp", "nonexistent=value"},
		true,
	},
	// Multiple selectors where first key is missing but second matches - should mismatch
	{
		[]string{"nonexistent=value", "app=myapp"},
		true,
	},
	// All selectors match existing labels - should match
	{
		[]string{"app=myapp", "tier=backend"},
		false,
	},
	// One of multiple selectors has wrong value - should mismatch
	{
		[]string{"app=myapp", "tier=frontend"},
		true,
	},
}

func TestCheckLabelSelectorMismatch(t *testing.T) {
	for _, test := range checkLabelSelectorMismatchTests {
		actual := util.CheckLabelSelectorMismatch(labels, test.selectors)
		if actual != test.expected {
			t.Errorf("CheckLabelSelectorMismatch(%s): expected: %t; actual: %t", test.selectors, test.expected, actual)
		}
	}
}
