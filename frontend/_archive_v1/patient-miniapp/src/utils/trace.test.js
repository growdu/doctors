// src/utils/trace.test.js
//
// newTraceId 格式 + 唯一性 + 前缀校验。
import { newTraceId, TRACE_PREFIX } from './trace.js';

describe('newTraceId', () => {
  test('前缀为 TRACE_PREFIX（mp）', () => {
    const id = newTraceId();
    expect(id.startsWith(`${TRACE_PREFIX}-`)).toBe(true);
  });

  test('格式为 mp-{ms}-{rand6}（3 段，rand 段定长 6 位）', () => {
    const id = newTraceId();
    const parts = id.split('-');
    expect(parts.length).toBe(3);
    expect(parts[0]).toBe(TRACE_PREFIX);
    expect(Number.isInteger(Number(parts[1]))).toBe(true);
    expect(parts[2].length).toBe(6);
    expect(Number.isInteger(Number(parts[2]))).toBe(true);
    expect(Number(parts[2])).toBeGreaterThanOrEqual(0);
    expect(Number(parts[2])).toBeLessThan(1_000_000);
  });

  test('连续两次调用 → 不同（ms / rand 任意一项不同）', () => {
    const a = newTraceId();
    const b = newTraceId();
    expect(a === b).toBe(false);
  });

  test('注入 now 参数 → ms 段 = 注入值', () => {
    const fixed = 1_727_187_600_123;
    const id = newTraceId(fixed);
    const msInId = Number(id.split('-')[1]);
    expect(msInId).toBe(fixed);
  });

  test('注入固定 now 后，rand 段仍可能不同（同 ms 内唯一性）', () => {
    const fixed = 1_727_187_600_123;
    const ids = new Set(Array.from({ length: 50 }, () => newTraceId(fixed)));
    // 50 次至少应有 >1 个不同 id（撞随机 1/1e6 可忽略）
    expect(ids.size).toBeGreaterThan(1);
  });

  test('ms 段与 Date.now() 接近（差 < 100ms）', () => {
    const before = Date.now();
    const id = newTraceId();
    const after = Date.now();
    const msInId = Number(id.split('-')[1]);
    expect(msInId).toBeGreaterThanOrEqual(before);
    expect(msInId).toBeLessThanOrEqual(after);
  });
});