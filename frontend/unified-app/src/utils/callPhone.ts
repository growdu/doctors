/**
 * 一键拨号工具（mobile batch 4b）。
 *
 * 跨端设计：
 *   - h5：优先创建 <a href="tel:xxx"> + 触发 click（多数手机浏览器会弹拨号面板）；
 *         不可用则回退 window.location.href = 'tel:xxx'（移动浏览器自动跳转）
 *   - app-plus：uni.makePhoneCall()（系统级拨号）
 *   - mp-weixin：暂不支持直接调起，UI 层应改为「展示号码 + 提示用户手动拨打或复制」
 *
 * 入参校验：phoneNumber 必须非空且是合法号码（含数字 + 可选 + - 空格）。
 *          失败统一返回 { ok:false, errMsg }，由调用方 toast 提示。
 */
export interface CallPhoneResult {
  ok: boolean;
  channel?: 'tel-link' | 'location-href' | 'uni-call';
  errMsg?: string;
}

const PHONE_REGEX = /^[+\d][\d\s\-()]{3,20}$/;

/**
 * 触发拨号。
 *
 * @param phoneNumber 目标号码（含国家区号，如 +86-138-0000-0000）
 */
export async function callPhone(phoneNumber: string): Promise<CallPhoneResult> {
  const phone = (phoneNumber ?? '').trim();
  if (!phone) return { ok: false, errMsg: 'empty-phone' };
  if (!PHONE_REGEX.test(phone)) return { ok: false, errMsg: 'invalid-format' };

  // ---- app-plus path: uni.makePhoneCall ----
  const uniApi = typeof uni !== 'undefined' ? (uni as unknown as {
    makePhoneCall?: (cfg: {
      phoneNumber: string;
      success: () => void;
      fail: (e: { errMsg?: string }) => void;
    }) => void;
  }) : undefined;

  if (uniApi?.makePhoneCall) {
    return new Promise((resolve) => {
      uniApi.makePhoneCall!({
        phoneNumber: phone,
        success: () => resolve({ ok: true, channel: 'uni-call' }),
        fail: (e) => resolve({ ok: false, errMsg: e?.errMsg ?? 'unknown' }),
      });
    });
  }

  // ---- h5 path: <a href="tel:..."> click + location.href fallback ----
  if (typeof document !== 'undefined') {
    try {
      const a = document.createElement('a');
      a.href = `tel:${phone}`;
      a.style.display = 'none';
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      return { ok: true, channel: 'tel-link' };
    } catch {
      // 落到 location.href 兜底
    }
  }

  if (typeof window !== 'undefined' && typeof window.location?.href === 'string') {
    try {
      window.location.href = `tel:${phone}`;
      return { ok: true, channel: 'location-href' };
    } catch {
      return { ok: false, errMsg: 'h5-tel-failed' };
    }
  }

  return { ok: false, errMsg: 'unsupported' };
}