/**
 * unified-app v2 设计 tokens（单一真相源 · TypeScript 常量）。
 *
 * 设计原则：
 *   - 所有组件 / store / 工具类只引用本文件的常量
 *   - 颜色用 hex 字符串（#RRGGBB），前端可直接拼接 + 后端验签
 *   - SCSS 文件（tokens.scss）写同名变量副本用于编译期；sync.spec.ts 验证两者一致
 *
 * 对应：plan 2026-09-28-unified-app-v2.md Phase 3 §0.1
 */

// ── 颜色 ──────────────────────────────────────────────────────────
export const uiColorPrimary = '#1677ff';
export const uiColorPrimaryHover = '#4096ff';
export const uiColorPrimaryActive = '#0958d9';
export const uiColorSuccess = '#52c41a';
export const uiColorWarning = '#faad14';
export const uiColorError = '#ff4d4f';
export const uiColorInfo = '#1677ff';

export const uiColorTextPrimary = 'rgba(0, 0, 0, 0.88)';
export const uiColorTextSecondary = 'rgba(0, 0, 0, 0.65)';
export const uiColorTextDisabled = 'rgba(0, 0, 0, 0.45)';
export const uiColorTextInverse = '#ffffff';

export const uiColorBgBase = '#f8f8f8';
export const uiColorBgCard = '#ffffff';
export const uiColorBgHover = '#f5f5f5';
export const uiColorBgMask = 'rgba(0, 0, 0, 0.5)';

export const uiColorBorder = '#e8e8e8';
export const uiColorBorderLight = '#f0f0f0';
export const uiColorDivider = '#f0f0f0';

// ── 间距（4px 步进）────────────────────────────────────────────────
export const uiSpace = {
  xxs: 2,
  xs: 4,
  sm: 8,
  md: 12,
  base: 16,
  lg: 20,
  xl: 24,
  xxl: 32,
} as const;

// ── 字号 ───────────────────────────────────────────────────────────
export const uiFontSize = {
  xs: 12,
  sm: 13,
  base: 14,
  md: 15,
  lg: 16,
  xl: 18,
  xxl: 20,
  display: 24,
} as const;

export const uiFontWeight = {
  regular: 400,
  medium: 500,
  semibold: 600,
  bold: 700,
} as const;

export const uiLineHeight = {
  tight: 1.2,
  base: 1.5,
  loose: 1.8,
} as const;

// ── 圆角 ───────────────────────────────────────────────────────────
export const uiRadius = {
  sm: 4,
  md: 8,
  lg: 12,
  pill: 9999,
} as const;

// ── 阴影 ───────────────────────────────────────────────────────────
export const uiShadow = {
  sm: '0 1px 4px rgba(0, 0, 0, 0.04)',
  md: '0 2px 8px rgba(0, 0, 0, 0.08)',
  lg: '0 4px 16px rgba(0, 0, 0, 0.12)',
} as const;

// ── z-index ────────────────────────────────────────────────────────
export const uiZIndex = {
  base: 1,
  sticky: 100,
  modal: 1000,
  toast: 2000,
} as const;

// ── 断点 ───────────────────────────────────────────────────────────
export const uiBreakpoint = {
  mobile: 320,
  tablet: 768,
  desktop: 1024,
} as const;

// ── 动画时长 ───────────────────────────────────────────────────────
export const uiDuration = {
  fast: 120,
  normal: 240,
  slow: 360,
} as const;

// ── 全部 token 列表（lint 用：新增 token 必须同步到 tokens.scss）──────
/**
 * CSS :root 变量块生成器。
 * 给 App.vue 的 root style 块用，让所有组件 var(--ui-color-primary) 都能取到值。
 * 为什么不直接 SCSS @import？因为 uni-app sass-loader 在处理 scoped style 时，
 * 相对路径解析 + 行号偏移会失败（Phase 3.0.6 调试发现）。
 * CSS 变量是 runtime 方案，不依赖 sass compile-time 解析，跨平台一致。
 */
