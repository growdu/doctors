/**
 * utils/callPhone.ts 单测（mobile batch 4b）。
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { callPhone } from './callPhone';

describe('callPhone · 入参校验', () => {
  it('空字符串 → ok=false, errMsg=empty-phone', async () => {
    const r = await callPhone('');
    expect(r).toEqual({ ok: false, errMsg: 'empty-phone' });
  });

  it('纯空白 → ok=false, errMsg=empty-phone', async () => {
    const r = await callPhone('   ');
    expect(r).toEqual({ ok: false, errMsg: 'empty-phone' });
  });

  it('非法字符（字母 x）→ ok=false, errMsg=invalid-format', async () => {
    const r = await callPhone('1380000xxxx');
    expect(r).toEqual({ ok: false, errMsg: 'invalid-format' });
  });

  it('太短（仅 3 位）→ ok=false, errMsg=invalid-format', async () => {
    const r = await callPhone('138');
    expect(r).toEqual({ ok: false, errMsg: 'invalid-format' });
  });

  it('合法号码（含 +86 区号、空格、横线） → 通过校验进入路径', async () => {
    const r = await callPhone('+86 138-0000-0000');
    // 不校验最终 ok（环境依赖），至少 errMsg 不是 invalid-format
    expect(r.errMsg).not.toBe('invalid-format');
    expect(r.errMsg).not.toBe('empty-phone');
  });
});

describe('callPhone · app-plus path（uni.makePhoneCall）', () => {
  let originalUni: unknown;

  beforeEach(() => {
    originalUni = (globalThis as { uni?: unknown }).uni;
  });

  afterEach(() => {
    (globalThis as { uni?: unknown }).uni = originalUni;
  });

  it('uni.makePhoneCall success → ok=true, channel=uni-call', async () => {
    const makePhoneCall = vi.fn((cfg: { phoneNumber: string; success: () => void }) => cfg.success());
    (globalThis as { uni?: unknown }).uni = { makePhoneCall };
    const r = await callPhone('13800000000');
    expect(r).toEqual({ ok: true, channel: 'uni-call' });
    expect(makePhoneCall).toHaveBeenCalledWith(expect.objectContaining({ phoneNumber: '13800000000' }));
  });

  it('uni.makePhoneCall fail → ok=false, errMsg 来自 err.errMsg', async () => {
    const makePhoneCall = vi.fn((cfg: { fail: (e: { errMsg: string }) => void }) =>
      cfg.fail({ errMsg: 'user-cancelled-call' }),
    );
    (globalThis as { uni?: unknown }).uni = { makePhoneCall };
    const r = await callPhone('13800000000');
    expect(r).toEqual({ ok: false, errMsg: 'user-cancelled-call' });
  });

  it('号码 trim 后传给 uni.makePhoneCall（去前后空白）', async () => {
    const makePhoneCall = vi.fn((cfg: { phoneNumber: string; success: () => void }) => cfg.success());
    (globalThis as { uni?: unknown }).uni = { makePhoneCall };
    await callPhone('  13800000000  ');
    expect(makePhoneCall.mock.calls[0][0].phoneNumber).toBe('13800000000');
  });
});

describe('callPhone · h5 path（document 创建 a + click）', () => {
  let originalUni: unknown;

  beforeEach(() => {
    originalUni = (globalThis as { uni?: unknown }).uni;
    // 屏蔽 app-plus path
    (globalThis as { uni?: unknown }).uni = {};
  });

  afterEach(() => {
    (globalThis as { uni?: unknown }).uni = originalUni;
  });

  it('document 存在时创建 a[href=tel:] 元素并 click → channel=tel-link', async () => {
    const click = vi.fn();
    const createElementSpy = vi.spyOn(document, 'createElement').mockImplementation(((tag: string) => {
      const el: Record<string, unknown> = {
        tagName: tag.toUpperCase(),
        style: {},
        click,
      };
      // 属性访问走 Proxy，转发 href / appendChild
      return new Proxy(el, {
        set(target, prop, value) {
          if (prop === 'href' || prop === 'style') target[prop as string] = value;
          return true;
        },
      });
    }) as unknown as typeof document.createElement);
    const appendChild = vi.spyOn(document.body, 'appendChild').mockImplementation((node: Node) => node);
    const removeChild = vi.spyOn(document.body, 'removeChild').mockImplementation((node: Node) => node);

    const r = await callPhone('13800000000');
    expect(r.channel).toBe('tel-link');
    expect(click).toHaveBeenCalled();
    expect(appendChild).toHaveBeenCalled();
    expect(removeChild).toHaveBeenCalled();

    createElementSpy.mockRestore();
    appendChild.mockRestore();
    removeChild.mockRestore();
  });
});