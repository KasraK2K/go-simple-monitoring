-- Script to manually convert existing tables to TimescaleDB hypertables
-- WARNING: This script will temporarily make tables unavailable during migration
-- Run this script only during maintenance windows

-- Step 1: Check which tables need conversion
SELECT 
    table_name,
    CASE 
        WHEN EXISTS (
            SELECT 1 FROM timescaledb_information.hypertables 
            WHERE hypertable_name = t.table_name
        ) THEN 'Already hypertable'
        ELSE 'Needs conversion'
    END as status
FROM information_schema.tables t
WHERE table_schema = 'public' 
AND table_type = 'BASE TABLE'
AND table_name NOT LIKE 'pg_%'
AND table_name NOT LIKE 'sql_%';

-- Step 2: Convert tables to hypertables (example for common table names)
-- Uncomment and run these one at a time during a maintenance window

/*
-- For the default monitoring table:
-- 1. Create a new hypertable with the same structure
CREATE TABLE default_new (LIKE default INCLUDING ALL);
SELECT create_hypertable('default_new', 'timestamp');

-- 2. Copy data in batches (adjust batch size as needed)
INSERT INTO default_new SELECT * FROM default ORDER BY timestamp;

-- 3. Rename tables (requires downtime)
ALTER TABLE default RENAME TO default_backup;
ALTER TABLE default_new RENAME TO default;

-- 4. Update any foreign key constraints if they exist
-- (Add constraint updates here if needed)

-- 5. After verification, drop the backup table
-- DROP TABLE default_backup;
*/

/*
-- For server-specific tables (replace 'your_table_name' with actual table name):
CREATE TABLE your_table_name_new (LIKE your_table_name INCLUDING ALL);
SELECT create_hypertable('your_table_name_new', 'timestamp');
INSERT INTO your_table_name_new SELECT * FROM your_table_name ORDER BY timestamp;
ALTER TABLE your_table_name RENAME TO your_table_name_backup;
ALTER TABLE your_table_name_new RENAME TO your_table_name;
-- DROP TABLE your_table_name_backup; -- After verification
*/

-- Step 3: Verify hypertables were created successfully
SELECT 
    hypertable_name,
    num_chunks,
    compression_enabled,
    replication_factor
FROM timescaledb_information.hypertables;

-- Step 4: Optional - Set up compression policies for older data (saves space)
/*
-- Enable compression on chunks older than 7 days
SELECT add_compression_policy('default', INTERVAL '7 days');

-- Set retention policy to automatically drop data older than 1 year
SELECT add_retention_policy('default', INTERVAL '1 year');
*/