// src/utils/auth.js
//
// Token 持久化 shim —— 实际实现见 ../../utils/auth.js（v1 骨架；worker A 创建）。
//
// 背景：
//   - plan Task 4 期望 `src/utils/auth.ts` 作为 tokenStorage 单一来源
//   - worker A 实际落地在根 `utils/auth.js`（v1 阶段；与 plan path 偏差）
//   - 为了让 src/ 内代码（stores / api / 后续页面）按 `@/utils/auth.js` 统一别名引入，
//     在此加 1 行 re-export 软链；后续 plan Task 4 完整版可让本文件持有实现并删除软链。
//
// 用法（src/ 内代码统一）：
//   import { getToken, setToken, clearToken } from '@/utils/auth.js';

export { getToken, setToken, clearToken } from '../../utils/auth.js';