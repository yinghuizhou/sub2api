-- Add foreign key constraints for payment and referral tables.
-- Use RESTRICT instead of CASCADE to protect financial records from accidental deletion.
-- Deleting a user with orders/commission records must be explicitly handled (soft-delete or manual cleanup).
-- Using DO blocks to make idempotent (skip if constraint already exists).

-- C1: payment_orders.user_id → users(id)
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_payment_orders_user') THEN
        ALTER TABLE payment_orders ADD CONSTRAINT fk_payment_orders_user
            FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT;
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_referrals_inviter') THEN
        ALTER TABLE referrals ADD CONSTRAINT fk_referrals_inviter
            FOREIGN KEY (inviter_id) REFERENCES users(id) ON DELETE RESTRICT;
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_referrals_invitee') THEN
        ALTER TABLE referrals ADD CONSTRAINT fk_referrals_invitee
            FOREIGN KEY (invitee_id) REFERENCES users(id) ON DELETE RESTRICT;
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_referral_commissions_inviter') THEN
        ALTER TABLE referral_commissions ADD CONSTRAINT fk_referral_commissions_inviter
            FOREIGN KEY (inviter_id) REFERENCES users(id) ON DELETE RESTRICT;
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_referral_commissions_invitee') THEN
        ALTER TABLE referral_commissions ADD CONSTRAINT fk_referral_commissions_invitee
            FOREIGN KEY (invitee_id) REFERENCES users(id) ON DELETE RESTRICT;
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_referral_commissions_order') THEN
        ALTER TABLE referral_commissions ADD CONSTRAINT fk_referral_commissions_order
            FOREIGN KEY (order_id) REFERENCES payment_orders(id) ON DELETE RESTRICT;
    END IF;
END $$;

-- M3: reseller_orders FK constraints (reseller_id → users, payment_order_id → payment_orders)
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_reseller_orders_reseller') THEN
        ALTER TABLE reseller_orders ADD CONSTRAINT fk_reseller_orders_reseller
            FOREIGN KEY (reseller_id) REFERENCES users(id) ON DELETE RESTRICT;
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_reseller_orders_payment') THEN
        ALTER TABLE reseller_orders ADD CONSTRAINT fk_reseller_orders_payment
            FOREIGN KEY (payment_order_id) REFERENCES payment_orders(id) ON DELETE RESTRICT;
    END IF;
END $$;
