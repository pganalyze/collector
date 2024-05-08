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
		[]string{"app==myapp"},
		false,
	},
	{
		[]string{"app!=myapp"},
		true,
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
}

func TestCheckLabelSelectorMismatch(t *testing.T) {
	for _, test := range checkLabelSelectorMismatchTests {
		actual := util.CheckLabelSelectorMismatch(labels, test.selectors)
		if actual != test.expected {
			t.Errorf("CheckLabelSelectorMismatch(%s): expected: %t; actual: %t", test.selectors, test.expected, actual)
		}
	}
}
