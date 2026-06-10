export const queryKeys = {
  auth: {
    me: ["auth", "me"] as const,
  },
  teamlead: {
    traders: (params?: Record<string, unknown>) => ["teamlead", "traders", params] as const,
    requisites: (params?: Record<string, unknown>) => ["teamlead", "requisites", params] as const,
    orders: (direction: "inbound" | "outbound", params?: Record<string, unknown>) =>
      ["teamlead", direction, "orders", params] as const,
    audit: (params?: Record<string, unknown>) => ["teamlead", "audit", params] as const,
  },
  trader: {
    currentShift: ["trader", "shift", "current"] as const,
    requisites: (params?: Record<string, unknown>) => ["trader", "requisites", params] as const,
    payouts: (params?: Record<string, unknown>) => ["trader", "payouts", params] as const,
    orders: (direction: "inbound" | "outbound", params?: Record<string, unknown>) =>
      ["trader", direction, "orders", params] as const,
  },
};
