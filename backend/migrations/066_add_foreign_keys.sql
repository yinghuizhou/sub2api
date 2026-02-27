-- Add foreign key constraints for referral tables
-- C1: Use RESTRICT instead of CASCADE to protect financial records from accidental deletion.
-- Deleting a user with commission records must be explicitly handled (soft-delete or manual cleanup).
-- Using DO blocks to make idempotent (skip if constraint already exists).

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
