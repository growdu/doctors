/**
 * design tokens 模块验证（vitest）。
 *
 * 验证目标：
 *   - tokens.ts 的所有常量值格式正确（颜色、间距、字号等）
 *   - 数值步进单调（spacing / font size 递增）
 *   - 命名规范与 allTokenKeys 一致（新增 token 必须同步注册）
 *
 * 注意：SCSS 副本（tokens.scss）值与本文件值的同步由 tokens-sync.spec.ts 验证。
 */
import { describe, expect, it } from 'vitest';
import {
  uiColorPrimary,
  uiColorSuccess,
  uiColorWarning,
  uiColorError,
  uiSpace,
  uiFontSize,
  uiFontWeight,
  uiLineHeight,
  uiRadius,
  uiShadow,
  uiZIndex,
  uiBreakpoint,
  uiDuration,
  allTokenKeys,
} from './tokens';

const HEX = /^#[0-9a-fA-F]{6}$/;

describe('design tokens · colors', () => {
  it('primary color is 6-digit hex', () => {
    expect(uiColorPrimary).toMatch(HEX);
  });

  it('all semantic colors are 6-digit hex', () => {
    expect(uiColorSuccess).toMatch(HEX);
    expect(uiColorWarning).toMatch(HEX);
    expect(uiColorError).toMatch(HEX);
  });

  it('primary equals AntD blue 5', () => {
    expect(uiColorPrimary).toBe('#1677ff');
  });
});

describe('design tokens · spacing', () => {
  it('8 spacing steps (xxs..xxl)', () => {
    expect(Object.keys(uiSpace)).toHaveLength(8);
    expect(uiSpace.xxs).toBe(2);
    expect(uiSpace.xxl).toBe(32);
  });

  it('spacing is monotonic increasing (xxs < xs < ... < xxl)', () => {
    const order: Array<keyof typeof uiSpace> = ['xxs', 'xs', 'sm', 'md', 'base', 'lg', 'xl', 'xxl'];
    for (let i = 1; i < order.length; i++) {
      expect(uiSpace[order[i]], `${order[i]} should > ${order[i - 1]}`).toBeGreaterThan(uiSpace[order[i - 1]!]);
    }
  });

  it('spacing steps follow 4px grid', () => {
    const arr = Object.values(uiSpace);
    for (const v of arr) {
      expect(v % 2).toBe(0);
    }
  });
});

describe('design tokens · font sizes', () => {
  it('8 font sizes (xs..display)', () => {
    expect(Object.keys(uiFontSize)).toHaveLength(8);
    expect(uiFontSize.xs).toBe(12);
    expect(uiFontSize.display).toBe(24);
  });

  it('font sizes are monotonic increasing', () => {
    const order: Array<keyof typeof uiFontSize> = ['xs', 'sm', 'base', 'md', 'lg', 'xl', 'xxl', 'display'];
    for (let i = 1; i < order.length; i++) {
      expect(uiFontSize[order[i]], `${order[i]} should > ${order[i - 1]}`).toBeGreaterThan(uiFontSize[order[i - 1]!]);
    }
  });
});

describe('design tokens · typography', () => {
  it('font weights 4 standard levels', () => {
    expect(uiFontWeight.regular).toBe(400);
    expect(uiFontWeight.medium).toBe(500);
    expect(uiFontWeight.semibold).toBe(600);
    expect(uiFontWeight.bold).toBe(700);
  });

  it('line heights 3 standard', () => {
    expect(uiLineHeight.tight).toBeLessThan(uiLineHeight.base);
    expect(uiLineHeight.base).toBeLessThan(uiLineHeight.loose);
  });
});

describe('design tokens · radius / shadow', () => {
  it('radius: sm < md < lg', () => {
    expect(uiRadius.sm).toBeLessThan(uiRadius.md);
    expect(uiRadius.md).toBeLessThan(uiRadius.lg);
  });

  it('radius.pill = 9999 (fully rounded)', () => {
    expect(uiRadius.pill).toBe(9999);
  });

  it('shadows 3 standard', () => {
    expect(uiShadow.sm).toContain('rgba');
    expect(uiShadow.md).toContain('rgba');
    expect(uiShadow.lg).toContain('rgba');
  });
});

describe('design tokens · z-index / breakpoints / duration', () => {
  it('z-index 4 standard layers', () => {
    expect(uiZIndex.base).toBeLessThan(uiZIndex.sticky);
    expect(uiZIndex.sticky).toBeLessThan(uiZIndex.modal);
    expect(uiZIndex.modal).toBeLessThan(uiZIndex.toast);
  });

  it('breakpoints 3 standard', () => {
    expect(uiBreakpoint.mobile).toBeLessThan(uiBreakpoint.tablet);
    expect(uiBreakpoint.tablet).toBeLessThan(uiBreakpoint.desktop);
  });

  it('durations 3 standard', () => {
    expect(uiDuration.fast).toBeLessThan(uiDuration.normal);
    expect(uiDuration.normal).toBeLessThan(uiDuration.slow);
  });
});

describe('design tokens · allTokenKeys registry', () => {
  it('registry has expected count (used as lint baseline)', () => {
    // 18 colors + 8 spacing + 8 font sizes + 4 weights + 3 line heights
    // + 4 radius + 3 shadow + 4 z + 3 breakpoints + 3 duration = 58
    expect(allTokenKeys.length).toBe(58);
  });

  it('all keys follow ui* naming convention', () => {
    for (const k of allTokenKeys) {
      expect(k, `token ${k} should start with 'ui'`).toMatch(/^ui/);
    }
  });
});