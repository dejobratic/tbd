-- Create orders table
CREATE TABLE IF NOT EXISTS orders (
    id TEXT PRIMARY KEY,
    customer_email TEXT NOT NULL,
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    status TEXT NOT NULL CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'canceled')),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Index for filtering by status
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);

-- Index for sorting and pagination by creation time
CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at DESC);

-- Composite index for common query pattern: filter by status and sort by created_at
CREATE INDEX IF NOT EXISTS idx_orders_status_created_at ON orders(status, created_at DESC);
