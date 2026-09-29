/**
 * utils/share.ts 单测（mobile batch 4a）。
 *
 * 覆盖 h5 path（navigator.share + clipboard fallback）+ app-plus path（uni.share）。
 * 不测 mp-weixin：见 ./share.mp-weixin.spec.ts。
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { shareContent } from './share';
import { buildMpShareMessage } from './share.mp-weixin';
import type { ShareOptions } from './share';

const opts: ShareOptions = {
  title: '订单详情',
  desc: '我的订单 #12345',
  href: '/pages/patient/order/detail?id=12345',
  imageUrl: 'https://example.com/share.png',
};

describe('shareContent · h5 path（navigator.share）', () => {
  let originalNav: unknown;

  beforeEach(() => {
    originalNav = (globalThis as { navigator?: unknown }).navigator;
  });

  afterEach(() => {
    Object.defineProperty(globalThis, 'navigator', { value: originalNav, configurable: true, writable: true });
  });

  it('navigator.share 成功 → ok=true, channel=native', async () => {
    const share = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(globalThis, 'navigator', {
      value: { share, clipboard: undefined },
      configurable: true,
      writable: true,
    });
    const r = await shareContent(opts);
    expect(r).toEqual({ ok: true, channel: 'native' });
    expect(share).toHaveBeenCalledWith({ title: opts.title, text: opts.desc, url: opts.href });
  });

  it('navigator.share 用户取消（AbortError） → ok=false, errMsg=user-cancelled（不回退 clipboard）', async () => {
    const share = vi.fn().mockRejectedValue({ name: 'AbortError' });
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(globalThis, 'navigator', {
      value: { share, clipboard: { writeText } },
      configurable: true,
      writable: true,
    });
    const r = await shareContent(opts);
    expect(r).toEqual({ ok: false, errMsg: 'user-cancelled' });
    expect(writeText).not.toHaveBeenCalled();
  });

  it('navigator.share 其他错误 → 回退 clipboard 成功 → channel=clipboard', async () => {
    const share = vi.fn().mockRejectedValue({ name: 'NotAllowedError' });
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(globalThis, 'navigator', {
      value: { share, clipboard: { writeText } },
      configurable: true,
      writable: true,
    });
    const r = await shareContent(opts);
    expect(r).toEqual({ ok: true, channel: 'clipboard' });
    expect(writeText).toHaveBeenCalledWith(opts.href);
  });

  it('clipboard.writeText 失败 → ok=false, errMsg=clipboard-failed', async () => {
    const writeText = vi.fn().mockRejectedValue(new Error('not allowed'));
    Object.defineProperty(globalThis, 'navigator', {
      value: { clipboard: { writeText } },
      configurable: true,
      writable: true,
    });
    const r = await shareContent(opts);
    expect(r).toEqual({ ok: false, errMsg: 'clipboard-failed' });
  });

  it('无 navigator.share 也无 clipboard → 走到 uni.share 或 unsupported', async () => {
    Object.defineProperty(globalThis, 'navigator', {
      value: {},
      configurable: true,
      writable: true,
    });
    const r = await shareContent(opts);
    // jsdom setup.ts mock 的 uni 没有 share，所以最终是 unsupported
    expect(r.ok).toBe(false);
    expect(['unsupported', 'unknown']).toContain(r.errMsg);
  });
});

describe('shareContent · app-plus path（uni.share）', () => {
  let originalUni: unknown;
  let originalNav: unknown;

  beforeEach(() => {
    originalUni = (globalThis as { uni?: unknown }).uni;
    originalNav = (globalThis as { navigator?: unknown }).navigator;
    // 屏蔽 h5 path
    Object.defineProperty(globalThis, 'navigator', {
      value: {},
      configurable: true,
      writable: true,
    });
  });

  afterEach(() => {
    (globalThis as { uni?: unknown }).uni = originalUni;
    Object.defineProperty(globalThis, 'navigator', { value: originalNav, configurable: true, writable: true });
  });

  it('uni.share success 回调 → ok=true, channel=system', async () => {
    const share = vi.fn((cfg: { success: () => void }) => cfg.success());
    (globalThis as { uni?: unknown }).uni = { share };
    const r = await shareContent(opts);
    expect(r).toEqual({ ok: true, channel: 'system' });
    expect(share).toHaveBeenCalledWith(expect.objectContaining({
      provider: 'weixin',
      scene: 'WXSceneSession',
      title: opts.title,
      href: opts.href,
    }));
  });

  it('uni.share fail 回调 → ok=false, errMsg 来自 err.errMsg', async () => {
    const share = vi.fn((cfg: { fail: (e: { errMsg: string }) => void }) =>
      cfg.fail({ errMsg: 'system-share-canceled' }),
    );
    (globalThis as { uni?: unknown }).uni = { share };
    const r = await shareContent(opts);
    expect(r).toEqual({ ok: false, errMsg: 'system-share-canceled' });
  });

  it('uni.share 缺 imageUrl 时传空字符串（uni.share 要求 imageUrl 字段）', async () => {
    const share = vi.fn((cfg: { success: () => void }) => cfg.success());
    (globalThis as { uni?: unknown }).uni = { share };
    await shareContent({ title: opts.title, desc: opts.desc, href: opts.href });
    expect((share.mock.calls[0][0] as unknown as { imageUrl: string }).imageUrl).toBe('');
  });
});

describe('buildMpShareMessage · mp-weixin onShareAppMessage helper', () => {
  it('把 ShareOptions 转为 { title, path, imageUrl } 结构', () => {
    const msg = buildMpShareMessage(opts);
    expect(msg).toEqual({
      title: opts.title,
      path: opts.href,
      imageUrl: opts.imageUrl,
    });
  });

  it('不传 imageUrl 时 imageUrl 为 undefined（mp-weixin 行为：使用默认截图）', () => {
    const msg = buildMpShareMessage({ title: 'T', desc: 'D', href: '/pages/x' });
    expect(msg.imageUrl).toBeUndefined();
    expect(msg.title).toBe('T');
    expect(msg.path).toBe('/pages/x');
  });
});