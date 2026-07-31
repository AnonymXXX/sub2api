import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";
import { createPinia } from "pinia";
import { createI18n } from "vue-i18n";
import SubscriptionPlanCard from "../SubscriptionPlanCard.vue";

const i18n = createI18n({
  legacy: false,
  locale: "en",
  fallbackWarn: false,
  missingWarn: false,
  messages: {
    en: {
      payment: {
        days: "days",
        models: "Models",
        planCard: {
          quota: "Quota",
          rate: "Rate",
          unlimited: "Unlimited",
        },
        subscribeNow: "Subscribe now",
        renewNow: "Renew now",
        renewalUnavailable: "Monthly quota not exhausted",
        activePlanLocked: "Current subscription is still active",
      },
    },
  },
});

const mountPlanCard = (groupPlatform: string) =>
  mount(SubscriptionPlanCard, {
    props: {
      plan: {
        id: 1,
        group_id: 10,
        group_platform: groupPlatform,
        name: "Pro",
        price: 10,
        amount: 1000,
        features: [],
        rate_multiplier: 1,
        validity_days: 30,
        validity_unit: "day",
        supported_model_scopes: ["claude", "gemini_text", "gemini_image"],
        is_active: true,
      },
    },
    global: { plugins: [i18n, createPinia()] },
  });

const activeSubscription = (overrides: Record<string, unknown> = {}) => ({
  id: 9,
  user_id: 1,
  group_id: 10,
  status: "active" as const,
  starts_at: "2026-07-01T00:00:00Z",
  expires_at: "2026-08-01T00:00:00Z",
  daily_usage_usd: 1,
  weekly_usage_usd: 1,
  monthly_usage_usd: 1,
  monthly_bonus_usd: 0,
  effective_monthly_limit_usd: 100,
  renewal_eligible: false,
  renewal_price: 10,
  daily_window_start: null,
  weekly_window_start: null,
  monthly_window_start: null,
  created_at: "2026-07-01T00:00:00Z",
  updated_at: "2026-07-01T00:00:00Z",
  ...overrides,
});

const mountPlanCardWithSubscription = (
  planGroupId: number,
  subscriptionOverrides: Record<string, unknown> = {},
) => mount(SubscriptionPlanCard, {
  props: {
    plan: {
      id: 1,
      group_id: planGroupId,
      group_platform: "openai",
      name: "Pro",
      price: 10,
      features: [],
      rate_multiplier: 1,
      validity_days: 30,
      validity_unit: "day",
      for_sale: true,
      sort_order: 0,
      description: "",
    },
    activeSubscriptions: [activeSubscription(subscriptionOverrides)],
  },
  global: { plugins: [i18n, createPinia()] },
});

describe("SubscriptionPlanCard", () => {
  it("does not show Antigravity model scopes for OpenAI plans", () => {
    const text = mountPlanCard("openai").text();

    expect(text).not.toContain("Claude");
    expect(text).not.toContain("Gemini");
    expect(text).not.toContain("Imagen");
  });

  it("shows model scopes for Antigravity plans", () => {
    const text = mountPlanCard("antigravity").text();

    expect(text).toContain("Claude");
    expect(text).toContain("Gemini");
    expect(text).toContain("Imagen");
  });

  it("disables the current plan until its monthly quota is exhausted", async () => {
    const wrapper = mountPlanCardWithSubscription(10);

    const button = wrapper.get("button");
    expect(button.attributes("disabled")).toBeDefined();
    expect(button.text()).toContain("payment.renewalUnavailable");
    await button.trigger("click");
    expect(wrapper.emitted("select")).toBeUndefined();
  });

  it("allows renewal for the current plan when the server marks it eligible", async () => {
    const wrapper = mountPlanCardWithSubscription(10, { renewal_eligible: true });

    const button = wrapper.get("button");
    expect(button.attributes("disabled")).toBeUndefined();
    expect(button.text()).toContain("payment.renewNow");
    await button.trigger("click");
    expect(wrapper.emitted("select")).toHaveLength(1);
  });

  it("disables other plans while a subscription is active", async () => {
    const wrapper = mountPlanCardWithSubscription(20);

    const button = wrapper.get("button");
    expect(button.attributes("disabled")).toBeDefined();
    expect(button.text()).toContain("payment.activePlanLocked");
    await button.trigger("click");
    expect(wrapper.emitted("select")).toBeUndefined();
  });
});
