package querysample

import (
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

type parseTagsTestpair struct {
	testName string
	query    string
	tags     map[string]string
}

var parseTagsTests = []parseTagsTestpair{
	{
		"No query tags",
		"SELECT 1",
		nil,
	},
	{
		"Query tag with key:value shape",
		"/*key1:value1,key2:value2*/ SELECT 1",
		map[string]string{"key1": "value1", "key2": "value2"},
	},
	{
		"Query tag with key=value shape",
		"/*key1=value1,key2=value2*/ SELECT 1",
		map[string]string{"key1": "value1", "key2": "value2"},
	},
	{
		"Query tag with key='value' shape",
		"/*key1='value1',key2='value2'*/ SELECT 1",
		map[string]string{"key1": "value1", "key2": "value2"},
	},
	{
		"Query tag with key='value' shape with meta characters and URL encoded",
		"/*key1='value1',key2='%2Fparam%20\\'first\\''*/ SELECT 1",
		map[string]string{"key1": "value1", "key2": "/param 'first'"},
	},
	{
		"Bad query",
		"SELECT BAD QUERY",
		nil,
	},
}

func TestParseTags(t *testing.T) {
	cfg := pretty.CompareConfig

	for _, pair := range parseTagsTests {
		tags := parseTags(pair.query)
		if diff := cfg.Compare(pair.tags, tags); diff != "" {
			t.Errorf("For %s: (-want +got)\n%s", pair.testName, diff)
		}
	}

}
