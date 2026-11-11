CREATE EXTENSION IF NOT EXISTS citus;

CREATE TABLE IF NOT EXISTS dialog_messages (
  dialog_id UUID NOT NULL,
  id UUID NOT NULL,
  sender_id UUID NOT NULL,
  recipient_id UUID NOT NULL,
  text TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (dialog_id, id),
  CHECK (sender_id <> recipient_id)
);

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_dist_partition
    WHERE logicalrelid = 'public.dialog_messages'::regclass
  ) THEN
    PERFORM create_distributed_table(
      'dialog_messages',
      'dialog_id',
      colocate_with => 'none',
      shard_count => 32
    );
  END IF;
END
$$;

CREATE INDEX IF NOT EXISTS idx_dialog_messages_dialog_created
  ON dialog_messages (dialog_id, created_at, id);
