// src/utils/format.test.js
//
// formatMoney / formatDateTime / maskPhone 三个工具函数全覆盖。
// 使用 jest + jsdom（jest.config.js 已配 testEnvironment: 'jsdom'）。
import { formatMoney, formatDateTime, maskPhone } from './format.js';

describe('formatMoney', () => {
  test('整数 → 加 .00', () => {
    expect(formatMoney(100)).toBe('¥100.00');
    expect(formatMoney(0)).toBe('¥0.00');
  });

  test('两位小数 → 原样', () => {
    expect(formatMoney(99.99)).toBe('¥99.99');
  });

  test('超过两位小数 → 四舍五入到两位', () => {
    expect(formatMoney(99.999)).toBe('¥100.00');
    expect(formatMoney(99.994)).toBe('¥99.99');
  });

  test('负数 → 抛 RangeError', () => {
    expect(() => formatMoney(-1)).toThrow(RangeError);
    expect(() => formatMoney(-0.01)).toThrow(RangeError);
  });

  test('整数类型 → 自动转字符串', () => {
    expect(formatMoney(300)).toBe('¥300.00');
  });

  test('非数字 → 抛 RangeError', () => {
    expect(() => formatMoney('100')).toThrow(RangeError);
    expect(() => formatMoney(NaN)).toThrow(RangeError);
    expect(() => formatMoney(null)).toThrow(RangeError);
    expect(() => formatMoney(undefined)).toThrow(RangeError);
  });
});

describe('formatDateTime', () => {
  test('标准 Date → yyyy-MM-dd HH:mm（本地时区）', () => {
    const dt = new Date(2026, 8, 24, 15, 30); // 月份 0-11
    expect(formatDateTime(dt)).toBe('2026-09-24 15:30');
  });

  test('月/日/时/分 < 10 → 自动补 0', () => {
    const dt = new Date(2026, 0, 5, 9, 3);
    expect(formatDateTime(dt)).toBe('2026-01-05 09:03');
  });

  test('ISO 字符串输入', () => {
    // 显式本地时区构造（避免时区敏感性：选择 UTC+8 视角下的 15:00）
    const dt = new Date('2026-09-24T15:30:00+08:00');
    expect(formatDateTime(dt)).toBe('2026-09-24 15:30');
  });

  test('跨年边界', () => {
    const dt = new Date(2025, 11, 31, 23, 59);
    expect(formatDateTime(dt)).toBe('2025-12-31 23:59');
  });

  test('null / undefined / 空串 → 占位 "-"', () => {
    expect(formatDateTime(null)).toBe('-');
    expect(formatDateTime(undefined)).toBe('-');
    expect(formatDateTime('')).toBe('-');
  });

  test('无效输入 → 占位 "-"（不抛错）', () => {
    expect(formatDateTime('not a date')).toBe('-');
    expect(formatDateTime(new Date('invalid'))).toBe('-');
  });
});

describe('maskPhone', () => {
  test('11 位手机号 → 前3****后4', () => {
    expect(maskPhone('13800138000')).toBe('138****8000');
    expect(maskPhone('19912345678')).toBe('199****5678');
  });

  test('非 11 位原样返回', () => {
    expect(maskPhone('12345')).toBe('12345');
    expect(maskPhone('123456789012')).toBe('123456789012');
  });

  test('空字符串 → 空字符串', () => {
    expect(maskPhone('')).toBe('');
  });

  test('非字符串 → 空字符串（兜底）', () => {
    expect(maskPhone(null)).toBe('');
    expect(maskPhone(undefined)).toBe('');
    expect(maskPhone(123)).toBe('');
  });
});