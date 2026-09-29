/**
 * mp-weixin 专用分享辅助（mobile batch 4a）。
 *
 * 微信小程序的分享机制与 h5 / app-plus 不同：
 *   - 不通过 API 调起；用户在页面右上角点击菜单，或页面级 `onShareAppMessage` 钩子触发
 *   - 页面 .vue 需 export 一个返回 share config 的 onShareAppMessage：
 *       export default { onShareAppMessage: () => buildMpShareMessage({...}) }
 *
 * 此文件只导出 `buildMpShareMessage`，调用方按需 import + 在页面内 #ifdef MP-WEIXIN 包裹。
 */
import type { ShareOptions } from './share';

export interface MpShareMessage {
  /** 分享标题（必填） */
  title: string;
  /** 分享路径（必填，pages/patient/order/detail?id=xxx） */
  path: string;
  /** 自定义图片路径 */
  imageUrl?: string;
}

/**
 * 把 ShareOptions 转换为 mp-weixin onShareAppMessage 返回结构。
 *
 * 注意：mp-weixin 的 `path` 不需要 host 前缀；`imageUrl` 必须是 base64 或本地路径（不支持外链）。
 */
export function buildMpShareMessage(opts: ShareOptions): MpShareMessage {
  return {
    title: opts.title,
    path: opts.href,
    imageUrl: opts.imageUrl,
  };
}