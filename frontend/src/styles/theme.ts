// Design token system per D-27, D-28, D-29, D-02.2, D-03.1~D-03.5
export const designTokens = {
  colors: {
    // D-03.2 / D-02.2 — single brand accent (deep teal), replaces antd default #1890ff
    accent: '#0F766E',
    // D-03.1 — neutral slate scale for surfaces/borders/muted text
    surface: '#f8fafc',
    border: '#e2e8f0',
    muted: '#94a3b8',
    // D-02.2 — primary is an alias to accent (no antd default blue)
    primary: '#0F766E',
    success: '#52c41a',
    warning: '#faad14',
    error: '#ff4d4f',
    text: {
      // D-03.1 — slate-900 / slate-600 / slate-400 (replaces rgba black values)
      primary: '#0f172a',
      secondary: '#475569',
      disabled: '#94a3b8',
    },
  },
  spacing: {
    xs: 4,
    sm: 8,
    md: 16,
    lg: 24,
    xl: 32,
    xxl: 48,
    xxxl: 64,
  },
  // D-03.4 — borderRadius expanded from single number to tiered object
  borderRadius: {
    base: 6,
    sm: 4,
    md: 6,
    lg: 10,
    pill: 999,
  },
  fontSize: {
    sm: 12,
    base: 14,
    lg: 16,
    xl: 20,
  },
  // D-03.3 — elevation tiers (antd-compatible boxShadow strings)
  elevation: {
    none: 'none',
    sm: '0 1px 2px 0 rgba(15, 23, 42, 0.04)',
    md: '0 4px 12px -2px rgba(15, 23, 42, 0.08)',
    lg: '0 12px 32px -8px rgba(15, 23, 42, 0.12)',
  },
  // D-03.5 — motion duration (ms, numbers) + easing curves
  motion: {
    duration: {
      fast: 120,
      base: 180,
      slow: 240,
    },
    easing: {
      standard: 'cubic-bezier(0.4, 0, 0.2, 1)',
      decelerate: 'cubic-bezier(0, 0, 0.2, 1)',
      accelerate: 'cubic-bezier(0.4, 0, 1, 1)',
    },
  },
}

export type ThemeTokens = typeof designTokens
