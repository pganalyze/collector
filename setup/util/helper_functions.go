package util

import (
	"crypto/md5"
	"fmt"
	"io"

	"github.com/lib/pq"
	"github.com/pganalyze/collector/setup/query"
)

type PGHelperFn struct {
	// function name
	name string
	// everything before prosrc in function definition sql, including
	// the opening quote before the function source content starts
	head string
	// the function body, prosrc
	body string
	// everything after prosrc, including the closing quote at the
	// beginning
	tail string
}

func (pgfn *PGHelperFn) GetDefinition() string {
	return pgfn.head + pgfn.body + pgfn.tail
}

func (pgfn *PGHelperFn) Matches(md5hash string) bool {
	h := md5.New()
	io.WriteString(h, pgfn.body)
	expected := fmt.Sprintf("%x", h.Sum(nil))
	return md5hash == expected
}

// The helper intentionally runs with the caller's search_path (so queries resolve application
// tables), so all objects it references itself are schema-qualified, including operators. This
// avoids picking up types from the caller's temporary schema (searched first for relations and
// types), as well as functions or operators from schemas the caller put ahead of pg_catalog.
var ExplainHelper = PGHelperFn{
	name: "explain",
	head: "CREATE OR REPLACE FUNCTION pganalyze.explain(query text, params text[]) RETURNS text AS $$",
	body: `DECLARE
	prepared_query pg_catalog.text;
	prepared_params pg_catalog.text;
	result pg_catalog.text;
BEGIN
	SELECT pg_catalog.regexp_replace(query, ';+\s*\Z', '') INTO prepared_query;
	IF prepared_query OPERATOR(pg_catalog.~~) '%;%' THEN
		RAISE EXCEPTION 'cannot run EXPLAIN when query contains semicolon';
	END IF;

	IF pg_catalog.array_length(params, 1) OPERATOR(pg_catalog.>) 0 THEN
		SELECT pg_catalog.array_to_string(
			ARRAY(
				SELECT pg_catalog.quote_literal(param) OPERATOR(pg_catalog.||) '::pg_catalog.unknown' FROM pg_catalog.unnest(params) p(param)
			),
			',',
			'NULL'
		) INTO prepared_params;

		EXECUTE 'PREPARE pganalyze_explain AS ' OPERATOR(pg_catalog.||) prepared_query;
		BEGIN
			EXECUTE 'EXPLAIN (VERBOSE, FORMAT JSON) EXECUTE pganalyze_explain(' OPERATOR(pg_catalog.||) prepared_params OPERATOR(pg_catalog.||) ')' INTO STRICT result;
		EXCEPTION WHEN OTHERS THEN
			DEALLOCATE pganalyze_explain;
			RAISE;
		END;
		DEALLOCATE pganalyze_explain;
	ELSE
		EXECUTE 'EXPLAIN (VERBOSE, FORMAT JSON) ' OPERATOR(pg_catalog.||) prepared_query INTO STRICT result;
	END IF;

	RETURN result;
END`,
	tail: "$$ LANGUAGE plpgsql VOLATILE SECURITY DEFINER;",
}

func ValidateHelperFunction(fn PGHelperFn, runner *query.Runner) (bool, error) {
	row, err := runner.QueryRow(
		fmt.Sprintf(
			`SELECT md5(btrim(prosrc, E' \\n\\r\\t'))
FROM pg_proc INNER JOIN pg_user ON (pg_proc.proowner = pg_user.usesysid)
WHERE proname = %s
	AND pronamespace::regnamespace::text = 'pganalyze'
	AND prosecdef
  AND pg_user.usesuper`,
			pq.QuoteLiteral(fn.name),
		),
	)
	if err == query.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	}
	actual := row.GetString(0)
	return fn.Matches(actual), nil
}
