---
## 22. escort-app 8 核心业务页接 API（escort-app v1 plan §E1-E8）

**目标**：按 escort-app v1 plan 落地 5 个新 API 模块 + 5 个新 Riverpod provider + 8 个真实业务页（login / profile / wallet / orders / training / order_detail / + auth_provider loginByPhone/loginByWx）。

**8 个 commit**：

| commit | 内容 | 新增测试 |
| :-- | :-- | :--: |
| `4c2f430` | `feat(escort-app)` api_client 5 个领域模块（profile/wallet/training/review/order）+ 16 单测 | 14 |
| `341527a` | `feat(escort-app)` providers 5 + models 4 + 32 单测 | 20+12 |
| `4df8e84` | login 登录页（短信 60s 倒计时 + 微信入口）+ auth_provider loginByPhone/loginByWx | 5+4 |
| `d2e2e3d` | profile 个人中心页（头像 + nickname + 实名状态 + 退出登录） | 4 |
| `7a643d0` | wallet 钱包页（余额 + 冻结 + 提现 + 流水 4 tab 过滤） | 6 |
| `edaf2a2` | orders 我的订单页（4 tab + CountdownBadge 复用） | 4 |
| `82dd30c` | training 培训页（统计卡片 + 进度条 + 状态 chip） | 3 |
| `dc6bdaf` | order_detail 订单详情页（6 节点进度 + 倒计时 + 确认/拒接 + 客户信息） | 7 |

**关键设计**：

1. **API 模式**：复用 `lib/services/api_client.dart`（已含 dio + 拦截器），新增端点按 static 路径常量 + free function
2. **provider 模式**：FutureProvider / StreamProvider / AsyncNotifier（与 v1.1 `invitation_provider` 一致）
3. **页面模式**：Material 3 + Card + ListView + 空态 / loading + Stepper 自绘
4. **测试模式**：flutter_test + data-test 属性（不实际跑）

**累计测试用例**：**65 个新测试**（API 14 + provider 32 + page 29）

**路由挂载**：
- `/auth/login`（E3）
- `/home/profile`（E4）
- `/home/wallet`（E5）
- `/home/orders`（E6）
- `/home/training`（E7）
- `/home/orders/:id`（E8 动态参数）

**未做（留给后续）**：

- 真实 wechat_kit / fluwx 拿 wxCode（v1 用占位）
- 实名认证图片上传走 OSS / 七牛预签名 URL
- 订单详情 GPS 签到 + 打卡
- 评价列表独立成页（reviewProvider 已就绪）
- 跑 flutter analyze + flutter test 验证

**端到端联通（v1.3 目标）**：

- escort-app 完整陪诊师流：login → invitations（v1.1）→ my-availability（v1.1）→ 订单详情（30s 倒计时确认/拒接）→ 服务执行（v2 GPS）→ wallet 提现 → training 课程 → profile 实名/退出
- 共享 X-Trace-Id：`escort-{ms}-{rand6}`（与 patient-miniapp `mp-` 不同）
- 后端 11 个 Go 服务 59 包 0 FAIL + escort-app 65 新测试
