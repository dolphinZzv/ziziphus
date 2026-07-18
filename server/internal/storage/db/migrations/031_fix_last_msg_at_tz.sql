-- 031_fix_last_msg_at_tz: Change last_msg_at from TIMESTAMP to TIMESTAMPTZ.
--
-- The USING clause interprets existing stored values as being in the current
-- session timezone (what NOW() used when storing them), so the conversion is
-- correct regardless of which timezone the server is configured with.
--
-- Wrapped in a DO block so it's safe to re-run on every server restart.
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name = 'conversations'
      AND column_name = 'last_msg_at'
      AND data_type = 'timestamp without time zone'
  ) THEN
    ALTER TABLE conversations ALTER COLUMN last_msg_at TYPE TIMESTAMPTZ
      USING last_msg_at AT TIME ZONE current_setting('TIMEZONE');
  END IF;
END $$;
