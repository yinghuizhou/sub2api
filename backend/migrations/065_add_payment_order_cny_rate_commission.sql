-- Add amount_cny, exchange_rate, and commission_status columns to payment_orders
-- B2: Use DEFAULT + NOT NULL to prevent NULL crashes on existing rows
ALTER TABLE payment_orders ADD COLUMN IF NOT EXISTS amount_cny DECIMAL(20,8) NOT NULL DEFAULT 0;
ALTER TABLE payment_orders ADD COLUMN IF NOT EXISTS exchange_rate DECIMAL(10,6) NOT NULL DEFAULT 7.2;
ALTER TABLE payment_orders ADD COLUMN IF NOT EXISTS commission_status VARCHAR(20) NOT NULL DEFAULT 'none';
