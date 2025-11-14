-- Enable TimescaleDB extension on first database initialization
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Optionally, preload toolkit (commented out by default)
-- CREATE EXTENSION IF NOT EXISTS timescaledb_toolkit;

-- Note: Application creates its own tables at runtime.
-- If you want to convert a specific table to a hypertable, run:
--   SELECT create_hypertable('<your_table_name>', 'timestamp', if_not_exists => TRUE);

