package querysample_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"regexp"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/pganalyze/collector/logs/querysample"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

type autoExplainQuerySampleTestpair struct {
	testName               string
	normalizedQueryTextOut string
	errOut                 error
}

var autoExplainQuerySampleTests = []autoExplainQuerySampleTestpair{
	// JSON examples
	{
		"json_simple",
		"SELECT abalance FROM pgbench_accounts WHERE aid = $1;",
		nil,
	},
	{
		"json_function_call",
		"/* pganalyze-collector */ \nSELECT dbid, userid, query, calls, total_exec_time, rows, shared_blks_hit, shared_blks_read,\n\t\t\t shared_blks_dirtied, shared_blks_written, local_blks_hit, local_blks_read,\n\t\t\t local_blks_dirtied, local_blks_written, temp_blks_read, temp_blks_written,\n\t\t\t blk_read_time, blk_write_time, queryid, min_exec_time, max_exec_time, mean_exec_time, stddev_exec_time\n\tFROM pganalyze.get_stat_statements()",
		nil,
	},
	{
		"json_insert_into_select",
		"INSERT INTO query_analysis_jobs (id) SELECT id FROM queries LEFT JOIN query_analyses ON query_analyses.query_id = queries.id WHERE queries.database_id = $1 AND last_occurred_at >= current_date - $2 AND statement_types && $3 AND statement_types <@ $4 AND (query_analyses.updated_at IS NULL OR query_analyses.updated_at < now() - interval $5) ORDER BY query_analyses.updated_at ASC NULLS FIRST LIMIT $6",
		nil,
	},
	{
		"json_parallel_plan",
		"SELECT \"query_explains_7d\".* FROM \"query_explains_7d\" INNER JOIN \"query_samples_7d\" USING (query_sample_id, query_fingerprint, database_id) WHERE \"query_explains_7d\".\"database_id\" = $1 AND \"query_explains_7d\".\"seen_at\" BETWEEN $2 AND $3 AND NOT (\"query_samples_7d\".\"query_text\" LIKE $4) ORDER BY seen_at DESC LIMIT $5;",
		nil,
	},
	{
		"json_incremental_sort",
		"select * from (select * from t order by a) s order by a, b limit $1 /* query from incremental_sort regression test, slightly modified output to model auto_explain */",
		nil,
	},
	{
		"json_memoize",
		"SELECT COUNT(*),AVG(t1.unique1) FROM tenk1 t1 INNER JOIN tenk1 t2 ON t1.unique1 = t2.twenty WHERE t2.unique1 < $1 /* query from memoize regression test, slightly modified output to model auto_explain */",
		nil,
	},
	{
		"json_tablesample",
		"SELECT count(*) FROM test_tablesample TABLESAMPLE SYSTEM ($1) REPEATABLE ($2+$3) /* query from tablesample regression test, slightly modified output to model auto_explain */",
		nil,
	},
	{
		"json_insert_conflict",
		"insert into insertconflicttest values ($1, $2) on conflict (key) do update set fruit = excluded.fruit where insertconflicttest.fruit != $3 returning *; /* query from insert_conflict regression test, slightly modified output to model auto_explain */",
		nil,
	},
	{
		"json_postgres_fdw",
		"SELECT * FROM local_tbl t1 LEFT JOIN (SELECT *, (SELECT count(*) FROM async_pt WHERE a < $1) FROM async_pt WHERE a < $2) t2 ON t1.a = t2.a; /* query from postgres_fdw regression test, slightly modified output to model auto_explain */",
		nil,
	},
	{
		"json_hash_aggregate",
		"select * from t1 inner join t2 on t1.a = t2.x and t1.b = t2.y group by t1.a,t1.b,t1.c,t1.d,t2.x,t2.y,t2.z; /* query from aggregates regression test, slightly modified output to model auto_explain */",
		nil,
	},
	{
		"json_xml",
		"SELECT * FROM xmltableview1; /* query from xml regression test, slightly modified output to model auto_explain */",
		nil,
	},
	{
		"json_groupingset_sorting",
		"select g100, g10, sum(g::numeric), count(*), max(g::text) from gs_data_1 group by cube (g1000, g100,g10); /* query from groupingsets regression test, slightly modified output to model auto_explain */",
		nil,
	},
	{
		"json_groupingset_hashagg",
		"select g100, g10, sum(g::numeric), count(*), max(g::text) from gs_data_1 group by cube (g1000, g100,g10); /* query from groupingsets regression test, slightly modified output to model auto_explain */",
		nil,
	},
	{
		"json_parallel_params_evaluated",
		"SELECT unique1 FROM tenk1 WHERE fivethous = (SELECT unique1 FROM tenk1 WHERE fivethous = $1 LIMIT $2) UNION ALL SELECT unique1 FROM tenk1 WHERE fivethous = (SELECT unique2 FROM tenk1 WHERE fivethous = $3 LIMIT $4) ORDER BY 1; /* query from select_parallel regression test, slightly modified output to model auto_explain */",
		nil,
	},
	{
		"json_union",
		"SELECT q2 FROM int8_tbl INTERSECT SELECT q1 FROM int8_tbl ORDER BY 1; /* query from union regression test, slightly modified output to model auto_explain */",
		nil,
	},
	{
		"json_with",
		"<unparsable query>",
		nil,
	},
	{
		"json_order_by",
		"SELECT rank() OVER (ORDER BY b <-> point $1) n, b <-> point $2 dist, id FROM quad_box_tbl; /* query from box regression test, slightly modified output to model auto_explain */",
		nil,
	},
	{
		"json_tidrangescan",
		"SELECT ctid FROM tidrangescan WHERE ctid < $1; /* query from tidrangescan regression test, slightly modified output to model auto_explain */",
		nil,
	},
	{
		"json_one_time_filter",
		"select * from int8_tbl t1 left join (select q1 as x, $1 as y from int8_tbl t2) ss on t1.q2 = ss.x where $2 = (select $3 from int8_tbl t3 where ss.y is not null limit $4) order by 1,2; /* query from join regression test, slightly modified output to model auto_explain */",
		nil,
	},
	{
		"json_custom_plan_citus",
		"SELECT l_quantity, count(*) count_quantity FROM lineitem GROUP BY l_quantity ORDER BY count_quantity, l_quantity; /* from https://github.com/citusdata/citus/blob/ace800851a88d691f694c86244dcccd72ea90d1d/src/test/regress/expected/multi_explain.out#L93 */",
		nil,
	},
	{
		"json_queryid",
		"select * from int8_tbl i8 /* query from explain regression test, slightly modified output to model auto_explain, and ensuring the Query Identifier can pass as a float64 (as our test logic unmarshals into an interface) */",
		nil,
	},
	{
		"json_trigger",
		"insert into parted_constr values ($1, $2); /* query from triggers regression test, slightly modified output to model auto_explain */",
		nil,
	},
	{
		"json_jit",
		"SELECT pg_sleep($1);",
		nil,
	},
	{
		"json_file_fdw",
		"SELECT * FROM agg_csv; /* query from file_fdw regression test, slightly modified output to model auto_explain */",
		nil,
	},
	// unsupported formats
	{
		"bad_format",
		"",
		fmt.Errorf("unsupported auto_explain format"),
	},
}

