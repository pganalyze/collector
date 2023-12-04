package querysample

import (
	"reflect"
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
		map[string]string{},
	},
	{
		"Query tag with key:value (marginalia) shape",
		"SELECT 1 /* abc:123, def:456 */",
		map[string]string{"abc": "123", "def": "456"},
	},
	{
		"Query tag with complex key:value (marginalia) shape",
		"SELECT 1 /*controller_with_namespace:Api::V1::SubmittedInspectionFormsController,action:index,line:/config/initializers/kaminari_total_count.rb:60:in `total_count'*/",
		map[string]string{
			"controller_with_namespace": "Api::V1::SubmittedInspectionFormsController",
			"action":                    "index",
			"line":                      "/config/initializers/kaminari_total_count.rb:60:in `total_count'",
		},
	},
	{
		"Query tag with key=value shape",
		"SELECT 1 /* abc=123,def=456 */",
		map[string]string{"abc": "123", "def": "456"},
	},
	{
		"Query tag with key=value shape with valueless key (ignore)",
		"SELECT 1 /* hello=world,foo */",
		map[string]string{"hello": "world"},
	},
	{
		"Query tag with key=value shape with valueless key in the middle (ignore)",
		"SELECT 1 /* hello: world, foo, bar: 123 */",
		map[string]string{"hello": "world", "bar": "123"},
	},
	{
		"Comment inside string",
		"SELECT '/* not a comment */' /* a:42 */",
		map[string]string{"a": "42"},
	},
	{
		"Multiple comments",
		"/* a:1,b:2 */ SELECT 1 /* c:3,d:4 */",
		map[string]string{
			"a": "1",
			"b": "2",
			"c": "3",
			"d": "4",
		},
	},
	{
		"Multiple comments with conflicting keys",
		"/* a:1,b:2 */ SELECT 1 /* c:3,a:4 */",
		map[string]string{
			"a": "4",
			"b": "2",
			"c": "3",
		},
	},
	{
		"Comment inside string",
		"SELECT '/* not a comment */' /* a:42 */",
		map[string]string{"a": "42"},
	},
	{
		"Query tag with key='value' (sqlcommenter) shape",
		"SELECT 1 /* foo='bar%20quux' */",
		map[string]string{"foo": "bar quux"},
	},
	{
		"Query tag with complex key='value' (sqlcommenter) shape",
		"SELECT 1, 'string', '/* ignore */' /* foo='bar%20quux',fred='http://example.org/a%20b%20c\\'',thud%20thud%25thud\\'='\\'%25%20%25 %20' */",
		map[string]string{
			"foo":             "bar quux",
			"fred":            "http://example.org/a b c'",
			"thud thud%thud'": "'% %  ",
		},
	},
	{
		"Query tag with key:value (marginalia) shape, with traceparent and tracestate",
		"SELECT 1 /* traceparent:00-7dd3a87ae5bdacc0c56f3ba452a22fed-b39c2eabd3993833-01,tracestate:pganalyze=t:1701420562.550783 */",
		map[string]string{"traceparent": "00-7dd3a87ae5bdacc0c56f3ba452a22fed-b39c2eabd3993833-01", "tracestate": "pganalyze=t:1701420562.550783"},
	},
	{
		"Query tag with key='value' (sqlcommenter) shape, with traceparent and tracestate",
		"SELECT 1 /* traceparent='00-7dd3a87ae5bdacc0c56f3ba452a22fed-b39c2eabd3993833-01',tracestate='pganalyze=t:1701420562.550783' */",
		map[string]string{"traceparent": "00-7dd3a87ae5bdacc0c56f3ba452a22fed-b39c2eabd3993833-01", "tracestate": "pganalyze=t:1701420562.550783"},
	},
	{
		"Query tag with key='value' (sqlcommenter) shape, with traceparent and tracestate (URL escaped)",
		"SELECT 1 /* traceparent='00-7dd3a87ae5bdacc0c56f3ba452a22fed-b39c2eabd3993833-01',tracestate='pganalyze%3Dt%3A1701420562.550783' */",
		map[string]string{"traceparent": "00-7dd3a87ae5bdacc0c56f3ba452a22fed-b39c2eabd3993833-01", "tracestate": "pganalyze=t:1701420562.550783"},
	},
}

func TestParseTags(t *testing.T) {
	cfg := pretty.CompareConfig

	for _, pair := range parseTagsTests {
		tags := parseTags(pair.query)
		if !reflect.DeepEqual(pair.tags, tags) {
			t.Errorf("For %s: (-want +got)\n%s", pair.testName, cfg.Compare(pair.tags, tags))
		}
	}

}
