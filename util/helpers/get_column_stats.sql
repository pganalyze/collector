DROP FUNCTION IF EXISTS pganalyze.get_column_stats;
CREATE FUNCTION pganalyze.get_column_stats() RETURNS TABLE(
  schemaname name, tablename name, attname name, inherited bool, null_frac real, avg_width int, n_distinct real, correlation real
) AS $$
  SELECT schemaname, tablename, attname, inherited, null_frac, avg_width, n_distinct, correlation
  FROM pg_catalog.pg_stats
  WHERE schemaname NOT IN ('pg_catalog', 'information_schema') AND tablename <> 'pg_subscription'
  /* pganalyze-collector */;
$$ LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path = pg_catalog, pg_temp;
