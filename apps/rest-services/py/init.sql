CREATE USER otelsample WITH PASSWORD 'samplepassword';
GRANT CREATE ON SCHEMA public TO otelsample;
GRANT USAGE ON SCHEMA public TO otelsample;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO otelsample;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO otelsample;
CREATE USER datadog WITH password 'datadogpassword';
ALTER ROLE datadog INHERIT;
CREATE SCHEMA datadog;
GRANT USAGE ON SCHEMA datadog TO datadog;
GRANT USAGE ON SCHEMA public TO datadog;
GRANT pg_monitor TO datadog;
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
CREATE OR REPLACE FUNCTION datadog.explain_statement(
   l_query TEXT,
   OUT explain JSON
)
RETURNS SETOF JSON AS
$$
DECLARE
curs REFCURSOR;
plan JSON;

BEGIN
   OPEN curs FOR EXECUTE pg_catalog.concat('EXPLAIN (FORMAT JSON) ', l_query);
   FETCH curs INTO plan;
   CLOSE curs;
   RETURN QUERY SELECT plan;
END;
$$
LANGUAGE 'plpgsql'
RETURNS NULL ON NULL INPUT
SECURITY DEFINER;
