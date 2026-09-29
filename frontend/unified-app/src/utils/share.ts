/**
 * 跨端分享工具（mobile batch 4a）。
 *
 * 设计目标：单一函数 `shareContent` 在 h5 + app-plus 都可用。
 *   - h5：优先 navigator.share()（Web Share API，Safari/Chrome 移动端）；
 *         不可用或被拒则回退 navigator.clipboard.writeText()（写链接 + 提示）
 *   - app-plus：uni.share（系统分享面板）
 *   - mp-weixin：不走此函数（见 ./share.mp-weixin.ts 的 buildMpShareMessage，
 *                  用于页面 onShareAppMessage 钩子）
 *
 * 返回结构统一为 ShareResult；调用方根据 ok 判断是否弹 toast。
 *
 * 跨端兼容：本文件只引 navigator / uni（运行时探测），不在编译期依赖 #ifdef。
 *          这保证 vitest jsdom 环境可直接覆盖（h5 路径）。
 */
export interface ShareOptions {
  /** 分享标题 */
  title: string;
  /** 分享描述（h5 Web Share API 映射为 text 字段） */
  desc?: string;
  /** 分享链接 */
  href: string;
  /** 缩略图（app-plus 必传，h5 选传） */
  imageUrl?: string;
}

export interface ShareResult {
  ok: boolean;
  /** 成功渠道：native（系统面板）/ clipboard（剪贴板）/ system（uni.share） */
  channel?: 'native' | 'clipboard' | 'system';
  /** 失败原因：user-cancelled / clipboard-failed / unsupported / other */
  errMsg?: string;
}

/**
 * 触发分享。
 * h5 → navigator.share() → 失败则 navigator.clipboard.writeText() 兜底
 * app-plus → uni.share() 调起系统分享面板
 * 其他/全失败 → ok=false
 */
export async function shareContent(opts: ShareOptions): Promise<ShareResult> {
  const nav = typeof navigator !== 'undefined' ? (navigator as unknown as {
    share?: (data: { title: string; text?: string; url?: string }) => Promise<void>;
    clipboard?: { writeText: (text: string) => Promise<void> };
  }) : undefined;

  // ---- h5 path: navigator.share + clipboard fallback ----
  if (nav?.share) {
    try {
      await nav.share({ title: opts.title, text: opts.desc, url: opts.href });
      return { ok: true, channel: 'native' };
    } catch (e) {
      const name = (e as { name?: string } | null)?.name;
      if (name === 'AbortError') {
        return { ok: false, errMsg: 'user-cancelled' };
      }
      // 其他错误（NotAllowedError 等）→ 继续往下走到 clipboard 兜底
    }
  }

  if (nav?.clipboard?.writeText) {
    try {
      await nav.clipboard.writeText(opts.href);
      return { ok: true, channel: 'clipboard' };
    } catch {
      return { ok: false, errMsg: 'clipboard-failed' };
    }
  }

  // ---- app-plus path: uni.share ----
  const uniApi = typeof uni !== 'undefined' ? (uni as unknown as {
    share?: (cfg: {
      provider: string;
      scene: string;
      type: number;
      title: string;
      summary: string;
      href: string;
      imageUrl: string;
      success: () => void;
      fail: (e: { errMsg?: string }) => void;
    }) => void;
  }) : undefined;

  if (uniApi?.share) {
    return new Promise((resolve) => {
      uniApi.share!({
        provider: 'weixin',
        scene: 'WXSceneSession',
        type: 0,
        title: opts.title,
        summary: opts.desc ?? '',
        href: opts.href,
        imageUrl: opts.imageUrl ?? '',
        success: () => resolve({ ok: true, channel: 'system' }),
        fail: (e) => resolve({ ok: false, errMsg: e?.errMsg ?? 'unknown' }),
      });
    });
  }

  return { ok: false, errMsg: 'unsupported' };
}