func TestAutoExplainQuerySample(t *testing.T) {
	cfg := pretty.CompareConfig
	cfg.SkipZeroFields = true
	jsonNewlineWhitespaceRegexp := regexp.MustCompile(`\s*\n\s*`)
	jsonKeyValueSeparatorRegexp := regexp.MustCompile(`": `)

	for _, pair := range autoExplainQuerySampleTests {
		explainIn, err := ioutil.ReadFile("testdata/" + pair.testName + ".in.json")
		if err != nil {
			t.Fatalf("Error loading input file: %s", err)
		}
		for _, variant := range []string{"normalize", "passthrough"} {
			fmt.Printf("%v (%s)\n", pair.testName, variant)
			sample, sampleErr := querysample.TransformAutoExplainToQuerySample(state.LogLine{}, string(explainIn), "0.123")
			if sampleErr == nil {
				if variant == "normalize" {
					sample.Query = util.NormalizeQuery(sample.Query, "unparseable", -1)
					sample.ExplainOutputJSON, err = querysample.NormalizeExplainJSON(sample.ExplainOutputJSON)
					if err != nil {
						t.Fatalf("For %v (%s): Error normalizing: %s", pair.testName, variant, err)
					}
				}
				explainOut, err := ioutil.ReadFile("testdata/" + pair.testName + ".out_" + variant + ".json")
				if err != nil {
					t.Fatalf("For %v (%s): Error loading output file: %s", pair.testName, variant, err)
				}

				marshaledOutput, err := json.Marshal(sample.ExplainOutputJSON)
				if err != nil {
					t.Fatalf("For %v (%s): Error marshaling sample JSON: %s", pair.testName, variant, err)
				}
				if diff := cfg.Compare(jsonKeyValueSeparatorRegexp.ReplaceAllString(jsonNewlineWhitespaceRegexp.ReplaceAllString(string(explainOut), ""), "\":"), "["+string(marshaledOutput)+"]"); diff != "" {
					t.Errorf("For %v (%s): text diff: (-want +got)\n%s", pair.testName, variant, diff)
				}

				// Run expected output through unmarshal+marshal to match whitespace to the actual output
				// Note that we also need to unmarshal the sample because we want to compare []interface{} against []interface{}
				var explainOutJson []interface{}
				var sampleJson []interface{}
				err = json.Unmarshal(explainOut, &explainOutJson)
				if err != nil {
					t.Fatalf("For %v (%s): Error unmarshaling output file: %s", pair.testName, variant, err)
				}
				err = json.Unmarshal([]byte("["+string(marshaledOutput)+"]"), &sampleJson)
				if err != nil {
					t.Fatalf("For %v (%s): Error unmarshaling sample: %s", pair.testName, variant, err)
				}

				if diff := cfg.Compare(explainOutJson, sampleJson); diff != "" {
					t.Errorf("For %v (%s): parsed json diff: (-want +got)\n%s", pair.testName, variant, diff)
				}
			}

			if diff := cfg.Compare(pair.errOut, sampleErr); diff != "" {
				t.Errorf("For %v: error diff: (-want +got)\n%s", pair.testName, diff)
			}

			if variant == "normalize" {
				if diff := cfg.Compare(string(pair.normalizedQueryTextOut), sample.Query); diff != "" {
					t.Errorf("For %v: query text diff: (-want +got)\n%s", pair.testName, diff)
				}
			}
		}
	}
}
