-- Enforce the prerequisites for single-subscription billing before adding new
-- order constraints. Existing conflicts require explicit operator resolution.
DO $$
BEGIN
    IF EXISTS (
        SELECT user_id
        FROM user_subscriptions
        WHERE deleted_at IS NULL
          AND status = 'active'
          AND expires_at > NOW()
        GROUP BY user_id
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION 'single-subscription migration blocked: overlapping active subscriptions exist';
    END IF;

    IF EXISTS (
        SELECT user_id
        FROM payment_orders
        WHERE order_type = 'subscription'
          AND status IN ('PENDING', 'PAID', 'RECHARGING')
        GROUP BY user_id
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION 'single-subscription migration blocked: duplicate unfinished subscription orders exist';
    END IF;

    IF EXISTS (
        SELECT group_id
        FROM subscription_plans
        WHERE for_sale = TRUE
        GROUP BY group_id
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION 'single-subscription migration blocked: multiple sale plans exist for one subscription group';
    END IF;
END $$;

ALTER TABLE user_subscriptions
    ADD COLUMN IF NOT EXISTS monthly_bonus_usd DECIMAL(20,10) NOT NULL DEFAULT 0;

ALTER TABLE payment_orders
    ADD COLUMN IF NOT EXISTS subscription_action VARCHAR(20),
    ADD COLUMN IF NOT EXISTS subscription_id BIGINT;

CREATE INDEX IF NOT EXISTS payment_orders_subscription_id
    ON payment_orders(subscription_id)
    WHERE subscription_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS payment_orders_user_unfinished_subscription_unique
    ON payment_orders(user_id)
    WHERE order_type = 'subscription'
      AND status IN ('PENDING', 'PAID', 'RECHARGING');

-- A subscription stores its product identity as group_id. Keeping only one
-- sellable plan per group makes renewal price and duration unambiguous while
-- retaining disabled historical plan rows.
CREATE UNIQUE INDEX IF NOT EXISTS subscription_plans_group_sale_unique
    ON subscription_plans(group_id)
    WHERE for_sale = TRUE;
