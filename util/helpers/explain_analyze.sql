CREATE OR REPLACE FUNCTION pganalyze.explain_analyze(query text, params text[], param_types text[], analyze_flags text[]) RETURNS text AS $$
DECLARE
  prepared_query text;
  params_str text;
  params_str2 text;
  param_types_str text;
  explain_prefix text;
  explain_flag text;
  result text;
BEGIN
  SET TRANSACTION READ ONLY;

  PERFORM 1 FROM pg_roles WHERE (rolname = current_user AND rolsuper) OR (pg_has_role(oid, 'MEMBER') AND rolname IN ('rds_superuser', 'azure_pg_admin', 'cloudsqlsuperuser'));
  IF FOUND THEN
    RAISE EXCEPTION 'cannot run: pganalyze.explain_analyze helper is owned by superuser - recreate function with lesser privileged user';
  END IF;

  SELECT pg_catalog.regexp_replace(query, ';+\s*\Z', '') INTO prepared_query;
  IF prepared_query LIKE '%;%' THEN
    RAISE EXCEPTION 'cannot run pganalyze.explain_analyze helper with a multi-statement query';
  END IF;

  explain_prefix := 'EXPLAIN (VERBOSE, FORMAT JSON';
  FOR explain_flag IN SELECT * FROM unnest(analyze_flags)
  LOOP
    IF explain_flag NOT SIMILAR TO '[A-z_ ]+' THEN
      RAISE EXCEPTION 'cannot run pganalyze.explain_analyze helper with invalid flag';
    END IF;
    explain_prefix := explain_prefix || ', ' || explain_flag;
  END LOOP;
  explain_prefix := explain_prefix || ') ';

  IF cardinality(params) > 0 THEN
    SELECT '(' || pg_catalog.array_to_string(
      ARRAY(
        SELECT pg_catalog.quote_literal(p)
        FROM pg_catalog.unnest(params) _(p)
      ),
      ',',
      'NULL'
    ) || ')' INTO params_str;
  ELSE
    SELECT '' INTO params_str;
  END IF;
  SELECT COALESCE('(' || pg_catalog.string_agg(
    CASE
      WHEN p ~ '^[a-z0-9_]+(\[\])?$' THEN p
      ELSE pg_catalog.quote_ident(p)
    END,
    ','
  ) || ')', '') FROM pg_catalog.unnest(param_types) _(p) INTO param_types_str;

  EXECUTE 'PREPARE pganalyze_explain_analyze ' || param_types_str || ' AS ' || prepared_query;
  BEGIN
    EXECUTE explain_prefix || 'EXECUTE pganalyze_explain_analyze' || params_str INTO STRICT result;
  EXCEPTION WHEN QUERY_CANCELED OR OTHERS THEN
    DEALLOCATE pganalyze_explain_analyze;
    RAISE;
  END;
  DEALLOCATE pganalyze_explain_analyze;

  RETURN result;
END
$$ LANGUAGE plpgsql VOLATILE SECURITY DEFINER;
