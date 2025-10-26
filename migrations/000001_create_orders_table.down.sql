-- Drop indexes first
DROP INDEX IF EXISTS idx_orders_status_created_at;
DROP INDEX IF EXISTS idx_orders_created_at;
DROP INDEX IF EXISTS idx_orders_status;

-- Drop orders table
DROP TABLE IF EXISTS orders;
