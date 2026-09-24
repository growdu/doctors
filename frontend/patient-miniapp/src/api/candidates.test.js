// src/api/candidates.test.js
//
// getCandidates 单测 —— 验证 URL 拼接 + request 调用 + 响应透传 + 缺参兜底。
//
// 测试策略：
//   - jest.doMock('../../utils/request.js') 替换根目录 utils/request.js 的 request 函数
//     （brief 提到 `@/utils/request.js`，但 `@/` 在 jest.config.js moduleNameMapper
//      映射到 `<rootDir>/src/utils/`，而本模块 import 的是根 `utils/`，故 mock 路径
//      必须与 api 模块实际 import 的路径一致 —— 这里走相对路径 '../..'）
//   - 配合 jest.resetModules() 隔离每个 case 的 mock 状态
//   - 动态 `await import('@/api/candidates.js')` 触发 doMock 重读模块

import { jest } from '@jest/globals';

describe('api/candidates', () => {
  beforeEach(() => {
    jest.resetModules();
  });

  it('getCandidates 调 request GET /orders/:id/candidates 并透传响应', async () => {
    const responseBody = {
      items: [
        {
          escortId: 11,
          nickname: '张三',
          rating: 4.9,
          completedOrders: 220,
          distanceKm: 1.2,
          tags: ['耐心', '三甲熟悉'],
        },
        {
          escortId: 12,
          nickname: '李四',
          rating: 4.7,
          completedOrders: 90,
          distanceKm: 3.4,
          tags: ['陪同手术'],
        },
      ],
      generated_at: '2026-09-24T15:30:00+08:00',
    };
    const requestMock = jest.fn(async () => responseBody);
    jest.doMock('../../utils/request.js', () => ({
      request: requestMock,
      ApiError: class ApiError extends Error {},
      newTraceId: () => 'mp-test',
    }));

    const { getCandidates } = await import('@/api/candidates.js');
    const r = await getCandidates(7);

    expect(requestMock).toHaveBeenCalledTimes(1);
    expect(requestMock).toHaveBeenCalledWith(
      expect.objectContaining({
        url: '/orders/7/candidates',
        method: 'GET',
      }),
    );
    // 透传：返回的就是 request mock 的解析值
    expect(r).toBe(responseBody);
    expect(r.items).toHaveLength(2);
    expect(r.items[0].escortId).toBe(11);
    expect(r.items[0].tags).toEqual(['耐心', '三甲熟悉']);
    expect(r.generated_at).toBe('2026-09-24T15:30:00+08:00');
  });

  it('getCandidates 支持字符串 orderId（路径直接拼接）', async () => {
    const requestMock = jest.fn(async () => ({ items: [], generated_at: '' }));
    jest.doMock('../../utils/request.js', () => ({
      request: requestMock,
    }));

    const { getCandidates } = await import('@/api/candidates.js');
    await getCandidates('abc-uuid');

    expect(requestMock).toHaveBeenCalledWith(
      expect.objectContaining({
        url: '/orders/abc-uuid/candidates',
        method: 'GET',
      }),
    );
  });

  it('getCandidates 缺 orderId → 同步 reject 且不发请求', async () => {
    const requestMock = jest.fn();
    jest.doMock('../../utils/request.js', () => ({
      request: requestMock,
    }));

    const { getCandidates } = await import('@/api/candidates.js');

    await expect(getCandidates(undefined)).rejects.toThrow(/orderId is required/);
    await expect(getCandidates(null)).rejects.toThrow(/orderId is required/);
    await expect(getCandidates('')).rejects.toThrow(/orderId is required/);
    expect(requestMock).not.toHaveBeenCalled();
  });

  it('getCandidates 失败时透传 request 抛出的 ApiError', async () => {
    const requestMock = jest.fn(async () => {
      // 模拟 request.js 在 401 / 业务码 11001 时 reject 的 ApiError
      const e = new Error('network down');
      e.code = 11001;
      e.statusCode = 401;
      throw e;
    });
    jest.doMock('../../utils/request.js', () => ({
      request: requestMock,
    }));

    const { getCandidates } = await import('@/api/candidates.js');
    await expect(getCandidates(7)).rejects.toThrow(/network down/);
  });

  it('getCandidates 透传 backend 返的空 items 数组', async () => {
    const requestMock = jest.fn(async () => ({ items: [], generated_at: '' }));
    jest.doMock('../../utils/request.js', () => ({
      request: requestMock,
    }));

    const { getCandidates } = await import('@/api/candidates.js');
    const r = await getCandidates(7);

    expect(r.items).toEqual([]);
    expect(r.generated_at).toBe('');
  });
});
