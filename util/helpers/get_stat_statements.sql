CREATE OR REPLACE FUNCTION pganalyze.get_stat_statements(showtext boolean = true) RETURNS SETOF pg_stat_statements AS
$$
    /* pganalyze-collector */ SELECT * FROM public.pg_stat_statements(showtext);
$$ LANGUAGE sql VOLATILE SECURITY DEFINER;
