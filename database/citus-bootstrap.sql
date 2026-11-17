DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_dist_node WHERE nodename = 'citus-worker-1' AND nodeport = 5432
  ) THEN
    PERFORM citus_add_node('citus-worker-1', 5432);
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_dist_node WHERE nodename = 'citus-worker-2' AND nodeport = 5432
  ) THEN
    PERFORM citus_add_node('citus-worker-2', 5432);
  END IF;
END
$$;
