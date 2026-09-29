/**
 * SCSS ↔ TS tokens 同步校验。
 *
 * 防止两份定义漂移：tokens.ts 是单一真相源（运行时可用 + 测试可验证），
 * tokens.scss 是 SCSS 副本（编译期可用，@media 嵌套规则等）。
 *
 * 验证策略：
 *   - 读 tokens.scss 文件内容
 *   - 正则提取每个 $ui-* 变量值
 *   - 与 tokens.ts 导出常量比对（容许 px 后缀差异）
 *
 * 跑通 = SCSS 副本与 TS 真相源 100% 一致；
 * 失败 = 有人改了 TS 但忘了同步 SCSS（或反向）。
 */
import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import {
  uiColorPrimary,
  uiColorSuccess,
  uiColorWarning,
  uiColorError,
  uiSpace,
  uiFontSize,
  uiFontWeight,
  uiRadius,
  uiBreakpoint,
  uiDuration,
} from './tokens';

const scssPath = resolve(__dirname, 'tokens.scss');
const scssContent = readFileSync(scssPath, 'utf-8');

/**
 * 从 tokens.scss 提取 $ui-var-name: value;
 * 例：$ui-color-primary: #1677ff; → ['ui-color-primary', '#1677ff']
 */
function extractScssVar(name: string): string | null {
  // 匹配 $ui-xxx: value;  value 可以是 hex / rgba / px / 数字 / 1.5 这种
  const re = new RegExp(`^\\$${name.replace(/[-]/g, '-')}\\s*:\\s*([^;]+);`, 'm');
  const m = scssContent.match(re);
  return m ? m[1]!.trim() : null;
}

describe('SCSS ↔ TS tokens sync', () => {
  it('scss file is readable', () => {
    expect(scssContent).toContain('$ui-color-primary:');
  });

  it('colors: hex values identical', () => {
    for (const [scssName, tsValue] of [
      ['ui-color-primary', uiColorPrimary],
      ['ui-color-success', uiColorSuccess],
      ['ui-color-warning', uiColorWarning],
      ['ui-color-error', uiColorError],
    ] as Array<[string, string]>) {
      expect(extractScssVar(scssName), `${scssName} from SCSS`).toBe(tsValue);
    }
  });

  it('spacing: px values identical', () => {
    const cases: Array<[string, number]> = [
      ['ui-space-xxs', uiSpace.xxs],
      ['ui-space-xs', uiSpace.xs],
      ['ui-space-sm', uiSpace.sm],
      ['ui-space-md', uiSpace.md],
      ['ui-space-base', uiSpace.base],
      ['ui-space-lg', uiSpace.lg],
      ['ui-space-xl', uiSpace.xl],
      ['ui-space-xxl', uiSpace.xxl],
    ];
    for (const [scssName, tsValue] of cases) {
      const raw = extractScssVar(scssName);
      expect(raw, `${scssName} exists in SCSS`).not.toBeNull();
      expect(parseInt(raw!, 10), `${scssName} value matches TS`).toBe(tsValue);
      expect(raw, `${scssName} has px unit`).toMatch(/px$/);
    }
  });

  it('font sizes: px values identical', () => {
    const cases: Array<[string, number]> = [
      ['ui-font-xs', uiFontSize.xs],
      ['ui-font-sm', uiFontSize.sm],
      ['ui-font-base', uiFontSize.base],
      ['ui-font-md', uiFontSize.md],
      ['ui-font-lg', uiFontSize.lg],
      ['ui-font-xl', uiFontSize.xl],
      ['ui-font-xxl', uiFontSize.xxl],
      ['ui-font-display', uiFontSize.display],
    ];
    for (const [scssName, tsValue] of cases) {
      const raw = extractScssVar(scssName);
      expect(raw, `${scssName} exists in SCSS`).not.toBeNull();
      expect(parseInt(raw!, 10), `${scssName} value matches TS`).toBe(tsValue);
    }
  });

  it('font weights: numeric values identical', () => {
    const cases: Array<[string, number]> = [
      ['ui-font-weight-regular', uiFontWeight.regular],
      ['ui-font-weight-medium', uiFontWeight.medium],
      ['ui-font-weight-semibold', uiFontWeight.semibold],
      ['ui-font-weight-bold', uiFontWeight.bold],
    ];
    for (const [scssName, tsValue] of cases) {
      const raw = extractScssVar(scssName);
      expect(raw, `${scssName} exists in SCSS`).not.toBeNull();
      expect(parseInt(raw!, 10), `${scssName} value matches TS`).toBe(tsValue);
    }
  });

  it('radius: px values identical', () => {
    const cases: Array<[string, number]> = [
      ['ui-radius-sm', uiRadius.sm],
      ['ui-radius-md', uiRadius.md],
      ['ui-radius-lg', uiRadius.lg],
      ['ui-radius-pill', uiRadius.pill],
    ];
    for (const [scssName, tsValue] of cases) {
      const raw = extractScssVar(scssName);
      expect(raw, `${scssName} exists in SCSS`).not.toBeNull();
      expect(parseInt(raw!, 10), `${scssName} value matches TS`).toBe(tsValue);
    }
  });

  it('breakpoints: px values identical', () => {
    const cases: Array<[string, number]> = [
      ['ui-breakpoint-mobile', uiBreakpoint.mobile],
      ['ui-breakpoint-tablet', uiBreakpoint.tablet],
      ['ui-breakpoint-desktop', uiBreakpoint.desktop],
    ];
    for (const [scssName, tsValue] of cases) {
      const raw = extractScssVar(scssName);
      expect(raw, `${scssName} exists in SCSS`).not.toBeNull();
      expect(parseInt(raw!, 10), `${scssName} value matches TS`).toBe(tsValue);
    }
  });

  it('durations: ms values identical', () => {
    const cases: Array<[string, number]> = [
      ['ui-duration-fast', uiDuration.fast],
      ['ui-duration-normal', uiDuration.normal],
      ['ui-duration-slow', uiDuration.slow],
    ];
    for (const [scssName, tsValue] of cases) {
      const raw = extractScssVar(scssName);
      expect(raw, `${scssName} exists in SCSS`).not.toBeNull();
      expect(parseInt(raw!, 10), `${scssName} value matches TS`).toBe(tsValue);
      expect(raw, `${scssName} has ms unit`).toMatch(/ms$/);
    }
  });
});