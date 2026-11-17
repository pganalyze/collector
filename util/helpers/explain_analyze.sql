CREATE OR REPLACE FUNCTION pganalyze.explain_analyze(query text, params text[], param_types text[], analyze_flags text[]) RETURNS text AS $$
DECLARE
  prepared_query pg_catalog.text;
  params_str pg_catalog.text;
  param_types_str pg_catalog.text;
  explain_prefix pg_catalog.text;
  explain_flag pg_catalog.text;
  result pg_catalog.text;
BEGIN
  SET TRANSACTION READ ONLY;

  PERFORM 1 FROM pg_catalog.pg_roles WHERE (rolname OPERATOR(pg_catalog.=) current_user AND rolsuper) OR (pg_catalog.pg_has_role(oid, 'MEMBER') AND rolname OPERATOR(pg_catalog.=) ANY (ARRAY['rds_superuser', 'azure_pg_admin', 'cloudsqlsuperuser']::pg_catalog.name[]));
  IF FOUND THEN
    RAISE EXCEPTION 'cannot run: pganalyze.explain_analyze helper is owned by superuser - recreate function with lesser privileged user';
  END IF;

  SELECT pg_catalog.regexp_replace(query, ';+\s*\Z', '') INTO prepared_query;
  IF prepared_query OPERATOR(pg_catalog.~~) '%;%' THEN
    RAISE EXCEPTION 'cannot run pganalyze.explain_analyze helper with a multi-statement query';
  END IF;

  explain_prefix := 'EXPLAIN (VERBOSE, FORMAT JSON';
  FOR explain_flag IN SELECT * FROM pg_catalog.unnest(analyze_flags)
  LOOP
    -- Equivalent to NOT SIMILAR TO '[A-z_ ]+', written as a regular expression to allow qualifying the operator
    IF explain_flag OPERATOR(pg_catalog.!~) '^(?:[A-z_ ]+)$' THEN
      RAISE EXCEPTION 'cannot run pganalyze.explain_analyze helper with invalid flag';
    END IF;
    explain_prefix := explain_prefix OPERATOR(pg_catalog.||) ', ' OPERATOR(pg_catalog.||) explain_flag;
  END LOOP;
  explain_prefix := explain_prefix OPERATOR(pg_catalog.||) ') ';

  IF pg_catalog.cardinality(params) OPERATOR(pg_catalog.>) 0 THEN
    SELECT '(' OPERATOR(pg_catalog.||) pg_catalog.array_to_string(
      ARRAY(
        SELECT pg_catalog.quote_literal(p)
        FROM pg_catalog.unnest(params) _(p)
      ),
      ',',
      'NULL'
    ) OPERATOR(pg_catalog.||) ')' INTO params_str;
  ELSE
    SELECT '' INTO params_str;
  END IF;
  SELECT COALESCE('(' OPERATOR(pg_catalog.||) pg_catalog.string_agg(
    CASE
      WHEN p OPERATOR(pg_catalog.~) '^[a-z_][a-z0-9_]*(\[\])?$' THEN p
      ELSE pg_catalog.quote_ident(p)
    END,
    ','
  ) OPERATOR(pg_catalog.||) ')', '') FROM pg_catalog.unnest(param_types) _(p) INTO param_types_str;

  EXECUTE 'PREPARE pganalyze_explain_analyze ' OPERATOR(pg_catalog.||) param_types_str OPERATOR(pg_catalog.||) ' AS ' OPERATOR(pg_catalog.||) prepared_query;
  BEGIN
    EXECUTE explain_prefix OPERATOR(pg_catalog.||) 'EXECUTE pganalyze_explain_analyze' OPERATOR(pg_catalog.||) params_str INTO STRICT result;
  EXCEPTION WHEN QUERY_CANCELED OR OTHERS THEN
    DEALLOCATE pganalyze_explain_analyze;
    RAISE;
  END;
  DEALLOCATE pganalyze_explain_analyze;

  RETURN result;
END
$$ LANGUAGE plpgsql VOLATILE SECURITY DEFINER;
