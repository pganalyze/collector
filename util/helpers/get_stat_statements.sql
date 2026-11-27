DROP FUNCTION IF EXISTS pganalyze.get_stat_statements;
CREATE FUNCTION pganalyze.get_stat_statements(showtext boolean = true) RETURNS SETOF pg_stat_statements AS
$$
    SELECT * FROM public.pg_stat_statements(showtext) /* pganalyze-collector */;
$$ LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path = pg_catalog, pg_temp;
