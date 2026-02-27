-- Add amount_cny, exchange_rate, and commission_status columns to payment_orders
ALTER TABLE payment_orders ADD COLUMN IF NOT EXISTS amount_cny DECIMAL(20,8);
ALTER TABLE payment_orders ADD COLUMN IF NOT EXISTS exchange_rate DECIMAL(10,6);
ALTER TABLE payment_orders ADD COLUMN IF NOT EXISTS commission_status VARCHAR(20) NOT NULL DEFAULT 'none';