export function generateCssVarsBlock(): string {
  return `:root {
  --ui-color-primary: ${uiColorPrimary};
  --ui-color-primary-hover: ${uiColorPrimaryHover};
  --ui-color-primary-active: ${uiColorPrimaryActive};
  --ui-color-success: ${uiColorSuccess};
  --ui-color-warning: ${uiColorWarning};
  --ui-color-error: ${uiColorError};
  --ui-color-info: ${uiColorInfo};
  --ui-color-text-primary: ${uiColorTextPrimary};
  --ui-color-text-secondary: ${uiColorTextSecondary};
  --ui-color-text-disabled: ${uiColorTextDisabled};
  --ui-color-text-inverse: ${uiColorTextInverse};
  --ui-color-bg-base: ${uiColorBgBase};
  --ui-color-bg-card: ${uiColorBgCard};
  --ui-color-bg-hover: ${uiColorBgHover};
  --ui-color-bg-mask: ${uiColorBgMask};
  --ui-color-border: ${uiColorBorder};
  --ui-color-border-light: ${uiColorBorderLight};
  --ui-color-divider: ${uiColorDivider};
  --ui-space-xxs: ${uiSpace.xxs}px;
  --ui-space-xs: ${uiSpace.xs}px;
  --ui-space-sm: ${uiSpace.sm}px;
  --ui-space-md: ${uiSpace.md}px;
  --ui-space-base: ${uiSpace.base}px;
  --ui-space-lg: ${uiSpace.lg}px;
  --ui-space-xl: ${uiSpace.xl}px;
  --ui-space-xxl: ${uiSpace.xxl}px;
  --ui-font-xs: ${uiFontSize.xs}px;
  --ui-font-sm: ${uiFontSize.sm}px;
  --ui-font-base: ${uiFontSize.base}px;
  --ui-font-md: ${uiFontSize.md}px;
  --ui-font-lg: ${uiFontSize.lg}px;
  --ui-font-xl: ${uiFontSize.xl}px;
  --ui-font-xxl: ${uiFontSize.xxl}px;
  --ui-font-display: ${uiFontSize.display}px;
  --ui-font-weight-regular: ${uiFontWeight.regular};
  --ui-font-weight-medium: ${uiFontWeight.medium};
  --ui-font-weight-semibold: ${uiFontWeight.semibold};
  --ui-font-weight-bold: ${uiFontWeight.bold};
  --ui-line-height-tight: ${uiLineHeight.tight};
  --ui-line-height-base: ${uiLineHeight.base};
  --ui-line-height-loose: ${uiLineHeight.loose};
  --ui-radius-sm: ${uiRadius.sm}px;
  --ui-radius-md: ${uiRadius.md}px;
  --ui-radius-lg: ${uiRadius.lg}px;
  --ui-radius-pill: ${uiRadius.pill}px;
  --ui-shadow-sm: ${uiShadow.sm};
  --ui-shadow-md: ${uiShadow.md};
  --ui-shadow-lg: ${uiShadow.lg};
  --ui-z-base: ${uiZIndex.base};
  --ui-z-sticky: ${uiZIndex.sticky};
  --ui-z-modal: ${uiZIndex.modal};
  --ui-z-toast: ${uiZIndex.toast};
  --ui-duration-fast: ${uiDuration.fast}ms;
  --ui-duration-normal: ${uiDuration.normal}ms;
  --ui-duration-slow: ${uiDuration.slow}ms;
}`;
}

export const allTokenKeys = [
  // colors
  'uiColorPrimary', 'uiColorPrimaryHover', 'uiColorPrimaryActive',
  'uiColorSuccess', 'uiColorWarning', 'uiColorError', 'uiColorInfo',
  'uiColorTextPrimary', 'uiColorTextSecondary', 'uiColorTextDisabled', 'uiColorTextInverse',
  'uiColorBgBase', 'uiColorBgCard', 'uiColorBgHover', 'uiColorBgMask',
  'uiColorBorder', 'uiColorBorderLight', 'uiColorDivider',
  // spacing
  'uiSpaceXxs', 'uiSpaceXs', 'uiSpaceSm', 'uiSpaceMd', 'uiSpaceBase', 'uiSpaceLg', 'uiSpaceXl', 'uiSpaceXxl',
  // font size
  'uiFontSizeXs', 'uiFontSizeSm', 'uiFontSizeBase', 'uiFontSizeMd', 'uiFontSizeLg', 'uiFontSizeXl', 'uiFontSizeXxl', 'uiFontSizeDisplay',
  // font weight
  'uiFontWeightRegular', 'uiFontWeightMedium', 'uiFontWeightSemibold', 'uiFontWeightBold',
  // line height
  'uiLineHeightTight', 'uiLineHeightBase', 'uiLineHeightLoose',
  // radius
  'uiRadiusSm', 'uiRadiusMd', 'uiRadiusLg', 'uiRadiusPill',
  // shadow
  'uiShadowSm', 'uiShadowMd', 'uiShadowLg',
  // z-index
  'uiZIndexBase', 'uiZIndexSticky', 'uiZIndexModal', 'uiZIndexToast',
  // breakpoint
  'uiBreakpointMobile', 'uiBreakpointTablet', 'uiBreakpointDesktop',
  // duration
  'uiDurationFast', 'uiDurationNormal', 'uiDurationSlow',
] as const;