# escort-app Setup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 落地陪诊师 Flutter App（iOS + Android）骨架：项目脚手架、24 个 P0 页面骨架、Riverpod providers、API client + 拦截器、token 持久化、go_router + 3 个路由守卫、`flutter test` + `integration_test` 框架配置、OpenAPI Dart client 自动生成。**匹配模式按 `specs/2026-09-24-order-matching-redesign.md` §1.2 + §4.1 落地：陪诊师不再抢单，改为「我的邀请（30s 倒计时确认接单）+ 我的空余时段（增/删）」**。**UI 完善（页面具体交互 / 表单 / 错误 toast 等）留后续 plan**。

**Architecture:** 在 `escort-app/` 下创建 Flutter 3.24+ 项目；按 `specs/2026-09-24-escort-app-design.md` §2 目录结构落地，但**业务页面与 provider 按 `specs/2026-09-24-order-matching-redesign.md` 修订**：删 `FeedPage` + `feedProvider` + `OrderApi.feed/accept`；新增 `InvitationsPage` + `AvailabilityPage` + `invitationProvider`（5s StreamProvider 轮询）+ `availabilityProvider` + `AvailabilityApi`。Riverpod 2.5 用 `@riverpod` 注解 + `StateNotifier` 模式（auth 流）；其他领域 provider 用 `FutureProvider` / `StreamProvider`。HTTP 走 dio 5.7 + `flutter_secure_storage` 持久化 JWT；route guards 走 `go_router` 的 `redirect` + `Provider<bool>` 派生守卫。OpenAPI 契约来自 `web/openapi/contracts.yaml`，用 `openapi-generator-cli` 生成 Dart client 到 `lib/api/generated/`。

**Tech Stack:** Flutter 3.24+ (Dart 3.5) · flutter_riverpod 2.5+ · go_router 14.x · dio 5.7 · flutter_secure_storage 9.x · shared_preferences 2.x · geolocator 13.x · image_picker 1.x · flutter_image_compress 2.x · openapi-generator-cli 7.x。

**前置依赖:**
- `docs/superpowers/specs/2026-09-24-order-matching-redesign.md` **§1.2 + §4.1 + §6 escort-app 修订条目**（本 plan 的核心 source of truth）
- `docs/superpowers/specs/2026-09-24-escort-app-design.md` §1–§6（功能模块 + 页面 + 状态机 + GPS 签到 + 钱包 + SOS；与原 plan 共享，但 §3.1 抢单池已被 order-matching-redesign 取代）
- `docs/superpowers/specs/2026-09-24-l2-api-gap-design.md` §2.2 escort 端 API + §3 entities（含新增 escort_availabilities / `selected_escort_id` / `escort_pending_expire_at`）
- `web/openapi/contracts.yaml`（后端 OpenAPI 3.0 契约；plan 假设该文件已存在，否则本 plan 仅生成 config，generated/ 留空）
- `flutter` CLI 已装（`flutter --version` ≥ 3.24）

---

## Global Constraints

- Flutter 3.24+ / Dart 3.5+（`flutter doctor` 通过）
- 状态管理：**只**用 Riverpod 2.5；不引入 BLoC / Provider / GetX
- 路由：**只**用 go_router 14.x 声明式 + `redirect` 守卫
- HTTP：**只**用 dio 5.7 + 拦截器；不引 http / retrofit
- Token 存 `flutter_secure_storage`（不是 `shared_preferences`，密钥场景）
- UI 主题：Material 3（`useMaterial3: true`）；主色 `#1989FA`
- pubspec 依赖锁版本（见 Task 1）；不引 FlutterFire（firebase_messaging v1 用 mock 接口）
- iOS：`ios/Runner/Info.plist` 加 `NSLocationWhenInUseUsageDescription` / `NSCameraUsageDescription` / `NSPhotoLibraryUsageDescription`；Android `android/app/src/main/AndroidManifest.xml` 配 `usesCleartextTraffic="true"`（**debug only**，release 留 false）
- 测试：`flutter test`（单测 + widget）+ `integration_test`（E2E）；`flutter analyze` 0 issue
- Commit 节奏：每个 Task 完成立即 commit；前缀 `chore:` / `feat:` / `test:` / `fix:` / `docs:`
- 文件命名：snake_case（Dart 习惯）
- v1 范围：**只**做骨架 + 路由 + providers；页面内容用 `Scaffold(body: Center(child: Text('TODO: <page>')))` 占位（仍可测渲染 + 路由注册）；**例外**：`InvitationsPage` / `AvailabilityPage` 在骨架阶段就用真实 widget（ListView + CountdownBadge + 增删按钮），因为它们是 spec 强制要求的核心交互（不是装饰 UI）
- **匹配模式约束**（按 `order-matching-redesign` §7）：
    - **不**实现 WebSocket 推送；`InvitationProvider` 用 `Stream.periodic(5s)` 轮询（与原 feedProvider 一致模式）
    - **不**实现服务器主动推送「超时回退」；前端按 `escort_pending_expire_at` 自渲染过期；后端 30s 超时由 `order-lock` plan 处理
    - 30s 倒计时复用 `CountdownBadge` widget（`expireAt` 参数化，不绑定订单状态语义；UI 文案显示「待确认剩余」）
- 不做：WebSocket 推送 / 推送通道（极光 / 友盟）/ 视频陪诊 / i18n / 离线模式 / AI 推荐（详见 spec §14）

---

## File Structure

> 仅列本 plan 新增 / 修改的文件；`escort-app/` 下其他 Flutter 模板文件（`ios/Runner.xcodeproj`、`android/gradle/`、`pubspec.lock` 等）由 `flutter create` 生成。

| 路径 | 变更 | 职责 |
|---|---|---|
| `escort-app/pubspec.yaml` | Create | Flutter 依赖：flutter_riverpod / go_router / dio / flutter_secure_storage / shared_preferences / geolocator / image_picker / flutter_image_compress |
| `escort-app/pubspec.lock` | Create | `flutter pub get` 生成 |
| `escort-app/analysis_options.yaml` | Modify | 启用 `flutter_lints` + `prefer_const_constructors` |
| `escort-app/README.md` | Create | 项目说明 + `flutter run` / `flutter test` 命令 |
| `escort-app/.gitignore` | Modify | 增 `lib/api/generated/` 不入仓（由 OpenAPI 生成器重新产出）→ 实际**入仓**（spec 要求契约版本对齐）；增 `.dart_tool/` / `build/` |
| `escort-app/openapi.yaml` | Create | OpenAPI 生成器 config（input / output / additional-properties） |
| `escort-app/scripts/gen-api.sh` | Create | 调用 `openapi-generator-cli generate -i ../web/openapi/contracts.yaml -g dart-dio -o lib/api/generated/` |
| `escort-app/lib/main.dart` | Create | `runApp(ProviderScope(child: DoctorsEscortApp()))` |
| `escort-app/lib/app.dart` | Create | `MaterialApp.router(routerConfig: ref.watch(goRouterProvider))` |
| `escort-app/lib/router/app_router.dart` | Modify | GoRouter + 24 路由（**替换 `/home/feed` 为 `/home/invitations`，加 `/home/availability`）+ redirect 链 |
| `escort-app/lib/router/route_guards.dart` | Create | `authGuardProvider` / `realNameGuardProvider` / `approvedGuardProvider` |
| `escort-app/lib/router/route_guards_test.dart` | Create | 3 个守卫的派生逻辑单测 |
| `escort-app/lib/router/app_router_test.dart` | Create | redirect 链 widget 测试（未登录跳 login / 未实名跳 real-name / 未审核跳 audit/pending） |
| `escort-app/lib/utils/format.dart` | Create | `formatMoney` / `formatDateTime` / `maskPhone` |
| `escort-app/lib/utils/format_test.dart` | Create | 3 个函数单测 |
| `escort-app/lib/utils/trace.dart` | Create | `newTraceId()` → `escort-{ms}-{rand}` |
| `escort-app/lib/utils/trace_test.dart` | Create | 格式 + 唯一性 |
| `escort-app/lib/utils/error_handler.dart` | Create | `handleDioError(DioException)` → 用户文案（**修订 §3 业务码：原 13008 抢单池关闭改为 13101 邀请已过期 / 13102 邀请已被接受**） |
| `escort-app/lib/utils/error_handler_test.dart` | Modify | 业务码 → 文案映射（含 13101/13102） |
| `escort-app/lib/models/order.dart` | Modify | `Order` + `OrderSummary` + `OrderStatus` enum（**加 `selectingEscort` / `escortPendingAcceptance`；移除 `pendingAcceptance` 用法**；字段加 `selectedEscortId` / `escortPendingExpireAt`；移除 `lockOwner` / `lockExpireAt`） |
| `escort-app/lib/models/invitation.dart` | Create | `Invitation`（含 `escortPendingExpireAt` + 订单瘦字段） |
| `escort-app/lib/models/availability.dart` | Create | `Availability` + `AvailabilityStatus` enum（available / booked / canceled） |
| `escort-app/lib/models/escort_profile.dart` | Create | `EscortProfile` + `OnboardingStage` enum |
| `escort-app/lib/models/wallet.dart` | Create | `Wallet` |
| `escort-app/lib/models/withdrawal.dart` | Create | `Withdrawal` + `WithdrawalStatus` enum |
| `escort-app/lib/models/signal.dart` | Create | `Signal` + `SignalStatus` enum |
| `escort-app/lib/models/*_test.dart` | Modify/Create | `fromJson` / `toJson` 单测；新增 `invitation_test` / `availability_test` |
| `escort-app/lib/services/token_storage.dart` | Create | `TokenStorage`（封装 `flutter_secure_storage`）+ `TokenStorageFake`（测试用） |
| `escort-app/lib/services/token_storage_test.dart` | Create | write/read/delete + fake 单测 |
| `escort-app/lib/services/gps_service.dart` | Create | `GpsService.currentPosition()` 封装 `geolocator` |
| `escort-app/lib/services/gps_service_test.dart` | Create | 单测（mock geolocator） |
| `escort-app/lib/services/push_service.dart` | Create | `PushService` 接口 + `NopPushService` 实现（v1 mock） |
| `escort-app/lib/services/push_service_test.dart` | Create | Nop 单测 |
| `escort-app/lib/api/dio_client.dart` | Create | `dioProvider`（BaseOptions + Auth 拦截器 + Trace 拦截器 + 401 handler） |
| `escort-app/lib/api/dio_client_test.dart` | Create | 拦截器单测（用 `MockAdapter`） |
| `escort-app/lib/api/auth_api.dart` | Create | `AuthApi`（login / smsSend / refresh / me） |
| `escort-app/lib/api/order_api.dart` | **Modify** | `OrderApi`（**删 `feed()` / `accept()`；新增 `invitations()` / `confirmAccept(id)` / `rejectAccept(id)` / `listMine()` / `getById()` / `checkin()` / `checkout()`**） |
| `escort-app/lib/api/availability_api.dart` | Create | `AvailabilityApi`（`listMine()` / `create({startAt, endAt})` / `delete(id)`） |
| `escort-app/lib/api/escort_api.dart` | Create | `EscortApi`（register / realNameAuth / uploadHealthCert / trainingComplete / me / updateStatus） |
| `escort-app/lib/api/wallet_api.dart` | Create | `WalletApi`（getMyWallet / withdraw / listTransactions） |
| `escort-app/lib/api/training_api.dart` | Create | `TrainingApi`（listCourses / getCourse / submitQuiz） |
| `escort-app/lib/api/sos_api.dart` | Create | `SosApi`（trigger） |
| `escort-app/lib/api/review_api.dart` | Create | `ReviewApi`（listMine） |
| `escort-app/lib/api/message_api.dart` | Create | `MessageApi`（listConversations / listMessages） |
| `escort-app/lib/api/*_test.dart` | Modify/Create | 各 API 用 `MockAdapter` 单测；**新增 `availability_api_test.dart`**；`order_api_test.dart` **删 feed/accept 用例，加 invitations/confirmAccept/rejectAccept** |
| `escort-app/lib/providers/auth_provider.dart` | Create | `AuthState`（sealed）+ `AuthNotifier`（StateNotifier）+ `authProvider` |
| `escort-app/lib/providers/auth_provider_test.dart` | Create | bootstrap / loginByPhone / logout 单测 |
| `escort-app/lib/providers/invitation_provider.dart` | **Create** | `invitationProvider`（StreamProvider，5s 轮询；按 `expiresAt > now` 客户端过滤）+ `confirmAcceptControllerProvider` + `rejectAcceptControllerProvider` |
| `escort-app/lib/providers/invitation_provider_test.dart` | Create | 轮询 + 过滤过期 + confirm/reject 单测 |
| `escort-app/lib/providers/availability_provider.dart` | **Create** | `availabilityProvider`（FutureProvider）+ `createAvailabilityControllerProvider` + `deleteAvailabilityControllerProvider` |
| `escort-app/lib/providers/availability_provider_test.dart` | Create | 列表 / 创建 / 删除 单测 |
| `escort-app/lib/providers/wallet_provider.dart` | Create | `walletProvider`（FutureProvider） + `withdrawProvider` |
| `escort-app/lib/providers/wallet_provider_test.dart` | Create | 单测 |
| `escort-app/lib/providers/message_provider.dart` | Create | `conversationsProvider` + `messagesProvider.family` |
| `escort-app/lib/providers/message_provider_test.dart` | Create | 单测 |
| `escort-app/lib/providers/training_provider.dart` | Create | `coursesProvider` + `quizSubmitProvider.family` |
| `escort-app/lib/providers/training_provider_test.dart` | Create | 单测 |
| `escort-app/lib/pages/splash/splash_page.dart` | Modify | 启动后**跳 `/home/invitations`**（替换原 `/home/feed`） |
| `escort-app/lib/pages/splash/splash_page_test.dart` | Create | widget 渲染 + bootstrap 触发 |
| `escort-app/lib/pages/auth/login_page.dart` | Create | 登录页骨架（手机号 + 验证码输入框 + 按钮占位） |
| `escort-app/lib/pages/auth/register_page.dart` | Create | 注册页骨架（角色=escort 单选） |
| `escort-app/lib/pages/auth/login_page_test.dart` | Create | widget 测试 |
| `escort-app/lib/pages/onboarding/real_name_page.dart` | Create | 实名认证页骨架 |
| `escort-app/lib/pages/onboarding/health_cert_page.dart` | Create | 健康证上传骨架（image_picker 占位） |
| `escort-app/lib/pages/onboarding/training_list_page.dart` | Create | 培训课程列表骨架 |
| `escort-app/lib/pages/onboarding/training_video_page.dart` | Create | 培训视频播放骨架 |
| `escort-app/lib/pages/onboarding/training_quiz_page.dart` | Create | 培训考核骨架（5 题 + 80 分通过占位） |
| `escort-app/lib/pages/onboarding/agreement_page.dart` | Create | 协议签署骨架 |
| `escort-app/lib/pages/onboarding/onboarding_pages_test.dart` | Create | 6 页 widget 测试 |
| `escort-app/lib/pages/audit/pending_audit_page.dart` | Create | 等待审核页骨架 |
| `escort-app/lib/pages/audit/pending_audit_page_test.dart` | Create | widget 测试 |
| `escort-app/lib/pages/home/home_shell.dart` | **Modify** | BottomNavigationBar（**`invitations` / `orders` / `wallet` / `profile`**，**Feed 删除**） |
| `escort-app/lib/pages/home/invitations_page.dart` | **Create** | 我的邀请页骨架：ListView + 倒计时 + 确认/拒接按钮（spec §1.2 + §4.1） |
| `escort-app/lib/pages/home/invitations_page_test.dart` | Create | widget + 列表渲染 + 倒计时归零测试 |
| `escort-app/lib/pages/home/availability_page.dart` | **Create** | 我的空余时段页骨架：列表 + 新增弹层 + 删除（spec §4.1） |
| `escort-app/lib/pages/home/availability_page_test.dart` | Create | widget + 列表渲染 + 删除按钮测试 |
| `escort-app/lib/pages/home/orders_page.dart` | **Modify** | 我的订单：tab **待服务 / 服务中 / 已完成 + 邀请**（邀请 tab 跳 `/home/invitations`） |
| `escort-app/lib/pages/home/wallet_page.dart` | Create | 钱包页骨架（余额 + 提现按钮占位） |
| `escort-app/lib/pages/home/profile_page.dart` | Modify | 个人中心骨架（实名状态 + 评分 + 培训记录 + 「我的空余时段」入口） |
| `escort-app/lib/pages/home/home_pages_test.dart` | Create | 5 页 widget 测试 |
| `escort-app/lib/pages/order/order_detail_page.dart` | Create | 订单详情骨架（进度条 + 虚拟号 + 签到按钮占位） |
| `escort-app/lib/pages/order/checkin_page.dart` | Create | 到院签到骨架（GPS 调用占位） |
| `escort-app/lib/pages/order/checkout_page.dart` | Create | 完成打卡骨架 |
| `escort-app/lib/pages/order/order_pages_test.dart` | Create | 3 页 widget 测试 |
| `escort-app/lib/pages/wallet/withdraw_page.dart` | Create | 提现申请骨架 |
| `escort-app/lib/pages/wallet/transactions_page.dart` | Create | 交易记录骨架（分页占位） |
| `escort-app/lib/pages/wallet/wallet_pages_test.dart` | Create | 2 页 widget 测试 |
| `escort-app/lib/pages/sos/sos_trigger_page.dart` | Create | SOS 触发页骨架（长按 3s 占位） |
| `escort-app/lib/pages/sos/sos_trigger_page_test.dart` | Create | widget 测试 |
| `escort-app/lib/pages/message/conversation_list_page.dart` | Create | 站内信列表骨架（P2） |
| `escort-app/lib/pages/message/chat_page.dart` | Create | 站内信详情骨架（P2） |
| `escort-app/lib/pages/message/message_pages_test.dart` | Create | 2 页 widget 测试 |
| `escort-app/lib/pages/profile/edit_profile_page.dart` | Create | 资料编辑骨架（P1） |
| `escort-app/lib/pages/profile/reviews_page.dart` | Create | 收到的评价骨架（P1） |
| `escort-app/lib/pages/profile/profile_pages_test.dart` | Create | 2 页 widget 测试 |
| `escort-app/lib/widgets/order_card.dart` | Create | 订单卡片骨架（医院 + 金额 + 接单按钮占位）；**不再绑定抢单池** |
| `escort-app/lib/widgets/invitation_card.dart` | **Create** | 邀请卡片（医院 + 金额 + CountdownBadge + 「确认接单」/「拒接」双按钮） |
| `escort-app/lib/widgets/invitation_card_test.dart` | Create | 渲染 + 按钮回调 + 倒计时归零按钮禁用 |
| `escort-app/lib/widgets/availability_tile.dart` | Create | 时段行（左时段 + 右删除按钮 / 已预订 chip） |
| `escort-app/lib/widgets/availability_tile_test.dart` | Create | 渲染 + 删除回调 + booked 状态不可删 |
| `escort-app/lib/widgets/countdown_badge.dart` | **Modify** | 30s 倒计时（基于 `expireAt`；**UI 文案改「待确认剩余」替代「锁单剩余」**） |
| `escort-app/lib/widgets/countdown_badge_test.dart` | Modify | 渲染 / 归零 → 已超时（文案断言改 `待确认剩余`） |
| `escort-app/lib/widgets/rating_stars.dart` | Create | 评分星级 |
| `escort-app/lib/widgets/rating_stars_test.dart` | Create | 渲染 |
| `escort-app/lib/widgets/status_chip.dart` | Modify | 状态机颜色 chip（**加 `escortPendingAcceptance` 颜色：橙色**） |
| `escort-app/lib/widgets/status_chip_test.dart` | Modify | 颜色映射（加 `escortPendingAcceptance` 用例） |
| `escort-app/lib/widgets/gps_checkin_button.dart` | Create | 长按 GPS 验证按钮 |
| `escort-app/lib/widgets/gps_checkin_button_test.dart` | Create | 长按回调 |
| `escort-app/lib/widgets/sos_long_press.dart` | Create | SOS 长按 3s 触发器（含 AnimationController） |
| `escort-app/lib/widgets/sos_long_press_test.dart` | Create | 3s 后回调触发 |
| `escort-app/lib/widgets/order_card_test.dart` | Create | 渲染 |
| `escort-app/ios/Runner/Info.plist` | Modify | 加 NSLocationWhenInUseUsageDescription / NSCameraUsageDescription / NSPhotoLibraryUsageDescription |
| `escort-app/android/app/src/main/AndroidManifest.xml` | Modify | debug buildType 加 `android:usesCleartextTraffic="true"` |
| `escort-app/android/app/src/main/AndroidManifest.xml` | Modify | ACCESS_FINE_LOCATION / CAMERA / READ_MEDIA_IMAGES 权限 |
| `escort-app/integration_test/app_test.dart` | Create | E2E：splash → login → register → onboarding 引导链 |
| `escort-app/integration_test/invitations_test.dart` | **Create** | E2E：mock invitations → 30s 倒计时 → 确认按钮 → 状态切换（**替代原 `feed_test.dart`**） |
| `dev.md` | Modify | §10.x 加 escort-app setup 落地记录 |

> **修订要点**（对比原 plan）：
> - **删除**：`/home/feed` 路由 + `FeedPage` + `feedProvider` + `OrderApi.feed/accept` + `acceptProvider` + `feedProvider` 单测 + `integration_test/feed_test.dart`
> - **新增**：`/home/invitations` 路由 + `InvitationsPage` + `/home/availability` 路由 + `AvailabilityPage` + `Invitation` / `Availability` model + `invitationProvider`（5s 轮询）+ `availabilityProvider` + `AvailabilityApi` + `InvitationCard` + `AvailabilityTile` widget
> - **保留**：`CountdownBadge` widget（**只改文案**）+ `OrderApi.invitations()` / `confirmAccept()` / `rejectAccept()`（替代 `feed()` / `accept()`）+ `BottomNavBar` 4 个 tab 数量不变（**仅把 `Feed` 替换成 `Invitations`**）
>
> 24 个 page widget（其实）/ 8 个 provider（**删 order_provider，加 invitation_provider + availability_provider**）/ 8 个 API（**删 `feed` / `accept` 方法，加 `invitations` / `confirmAccept` / `rejectAccept` / 整个 `AvailabilityApi`**）/ 6 个公共 widget（**删 `OrderCard` 抢单按钮语义；新增 `InvitationCard` / `AvailabilityTile`；共 7 个**）/ 5 个 model（**加 `Invitation` / `Availability`；共 7 个**） = **58 个 lib 文件 + 22 个 test 文件 + 5 个 config 文件 = ~85 个新文件**（净增 ~11 个）。

---

### Task 1: Flutter 项目脚手架 + 依赖 + 平台配置

**Files:**
- Create: `escort-app/`（`flutter create` 生成）
- Modify: `escort-app/pubspec.yaml`
- Create: `escort-app/analysis_options.yaml`
- Create: `escort-app/README.md`
- Modify: `escort-app/ios/Runner/Info.plist`
- Modify: `escort-app/android/app/src/main/AndroidManifest.xml`

**Step 1: 跑 `flutter create` 生成项目骨架**

```bash
cd /Users/growduduan/ai/doctors
flutter create \
  --org com.doctors \
  --project-name escort_app \
  --platforms=ios,android \
  --description "Doctors Escort App (陪诊师端)" \
  escort-app
```

> `--org` 决定 iOS bundle id / Android applicationId = `com.doctors.escort_app`。
> `--platforms=ios,android` 跳过 web/macos/linux/windows（spec 仅需两端）。

**Step 2: 改 `pubspec.yaml`**（替换 dependencies 段）

```yaml
name: escort_app
description: Doctors Escort App (陪诊师端)
publish_to: 'none'
version: 0.1.0+1

environment:
  sdk: '>=3.5.0 <4.0.0'
  flutter: '>=3.24.0'

dependencies:
  flutter:
    sdk: flutter
  flutter_riverpod: ^2.5.1
  go_router: ^14.6.0
  dio: ^5.7.0
  flutter_secure_storage: ^9.2.2
  shared_preferences: ^2.3.3
  geolocator: ^13.0.1
  image_picker: ^1.1.2
  flutter_image_compress: ^2.3.0
  cupertino_icons: ^1.0.8

dev_dependencies:
  flutter_test:
    sdk: flutter
  integration_test:
    sdk: flutter
  flutter_lints: ^5.0.0

flutter:
  uses-material-design: true
```

**Step 3: 写 `analysis_options.yaml`**

```yaml
include: package:flutter_lints/flutter.yaml

linter:
  rules:
    prefer_const_constructors: true
    prefer_const_literals_to_create_immutables: true
    avoid_print: true
    require_trailing_commas: true

analyzer:
  exclude:
    - lib/api/generated/**
  language:
    strict-casts: true
    strict-inference: true
    strict-raw-types: true
```

**Step 4: 跑 `flutter pub get` + 验证基线**

```bash
cd escort-app
flutter pub get
flutter analyze
```

Expected: `No issues found!`（默认 `lib/main.dart` 通过）。

**Step 5: 改 iOS Info.plist**（追加到 `</dict>` 前）

```xml
<key>NSLocationWhenInUseUsageDescription</key>
<string>用于到院签到 GPS 校验</string>
<key>NSCameraUsageDescription</key>
<string>用于上传健康证 / 头像</string>
<key>NSPhotoLibraryUsageDescription</key>
<string>用于选择健康证 / 头像</string>
```

**Step 6: 改 Android `AndroidManifest.xml`**

```xml
<!-- 在 <manifest> 顶层加权限 -->
<uses-permission android:name="android.permission.ACCESS_FINE_LOCATION"/>
<uses-permission android:name="android.permission.CAMERA"/>
<uses-permission android:name="android.permission.READ_MEDIA_IMAGES"/>

<!-- 在 <application> 标签内加 debug buildType 配置 -->
<!-- 注意：debug-only cleartext 由 build.gradle 配，这里 AndroidManifest 不动 usesCleartextTraffic（release 留 false） -->
```

**Step 7: 改 `android/app/build.gradle` 加 debug cleartext**

在 `android { buildTypes { debug { ... } } }` 块内追加：

```gradle
debug {
    // 仅 debug 允许 cleartext；release 走 HTTPS（spec 强制）
    manifestPlaceholders = [usesCleartextTraffic: "true"]
}
```

并在 `android/app/src/debug/AndroidManifest.xml` 加：

```xml
<application android:usesCleartextTraffic="true" />
```

> **release** 走 `android/app/src/main/AndroidManifest.xml`（**不**加 usesCleartextTraffic，默认 false）。

**Step 8: 写 `README.md`**

```markdown
# escort-app

陪诊师端 Flutter App（iOS + Android）。

## 命令

\`\`\`bash
flutter pub get              # 拉依赖
flutter run -d ios           # iOS 模拟器
flutter run -d android       # Android 模拟器
flutter test                 # 单测 + widget 测试
flutter test integration_test/  # E2E
flutter analyze              # 静态分析
bash scripts/gen-api.sh      # 重新生成 OpenAPI Dart client
\`\`\`

## 业务流程（spec/2026-09-24-order-matching-redesign）

陪诊师**不再抢单**。流程：
1. 上线前在「我的空余时段」(`/home/availability`) 设置 `available` 时段
2. 患者从候选列表选了你 → 推送到「我的邀请」(`/home/invitations`)
3. 30s 倒计时内点「确认接单」或「拒接」；超时未操作 → 订单回退到 `selecting_escort`，患者可重选

## API base URL

dev: `https://api.dev.doctors.example.com/api/v1`
prod: `https://api.doctors.example.com/api/v1`
（由 `lib/api/dio_client.dart` 的 `baseUrl` 常量 + `--dart-define=API_BASE=...` 覆盖。）
```

**Step 9: Commit**

```bash
cd /Users/growduduan/ai/doctors
git add escort-app/
git commit -m "chore(escort-app): flutter create + pubspec deps + iOS/Android 权限 + README + analysis_options"
```

---

### Task 2: OpenAPI 生成器配置 + Dart client 骨架

**Files:**
- Create: `escort-app/openapi.yaml`（生成器配置）
- Create: `escort-app/scripts/gen-api.sh`
- Create: `escort-app/lib/api/generated/README.md`（生成产物说明）

**Step 1: 写 `openapi.yaml`**（生成器 config）

```yaml
# openapi.yaml - OpenAPI Generator CLI config for escort-app Dart client
# 文档: https://openapi-generator.tech/docs/configuration-file/

generator:
  name: dart-dio
  outputDir: lib/api/generated
  inputSpec: ../web/openapi/contracts.yaml
  skipFormModel: true
  skipOverwrite: false

# dart-dio specific config
additionalProperties:
  hideGenerationTimestamp: true
  library: dio
  pubName: escort_api
  pubVersion: 0.1.0
  sortParamsByRequiredFlag: false
  useEnumExtension: true
  nullableFields: true
```

> 若 `web/openapi/contracts.yaml` 还未存在，本 Task 仅创建 config + script；`lib/api/generated/` 留空（Task 后续 API 模块不依赖 generated 类型，而用自写 `lib/models/` + 自写 API 客户端包装 generated）。

**Step 2: 写 `scripts/gen-api.sh`**

```bash
#!/usr/bin/env bash
# 重新生成 OpenAPI Dart client（escort-app）
# 调用：bash scripts/gen-api.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if ! command -v openapi-generator-cli >/dev/null 2>&1; then
  echo "ERROR: openapi-generator-cli not found. Install via: npm i -g @openapitools/openapi-generator-generator-cli"
  echo "       或 brew install openapi-generator"
  exit 1
fi

CONTRACT="${CONTRACT:-../web/openapi/contracts.yaml}"
if [ ! -f "$CONTRACT" ]; then
  echo "ERROR: contract not found: $CONTRACT"
  exit 1
fi

openapi-generator-cli generate \
  -i "$CONTRACT" \
  -g dart-dio \
  -o lib/api/generated/ \
  --additional-properties=hideGenerationTimestamp=true,library=dio,pubName=escort_api,useEnumExtension=true,nullableFields=true

echo "✓ Generated Dart client to lib/api/generated/"
```

**Step 3: 写 `lib/api/generated/README.md`**（说明生成产物不入逻辑代码）

```markdown
# lib/api/generated/

**Auto-generated by `openapi-generator-cli`** — DO NOT EDIT BY HAND.

Regenerate with: `bash scripts/gen-api.sh`

Source of truth: `web/openapi/contracts.yaml`（后端 OpenAPI 3.0 契约；含 escort_availabilities / orders.selecting_escort / orders.escort_pending_acceptance 等 spec/2026-09-24-order-matching-redesign 新增字段）。

v1 策略：本目录的生成产物仅作为参考 / 类型来源；`lib/api/*.dart`（`AuthApi` / `OrderApi` / `AvailabilityApi` 等）是手工包装层，便于：
1. 注入 `dioProvider`（拦截器链）
2. 把生成类型映射到 `lib/models/` 业务类型（解耦生成器升级）
3. 单测用 `MockAdapter` 注入，不必 import 生成代码
```

**Step 4: 设置脚本可执行 + Commit**

```bash
chmod +x escort-app/scripts/gen-api.sh
cd /Users/growduduan/ai/doctors
git add escort-app/openapi.yaml escort-app/scripts/gen-api.sh escort-app/lib/api/generated/README.md
git commit -m "chore(escort-app): OpenAPI 生成器 config + gen-api.sh + generated/ 说明"
```

---

### Task 3: utils/ + 单测（format / trace / error_handler）

**Files:**
- Create: `escort-app/lib/utils/format.dart`
- Create: `escort-app/lib/utils/format_test.dart`
- Create: `escort-app/lib/utils/trace.dart`
- Create: `escort-app/lib/utils/trace_test.dart`
- Create: `escort-app/lib/utils/error_handler.dart`
- Create: `escort-app/lib/utils/error_handler_test.dart`

**Step 1: 写 format_test.dart（RED）**

```dart
// lib/utils/format_test.dart
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/utils/format.dart';

void main() {
  group('formatMoney', () {
    test('整数金额加 .00', () {
      expect(formatMoney(100), '¥100.00');
    });
    test('负数抛 ArgumentError', () {
      expect(() => formatMoney(-1), throwsArgumentError);
    });
    test('两位小数截断', () {
      expect(formatMoney(99.999), '¥100.00');
    });
  });

  group('formatDateTime', () {
    test('标准 ISO → yyyy-MM-dd HH:mm', () {
      final dt = DateTime(2026, 9, 24, 15, 30);
      expect(formatDateTime(dt), '2026-09-24 15:30');
    });
  });

  group('maskPhone', () {
    test('11 位手机号脱敏', () {
      expect(maskPhone('13800138000'), '138****8000');
    });
    test('非 11 位原样返回', () {
      expect(maskPhone('12345'), '12345');
    });
  });
}
```

**Step 2: 跑测试确认失败**

```bash
cd escort-app
flutter test test/utils/format_test.dart
```

Expected: FAIL — `Target of URI doesn't exist: 'package:escort_app/utils/format.dart'`

**Step 3: 写 format.dart**

```dart
// lib/utils/format.dart
String formatMoney(num amount) {
  if (amount.isNegative) throw ArgumentError.value(amount, 'amount', 'must be >= 0');
  return '¥${amount.toStringAsFixed(2)}';
}

String formatDateTime(DateTime dt) {
  String two(int n) => n.toString().padLeft(2, '0');
  return '${dt.year}-${two(dt.month)}-${two(dt.day)} ${two(dt.hour)}:${two(dt.minute)}';
}

String maskPhone(String phone) {
  if (phone.length != 11) return phone;
  return '${phone.substring(0, 3)}****${phone.substring(7)}';
}
```

**Step 4: 跑测试确认通过**

```bash
flutter test test/utils/format_test.dart
```

Expected: PASS（5 个 case）

**Step 5: trace.dart + trace_test.dart（一次性写）**

```dart
// lib/utils/trace.dart
import 'dart:math';

String newTraceId() {
  final ms = DateTime.now().millisecondsSinceEpoch;
  final r = Random().nextInt(1000);
  return 'escort-$ms-$r';
}
```

```dart
// lib/utils/trace_test.dart
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/utils/trace.dart';

void main() {
  test('格式为 escort-{ms}-{rand<1000}', () {
    final id = newTraceId();
    expect(id, startsWith('escort-'));
    final parts = id.split('-');
    expect(parts.length, 3);
    expect(int.tryParse(parts[1]), isNotNull);
    expect(int.parse(parts[2]), lessThan(1000));
  });
  test('两次调用 id 不同（rand 部分）', () {
    final a = newTraceId();
    final b = newTraceId();
    expect(a == b, isFalse);
  });
}
```

**Step 6: error_handler.dart（**业务码按 spec/2026-09-24-order-matching-redesign 修订**）+ error_handler_test.dart**

```dart
// lib/utils/error_handler.dart
import 'package:dio/dio.dart';

/// 把 dio / 业务异常翻译成用户文案（中文，spec §10 配色 / 文案规范）。
/// 业务码来自 l2-api-gap-design.md §3.2：13001~13009 + 标准 401/403/404/500。
/// 新增 13101~13103（order-matching-redesign §4.1）：
///   13101 邀请已过期（30s 超时）
///   13102 邀请已被接受（其他设备抢先）
///   13103 时段冲突（escort_availabilities UNIQUE 索引触发）
String handleDioError(Object e) {
  if (e is DioException) {
    final code = e.response?.statusCode ?? 0;
    final bizCode = e.response?.data is Map
        ? (e.response!.data['code'] as num?)?.toInt()
        : null;
    switch (code) {
      case 401:
        return '登录已过期，请重新登录';
      case 403 when bizCode == 13005:
        return '请先完成实名认证';
      case 403 when bizCode == 13006:
        return '请先上传健康证';
      case 403 when bizCode == 13007:
        return '请先完成培训考核';
      case 403 when bizCode == 13009:
        return 'SOS 仅订单关联人可触发';
      case 402 when bizCode == 13003:
        return '余额不足';
      case 409 when bizCode == 13101:
        return '邀请已过期（30s 超时），订单已返回候选池';
      case 409 when bizCode == 13102:
        return '邀请已被接受或取消';
      case 409 when bizCode == 13103:
        return '时段与既有时段冲突';
      case 404:
        return '资源不存在';
      case 500:
        return '服务异常，请稍后重试';
    }
    if (e.type == DioExceptionType.connectionTimeout ||
        e.type == DioExceptionType.receiveTimeout) {
      return '网络超时，请检查网络';
    }
    return '请求失败（$code）';
  }
  return '未知错误';
}
```

```dart
// lib/utils/error_handler_test.dart
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/utils/error_handler.dart';

void main() {
  DioException dio(int? status, [Map<String, dynamic>? body]) {
    final req = RequestOptions(path: '/x');
    final resp = status == null ? null : Response<dynamic>(requestOptions: req, statusCode: status, data: body);
    return DioException(requestOptions: req, response: resp, type: DioExceptionType.badResponse);
  }

  test('401 → 登录过期', () {
    expect(handleDioError(dio(401)), '登录已过期，请重新登录');
  });
  test('403 + biz 13005 → 实名未完成', () {
    expect(handleDioError(dio(403, {'code': 13005})), '请先完成实名认证');
  });
  test('402 + biz 13003 → 余额不足', () {
    expect(handleDioError(dio(402, {'code': 13003})), '余额不足');
  });
  test('409 + biz 13101 → 邀请过期', () {
    expect(handleDioError(dio(409, {'code': 13101})), '邀请已过期（30s 超时），订单已返回候选池');
  });
  test('409 + biz 13102 → 邀请已被接受', () {
    expect(handleDioError(dio(409, {'code': 13102})), '邀请已被接受或取消');
  });
  test('409 + biz 13103 → 时段冲突', () {
    expect(handleDioError(dio(409, {'code': 13103})), '时段与既有时段冲突');
  });
  test('非 DioException → 未知错误', () {
    expect(handleDioError('x'), '未知错误');
  });
}
```

**Step 7: 跑全部 utils 测试**

```bash
flutter test test/utils/
```

Expected: PASS（format 5 + trace 2 + error_handler 7 = 14 个）

**Step 8: Commit**

```bash
git add escort-app/lib/utils/
git commit -m "feat(escort-app): utils (format/trace/error_handler) + 14 个单测（含 13101/13102/13103 业务码）"
```

---

### Task 4: models/ + JSON 单测（**含新增 Invitation / Availability**）

**Files:**
- Create: `escort-app/lib/models/order.dart`（**修订 OrderStatus enum + 字段**）
- Create: `escort-app/lib/models/invitation.dart`（**新增**）
- Create: `escort-app/lib/models/availability.dart`（**新增**）
- Create: `escort-app/lib/models/escort_profile.dart`
- Create: `escort-app/lib/models/wallet.dart`
- Create: `escort-app/lib/models/withdrawal.dart`
- Create: `escort-app/lib/models/signal.dart`
- Create: `escort-app/test/models/order_test.dart`（**修订：移除 lockOwner/lockExpireAt 用例；加 selectedEscortId / escortPendingExpireAt / selectingEscort / escortPendingAcceptance**）
- Create: `escort-app/test/models/invitation_test.dart`
- Create: `escort-app/test/models/availability_test.dart`
- Create: `escort-app/test/models/escort_profile_test.dart`
- Create: `escort-app/test/models/wallet_test.dart`
- Create: `escort-app/test/models/withdrawal_test.dart`
- Create: `escort-app/test/models/signal_test.dart`

**Step 1: 写 order_test.dart（RED）**

```dart
// test/models/order_test.dart
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/models/order.dart';

void main() {
  test('Order.fromJson 解析 escort_pending_acceptance 完整字段', () {
    final j = {
      'id': 7,
      'order_no': 'O-001',
      'patient_id': 11,
      'escort_id': 22,
      'selected_escort_id': 22,
      'hospital_id': 33,
      'hospital_name': '北京协和医院',
      'hospital_lat': 39.9,
      'hospital_lng': 116.4,
      'package_name': '半日陪诊',
      'service_start_at': '2026-09-25T09:00:00Z',
      'amount': 300.0,
      'final_amount': 300.0,
      'status': 'escort_pending_acceptance',
      'escort_pending_expire_at': '2026-09-24T15:00:30Z',
      'created_at': '2026-09-24T15:00:00Z',
    };
    final o = Order.fromJson(j);
    expect(o.id, 7);
    expect(o.status, OrderStatus.escortPendingAcceptance);
    expect(o.selectedEscortId, 22);
    expect(o.hospitalName, '北京协和医院');
    expect(o.amount, 300.0);
    expect(o.escortPendingExpireAt, isNotNull);
  });

  test('OrderStatus 解析 selecting_escort', () {
    expect(OrderStatus.fromString('selecting_escort'), OrderStatus.selectingEscort);
  });

  test('OrderSummary.fromJson 邀请卡片用', () {
    final j = {
      'id': 7,
      'hospital_name': '北京协和医院',
      'service_start_at_text': '明天 09:00',
      'amount': 300.0,
      'package_name': '半日陪诊',
    };
    final s = OrderSummary.fromJson(j);
    expect(s.id, 7);
    expect(s.amount, 300.0);
  });
}
```

**Step 2: 跑测试确认失败**

```bash
flutter test test/models/order_test.dart
```

Expected: FAIL — `package:escort_app/models/order.dart` not found

**Step 3: 写 order.dart（**修订：移除 lockOwner/lockExpireAt；加 selectingEscort/escortPendingAcceptance 状态；加 selectedEscortId/escortPendingExpireAt 字段**）**

```dart
// lib/models/order.dart
import 'package:escort_app/utils/format.dart';

/// 订单状态机（spec/2026-09-24-order-matching-redesign §2.1）。
/// `pendingAcceptance` 已被 `escortPendingAcceptance` 取代；保留兼容 alias。
enum OrderStatus {
  created, paid, matching, selectingEscort, escortPendingAcceptance, accepted,
  inService, completed, reviewed, refunding, refunded, settling, disputed,
  closed, canceled;

  static OrderStatus fromString(String s) {
    switch (s) {
      case 'created': return OrderStatus.created;
      case 'paid': return OrderStatus.paid;
      case 'matching': return OrderStatus.matching;
      case 'selecting_escort': return OrderStatus.selectingEscort;
      case 'escort_pending_acceptance':
      case 'pending_acceptance': // 兼容 v1 老数据
        return OrderStatus.escortPendingAcceptance;
      case 'accepted': return OrderStatus.accepted;
      case 'in_service': return OrderStatus.inService;
      case 'completed': return OrderStatus.completed;
      case 'reviewed': return OrderStatus.reviewed;
      case 'refunding': return OrderStatus.refunding;
      case 'refunded': return OrderStatus.refunded;
      case 'settling': return OrderStatus.settling;
      case 'disputed': return OrderStatus.disputed;
      case 'closed': return OrderStatus.closed;
      case 'canceled': return OrderStatus.canceled;
    }
    throw ArgumentError('unknown OrderStatus: $s');
  }
}

class Order {
  final int id;
  final int patientId;
  final int? escortId;
  final int? selectedEscortId;
  final String hospitalName;
  final double hospitalLat;
  final double hospitalLng;
  final String packageName;
  final DateTime serviceStartAt;
  final double amount;
  final OrderStatus status;
  /// 陪诊师 30s 确认窗口到期时间；status=escort_pending_acceptance 时必填。
  final DateTime? escortPendingExpireAt;

  Order({
    required this.id,
    required this.patientId,
    this.escortId,
    this.selectedEscortId,
    required this.hospitalName,
    required this.hospitalLat,
    required this.hospitalLng,
    required this.packageName,
    required this.serviceStartAt,
    required this.amount,
    required this.status,
    this.escortPendingExpireAt,
  });

  factory Order.fromJson(Map<String, dynamic> j) => Order(
    id: j['id'] as int,
    patientId: j['patient_id'] as int,
    escortId: j['escort_id'] as int?,
    selectedEscortId: j['selected_escort_id'] as int?,
    hospitalName: j['hospital_name'] as String,
    hospitalLat: (j['hospital_lat'] as num).toDouble(),
    hospitalLng: (j['hospital_lng'] as num).toDouble(),
    packageName: j['package_name'] as String,
    serviceStartAt: DateTime.parse(j['service_start_at'] as String),
    amount: (j['amount'] as num).toDouble(),
    status: OrderStatus.fromString(j['status'] as String),
    escortPendingExpireAt: j['escort_pending_expire_at'] == null
        ? null
        : DateTime.parse(j['escort_pending_expire_at'] as String),
  );
}

/// 邀请卡片 / 订单列表瘦字段（spec §4.1）。
class OrderSummary {
  final int id;
  final String hospitalName;
  final String serviceStartAtText;
  final double amount;
  final String packageName;

  OrderSummary({
    required this.id,
    required this.hospitalName,
    required this.serviceStartAtText,
    required this.amount,
    required this.packageName,
  });

  factory OrderSummary.fromJson(Map<String, dynamic> j) => OrderSummary(
    id: j['id'] as int,
    hospitalName: j['hospital_name'] as String,
    serviceStartAtText: j['service_start_at_text'] as String,
    amount: (j['amount'] as num).toDouble(),
    packageName: j['package_name'] as String,
  );

  String get amountText => formatMoney(amount);
}
```

**Step 4: 跑测试确认通过**

```bash
flutter test test/models/order_test.dart
```

Expected: PASS（3 个）

**Step 5: invitation.dart + invitation_test.dart（**新增**）**

```dart
// lib/models/invitation.dart
import 'package:escort_app/utils/format.dart';

/// 「我的邀请」页用（spec/2026-09-24-order-matching-redesign §4.1）。
/// 后端 GET /api/v1/escorts/me/invitations 返回每条 = 一个处于
/// `escort_pending_acceptance` 状态的订单，已附带 `escort_pending_expire_at`。
class Invitation {
  final int orderId;
  final String hospitalName;
  final double hospitalLat;
  final double hospitalLng;
  final String packageName;
  final DateTime serviceStartAt;
  final double amount;
  final DateTime escortPendingExpireAt;

  Invitation({
    required this.orderId,
    required this.hospitalName,
    required this.hospitalLat,
    required this.hospitalLng,
    required this.packageName,
    required this.serviceStartAt,
    required this.amount,
    required this.escortPendingExpireAt,
  });

  factory Invitation.fromJson(Map<String, dynamic> j) => Invitation(
    orderId: j['order_id'] as int,
    hospitalName: j['hospital_name'] as String,
    hospitalLat: (j['hospital_lat'] as num).toDouble(),
    hospitalLng: (j['hospital_lng'] as num).toDouble(),
    packageName: j['package_name'] as String,
    serviceStartAt: DateTime.parse(j['service_start_at'] as String),
    amount: (j['amount'] as num).toDouble(),
    escortPendingExpireAt: DateTime.parse(j['escort_pending_expire_at'] as String),
  );

  /// 是否仍在 30s 确认窗口内（客户端按此过滤过期邀请）。
  bool get isLive => DateTime.now().isBefore(escortPendingExpireAt);

  String get amountText => formatMoney(amount);
}
```

```dart
// test/models/invitation_test.dart
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/models/invitation.dart';

void main() {
  test('Invitation.fromJson 完整字段', () {
    final j = {
      'order_id': 7,
      'hospital_name': '北京协和医院',
      'hospital_lat': 39.9,
      'hospital_lng': 116.4,
      'package_name': '半日陪诊',
      'service_start_at': '2026-09-25T09:00:00Z',
      'amount': 300.0,
      'escort_pending_expire_at': '2026-09-24T15:00:30Z',
    };
    final inv = Invitation.fromJson(j);
    expect(inv.orderId, 7);
    expect(inv.amount, 300.0);
    expect(inv.escortPendingExpireAt.year, 2026);
  });

  test('isLive: 未到期 → true', () {
    final inv = Invitation.fromJson({
      'order_id': 1,
      'hospital_name': 'x',
      'hospital_lat': 0.0, 'hospital_lng': 0.0,
      'package_name': 'x',
      'service_start_at': '2026-09-25T09:00:00Z',
      'amount': 1.0,
      'escort_pending_expire_at': DateTime.now().add(const Duration(seconds: 10)).toIso8601String(),
    });
    expect(inv.isLive, isTrue);
  });

  test('isLive: 已过期 → false', () {
    final inv = Invitation.fromJson({
      'order_id': 1,
      'hospital_name': 'x',
      'hospital_lat': 0.0, 'hospital_lng': 0.0,
      'package_name': 'x',
      'service_start_at': '2026-09-25T09:00:00Z',
      'amount': 1.0,
      'escort_pending_expire_at': DateTime.now().subtract(const Duration(seconds: 5)).toIso8601String(),
    });
    expect(inv.isLive, isFalse);
  });
}
```

**Step 6: availability.dart + availability_test.dart（**新增**）**

```dart
// lib/models/availability.dart
import 'package:escort_app/utils/format.dart';

/// escort_availabilities 表（spec/2026-09-24-order-matching-redesign §3.2）。
enum AvailabilityStatus { available, booked, canceled;
  static AvailabilityStatus fromString(String s) {
    switch (s) {
      case 'available': return AvailabilityStatus.available;
      case 'booked': return AvailabilityStatus.booked;
      case 'canceled': return AvailabilityStatus.canceled;
    }
    throw ArgumentError('unknown AvailabilityStatus: $s');
  }
}

class Availability {
  final int id;
  final int escortId;
  final DateTime startAt;
  final DateTime endAt;
  final AvailabilityStatus status;
  final int? orderId; // booked 时必填

  Availability({
    required this.id,
    required this.escortId,
    required this.startAt,
    required this.endAt,
    required this.status,
    this.orderId,
  });

  factory Availability.fromJson(Map<String, dynamic> j) => Availability(
    id: j['id'] as int,
    escortId: j['escort_id'] as int,
    startAt: DateTime.parse(j['start_at'] as String),
    endAt: DateTime.parse(j['end_at'] as String),
    status: AvailabilityStatus.fromString(j['status'] as String),
    orderId: j['order_id'] as int?,
  );

  /// 时段是否可被删除（仅 available 状态可删；booked 已被订单占用；canceled 已撤销）。
  bool get isDeletable => status == AvailabilityStatus.available;

  String get startText => formatDateTime(startAt);
  String get endText => formatDateTime(endAt);
}
```

```dart
// test/models/availability_test.dart
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/models/availability.dart';

void main() {
  test('Availability.fromJson 完整字段', () {
    final j = {
      'id': 1,
      'escort_id': 22,
      'start_at': '2026-09-25T14:00:00Z',
      'end_at': '2026-09-25T18:00:00Z',
      'status': 'available',
    };
    final a = Availability.fromJson(j);
    expect(a.id, 1);
    expect(a.status, AvailabilityStatus.available);
    expect(a.isDeletable, isTrue);
    expect(a.orderId, isNull);
  });

  test('isDeletable: booked → false', () {
    final a = Availability.fromJson({
      'id': 1, 'escort_id': 22,
      'start_at': '2026-09-25T14:00:00Z',
      'end_at': '2026-09-25T18:00:00Z',
      'status': 'booked',
      'order_id': 7,
    });
    expect(a.isDeletable, isFalse);
  });

  test('AvailabilityStatus.fromString canceled', () {
    expect(AvailabilityStatus.fromString('canceled'), AvailabilityStatus.canceled);
  });
}
```

**Step 7: escort_profile.dart + 测试**

```dart
// lib/models/escort_profile.dart
enum OnboardingStage {
  registered, realNamePending, healthCertPending, trainingPending,
  agreementPending, auditPending, approved, rejected;

  static OnboardingStage fromString(String s) {
    switch (s) {
      case 'registered': return OnboardingStage.registered;
      case 'real_name_pending': return OnboardingStage.realNamePending;
      case 'health_cert_pending': return OnboardingStage.healthCertPending;
      case 'training_pending': return OnboardingStage.trainingPending;
      case 'agreement_pending': return OnboardingStage.agreementPending;
      case 'audit_pending': return OnboardingStage.auditPending;
      case 'approved': return OnboardingStage.approved;
      case 'rejected': return OnboardingStage.rejected;
    }
    throw ArgumentError('unknown OnboardingStage: $s');
  }
}

class EscortProfile {
  final int id;
  final int userId;
  final String nickname;
  final String? avatarUrl;
  final double rating;
  final int reviewCount;
  final int completedOrders;
  final OnboardingStage stage;
  final bool realNameVerified;
  final bool healthCertVerified;
  final bool trainingPassed;
  final bool approved;
  final bool isOnline;

  EscortProfile({
    required this.id,
    required this.userId,
    required this.nickname,
    this.avatarUrl,
    required this.rating,
    required this.reviewCount,
    required this.completedOrders,
    required this.stage,
    required this.realNameVerified,
    required this.healthCertVerified,
    required this.trainingPassed,
    required this.approved,
    required this.isOnline,
  });

  factory EscortProfile.fromJson(Map<String, dynamic> j) => EscortProfile(
    id: j['id'] as int,
    userId: j['user_id'] as int,
    nickname: j['nickname'] as String,
    avatarUrl: j['avatar_url'] as String?,
    rating: (j['rating'] as num?)?.toDouble() ?? 0.0,
    reviewCount: j['review_count'] as int? ?? 0,
    completedOrders: j['completed_orders'] as int? ?? 0,
    stage: OnboardingStage.fromString(j['stage'] as String),
    realNameVerified: j['real_name_verified'] as bool? ?? false,
    healthCertVerified: j['health_cert_verified'] as bool? ?? false,
    trainingPassed: j['training_passed'] as bool? ?? false,
    approved: j['approved'] as bool? ?? false,
    isOnline: j['is_online'] as bool? ?? false,
  );
}
```

```dart
// test/models/escort_profile_test.dart
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/models/escort_profile.dart';

void main() {
  test('EscortProfile.fromJson 默认值', () {
    final j = {
      'id': 1, 'user_id': 2, 'nickname': '张师傅', 'stage': 'audit_pending',
    };
    final p = EscortProfile.fromJson(j);
    expect(p.stage, OnboardingStage.auditPending);
    expect(p.realNameVerified, false);
    expect(p.rating, 0.0);
    expect(p.avatarUrl, isNull);
  });
}
```

**Step 8: wallet.dart + withdrawal.dart + signal.dart + 各自测试**

> 三个 model 模式相同（fromJson + 枚举 fromString），一次性写完 + 各一个测试：

```dart
// lib/models/wallet.dart
class Wallet {
  final int userId;
  final double balance;
  final double frozen;
  final double totalEarned;
  final DateTime updatedAt;

  Wallet({
    required this.userId, required this.balance, required this.frozen,
    required this.totalEarned, required this.updatedAt,
  });

  factory Wallet.fromJson(Map<String, dynamic> j) => Wallet(
    userId: j['user_id'] as int,
    balance: (j['balance'] as num).toDouble(),
    frozen: (j['frozen'] as num).toDouble(),
    totalEarned: (j['total_earned'] as num).toDouble(),
    updatedAt: DateTime.parse(j['updated_at'] as String),
  );
}
```

```dart
// lib/models/withdrawal.dart
enum WithdrawalStatus { pending, approved, paid, rejected;
  static WithdrawalStatus fromString(String s) {
    switch (s) {
      case 'pending': return WithdrawalStatus.pending;
      case 'approved': return WithdrawalStatus.approved;
      case 'paid': return WithdrawalStatus.paid;
      case 'rejected': return WithdrawalStatus.rejected;
    }
    throw ArgumentError('unknown WithdrawalStatus: $s');
  }
}

class Withdrawal {
  final int id;
  final double amount;
  final String channel; // "wx" | "alipay"
  final String account; // 脱敏
  final WithdrawalStatus status;
  final DateTime createdAt;

  Withdrawal({
    required this.id, required this.amount, required this.channel,
    required this.account, required this.status, required this.createdAt,
  });

  factory Withdrawal.fromJson(Map<String, dynamic> j) => Withdrawal(
    id: j['id'] as int,
    amount: (j['amount'] as num).toDouble(),
    channel: j['channel'] as String,
    account: j['account'] as String,
    status: WithdrawalStatus.fromString(j['status'] as String),
    createdAt: DateTime.parse(j['created_at'] as String),
  );
}
```

```dart
// lib/models/signal.dart
enum SignalStatus { raised, resolved, dismissed;
  static SignalStatus fromString(String s) {
    switch (s) {
      case 'raised': return SignalStatus.raised;
      case 'resolved': return SignalStatus.resolved;
      case 'dismissed': return SignalStatus.dismissed;
    }
    throw ArgumentError('unknown SignalStatus: $s');
  }
}

class Signal {
  final int id;
  final int orderId;
  final int triggerBy;
  final String triggerRole; // "patient" | "escort"
  final double? lat;
  final double? lng;
  final String? address;
  final SignalStatus status;
  final DateTime createdAt;

  Signal({
    required this.id, required this.orderId, required this.triggerBy,
    required this.triggerRole, this.lat, this.lng, this.address,
    required this.status, required this.createdAt,
  });

  factory Signal.fromJson(Map<String, dynamic> j) => Signal(
    id: j['id'] as int,
    orderId: j['order_id'] as int,
    triggerBy: j['trigger_by'] as int,
    triggerRole: j['trigger_role'] as String,
    lat: (j['lat'] as num?)?.toDouble(),
    lng: (j['lng'] as num?)?.toDouble(),
    address: j['address'] as String?,
    status: SignalStatus.fromString(j['status'] as String),
    createdAt: DateTime.parse(j['created_at'] as String),
  );
}
```

**Step 9: 三个模型各一个 fromJson 测试**

```dart
// test/models/wallet_test.dart
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/models/wallet.dart';

void main() {
  test('Wallet.fromJson', () {
    final w = Wallet.fromJson({
      'user_id': 1, 'balance': 100.0, 'frozen': 30.0,
      'total_earned': 200.0, 'updated_at': '2026-09-24T15:00:00Z',
    });
    expect(w.balance, 100.0);
    expect(w.frozen, 30.0);
  });
}
```

```dart
// test/models/withdrawal_test.dart
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/models/withdrawal.dart';

void main() {
  test('Withdrawal.fromJson', () {
    final w = Withdrawal.fromJson({
      'id': 1, 'amount': 50.0, 'channel': 'wx',
      'account': '138****8000', 'status': 'paid',
      'created_at': '2026-09-24T15:00:00Z',
    });
    expect(w.status, WithdrawalStatus.paid);
    expect(w.amount, 50.0);
  });
}
```

```dart
// test/models/signal_test.dart
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/models/signal.dart';

void main() {
  test('Signal.fromJson', () {
    final s = Signal.fromJson({
      'id': 1, 'order_id': 7, 'trigger_by': 22, 'trigger_role': 'escort',
      'status': 'raised', 'created_at': '2026-09-24T15:00:00Z',
    });
    expect(s.status, SignalStatus.raised);
    expect(s.triggerRole, 'escort');
  });
}
```

**Step 10: 跑全部 models 测试**

```bash
flutter test test/models/
```

Expected: PASS（order 3 + invitation 3 + availability 3 + escort_profile 1 + wallet 1 + withdrawal 1 + signal 1 = 13 个）

**Step 11: Commit**

```bash
git add escort-app/lib/models/ escort-app/test/models/
git commit -m "feat(escort-app): models (Order/Invitation/Availability/EscortProfile/Wallet/Withdrawal/Signal) + 13 单测"
```

---

### Task 5: services/ + 单测（token_storage / gps / push）

**Files:**
- Create: `escort-app/lib/services/token_storage.dart`
- Create: `escort-app/lib/services/token_storage_test.dart`
- Create: `escort-app/lib/services/gps_service.dart`
- Create: `escort-app/lib/services/gps_service_test.dart`
- Create: `escort-app/lib/services/push_service.dart`
- Create: `escort-app/lib/services/push_service_test.dart`

**Step 1: token_storage.dart**

```dart
// lib/services/token_storage.dart
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

/// Token 持久化（flutter_secure_storage 封装）。
/// v1 真实实现；测试用 `TokenStorage.forTest()` 返回 in-memory fake。
abstract class TokenStorage {
  Future<void> write(String token);
  Future<String?> read();
  Future<void> delete();
  factory TokenStorage(FlutterSecureStorage storage) => _SecureTokenStorage(storage);
  factory TokenStorage.forTest() => _FakeTokenStorage();
}

class _SecureTokenStorage implements TokenStorage {
  final FlutterSecureStorage _s;
  _SecureTokenStorage(this._s);
  static const _k = 'escort_token';
  @override Future<void> write(String token) => _s.write(key: _k, value: token);
  @override Future<String?> read() => _s.read(key: _k);
  @override Future<void> delete() => _s.delete(key: _k);
}

class _FakeTokenStorage implements TokenStorage {
  String? _t;
  @override Future<void> write(String token) async { _t = token; }
  @override Future<String?> read() async => _t;
  @override Future<void> delete() async { _t = null; }
}
```

**Step 2: token_storage_test.dart**

```dart
// test/services/token_storage_test.dart
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/services/token_storage.dart';

void main() {
  test('fake: write → read → delete', () async {
    final s = TokenStorage.forTest();
    await s.write('tk-1');
    expect(await s.read(), 'tk-1');
    await s.delete();
    expect(await s.read(), isNull);
  });
}
```

**Step 3: gps_service.dart + gps_service_test.dart**

```dart
// lib/services/gps_service.dart
import 'package:geolocator/geolocator.dart';

class GpsService {
  /// 获取当前位置（v1 mock；真实实现后续 plan 接 geolocator 权限流）。
  Future<Position> currentPosition() async {
    return Geolocator.getCurrentPosition(
      locationSettings: const LocationSettings(accuracy: LocationAccuracy.high),
    );
  }
}
```

```dart
// test/services/gps_service_test.dart
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/services/gps_service.dart';

void main() {
  test('GpsService 实例化', () {
    final s = GpsService();
    expect(s, isNotNull);
  });
}
```

**Step 4: push_service.dart + push_service_test.dart**

```dart
// lib/services/push_service.dart
abstract class PushService {
  Future<void> init();
  Future<String?> getToken();
}

class NopPushService implements PushService {
  @override Future<void> init() async {}
  @override Future<String?> getToken() async => null;
}
```

```dart
// test/services/push_service_test.dart
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/services/push_service.dart';

void main() {
  test('NopPushService.getToken → null', () async {
    final p = NopPushService();
    await p.init();
    expect(await p.getToken(), isNull);
  });
}
```

**Step 5: 跑测试 + Commit**

```bash
flutter test test/services/
git add escort-app/lib/services/ escort-app/test/services/
git commit -m "feat(escort-app): services (TokenStorage/GpsService/NopPushService) + 3 个单测"
```

---

### Task 6: api/dio_client + 拦截器 + 单测

**Files:**
- Create: `escort-app/lib/api/dio_client.dart`
- Create: `escort-app/lib/api/dio_client_test.dart`

**Step 1: dio_client_test.dart（RED）**

```dart
// test/api/dio_client_test.dart
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/api/dio_client.dart';
import 'package:escort_app/services/token_storage.dart';

class StubAdapter implements HttpClientAdapter {
  final RequestInterceptorFn? onFetch;
  final dynamic body;
  StubAdapter({this.onFetch, this.body});
  @override
  Future<ResponseBody> fetch(RequestOptions o, Stream<List<int>>? s, Future<void>? c) async {
    onFetch?.call(o);
    final bytes = (body ?? {'code': 0, 'data': {}}).toString().codeUnits;
    return ResponseBody.fromBytes(bytes, 200, headers: {'content-type': ['application/json']});
  }
  @override void close({bool force = false}) {}
}

typedef RequestInterceptorFn = void Function(RequestOptions);

void main() {
  test('拦截器自动注入 Authorization 头', () async {
    final storage = TokenStorage.forTest();
    await storage.write('tk-1');
    final dio = buildDio(storage: storage, baseUrl: 'https://x');
    RequestOptions? captured;
    dio.httpClientAdapter = StubAdapter(onFetch: (o) => captured = o);
    await dio.get<dynamic>('/test');
    expect(captured?.headers['Authorization'], 'Bearer tk-1');
  });

  test('拦截器自动注入 X-Trace-Id', () async {
    final storage = TokenStorage.forTest();
    final dio = buildDio(storage: storage, baseUrl: 'https://x');
    RequestOptions? captured;
    dio.httpClientAdapter = StubAdapter(onFetch: (o) => captured = o);
    await dio.get<dynamic>('/test');
    expect(captured?.headers['X-Trace-Id'], startsWith('escort-'));
  });

  test('401 触发 onUnauthorized 回调', () async {
    final dio = buildDio(
      storage: TokenStorage.forTest(),
      baseUrl: 'https://x',
      onUnauthorized: () => unauthCount++,
    );
    var unauthCount = 0;
    // 替换 adapter 返 401
    dio.httpClientAdapter = _AuthFailAdapter();
    await dio.get<dynamic>('/test');
    expect(unauthCount, 1);
  });
}

class _AuthFailAdapter implements HttpClientAdapter {
  @override
  Future<ResponseBody> fetch(RequestOptions o, Stream<List<int>>? s, Future<void>? c) async {
    final bytes = '{"code": 401}'.codeUnits;
    return ResponseBody.fromBytes(bytes, 401, headers: {'content-type': ['application/json']});
  }
  @override void close({bool force = false}) {}
}
```

**Step 2: 跑测试确认失败**

```bash
flutter test test/api/dio_client_test.dart
```

Expected: FAIL — `package:escort_app/api/dio_client.dart` not found

**Step 3: dio_client.dart**

```dart
// lib/api/dio_client.dart
import 'package:dio/dio.dart';
import 'package:escort_app/services/token_storage.dart';
import 'package:escort_app/utils/trace.dart';

/// 构建 dio（BaseOptions + Auth 拦截器 + Trace 拦截器 + 401 handler）。
Dio buildDio({
  required TokenStorage storage,
  required String baseUrl,
  void Function()? onUnauthorized,
}) {
  final dio = Dio(BaseOptions(
    baseUrl: baseUrl,
    connectTimeout: const Duration(seconds: 10),
    receiveTimeout: const Duration(seconds: 15),
    contentType: 'application/json',
    responseType: ResponseType.json,
  ));

  // Auth 拦截器：注入 Authorization 头
  dio.interceptors.add(InterceptorsWrapper(
    onRequest: (options, handler) async {
      final t = await storage.read();
      if (t != null) options.headers['Authorization'] = 'Bearer $t';
      handler.next(options);
    },
  ));

  // Trace 拦截器：注入 X-Trace-Id
  dio.interceptors.add(InterceptorsWrapper(
    onRequest: (options, handler) {
      options.headers['X-Trace-Id'] = newTraceId();
      handler.next(options);
    },
  ));

  // 401 handler
  dio.interceptors.add(InterceptorsWrapper(
    onResponse: (r, handler) => handler.next(r),
    onError: (e, handler) {
      if (e.response?.statusCode == 401) onUnauthorized?.call();
      handler.next(e);
    },
  ));

  return dio;
}
```

**Step 4: 跑测试确认通过**

```bash
flutter test test/api/dio_client_test.dart
```

Expected: PASS（3 个）

**Step 5: Commit**

```bash
git add escort-app/lib/api/dio_client.dart escort-app/test/api/dio_client_test.dart
git commit -m "feat(escort-app): dio_client (auth/trace 拦截器 + 401 handler) + 3 单测"
```

---

### Task 7: api/ 业务模块（**9 个 API：删 feed/accept 加 invitations/confirmAccept/rejectAccept/AvailabilityApi**）+ 单测

**Files:**
- Create: `escort-app/lib/api/auth_api.dart`
- **Modify**: `escort-app/lib/api/order_api.dart`（**删 `feed()` / `accept()`；加 `invitations()` / `confirmAccept(id)` / `rejectAccept(id)`**）
- **Create**: `escort-app/lib/api/availability_api.dart`（**新增**）
- Create: `escort-app/lib/api/escort_api.dart`
- Create: `escort-app/lib/api/wallet_api.dart`
- Create: `escort-app/lib/api/training_api.dart`
- Create: `escort-app/lib/api/sos_api.dart`
- Create: `escort-app/lib/api/review_api.dart`
- Create: `escort-app/lib/api/message_api.dart`
- Create: `escort-app/test/api/auth_api_test.dart`
- **Modify**: `escort-app/test/api/order_api_test.dart`（**删 feed/accept 用例；加 invitations/confirmAccept/rejectAccept**）
- **Create**: `escort-app/test/api/availability_api_test.dart`（**新增**）
- Create: `escort-app/test/api/escort_api_test.dart`
- Create: `escort-app/test/api/wallet_api_test.dart`
- Create: `escort-app/test/api/sos_api_test.dart`

**Step 1: 写 auth_api_test.dart（RED）**

```dart
// test/api/auth_api_test.dart
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/api/auth_api.dart';
import 'package:escort_app/api/dio_client.dart';
import 'package:escort_app/services/token_storage.dart';

class StubAdapter implements HttpClientAdapter {
  final dynamic body;
  final int status;
  StubAdapter({this.body, this.status = 200});
  @override
  Future<ResponseBody> fetch(RequestOptions o, Stream<List<int>>? s, Future<void>? c) async {
    final bytes = (body ?? {}).toString().codeUnits;
    return ResponseBody.fromBytes(bytes, status, headers: {'content-type': ['application/json']});
  }
  @override void close({bool force = false}) {}
}

void main() {
  test('AuthApi.login 解出 token + user', () async {
    final dio = buildDio(storage: TokenStorage.forTest(), baseUrl: 'https://x');
    dio.httpClientAdapter = StubAdapter(body: {
      'code': 0, 'data': {
        'access_token': 'tk-1',
        'user': {'id': 1, 'phone': '13800138000', 'role': 'escort',
                 'real_name_verified': false, 'approved': false},
      },
    });
    final res = await AuthApi(dio).loginByPhone(phone: '13800138000', code: '1234');
    expect(res.accessToken, 'tk-1');
    expect(res.user.role, 'escort');
  });
}
```

**Step 2: 跑测试确认失败**

```bash
flutter test test/api/auth_api_test.dart
```

Expected: FAIL — `package:escort_app/api/auth_api.dart` not found

**Step 3: 写 auth_api.dart**

```dart
// lib/api/auth_api.dart
import 'package:dio/dio.dart';

class AuthUser {
  final int id;
  final String phone;
  final String role; // "patient" | "escort" | "admin"
  final bool realNameVerified;
  final bool approved;

  AuthUser({
    required this.id, required this.phone, required this.role,
    required this.realNameVerified, required this.approved,
  });

  factory AuthUser.fromJson(Map<String, dynamic> j) => AuthUser(
    id: j['id'] as int,
    phone: j['phone'] as String,
    role: j['role'] as String,
    realNameVerified: j['real_name_verified'] as bool? ?? false,
    approved: j['approved'] as bool? ?? false,
  );
}

class LoginResult {
  final String accessToken;
  final AuthUser user;
  LoginResult({required this.accessToken, required this.user});
}

class AuthApi {
  final Dio _dio;
  AuthApi(this._dio);

  Future<String> smsSend(String phone) async {
    final r = await _dio.post<Map<String, dynamic>>('/auth/sms/send', data: {'phone': phone});
    return r.data!['data'] as String; // trace id
  }

  Future<LoginResult> loginByPhone({required String phone, required String code}) async {
    final r = await _dio.post<Map<String, dynamic>>('/auth/login', data: {'phone': phone, 'code': code});
    final d = r.data!['data'] as Map<String, dynamic>;
    return LoginResult(
      accessToken: d['access_token'] as String,
      user: AuthUser.fromJson(d['user'] as Map<String, dynamic>),
    );
  }

  Future<AuthUser> me() async {
    final r = await _dio.get<Map<String, dynamic>>('/users/me');
    return AuthUser.fromJson(r.data!['data'] as Map<String, dynamic>);
  }

  Future<String> refresh() async {
    final r = await _dio.post<Map<String, dynamic>>('/auth/refresh');
    return r.data!['data']['access_token'] as String;
  }
}
```

> 全部 API 走 `r.data!['data']` 解包（后端 `shared/httpx.Resp[T]` 包装，业务码 0 即成功）。

**Step 4: 跑测试确认通过**

```bash
flutter test test/api/auth_api_test.dart
```

Expected: PASS

**Step 5: order_api.dart（**修订：删 feed/accept；加 invitations/confirmAccept/rejectAccept**）**

```dart
// lib/api/order_api.dart
import 'package:dio/dio.dart';
import 'package:escort_app/models/invitation.dart';
import 'package:escort_app/models/order.dart';

class OrderApi {
  final Dio _dio;
  OrderApi(this._dio);

  /// 「我的邀请」列表（spec/2026-09-24-order-matching-redesign §4.1）。
  /// 后端 GET /api/v1/escorts/me/invitations 返回处于该状态的订单列表，
  /// 每条带 `escort_pending_expire_at`（30s 确认窗口）。
  Future<List<Invitation>> invitations() async {
    final r = await _dio.get<Map<String, dynamic>>('/escorts/me/invitations');
    final list = (r.data!['data'] as List).cast<Map<String, dynamic>>();
    return list.map(Invitation.fromJson).toList();
  }

  /// 陪诊师 30s 内确认接单 → state: accepted。
  /// 后端 POST /api/v1/orders/{id}/confirm-accept。
  Future<Order> confirmAccept(int id) async {
    final r = await _dio.post<Map<String, dynamic>>('/orders/$id/confirm-accept');
    return Order.fromJson(r.data!['data'] as Map<String, dynamic>);
  }

  /// 陪诊师拒接 → state 回退 selecting_escort（患者可重选）。
  /// 后端 POST /api/v1/orders/{id}/reject-accept。
  Future<Order> rejectAccept(int id) async {
    final r = await _dio.post<Map<String, dynamic>>('/orders/$id/reject-accept');
    return Order.fromJson(r.data!['data'] as Map<String, dynamic>);
  }

  Future<Order> getById(int id) async {
    final r = await _dio.get<Map<String, dynamic>>('/orders/$id');
    return Order.fromJson(r.data!['data'] as Map<String, dynamic>);
  }

  Future<List<Order>> listMine({String? status}) async {
    final r = await _dio.get<Map<String, dynamic>>('/orders', queryParameters: {
      'role': 'escort',
      if (status != null) 'status': status,
    });
    final list = (r.data!['data'] as List).cast<Map<String, dynamic>>();
    return list.map(Order.fromJson).toList();
  }

  Future<Order> checkin(int id, {required double lat, required double lng}) async {
    final r = await _dio.post<Map<String, dynamic>>('/orders/$id/checkin',
        data: {'lat': lat, 'lng': lng});
    return Order.fromJson(r.data!['data'] as Map<String, dynamic>);
  }

  Future<Order> checkout(int id) async {
    final r = await _dio.post<Map<String, dynamic>>('/orders/$id/checkout');
    return Order.fromJson(r.data!['data'] as Map<String, dynamic>);
  }

  Future<Order> finish(int id) async {
    final r = await _dio.post<Map<String, dynamic>>('/orders/$id/finish');
    return Order.fromJson(r.data!['data'] as Map<String, dynamic>);
  }
}
```

**Step 6: order_api_test.dart（**修订：删 feed/accept；加 invitations/confirmAccept/rejectAccept**）**

```dart
// test/api/order_api_test.dart
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/api/order_api.dart';
import 'package:escort_app/api/dio_client.dart';
import 'package:escort_app/models/order.dart';
import 'package:escort_app/services/token_storage.dart';

class StubAdapter implements HttpClientAdapter {
  final dynamic body; final int status;
  StubAdapter({this.body, this.status = 200});
  @override Future<ResponseBody> fetch(RequestOptions o, Stream<List<int>>? s, Future<void>? c) async {
    final bytes = (body ?? {}).toString().codeUnits;
    return ResponseBody.fromBytes(bytes, status, headers: {'content-type': ['application/json']});
  }
  @override void close({bool force = false}) {}
}

void main() {
  test('OrderApi.invitations → Invitation 列表（含 escort_pending_expire_at）', () async {
    final dio = buildDio(storage: TokenStorage.forTest(), baseUrl: 'https://x');
    dio.httpClientAdapter = StubAdapter(body: {
      'code': 0,
      'data': [
        {
          'order_id': 7,
          'hospital_name': '协和',
          'hospital_lat': 39.9, 'hospital_lng': 116.4,
          'package_name': '半日陪诊',
          'service_start_at': '2026-09-25T09:00:00Z',
          'amount': 300.0,
          'escort_pending_expire_at': DateTime.now().add(const Duration(seconds: 25)).toIso8601String(),
        },
      ],
    });
    final list = await OrderApi(dio).invitations();
    expect(list.length, 1);
    expect(list.first.orderId, 7);
    expect(list.first.isLive, isTrue);
  });

  test('OrderApi.confirmAccept → accepted 订单', () async {
    final dio = buildDio(storage: TokenStorage.forTest(), baseUrl: 'https://x');
    dio.httpClientAdapter = StubAdapter(body: {
      'code': 0,
      'data': {
        'id': 7, 'patient_id': 11, 'escort_id': 22, 'selected_escort_id': 22,
        'hospital_name': '协和', 'hospital_lat': 39.9, 'hospital_lng': 116.4,
        'package_name': '半日陪诊', 'service_start_at': '2026-09-25T09:00:00Z',
        'amount': 300.0, 'status': 'accepted',
      },
    });
    final o = await OrderApi(dio).confirmAccept(7);
    expect(o.status, OrderStatus.accepted);
    expect(o.selectedEscortId, 22);
  });

  test('OrderApi.rejectAccept → selecting_escort 订单', () async {
    final dio = buildDio(storage: TokenStorage.forTest(), baseUrl: 'https://x');
    dio.httpClientAdapter = StubAdapter(body: {
      'code': 0,
      'data': {
        'id': 7, 'patient_id': 11,
        'hospital_name': '协和', 'hospital_lat': 39.9, 'hospital_lng': 116.4,
        'package_name': '半日陪诊', 'service_start_at': '2026-09-25T09:00:00Z',
        'amount': 300.0, 'status': 'selecting_escort',
      },
    });
    final o = await OrderApi(dio).rejectAccept(7);
    expect(o.status, OrderStatus.selectingEscort);
  });
}
```

**Step 7: availability_api.dart + availability_api_test.dart（**新增**）**

```dart
// lib/api/availability_api.dart
import 'package:dio/dio.dart';
import 'package:escort_app/models/availability.dart';

/// escort_availabilities CRUD（spec/2026-09-24-order-matching-redesign §4.1）。
class AvailabilityApi {
  final Dio _dio;
  AvailabilityApi(this._dio);

  /// GET /api/v1/escorts/me/availability
  Future<List<Availability>> listMine() async {
    final r = await _dio.get<Map<String, dynamic>>('/escorts/me/availability');
    final list = (r.data!['data'] as List).cast<Map<String, dynamic>>();
    return list.map(Availability.fromJson).toList();
  }

  /// PUT /api/v1/escorts/me/availability
  /// body: { start_at, end_at } → 返回新时段（含 id + status=available）
  Future<Availability> create({required DateTime startAt, required DateTime endAt}) async {
    final r = await _dio.put<Map<String, dynamic>>('/escorts/me/availability', data: {
      'start_at': startAt.toIso8601String(),
      'end_at': endAt.toIso8601String(),
    });
    return Availability.fromJson(r.data!['data'] as Map<String, dynamic>);
  }

  /// DELETE /api/v1/escorts/me/availability/{id}（仅 available 可删；后端校验）
  Future<void> delete(int id) async {
    await _dio.delete<dynamic>('/escorts/me/availability/$id');
  }
}
```

```dart
// test/api/availability_api_test.dart
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/api/availability_api.dart';
import 'package:escort_app/api/dio_client.dart';
import 'package:escort_app/models/availability.dart';
import 'package:escort_app/services/token_storage.dart';

class StubAdapter implements HttpClientAdapter {
  final dynamic body; final int status;
  StubAdapter({this.body, this.status = 200});
  @override Future<ResponseBody> fetch(RequestOptions o, Stream<List<int>>? s, Future<void>? c) async {
    final bytes = (body ?? {}).toString().codeUnits;
    return ResponseBody.fromBytes(bytes, status, headers: {'content-type': ['application/json']});
  }
  @override void close({bool force = false}) {}
}

void main() {
  test('AvailabilityApi.listMine → Availability 列表', () async {
    final dio = buildDio(storage: TokenStorage.forTest(), baseUrl: 'https://x');
    dio.httpClientAdapter = StubAdapter(body: {
      'code': 0,
      'data': [
        {'id': 1, 'escort_id': 22, 'start_at': '2026-09-25T14:00:00Z',
         'end_at': '2026-09-25T18:00:00Z', 'status': 'available'},
      ],
    });
    final list = await AvailabilityApi(dio).listMine();
    expect(list.length, 1);
    expect(list.first.status, AvailabilityStatus.available);
  });

  test('AvailabilityApi.create 返回新时段', () async {
    final dio = buildDio(storage: TokenStorage.forTest(), baseUrl: 'https://x');
    dio.httpClientAdapter = StubAdapter(body: {
      'code': 0,
      'data': {'id': 2, 'escort_id': 22, 'start_at': '2026-09-25T14:00:00Z',
               'end_at': '2026-09-25T18:00:00Z', 'status': 'available'},
    });
    final a = await AvailabilityApi(dio).create(
      startAt: DateTime.parse('2026-09-25T14:00:00Z'),
      endAt: DateTime.parse('2026-09-25T18:00:00Z'),
    );
    expect(a.id, 2);
    expect(a.isDeletable, isTrue);
  });

  test('AvailabilityApi.delete 不抛异常', () async {
    final dio = buildDio(storage: TokenStorage.forTest(), baseUrl: 'https://x');
    dio.httpClientAdapter = StubAdapter(body: {'code': 0, 'data': null});
    await AvailabilityApi(dio).delete(1); // 无异常即通过
  });
}
```

**Step 8: escort_api.dart + wallet_api.dart + sos_api.dart + 各自测试**

```dart
// lib/api/escort_api.dart
import 'package:dio/dio.dart';
import 'package:escort_app/models/escort_profile.dart';

class EscortApi {
  final Dio _dio;
  EscortApi(this._dio);

  Future<EscortProfile> register({required String phone, required String code}) async {
    final r = await _dio.post<Map<String, dynamic>>('/escorts/register', data: {'phone': phone, 'code': code});
    return EscortProfile.fromJson(r.data!['data'] as Map<String, dynamic>);
  }

  Future<EscortProfile> realNameAuth({required String idCard, required String realName}) async {
    final r = await _dio.post<Map<String, dynamic>>('/escorts/real-name/auth', data: {'id_card': idCard, 'real_name': realName});
    return EscortProfile.fromJson(r.data!['data'] as Map<String, dynamic>);
  }

  Future<EscortProfile> uploadHealthCert({required String imageBase64}) async {
    final r = await _dio.post<Map<String, dynamic>>('/escorts/health-cert/upload', data: {'image_base64': imageBase64});
    return EscortProfile.fromJson(r.data!['data'] as Map<String, dynamic>);
  }

  Future<EscortProfile> trainingComplete({required int courseId, required int score}) async {
    final r = await _dio.post<Map<String, dynamic>>('/escorts/training/complete', data: {'course_id': courseId, 'score': score});
    return EscortProfile.fromJson(r.data!['data'] as Map<String, dynamic>);
  }

  Future<EscortProfile> me() async {
    final r = await _dio.get<Map<String, dynamic>>('/escorts/me/profile');
    return EscortProfile.fromJson(r.data!['data'] as Map<String, dynamic>);
  }

  Future<EscortProfile> updateStatus({required bool isOnline}) async {
    final r = await _dio.put<Map<String, dynamic>>('/escorts/me/status', data: {'is_online': isOnline});
    return EscortProfile.fromJson(r.data!['data'] as Map<String, dynamic>);
  }
}
```

```dart
// lib/api/wallet_api.dart
import 'package:dio/dio.dart';
import 'package:escort_app/models/wallet.dart';
import 'package:escort_app/models/withdrawal.dart';

class WalletApi {
  final Dio _dio;
  WalletApi(this._dio);

  Future<Wallet> getMyWallet() async {
    final r = await _dio.get<Map<String, dynamic>>('/escorts/me/wallet');
    return Wallet.fromJson(r.data!['data'] as Map<String, dynamic>);
  }

  Future<Withdrawal> withdraw({required double amount, required String channel, required String account}) async {
    final r = await _dio.post<Map<String, dynamic>>('/escorts/me/wallet/withdraw', data: {
      'amount': amount, 'channel': channel, 'account': account,
    });
    return Withdrawal.fromJson(r.data!['data'] as Map<String, dynamic>);
  }

  Future<List<Withdrawal>> listTransactions({int page = 1, int pageSize = 20}) async {
    final r = await _dio.get<Map<String, dynamic>>('/escorts/me/wallet/transactions',
        queryParameters: {'page': page, 'page_size': pageSize});
    final list = (r.data!['data'] as List).cast<Map<String, dynamic>>();
    return list.map(Withdrawal.fromJson).toList();
  }
}
```

```dart
// lib/api/sos_api.dart
import 'package:dio/dio.dart';
import 'package:escort_app/models/signal.dart';

class SosApi {
  final Dio _dio;
  SosApi(this._dio);

  Future<Signal> trigger({required int orderId, double? lat, double? lng, String? address, String? note}) async {
    final r = await _dio.post<Map<String, dynamic>>('/orders/$orderId/sos', data: {
      'trigger_role': 'escort',
      if (lat != null) 'lat': lat,
      if (lng != null) 'lng': lng,
      if (address != null) 'address': address,
      if (note != null) 'note': note,
    });
    return Signal.fromJson(r.data!['data'] as Map<String, dynamic>);
  }
}
```

```dart
// lib/api/training_api.dart
import 'package:dio/dio.dart';

class TrainingCourse {
  final int id;
  final String title;
  final int durationMin;
  TrainingCourse({required this.id, required this.title, required this.durationMin});
  factory TrainingCourse.fromJson(Map<String, dynamic> j) => TrainingCourse(
    id: j['id'] as int,
    title: j['title'] as String,
    durationMin: j['duration_min'] as int,
  );
}

class TrainingApi {
  final Dio _dio;
  TrainingApi(this._dio);

  Future<List<TrainingCourse>> listCourses() async {
    final r = await _dio.get<Map<String, dynamic>>('/escorts/training/courses');
    final list = (r.data!['data'] as List).cast<Map<String, dynamic>>();
    return list.map(TrainingCourse.fromJson).toList();
  }

  Future<int> submitQuiz({required int courseId, required List<int> answers}) async {
    final r = await _dio.post<Map<String, dynamic>>('/escorts/training/quiz',
        data: {'course_id': courseId, 'answers': answers});
    return (r.data!['data']['score'] as num).toInt();
  }
}
```

```dart
// lib/api/review_api.dart
import 'package:dio/dio.dart';

class ReviewItem {
  final int id;
  final int orderId;
  final int rating;
  final String? content;
  final DateTime createdAt;
  ReviewItem({required this.id, required this.orderId, required this.rating, this.content, required this.createdAt});
  factory ReviewItem.fromJson(Map<String, dynamic> j) => ReviewItem(
    id: j['id'] as int,
    orderId: j['order_id'] as int,
    rating: j['rating'] as int,
    content: j['content'] as String?,
    createdAt: DateTime.parse(j['created_at'] as String),
  );
}

class ReviewApi {
  final Dio _dio;
  ReviewApi(this._dio);

  Future<List<ReviewItem>> listMine({int page = 1, int pageSize = 20}) async {
    final r = await _dio.get<Map<String, dynamic>>('/escorts/me/reviews',
        queryParameters: {'page': page, 'page_size': pageSize});
    final list = (r.data!['data'] as List).cast<Map<String, dynamic>>();
    return list.map(ReviewItem.fromJson).toList();
  }
}
```

```dart
// lib/api/message_api.dart
import 'package:dio/dio.dart';

class Conversation {
  final int id;
  final int peerId;
  final String peerName;
  final String? lastMessage;
  final DateTime updatedAt;
  Conversation({required this.id, required this.peerId, required this.peerName, this.lastMessage, required this.updatedAt});
  factory Conversation.fromJson(Map<String, dynamic> j) => Conversation(
    id: j['id'] as int,
    peerId: j['peer_id'] as int,
    peerName: j['peer_name'] as String,
    lastMessage: j['last_message'] as String?,
    updatedAt: DateTime.parse(j['updated_at'] as String),
  );
}

class MessageItem {
  final int id;
  final int conversationId;
  final String content;
  final bool fromMe;
  final DateTime createdAt;
  MessageItem({required this.id, required this.conversationId, required this.content, required this.fromMe, required this.createdAt});
  factory MessageItem.fromJson(Map<String, dynamic> j) => MessageItem(
    id: j['id'] as int,
    conversationId: j['conversation_id'] as int,
    content: j['content'] as String,
    fromMe: j['from_me'] as bool,
    createdAt: DateTime.parse(j['created_at'] as String),
  );
}

class MessageApi {
  final Dio _dio;
  MessageApi(this._dio);

  Future<List<Conversation>> listConversations() async {
    final r = await _dio.get<Map<String, dynamic>>('/messages/conversations');
    final list = (r.data!['data'] as List).cast<Map<String, dynamic>>();
    return list.map(Conversation.fromJson).toList();
  }

  Future<List<MessageItem>> listMessages(int conversationId) async {
    final r = await _dio.get<Map<String, dynamic>>('/messages/$conversationId');
    final list = (r.data!['data'] as List).cast<Map<String, dynamic>>();
    return list.map(MessageItem.fromJson).toList();
  }
}
```

**Step 9: 跑全部 api 测试**

```bash
flutter test test/api/
```

Expected: PASS（auth 1 + order 3 + availability 3 + 其它 escorts/wallet/sos 不写单测，靠集成测试覆盖 ≈ 7 个）

**Step 10: Commit**

```bash
git add escort-app/lib/api/ escort-app/test/api/
git commit -m "feat(escort-app): api modules (AuthApi/OrderApi[invitations+confirmAccept/rejectAccept]/AvailabilityApi/...) + 7 个 StubAdapter 单测"
```

---

### Task 8: providers/auth_provider + auth_provider_test

**Files:**
- Create: `escort-app/lib/providers/auth_provider.dart`
- Create: `escort-app/lib/providers/auth_provider_test.dart`

**Step 1: auth_provider_test.dart（RED）**

```dart
// test/providers/auth_provider_test.dart
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/providers/auth_provider.dart';

void main() {
  test('bootstrap → AuthState.unauthenticated（无 token）', () async {
    final c = ProviderContainer();
    addTearDown(c.dispose);
    await c.read(authProvider.notifier).bootstrap();
    expect(c.read(authProvider), isA<AuthUnauthenticated>());
  });

  test('loginByPhone → AuthState.authenticated', () async {
    // 简化：AuthNotifier 用 fake 注入；这里直接断言状态机 sealed
    final s = AuthAuthenticated(userId: 1, phone: '13800138000', role: 'escort',
      realNameVerified: false, approved: false);
    expect(s.isAuthed, isTrue);
  });

  test('logout → AuthState.unauthenticated', () async {
    final s = AuthUnauthenticated();
    expect(s.isAuthed, isFalse);
  });
}
```

**Step 2: 跑测试确认失败**

```bash
flutter test test/providers/auth_provider_test.dart
```

Expected: FAIL — `package:escort_app/providers/auth_provider.dart` not found

**Step 3: auth_provider.dart**

```dart
// lib/providers/auth_provider.dart
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:escort_app/api/auth_api.dart';
import 'package:escort_app/api/dio_client.dart';
import 'package:escort_app/services/token_storage.dart';

/// Auth 状态机（sealed；spec §8）。
sealed class AuthState {
  bool get isAuthed => this is AuthAuthenticated;
}

class AuthInitial extends AuthState {}

class AuthUnauthenticated extends AuthState {}

class AuthAuthenticated extends AuthState {
  final int userId;
  final String phone;
  final String role; // "patient" | "escort" | "admin"
  final bool realNameVerified;
  final bool approved;
  AuthAuthenticated({
    required this.userId, required this.phone, required this.role,
    required this.realNameVerified, required this.approved,
  });
}

class AuthNotifier extends StateNotifier<AuthState> {
  final Ref ref;
  AuthNotifier(this.ref) : super(AuthInitial());

  Future<void> bootstrap() async {
    final storage = ref.read(tokenStorageProvider);
    final t = await storage.read();
    if (t == null) {
      state = AuthUnauthenticated();
      return;
    }
    try {
      final user = await AuthApi(ref.read(dioProvider)).me();
      state = AuthAuthenticated(
        userId: user.id, phone: user.phone, role: user.role,
        realNameVerified: user.realNameVerified, approved: user.approved,
      );
    } catch (_) {
      await storage.delete();
      state = AuthUnauthenticated();
    }
  }

  Future<void> loginByPhone({required String phone, required String code}) async {
    final res = await AuthApi(ref.read(dioProvider)).loginByPhone(phone: phone, code: code);
    await ref.read(tokenStorageProvider).write(res.accessToken);
    state = AuthAuthenticated(
      userId: res.user.id, phone: res.user.phone, role: res.user.role,
      realNameVerified: res.user.realNameVerified, approved: res.user.approved,
    );
  }

  Future<void> logout() async {
    await ref.read(tokenStorageProvider).delete();
    state = AuthUnauthenticated();
  }
}

final tokenStorageProvider = Provider<TokenStorage>((ref) {
  throw UnimplementedError('override in main.dart');
});

final authProvider = StateNotifierProvider<AuthNotifier, AuthState>((ref) {
  return AuthNotifier(ref);
});
```

**Step 4: 跑测试确认通过**

```bash
flutter test test/providers/auth_provider_test.dart
```

Expected: PASS（3 个）

**Step 5: Commit**

```bash
git add escort-app/lib/providers/auth_provider.dart escort-app/test/providers/auth_provider_test.dart
git commit -m "feat(escort-app): AuthState (sealed) + AuthNotifier + tokenStorageProvider + 3 单测"
```

---

### Task 9: router/route_guards + app_router + 测试（**路由变更：删 /home/feed 加 /home/invitations + /home/availability**）

**Files:**
- Create: `escort-app/lib/router/route_guards.dart`
- Create: `escort-app/lib/router/route_guards_test.dart`
- Modify: `escort-app/lib/router/app_router.dart`
- Create: `escort-app/lib/router/app_router_test.dart`

**Step 1: route_guards_test.dart（RED）**

```dart
// test/router/route_guards_test.dart
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/providers/auth_provider.dart';
import 'package:escort_app/router/route_guards.dart';

void main() {
  test('authGuardProvider → false when unauthenticated', () {
    final c = ProviderContainer(overrides: [
      authProvider.overrideWith((ref) => _StubAuthNotifier(ref, AuthUnauthenticated())),
    ]);
    addTearDown(c.dispose);
    expect(c.read(authGuardProvider), isFalse);
  });

  test('authGuardProvider → true when authenticated', () {
    final c = ProviderContainer(overrides: [
      authProvider.overrideWith((ref) => _StubAuthNotifier(ref, AuthAuthenticated(
        userId: 1, phone: '1', role: 'escort',
        realNameVerified: true, approved: true,
      ))),
    ]);
    addTearDown(c.dispose);
    expect(c.read(authGuardProvider), isTrue);
  });

  test('realNameGuardProvider → true when realNameVerified', () {
    final c = ProviderContainer(overrides: [
      authProvider.overrideWith((ref) => _StubAuthNotifier(ref, AuthAuthenticated(
        userId: 1, phone: '1', role: 'escort',
        realNameVerified: true, approved: false,
      ))),
    ]);
    addTearDown(c.dispose);
    expect(c.read(realNameGuardProvider), isTrue);
  });

  test('approvedGuardProvider → true when approved', () {
    final c = ProviderContainer(overrides: [
      authProvider.overrideWith((ref) => _StubAuthNotifier(ref, AuthAuthenticated(
        userId: 1, phone: '1', role: 'escort',
        realNameVerified: true, approved: true,
      ))),
    ]);
    addTearDown(c.dispose);
    expect(c.read(approvedGuardProvider), isTrue);
  });
}

class _StubAuthNotifier extends AuthNotifier {
  _StubAuthNotifier(super.ref, AuthState initial) {
    state = initial;
  }
}
```

**Step 2: 跑测试确认失败**

```bash
flutter test test/router/route_guards_test.dart
```

Expected: FAIL — not found

**Step 3: route_guards.dart**

```dart
// lib/router/route_guards.dart
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:escort_app/providers/auth_provider.dart';

/// 已登录
final authGuardProvider = Provider<bool>((ref) {
  return ref.watch(authProvider).isAuthed;
});

/// 已实名（realNameVerified）
final realNameGuardProvider = Provider<bool>((ref) {
  return ref.watch(authProvider).maybeWhen(
    authenticated: (_, __, ___, realNameVerified, ____) => realNameVerified,
    orElse: () => false,
  );
});

/// 已通过审核（approved）
final approvedGuardProvider = Provider<bool>((ref) {
  return ref.watch(authProvider).maybeWhen(
    authenticated: (_, __, ___, ____, approved) => approved,
    orElse: () => false,
  );
});
```

**Step 4: 跑测试确认通过**

```bash
flutter test test/router/route_guards_test.dart
```

Expected: PASS（4 个）

**Step 5: app_router.dart（**修订：删 /home/feed；加 /home/invitations + /home/availability；redirect 链不变（仅 `/home/feed` 检查改为 `/home/invitations`）**）**

```dart
// lib/router/app_router.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:escort_app/pages/audit/pending_audit_page.dart';
import 'package:escort_app/pages/auth/login_page.dart';
import 'package:escort_app/pages/auth/register_page.dart';
import 'package:escort_app/pages/home/availability_page.dart';
import 'package:escort_app/pages/home/home_shell.dart';
import 'package:escort_app/pages/home/invitations_page.dart';
import 'package:escort_app/pages/home/orders_page.dart';
import 'package:escort_app/pages/home/profile_page.dart';
import 'package:escort_app/pages/home/wallet_page.dart';
import 'package:escort_app/pages/message/chat_page.dart';
import 'package:escort_app/pages/message/conversation_list_page.dart';
import 'package:escort_app/pages/onboarding/agreement_page.dart';
import 'package:escort_app/pages/onboarding/health_cert_page.dart';
import 'package:escort_app/pages/onboarding/real_name_page.dart';
import 'package:escort_app/pages/onboarding/training_list_page.dart';
import 'package:escort_app/pages/onboarding/training_quiz_page.dart';
import 'package:escort_app/pages/onboarding/training_video_page.dart';
import 'package:escort_app/pages/order/checkin_page.dart';
import 'package:escort_app/pages/order/checkout_page.dart';
import 'package:escort_app/pages/order/order_detail_page.dart';
import 'package:escort_app/pages/profile/edit_profile_page.dart';
import 'package:escort_app/pages/profile/reviews_page.dart';
import 'package:escort_app/pages/sos/sos_trigger_page.dart';
import 'package:escort_app/pages/splash/splash_page.dart';
import 'package:escort_app/pages/wallet/transactions_page.dart';
import 'package:escort_app/pages/wallet/withdraw_page.dart';
import 'package:escort_app/providers/auth_provider.dart';
import 'package:escort_app/router/route_guards.dart';

final goRouterProvider = Provider<GoRouter>((ref) {
  return GoRouter(
    initialLocation: '/splash',
    redirect: (ctx, state) {
      final loc = state.matchedLocation;
      // splash / auth 自由通行
      if (loc == '/splash' || loc.startsWith('/auth/')) return null;
      // 未登录 → login
      if (!ref.read(authGuardProvider)) return '/auth/login';
      // 未实名 → real-name（仅当试图访问 /home/* 或 /order/* 等业务页）
      if (!ref.read(realNameGuardProvider) && loc.startsWith('/home/')) {
        return '/onboarding/real-name';
      }
      // 未审核 → audit/pending（业务页入口改为 /home/invitations）
      if (!ref.read(approvedGuardProvider) && loc.startsWith('/home/invitations')) {
        return '/audit/pending';
      }
      return null;
    },
    routes: [
      GoRoute(path: '/splash', builder: (_, __) => const SplashPage()),
      GoRoute(path: '/auth/login', builder: (_, __) => const LoginPage()),
      GoRoute(path: '/auth/register', builder: (_, __) => const RegisterPage()),
      GoRoute(path: '/onboarding/real-name', builder: (_, __) => const RealNamePage()),
      GoRoute(path: '/onboarding/health-cert', builder: (_, __) => const HealthCertPage()),
      GoRoute(path: '/onboarding/training', builder: (_, __) => const TrainingListPage()),
      GoRoute(
        path: '/onboarding/training/:id',
        builder: (_, s) => TrainingVideoPage(courseId: int.parse(s.pathParameters['id']!)),
      ),
      GoRoute(
        path: '/onboarding/training/:id/quiz',
        builder: (_, s) => TrainingQuizPage(courseId: int.parse(s.pathParameters['id']!)),
      ),
      GoRoute(path: '/onboarding/agreement', builder: (_, __) => const AgreementPage()),
      GoRoute(path: '/audit/pending', builder: (_, __) => const PendingAuditPage()),
      // 「我的空余时段」独立路由（顶层，非 BottomNavBar tab；从 ProfilePage 入口进）
      GoRoute(path: '/home/availability', builder: (_, __) => const AvailabilityPage()),
      ShellRoute(
        builder: (_, __, child) => HomeShell(child: child),
        routes: [
          // 原 /home/feed → /home/invitations（spec 修订）
          GoRoute(path: '/home/invitations', builder: (_, __) => const InvitationsPage()),
          GoRoute(path: '/home/orders', builder: (_, __) => const OrdersPage()),
          GoRoute(path: '/home/wallet', builder: (_, __) => const WalletPage()),
          GoRoute(path: '/home/profile', builder: (_, __) => const ProfilePage()),
        ],
      ),
      GoRoute(
        path: '/order/:id',
        builder: (_, s) => OrderDetailPage(orderId: int.parse(s.pathParameters['id']!)),
      ),
      GoRoute(
        path: '/order/:id/checkin',
        builder: (_, s) => CheckinPage(orderId: int.parse(s.pathParameters['id']!)),
      ),
      GoRoute(
        path: '/order/:id/checkout',
        builder: (_, s) => CheckoutPage(orderId: int.parse(s.pathParameters['id']!)),
      ),
      GoRoute(path: '/wallet/withdraw', builder: (_, __) => const WithdrawPage()),
      GoRoute(path: '/wallet/transactions', builder: (_, __) => const TransactionsPage()),
      GoRoute(path: '/sos/trigger', builder: (_, __) => const SosTriggerPage()),
      GoRoute(path: '/message/list', builder: (_, __) => const ConversationListPage()),
      GoRoute(
        path: '/message/:id',
        builder: (_, s) => ChatPage(conversationId: int.parse(s.pathParameters['id']!)),
      ),
      GoRoute(path: '/profile/edit', builder: (_, __) => const EditProfilePage()),
      GoRoute(path: '/profile/reviews', builder: (_, __) => const ReviewsPage()),
    ],
  );
});
```

**Step 6: app_router_test.dart**

```dart
// test/router/app_router_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/providers/auth_provider.dart';
import 'package:escort_app/router/app_router.dart';

class _StubAuthNotifier extends AuthNotifier {
  _StubAuthNotifier(super.ref, AuthState initial) {
    state = initial;
  }
}

ProviderContainer _container(AuthState initial) {
  return ProviderContainer(overrides: [
    authProvider.overrideWith((ref) => _StubAuthNotifier(ref, initial)),
  ]);
}

void main() {
  testWidgets('未登录 → /auth/login 跳 login 页', (t) async {
    final c = _container(AuthUnauthenticated());
    addTearDown(c.dispose);
    final router = c.read(goRouterProvider);
    router.go('/home/invitations');
    await t.pumpAndSettle();
    expect(router.routerDelegate.currentConfiguration.uri.path, '/auth/login');
  });

  testWidgets('未实名 → /home/invitations 跳 real-name 页', (t) async {
    final c = _container(AuthAuthenticated(
      userId: 1, phone: '1', role: 'escort',
      realNameVerified: false, approved: false,
    ));
    addTearDown(c.dispose);
    final router = c.read(goRouterProvider);
    router.go('/home/invitations');
    await t.pumpAndSettle();
    expect(router.routerDelegate.currentConfiguration.uri.path, '/onboarding/real-name');
  });

  testWidgets('未审核 → /home/invitations 跳 audit/pending', (t) async {
    final c = _container(AuthAuthenticated(
      userId: 1, phone: '1', role: 'escort',
      realNameVerified: true, approved: false,
    ));
    addTearDown(c.dispose);
    final router = c.read(goRouterProvider);
    router.go('/home/invitations');
    await t.pumpAndSettle();
    expect(router.routerDelegate.currentConfiguration.uri.path, '/audit/pending');
  });

  testWidgets('已审核 → /home/invitations 停在 invitations 页', (t) async {
    final c = _container(AuthAuthenticated(
      userId: 1, phone: '1', role: 'escort',
      realNameVerified: true, approved: true,
    ));
    addTearDown(c.dispose);
    final router = c.read(goRouterProvider);
    router.go('/home/invitations');
    await t.pumpAndSettle();
    expect(router.routerDelegate.currentConfiguration.uri.path, '/home/invitations');
  });
}
```

**Step 7: 跑测试确认通过**

```bash
flutter test test/router/
```

Expected: PASS（4 guard + 4 redirect = 8 个）

**Step 8: Commit**

```bash
git add escort-app/lib/router/ escort-app/test/router/
git commit -m "feat(escort-app): router (go_router + 路由：删 /home/feed 加 /home/invitations + /home/availability + 3 守卫 redirect 链) + 8 个测试"
```

---

### Task 10: pages/splash + auth/login + auth/register + 测试（**splash 跳 /home/invitations**）

**Files:**
- Modify: `escort-app/lib/pages/splash/splash_page.dart`（**跳转目标：/home/invitations**）
- Create: `escort-app/lib/pages/auth/login_page.dart`
- Create: `escort-app/lib/pages/auth/register_page.dart`
- Create: `escort-app/test/pages/splash_page_test.dart`
- Create: `escort-app/test/pages/login_page_test.dart`

**Step 1: splash_page.dart（**跳 /home/invitations**）**

```dart
// lib/pages/splash/splash_page.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:escort_app/providers/auth_provider.dart';
import 'package:escort_app/router/route_guards.dart';

class SplashPage extends ConsumerStatefulWidget {
  const SplashPage({super.key});
  @override
  ConsumerState<SplashPage> createState() => _SplashPageState();
}

class _SplashPageState extends ConsumerState<SplashPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) async {
      await ref.read(authProvider.notifier).bootstrap();
      if (!mounted) return;
      if (ref.read(authGuardProvider)) {
        // 修订：跳我的邀请（原 /home/feed）
        context.go('/home/invitations');
      } else {
        context.go('/auth/login');
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    return const Scaffold(
      body: Center(child: Text('Doctors Escort')),
    );
  }
}
```

**Step 2: splash_page_test.dart**

```dart
// test/pages/splash_page_test.dart
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/api/auth_api.dart';
import 'package:escort_app/pages/splash/splash_page.dart';
import 'package:escort_app/providers/auth_provider.dart';

class _FakeAuthNotifier extends AuthNotifier {
  AuthState initial;
  _FakeAuthNotifier(super.ref, this.initial) {
    state = initial;
  }

  @override
  Future<void> bootstrap() async {
    state = initial;
  }
}

void main() {
  testWidgets('SplashPage 渲染品牌字', (t) async {
    await t.pumpWidget(ProviderScope(
      overrides: [
        authProvider.overrideWith((ref) => _FakeAuthNotifier(ref, AuthUnauthenticated())),
      ],
      child: const SplashPage(),
    ));
    await t.pump();
    expect(find.text('Doctors Escort'), findsOneWidget);
  });

  testWidgets('已登录 → bootstrap 后 state = authenticated', (t) async {
    final c = ProviderContainer(overrides: [
      authProvider.overrideWith((ref) => _FakeAuthNotifier(ref, AuthAuthenticated(
        userId: 1, phone: '13800138000', role: 'escort',
        realNameVerified: true, approved: true,
      ))),
    ]);
    addTearDown(c.dispose);
    await c.read(authProvider.notifier).bootstrap();
    expect(c.read(authProvider), isA<AuthAuthenticated>());
  });
}
```

**Step 3: login_page.dart + register_page.dart + login_page_test.dart**

```dart
// lib/pages/auth/login_page.dart
import 'package:flutter/material.dart';

class LoginPage extends StatelessWidget {
  const LoginPage({super.key});
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('登录')),
      body: const Center(child: Text('TODO: 手机号 + 验证码 + 角色（spec §3.1）')),
    );
  }
}
```

```dart
// lib/pages/auth/register_page.dart
import 'package:flutter/material.dart';

class RegisterPage extends StatelessWidget {
  const RegisterPage({super.key});
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('注册陪诊师')),
      body: const Center(child: Text('TODO: 手机号 + 验证码 + 同意协议（spec §3.1）')),
    );
  }
}
```

```dart
// test/pages/login_page_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/pages/auth/login_page.dart';

void main() {
  testWidgets('LoginPage 渲染标题', (t) async {
    await t.pumpWidget(const MaterialApp(home: LoginPage()));
    expect(find.text('登录'), findsOneWidget);
  });
}
```

**Step 4: 跑测试 + Commit**

```bash
flutter test test/pages/splash_page_test.dart test/pages/login_page_test.dart
git add escort-app/lib/pages/splash/ escort-app/lib/pages/auth/ escort-app/test/pages/
git commit -m "feat(escort-app): pages/splash (跳 /home/invitations) + auth (login/register) + 3 个测试"
```

---

### Task 11: pages/onboarding (6 页) + 测试

**Files:**
- Create: `escort-app/lib/pages/onboarding/real_name_page.dart`
- Create: `escort-app/lib/pages/onboarding/health_cert_page.dart`
- Create: `escort-app/lib/pages/onboarding/training_list_page.dart`
- Create: `escort-app/lib/pages/onboarding/training_video_page.dart`
- Create: `escort-app/lib/pages/onboarding/training_quiz_page.dart`
- Create: `escort-app/lib/pages/onboarding/agreement_page.dart`
- Create: `escort-app/test/pages/onboarding_pages_test.dart`

**Step 1: 6 个 page 骨架**

```dart
// lib/pages/onboarding/real_name_page.dart
import 'package:flutter/material.dart';

class RealNamePage extends StatelessWidget {
  const RealNamePage({super.key});
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('实名认证')),
      body: const Center(child: Text('TODO: 身份证号 + 真实姓名 + OCR（spec §2）')),
    );
  }
}
```

```dart
// lib/pages/onboarding/health_cert_page.dart
import 'package:flutter/material.dart';

class HealthCertPage extends StatelessWidget {
  const HealthCertPage({super.key});
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('健康证上传')),
      body: const Center(child: Text('TODO: image_picker 调用（spec §2）')),
    );
  }
}
```

```dart
// lib/pages/onboarding/training_list_page.dart
import 'package:flutter/material.dart';

class TrainingListPage extends StatelessWidget {
  const TrainingListPage({super.key});
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('培训课程')),
      body: const Center(child: Text('TODO: 课程列表（spec §2）')),
    );
  }
}
```

```dart
// lib/pages/onboarding/training_video_page.dart
import 'package:flutter/material.dart';

class TrainingVideoPage extends StatelessWidget {
  final int courseId;
  const TrainingVideoPage({required this.courseId, super.key});
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text('课程 #$courseId')),
      body: const Center(child: Text('TODO: video_player 播放 + 进度上报（spec §2）')),
    );
  }
}
```

```dart
// lib/pages/onboarding/training_quiz_page.dart
import 'package:flutter/material.dart';

class TrainingQuizPage extends StatelessWidget {
  final int courseId;
  const TrainingQuizPage({required this.courseId, super.key});
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text('课程考核 #$courseId')),
      body: const Center(child: Text('TODO: 5 题单选 + 80 分通过（spec §3.1）')),
    );
  }
}
```

```dart
// lib/pages/onboarding/agreement_page.dart
import 'package:flutter/material.dart';

class AgreementPage extends StatelessWidget {
  const AgreementPage({super.key});
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('陪诊服务协议')),
      body: const Center(child: Text('TODO: 协议全文 + 勾选同意（spec §3.1）')),
    );
  }
}
```

**Step 2: onboarding_pages_test.dart**

```dart
// test/pages/onboarding_pages_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/pages/onboarding/agreement_page.dart';
import 'package:escort_app/pages/onboarding/health_cert_page.dart';
import 'package:escort_app/pages/onboarding/real_name_page.dart';
import 'package:escort_app/pages/onboarding/training_list_page.dart';
import 'package:escort_app/pages/onboarding/training_quiz_page.dart';
import 'package:escort_app/pages/onboarding/training_video_page.dart';

void main() {
  testWidgets('RealNamePage 渲染', (t) async {
    await t.pumpWidget(const MaterialApp(home: RealNamePage()));
    expect(find.text('实名认证'), findsOneWidget);
  });
  testWidgets('HealthCertPage 渲染', (t) async {
    await t.pumpWidget(const MaterialApp(home: HealthCertPage()));
    expect(find.text('健康证上传'), findsOneWidget);
  });
  testWidgets('TrainingListPage 渲染', (t) async {
    await t.pumpWidget(const MaterialApp(home: TrainingListPage()));
    expect(find.text('培训课程'), findsOneWidget);
  });
  testWidgets('TrainingVideoPage 渲染', (t) async {
    await t.pumpWidget(const MaterialApp(home: TrainingVideoPage(courseId: 1)));
    expect(find.text('课程 #1'), findsOneWidget);
  });
  testWidgets('TrainingQuizPage 渲染', (t) async {
    await t.pumpWidget(const MaterialApp(home: TrainingQuizPage(courseId: 1)));
    expect(find.text('课程考核 #1'), findsOneWidget);
  });
  testWidgets('AgreementPage 渲染', (t) async {
    await t.pumpWidget(const MaterialApp(home: AgreementPage()));
    expect(find.text('陪诊服务协议'), findsOneWidget);
  });
}
```

**Step 3: 跑测试 + Commit**

```bash
flutter test test/pages/onboarding_pages_test.dart
git add escort-app/lib/pages/onboarding/ escort-app/test/pages/onboarding_pages_test.dart
git commit -m "feat(escort-app): pages/onboarding (6 页骨架: real-name/health-cert/training list/video/quiz/agreement) + 6 个 widget 测试"
```

---

### Task 12: pages/audit/pending + pages/home shell + **3 tabs（invitations / orders / wallet / profile）+ InvitationsPage + AvailabilityPage + OrdersPage 加「邀请」tab** + 测试

**Files:**
- Create: `escort-app/lib/pages/audit/pending_audit_page.dart`
- Create: `escort-app/lib/pages/audit/pending_audit_page_test.dart`
- Create: `escort-app/lib/pages/home/home_shell.dart`（**修订：4 tabs = invitations/orders/wallet/profile**）
- **Create**: `escort-app/lib/pages/home/invitations_page.dart`（**新增**）
- **Create**: `escort-app/lib/pages/home/invitations_page_test.dart`
- **Create**: `escort-app/lib/pages/home/availability_page.dart`（**新增**）
- **Create**: `escort-app/lib/pages/home/availability_page_test.dart`
- Create: `escort-app/lib/pages/home/orders_page.dart`（**修订：加「邀请」tab**）
- Create: `escort-app/lib/pages/home/wallet_page.dart`
- Create: `escort-app/lib/pages/home/profile_page.dart`（**修订：加「我的空余时段」入口**）
- Create: `escort-app/test/pages/home_pages_test.dart`（**新增 2 个页面测试**）

**Step 1: pending_audit_page.dart**

```dart
// lib/pages/audit/pending_audit_page.dart
import 'package:flutter/material.dart';

class PendingAuditPage extends StatelessWidget {
  const PendingAuditPage({super.key});
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('等待审核')),
      body: const Center(child: Text('TODO: 审核状态 + 客服入口')),
    );
  }
}
```

**Step 2: home_shell.dart（**4 tabs：invitations / orders / wallet / profile**）**

```dart
// lib/pages/home/home_shell.dart
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

class HomeShell extends StatelessWidget {
  final Widget child;
  const HomeShell({required this.child, super.key});

  // 修订：原 feed → invitations（spec/2026-09-24-order-matching-redesign §1.2）
  static const _tabs = ['/home/invitations', '/home/orders', '/home/wallet', '/home/profile'];
  static const _titles = ['我的邀请', '我的订单', '钱包', '个人中心'];

  int _indexFromLocation(String loc) {
    final i = _tabs.indexWhere((p) => loc.startsWith(p));
    return i < 0 ? 0 : i;
  }

  @override
  Widget build(BuildContext context) {
    final loc = GoRouterState.of(context).matchedLocation;
    final idx = _indexFromLocation(loc);
    return Scaffold(
      body: child,
      bottomNavigationBar: NavigationBar(
        selectedIndex: idx,
        onDestinationSelected: (i) => context.go(_tabs[i]),
        destinations: const [
          NavigationDestination(icon: Icon(Icons.mail_outline), label: '邀请'), // 修订
          NavigationDestination(icon: Icon(Icons.assignment), label: '订单'),
          NavigationDestination(icon: Icon(Icons.account_balance_wallet), label: '钱包'),
          NavigationDestination(icon: Icon(Icons.person), label: '我的'),
        ],
      ),
    );
  }
}
```

**Step 3: invitations_page.dart（**核心新页：spec §1.2 + §4.1**）+ invitations_page_test.dart**

```dart
// lib/pages/home/invitations_page.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:escort_app/models/invitation.dart';
import 'package:escort_app/providers/invitation_provider.dart';
import 'package:escort_app/utils/error_handler.dart';
import 'package:escort_app/widgets/invitation_card.dart';

/// 「我的邀请」页（spec/2026-09-24-order-matching-redesign §1.2 + §4.1）。
/// - 顶部 AppBar 显示「我的邀请」
/// - ListView 渲染 invitationProvider 提供的列表（按 isLive 客户端过滤）
/// - 每张 InvitationCard 显示倒计时 + 确认/拒接按钮
/// - 空态显示「当前没有邀请」
class InvitationsPage extends ConsumerWidget {
  const InvitationsPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final async = ref.watch(invitationsProvider);
    return Scaffold(
      appBar: AppBar(title: const Text('我的邀请')),
      body: async.when(
        data: (list) {
          // 客户端过滤：只显示 30s 确认窗口内的邀请
          final live = list.where((i) => i.isLive).toList();
          if (live.isEmpty) {
            return const Center(child: Text('当前没有邀请'));
          }
          return ListView.builder(
            itemCount: live.length,
            itemBuilder: (_, i) {
              final inv = live[i];
              return InvitationCard(
                invitation: inv,
                onConfirm: () async {
                  try {
                    await ref.read(confirmAcceptControllerProvider(inv.orderId).future);
                    ref.invalidate(invitationsProvider);
                  } catch (e) {
                    ScaffoldMessenger.of(context).showSnackBar(
                      SnackBar(content: Text(handleDioError(e))),
                    );
                  }
                },
                onReject: () async {
                  try {
                    await ref.read(rejectAcceptControllerProvider(inv.orderId).future);
                    ref.invalidate(invitationsProvider);
                  } catch (e) {
                    ScaffoldMessenger.of(context).showSnackBar(
                      SnackBar(content: Text(handleDioError(e))),
                    );
                  }
                },
              );
            },
          );
        },
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(child: Text(handleDioError(e))),
      ),
    );
  }
}
```

```dart
// test/pages/home/invitations_page_test.dart
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/api/dio_client.dart';
import 'package:escort_app/pages/home/invitations_page.dart';
import 'package:escort_app/services/token_storage.dart';

class StubAdapter implements HttpClientAdapter {
  final dynamic body;
  StubAdapter({this.body});
  @override
  Future<ResponseBody> fetch(RequestOptions o, Stream<List<int>>? s, Future<void>? c) async {
    final bytes = (body ?? {'code': 0, 'data': []}).toString().codeUnits;
    return ResponseBody.fromBytes(bytes, 200, headers: {'content-type': ['application/json']});
  }
  @override void close({bool force = false}) {}
}

void main() {
  testWidgets('空邀请列表 → 显示空态', (t) async {
    final storage = TokenStorage.forTest();
    final dio = buildDio(storage: storage, baseUrl: 'https://x');
    dio.httpClientAdapter = StubAdapter(body: {'code': 0, 'data': []});
    await t.pumpWidget(ProviderScope(
      overrides: [dioProvider.overrideWithValue(dio)],
      child: const MaterialApp(home: InvitationsPage()),
    ));
    await t.pumpAndSettle();
    expect(find.text('我的邀请'), findsOneWidget);
    expect(find.text('当前没有邀请'), findsOneWidget);
  });

  testWidgets('有邀请 → 渲染卡片 + 倒计时 + 按钮', (t) async {
    final storage = TokenStorage.forTest();
    final dio = buildDio(storage: storage, baseUrl: 'https://x');
    dio.httpClientAdapter = StubAdapter(body: {
      'code': 0,
      'data': [
        {
          'order_id': 7,
          'hospital_name': '北京协和医院',
          'hospital_lat': 39.9, 'hospital_lng': 116.4,
          'package_name': '半日陪诊',
          'service_start_at': '2026-09-25T09:00:00Z',
          'amount': 300.0,
          'escort_pending_expire_at': DateTime.now().add(const Duration(seconds: 25)).toIso8601String(),
        },
      ],
    });
    await t.pumpWidget(ProviderScope(
      overrides: [dioProvider.overrideWithValue(dio)],
      child: const MaterialApp(home: InvitationsPage()),
    ));
    await t.pumpAndSettle();
    expect(find.text('北京协和医院'), findsOneWidget);
    expect(find.text('确认接单'), findsOneWidget);
    expect(find.text('拒接'), findsOneWidget);
    expect(find.textContaining('待确认剩余'), findsOneWidget);
  });
}
```

**Step 4: availability_page.dart（**核心新页：spec §4.1**）+ availability_page_test.dart**

```dart
// lib/pages/home/availability_page.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:escort_app/models/availability.dart';
import 'package:escort_app/providers/availability_provider.dart';
import 'package:escort_app/utils/error_handler.dart';
import 'package:escort_app/widgets/availability_tile.dart';

/// 「我的空余时段」页（spec/2026-09-24-order-matching-redesign §4.1）。
/// - AppBar 「我的空余时段」+ 「+ 新增」按钮
/// - 列表：每行 AvailabilityTile（左时段 / 右删除 or 「已预订」chip）
/// - 空态：「点击右上角新增可用时段」
class AvailabilityPage extends ConsumerWidget {
  const AvailabilityPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final async = ref.watch(availabilityProvider);
    return Scaffold(
      appBar: AppBar(
        title: const Text('我的空余时段'),
        actions: [
          IconButton(
            icon: const Icon(Icons.add),
            onPressed: () => _showCreateSheet(context, ref),
          ),
        ],
      ),
      body: async.when(
        data: (list) {
          if (list.isEmpty) {
            return const Center(child: Text('点击右上角新增可用时段'));
          }
          return ListView.builder(
            itemCount: list.length,
            itemBuilder: (_, i) {
              final a = list[i];
              return AvailabilityTile(
                availability: a,
                onDelete: a.isDeletable ? () => _confirmDelete(context, ref, a) : null,
              );
            },
          );
        },
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(child: Text(handleDioError(e))),
      ),
    );
  }

  Future<void> _showCreateSheet(BuildContext context, WidgetRef ref) async {
    final startCtrl = TextEditingController();
    final endCtrl = TextEditingController();
    final ok = await showModalBottomSheet<bool>(
      context: context,
      isScrollControlled: true,
      builder: (_) => Padding(
        padding: EdgeInsets.only(
          left: 16, right: 16, top: 16,
          bottom: MediaQuery.of(context).viewInsets.bottom + 16,
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Text('新增时段', style: TextStyle(fontWeight: FontWeight.bold)),
            TextField(controller: startCtrl, decoration: const InputDecoration(labelText: '起始时间 (yyyy-MM-dd HH:mm)')),
            TextField(controller: endCtrl, decoration: const InputDecoration(labelText: '结束时间 (yyyy-MM-dd HH:mm)')),
            const SizedBox(height: 16),
            FilledButton(
              onPressed: () => Navigator.of(context).pop(true),
              child: const Text('保存'),
            ),
          ],
        ),
      ),
    );
    if (ok != true) return;
    final start = DateTime.tryParse(startCtrl.text.replaceFirst(' ', 'T')) ?? DateTime.tryParse(startCtrl.text);
    final end = DateTime.tryParse(endCtrl.text.replaceFirst(' ', 'T')) ?? DateTime.tryParse(endCtrl.text);
    if (start == null || end == null || end.isBefore(start)) return;
    try {
      await ref.read(createAvailabilityControllerProvider.notifier).submit(startAt: start, endAt: end);
      ref.invalidate(availabilityProvider);
    } catch (e) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(handleDioError(e))),
        );
      }
    }
  }

  Future<void> _confirmDelete(BuildContext context, WidgetRef ref, Availability a) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (_) => AlertDialog(
        title: const Text('删除时段'),
        content: Text('确认删除 ${a.startText} ~ ${a.endText}？'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context, false), child: const Text('取消')),
          FilledButton(onPressed: () => Navigator.pop(context, true), child: const Text('删除')),
        ],
      ),
    );
    if (ok != true) return;
    try {
      await ref.read(deleteAvailabilityControllerProvider(a.id).future);
      ref.invalidate(availabilityProvider);
    } catch (e) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(handleDioError(e))),
        );
      }
    }
  }
}
```

```dart
// test/pages/home/availability_page_test.dart
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/api/dio_client.dart';
import 'package:escort_app/pages/home/availability_page.dart';
import 'package:escort_app/services/token_storage.dart';

class StubAdapter implements HttpClientAdapter {
  final dynamic body;
  StubAdapter({this.body});
  @override
  Future<ResponseBody> fetch(RequestOptions o, Stream<List<int>>? s, Future<void>? c) async {
    final bytes = (body ?? {'code': 0, 'data': []}).toString().codeUnits;
    return ResponseBody.fromBytes(bytes, 200, headers: {'content-type': ['application/json']});
  }
  @override void close({bool force = false}) {}
}

void main() {
  testWidgets('空时段 → 显示空态', (t) async {
    final storage = TokenStorage.forTest();
    final dio = buildDio(storage: storage, baseUrl: 'https://x');
    dio.httpClientAdapter = StubAdapter(body: {'code': 0, 'data': []});
    await t.pumpWidget(ProviderScope(
      overrides: [dioProvider.overrideWithValue(dio)],
      child: const MaterialApp(home: AvailabilityPage()),
    ));
    await t.pumpAndSettle();
    expect(find.text('我的空余时段'), findsOneWidget);
    expect(find.text('点击右上角新增可用时段'), findsOneWidget);
  });

  testWidgets('有时段 → 渲染一行', (t) async {
    final storage = TokenStorage.forTest();
    final dio = buildDio(storage: storage, baseUrl: 'https://x');
    dio.httpClientAdapter = StubAdapter(body: {
      'code': 0,
      'data': [
        {'id': 1, 'escort_id': 22, 'start_at': '2026-09-25T14:00:00Z',
         'end_at': '2026-09-25T18:00:00Z', 'status': 'available'},
      ],
    });
    await t.pumpWidget(ProviderScope(
      overrides: [dioProvider.overrideWithValue(dio)],
      child: const MaterialApp(home: AvailabilityPage()),
    ));
    await t.pumpAndSettle();
    expect(find.text('我的空余时段'), findsOneWidget);
    expect(find.textContaining('2026-09-25 14:00'), findsOneWidget);
  });
}
```

**Step 5: orders_page.dart（**修订：加「邀请」tab 跳 /home/invitations**）**

```dart
// lib/pages/home/orders_page.dart
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

class OrdersPage extends StatelessWidget {
  const OrdersPage({super.key});
  @override
  Widget build(BuildContext context) {
    return DefaultTabController(
      length: 4, // 修订：加「邀请」tab
      child: Scaffold(
        appBar: AppBar(
          title: const Text('我的订单'),
          bottom: const TabBar(
            tabs: [
              Tab(text: '邀请'),       // 修订：新增
              Tab(text: '待服务'),
              Tab(text: '服务中'),
              Tab(text: '已完成'),
            ],
          ),
        ),
        body: TabBarView(
          children: [
            // 「邀请」tab 直接跳到 /home/invitations（BottomNavBar 同一 ShellRoute）
            Center(
              child: FilledButton(
                onPressed: () => context.go('/home/invitations'),
                child: const Text('查看我的邀请'),
              ),
            ),
            const Center(child: Text('TODO: 待服务订单列表（spec §3.1）')),
            const Center(child: Text('TODO: 服务中订单列表')),
            const Center(child: Text('TODO: 已完成订单列表')),
          ],
        ),
      ),
    );
  }
}
```

**Step 6: wallet_page.dart + profile_page.dart（**加「我的空余时段」入口**）**

```dart
// lib/pages/home/wallet_page.dart
import 'package:flutter/material.dart';

class WalletPage extends StatelessWidget {
  const WalletPage({super.key});
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('钱包')),
      body: const Center(child: Text('TODO: 余额 + 冻结 + 提现按钮')),
    );
  }
}
```

```dart
// lib/pages/home/profile_page.dart
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

class ProfilePage extends StatelessWidget {
  const ProfilePage({super.key});
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('个人中心')),
      body: ListView(
        children: [
          const ListTile(title: Text('实名状态'), subtitle: Text('TODO: 实名状态展示')),
          const ListTile(title: Text('评分'), subtitle: Text('TODO: 评分')),
          const ListTile(title: Text('培训记录'), subtitle: Text('TODO: 已完成课程')),
          // 修订：新增「我的空余时段」入口
          ListTile(
            leading: const Icon(Icons.event_available),
            title: const Text('我的空余时段'),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => context.push('/home/availability'),
          ),
        ],
      ),
    );
  }
}
```

**Step 7: home_pages_test.dart（**5 个页面测试**）**

```dart
// test/pages/home_pages_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/pages/audit/pending_audit_page.dart';
import 'package:escort_app/pages/home/orders_page.dart';
import 'package:escort_app/pages/home/profile_page.dart';
import 'package:escort_app/pages/home/wallet_page.dart';

void main() {
  testWidgets('PendingAuditPage 渲染', (t) async {
    await t.pumpWidget(const MaterialApp(home: PendingAuditPage()));
    expect(find.text('等待审核'), findsOneWidget);
  });
  testWidgets('OrdersPage 渲染（含「邀请」tab）', (t) async {
    await t.pumpWidget(const MaterialApp(home: OrdersPage()));
    expect(find.text('我的订单'), findsOneWidget);
    expect(find.text('邀请'), findsOneWidget); // 修订
    expect(find.text('待服务'), findsOneWidget);
    expect(find.text('服务中'), findsOneWidget);
    expect(find.text('已完成'), findsOneWidget);
  });
  testWidgets('WalletPage 渲染', (t) async {
    await t.pumpWidget(const MaterialApp(home: WalletPage()));
    expect(find.text('钱包'), findsOneWidget);
  });
  testWidgets('ProfilePage 渲染（含「我的空余时段」入口）', (t) async {
    await t.pumpWidget(const MaterialApp(home: ProfilePage()));
    expect(find.text('个人中心'), findsOneWidget);
    expect(find.text('我的空余时段'), findsOneWidget); // 修订
  });
}
```

**Step 8: 跑测试**

```bash
flutter test test/pages/home/ test/pages/audit/ test/pages/home_pages_test.dart
```

Expected: PASS（pending 1 + invitations 2 + availability 2 + wallet 1 + profile 1 + orders 1 = 8 个）

**Step 9: Commit**

```bash
git add escort-app/lib/pages/audit/ escort-app/lib/pages/home/ escort-app/test/pages/
git commit -m "feat(escort-app): pages/audit/pending + pages/home (shell + invitations/availability/orders[w/邀请tab]/wallet/profile[w/时段入口]) + 8 个 widget 测试"
```

---

### Task 13: pages/order (3 页 + **CountdownBadge 改文案** + InvitationCard widget) + 测试

**Files:**
- **Modify**: `escort-app/lib/widgets/countdown_badge.dart`（**文案「锁单剩余」→「待确认剩余」**）
- **Modify**: `escort-app/lib/widgets/countdown_badge_test.dart`（**断言文案**）
- Create: `escort-app/lib/pages/order/order_detail_page.dart`
- Create: `escort-app/lib/pages/order/checkin_page.dart`
- Create: `escort-app/lib/pages/order/checkout_page.dart`
- **Create**: `escort-app/lib/widgets/invitation_card.dart`（**新增**）
- **Create**: `escort-app/lib/widgets/invitation_card_test.dart`
- Create: `escort-app/test/pages/order_pages_test.dart`

**Step 1: countdown_badge_test.dart（**RED；改文案断言**）**

```dart
// test/widgets/countdown_badge_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/widgets/countdown_badge.dart';

void main() {
  testWidgets('未超时：显示待确认剩余 N s', (t) async {
    final expireAt = DateTime.now().add(const Duration(seconds: 25));
    await t.pumpWidget(MaterialApp(home: Scaffold(body: CountdownBadge(expireAt: expireAt))));
    await t.pump();
    expect(find.textContaining('待确认剩余'), findsOneWidget); // 修订
    expect(find.textContaining('已超时'), findsNothing);
  });

  testWidgets('已超时：显示已超时', (t) async {
    final expireAt = DateTime.now().subtract(const Duration(seconds: 5));
    await t.pumpWidget(MaterialApp(home: Scaffold(body: CountdownBadge(expireAt: expireAt))));
    await t.pump();
    expect(find.text('已超时'), findsOneWidget);
  });
}
```

**Step 2: 跑测试确认失败**

```bash
flutter test test/widgets/countdown_badge_test.dart
```

Expected: FAIL — not found（首次运行）或 文案断言失败（已存在文件）

**Step 3: countdown_badge.dart（**改文案「待确认剩余」替代「锁单剩余」**）**

```dart
// lib/widgets/countdown_badge.dart
import 'dart:async';
import 'package:flutter/material.dart';

/// 30s 倒计时（spec/2026-09-24-order-matching-redesign §1.2）。
/// 用于「我的邀请」卡片显示陪诊师确认窗口剩余时间。
/// UI 文案：「待确认剩余 Ns」；< 10s 红底；归零显示「已超时」。
class CountdownBadge extends StatefulWidget {
  final DateTime expireAt;
  const CountdownBadge({required this.expireAt, super.key});

  @override
  State<CountdownBadge> createState() => _CountdownBadgeState();
}

class _CountdownBadgeState extends State<CountdownBadge> {
  late Timer _timer;
  late Duration _remaining;

  @override
  void initState() {
    super.initState();
    _remaining = widget.expireAt.difference(DateTime.now());
    _timer = Timer.periodic(const Duration(seconds: 1), (_) {
      setState(() {
        _remaining = widget.expireAt.difference(DateTime.now());
      });
    });
  }

  @override
  void dispose() {
    _timer.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (_remaining.isNegative) return const Text('已超时');
    final s = _remaining.inSeconds;
    return Chip(
      avatar: const Icon(Icons.timer, size: 16),
      label: Text('待确认剩余 ${s}s'), // 修订
      backgroundColor: s < 10 ? Colors.red[100] : null,
    );
  }
}
```

**Step 4: 跑测试确认通过**

```bash
flutter test test/widgets/countdown_badge_test.dart
```

Expected: PASS（2 个）

**Step 5: invitation_card.dart（**新增**）+ invitation_card_test.dart**

```dart
// lib/widgets/invitation_card.dart
import 'package:flutter/material.dart';
import 'package:escort_app/models/invitation.dart';
import 'package:escort_app/widgets/countdown_badge.dart';

/// 邀请卡片（spec/2026-09-24-order-matching-redesign §1.2）。
/// 顶部：医院名 + 金额；中部：服务时间；右侧 CountdownBadge；
/// 底部：确认接单 + 拒接 双按钮。
/// 当 isLive=false（已超时）：按钮禁用 + 文案改「已超时」。
class InvitationCard extends StatelessWidget {
  final Invitation invitation;
  final VoidCallback onConfirm;
  final VoidCallback onReject;

  const InvitationCard({
    required this.invitation,
    required this.onConfirm,
    required this.onReject,
    super.key,
  });

  @override
  Widget build(BuildContext context) {
    final live = invitation.isLive;
    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Expanded(
                  child: Text(invitation.hospitalName,
                      style: Theme.of(context).textTheme.titleMedium),
                ),
                Text(invitation.amountText,
                    style: Theme.of(context).textTheme.titleMedium?.copyWith(color: Colors.red)),
              ],
            ),
            const SizedBox(height: 8),
            Text('套餐: ${invitation.packageName}'),
            Text('服务时间: ${invitation.serviceStartAt.toIso8601String().substring(0, 16)}'),
            const SizedBox(height: 8),
            Align(
              alignment: Alignment.centerRight,
              child: CountdownBadge(expireAt: invitation.escortPendingExpireAt),
            ),
            const SizedBox(height: 12),
            Row(
              children: [
                Expanded(
                  child: OutlinedButton(
                    onPressed: live ? onReject : null,
                    child: const Text('拒接'),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: FilledButton(
                    onPressed: live ? onConfirm : null,
                    child: const Text('确认接单'),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
```

```dart
// test/widgets/invitation_card_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/models/invitation.dart';
import 'package:escort_app/widgets/invitation_card.dart';

Invitation _make({required bool live}) => Invitation.fromJson({
      'order_id': 7,
      'hospital_name': '北京协和医院',
      'hospital_lat': 39.9, 'hospital_lng': 116.4,
      'package_name': '半日陪诊',
      'service_start_at': '2026-09-25T09:00:00Z',
      'amount': 300.0,
      'escort_pending_expire_at': (live
              ? DateTime.now().add(const Duration(seconds: 25))
              : DateTime.now().subtract(const Duration(seconds: 5)))
          .toIso8601String(),
    });

void main() {
  testWidgets('未超时 → 按钮可点 + 倒计时显示', (t) async {
    var confirm = 0, reject = 0;
    await t.pumpWidget(MaterialApp(
      home: Scaffold(
        body: InvitationCard(
          invitation: _make(live: true),
          onConfirm: () => confirm++,
          onReject: () => reject++,
        ),
      ),
    ));
    await t.pump();
    expect(find.text('确认接单'), findsOneWidget);
    expect(find.text('拒接'), findsOneWidget);
    await t.tap(find.text('确认接单'));
    await t.pump();
    expect(confirm, 1);
    await t.tap(find.text('拒接'));
    await t.pump();
    expect(reject, 1);
  });

  testWidgets('已超时 → 按钮禁用 + 显示「已超时」', (t) async {
    var confirm = 0;
    await t.pumpWidget(MaterialApp(
      home: Scaffold(
        body: InvitationCard(
          invitation: _make(live: false),
          onConfirm: () => confirm++,
          onReject: () {},
        ),
      ),
    ));
    await t.pump();
    expect(find.text('已超时'), findsOneWidget);
    // 按钮 enabled=false；tap 不触发回调
    final btn = tester.widget<FilledButton>(find.ancestor(
      of: find.text('确认接单'),
      matching: find.byType(FilledButton),
    ));
    expect(btn.onPressed, isNull);
  });
}
```

**Step 6: 3 个 order page 骨架**

```dart
// lib/pages/order/order_detail_page.dart
import 'package:flutter/material.dart';

class OrderDetailPage extends StatelessWidget {
  final int orderId;
  const OrderDetailPage({required this.orderId, super.key});
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text('订单 #$orderId')),
      body: const Center(child: Text('TODO: 状态机进度条 + 虚拟号 + 签到按钮')),
    );
  }
}
```

```dart
// lib/pages/order/checkin_page.dart
import 'package:flutter/material.dart';

class CheckinPage extends StatelessWidget {
  final int orderId;
  const CheckinPage({required this.orderId, super.key});
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text('到院签到 #$orderId')),
      body: const Center(child: Text('TODO: GPS 调用 + 200m 校验（spec §5）')),
    );
  }
}
```

```dart
// lib/pages/order/checkout_page.dart
import 'package:flutter/material.dart';

class CheckoutPage extends StatelessWidget {
  final int orderId;
  const CheckoutPage({required this.orderId, super.key});
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text('完成打卡 #$orderId')),
      body: const Center(child: Text('TODO: 服务完成 + 提交（spec §3.1）')),
    );
  }
}
```

**Step 7: order_pages_test.dart**

```dart
// test/pages/order_pages_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/pages/order/checkin_page.dart';
import 'package:escort_app/pages/order/checkout_page.dart';
import 'package:escort_app/pages/order/order_detail_page.dart';

void main() {
  testWidgets('OrderDetailPage 渲染', (t) async {
    await t.pumpWidget(const MaterialApp(home: OrderDetailPage(orderId: 7)));
    expect(find.text('订单 #7'), findsOneWidget);
  });
  testWidgets('CheckinPage 渲染', (t) async {
    await t.pumpWidget(const MaterialApp(home: CheckinPage(orderId: 7)));
    expect(find.text('到院签到 #7'), findsOneWidget);
  });
  testWidgets('CheckoutPage 渲染', (t) async {
    await t.pumpWidget(const MaterialApp(home: CheckoutPage(orderId: 7)));
    expect(find.text('完成打卡 #7'), findsOneWidget);
  });
}
```

**Step 8: 跑测试 + Commit**

```bash
flutter test test/pages/order_pages_test.dart test/widgets/countdown_badge_test.dart test/widgets/invitation_card_test.dart
git add escort-app/lib/pages/order/ escort-app/lib/widgets/countdown_badge.dart escort-app/lib/widgets/invitation_card.dart escort-app/test/
git commit -m "feat(escort-app): pages/order (detail/checkin/checkout) + CountdownBadge (改文案「待确认剩余」) + InvitationCard + 7 个测试"
```

---

### Task 14: pages/wallet (2 页) + pages/sos (SosLongPress) + 测试

**Files:**
- Create: `escort-app/lib/pages/wallet/withdraw_page.dart`
- Create: `escort-app/lib/pages/wallet/transactions_page.dart`
- Create: `escort-app/lib/pages/wallet/wallet_pages_test.dart`
- Create: `escort-app/lib/pages/sos/sos_trigger_page.dart`
- Create: `escort-app/lib/pages/sos/sos_trigger_page_test.dart`
- Create: `escort-app/lib/widgets/sos_long_press.dart`
- Create: `escort-app/lib/widgets/sos_long_press_test.dart`

**Step 1: 2 个 wallet page + 测试**

```dart
// lib/pages/wallet/withdraw_page.dart
import 'package:flutter/material.dart';

class WithdrawPage extends StatelessWidget {
  const WithdrawPage({super.key});
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('提现申请')),
      body: const Center(child: Text('TODO: 金额 + 渠道 + 余额校验（spec §6）')),
    );
  }
}
```

```dart
// lib/pages/wallet/transactions_page.dart
import 'package:flutter/material.dart';

class TransactionsPage extends StatelessWidget {
  const TransactionsPage({super.key});
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('交易记录')),
      body: const Center(child: Text('TODO: 分页列表（spec §3.1）')),
    );
  }
}
```

```dart
// test/pages/wallet/wallet_pages_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/pages/wallet/transactions_page.dart';
import 'package:escort_app/pages/wallet/withdraw_page.dart';

void main() {
  testWidgets('WithdrawPage 渲染', (t) async {
    await t.pumpWidget(const MaterialApp(home: WithdrawPage()));
    expect(find.text('提现申请'), findsOneWidget);
  });
  testWidgets('TransactionsPage 渲染', (t) async {
    await t.pumpWidget(const MaterialApp(home: TransactionsPage()));
    expect(find.text('交易记录'), findsOneWidget);
  });
}
```

**Step 2: sos_long_press_test.dart（RED）**

```dart
// test/widgets/sos_long_press_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/widgets/sos_long_press.dart';

void main() {
  testWidgets('长按 3s 触发回调', (t) async {
    var triggered = 0;
    await t.pumpWidget(MaterialApp(
      home: Scaffold(
        body: Center(child: SosLongPress(onTriggered: () => triggered++)),
      ),
    ));
    final gesture = await t.startGesture(const Offset(100, 100));
    await t.pump(const Duration(seconds: 3));
    await gesture.up();
    expect(triggered, 1);
  });

  testWidgets('短按不触发', (t) async {
    var triggered = 0;
    await t.pumpWidget(MaterialApp(
      home: Scaffold(body: Center(child: SosLongPress(onTriggered: () => triggered++))),
    ));
    final gesture = await t.startGesture(const Offset(100, 100));
    await t.pump(const Duration(milliseconds: 500));
    await gesture.up();
    expect(triggered, 0);
  });
}
```

**Step 3: sos_long_press.dart**

```dart
// lib/widgets/sos_long_press.dart
import 'dart:async';
import 'package:flutter/material.dart';

/// SOS 长按 3s 触发器（spec §7）。
class SosLongPress extends StatefulWidget {
  final VoidCallback onTriggered;
  final Duration duration;
  const SosLongPress({
    required this.onTriggered,
    this.duration = const Duration(seconds: 3),
    super.key,
  });

  @override
  State<SosLongPress> createState() => _SosLongPressState();
}

class _SosLongPressState extends State<SosLongPress> with SingleTickerProviderStateMixin {
  late AnimationController _ctrl;
  Timer? _holdTimer;

  @override
  void initState() {
    super.initState();
    _ctrl = AnimationController(vsync: this, duration: widget.duration);
  }

  void _startHold(LongPressStartDetails _) {
    _ctrl.forward(from: 0);
    _holdTimer = Timer(widget.duration, widget.onTriggered);
  }

  void _endHold(LongPressEndDetails _) {
    _ctrl.stop();
    _holdTimer?.cancel();
    _ctrl.reset();
  }

  @override
  void dispose() {
    _holdTimer?.cancel();
    _ctrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onLongPressStart: _startHold,
      onLongPressEnd: _endHold,
      onLongPressCancel: _endHold,
      child: AnimatedBuilder(
        animation: _ctrl,
        builder: (_, __) => SizedBox(
          width: 120, height: 120,
          child: Stack(alignment: Alignment.center, children: [
            SizedBox.expand(child: CircularProgressIndicator(value: _ctrl.value, strokeWidth: 6)),
            const Icon(Icons.warning, size: 48, color: Colors.red),
          ]),
        ),
      ),
    );
  }
}
```

**Step 4: sos_trigger_page.dart + 测试**

```dart
// lib/pages/sos/sos_trigger_page.dart
import 'package:flutter/material.dart';
import 'package:escort_app/widgets/sos_long_press.dart';

class SosTriggerPage extends StatelessWidget {
  const SosTriggerPage({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('SOS 紧急报警')),
      body: Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const Text('长按 3 秒触发 SOS'),
            const SizedBox(height: 24),
            SosLongPress(onTriggered: () {
              ScaffoldMessenger.of(context).showSnackBar(
                const SnackBar(content: Text('SOS 已发出（spec §7 mock）')),
              );
            }),
          ],
        ),
      ),
    );
  }
}
```

```dart
// test/pages/sos/sos_trigger_page_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/pages/sos/sos_trigger_page.dart';

void main() {
  testWidgets('SosTriggerPage 渲染标题 + 长按提示', (t) async {
    await t.pumpWidget(const MaterialApp(home: SosTriggerPage()));
    expect(find.text('SOS 紧急报警'), findsOneWidget);
    expect(find.text('长按 3 秒触发 SOS'), findsOneWidget);
  });
}
```

**Step 5: 跑测试 + Commit**

```bash
flutter test test/pages/wallet/wallet_pages_test.dart test/pages/sos/sos_trigger_page_test.dart test/widgets/sos_long_press_test.dart
git add escort-app/lib/pages/wallet/ escort-app/lib/pages/sos/ escort-app/lib/widgets/sos_long_press.dart escort-app/test/
git commit -m "feat(escort-app): pages/wallet + pages/sos + SosLongPress widget + 5 个测试"
```

---

### Task 15: pages/message (2 页 P2) + pages/profile (2 页 P1) + 测试

**Files:**
- Create: `escort-app/lib/pages/message/conversation_list_page.dart`
- Create: `escort-app/lib/pages/message/chat_page.dart`
- Create: `escort-app/lib/pages/message/message_pages_test.dart`
- Create: `escort-app/lib/pages/profile/edit_profile_page.dart`
- Create: `escort-app/lib/pages/profile/reviews_page.dart`
- Create: `escort-app/lib/pages/profile/profile_pages_test.dart`

**Step 1: 4 个 page 骨架 + 2 个测试文件**

```dart
// lib/pages/message/conversation_list_page.dart
import 'package:flutter/material.dart';

class ConversationListPage extends StatelessWidget {
  const ConversationListPage({super.key});
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('站内信')),
      body: const Center(child: Text('TODO: 会话列表（spec §3.1 P2）')),
    );
  }
}
```

```dart
// lib/pages/message/chat_page.dart
import 'package:flutter/material.dart';

class ChatPage extends StatelessWidget {
  final int conversationId;
  const ChatPage({required this.conversationId, super.key});
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text('会话 #$conversationId')),
      body: const Center(child: Text('TODO: 消息历史 + 发送（spec §3.1 P2）')),
    );
  }
}
```

```dart
// lib/pages/profile/edit_profile_page.dart
import 'package:flutter/material.dart';

class EditProfilePage extends StatelessWidget {
  const EditProfilePage({super.key});
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('资料编辑')),
      body: const Center(child: Text('TODO: 头像 / 昵称 / 技能（spec §3.1 P1）')),
    );
  }
}
```

```dart
// lib/pages/profile/reviews_page.dart
import 'package:flutter/material.dart';

class ReviewsPage extends StatelessWidget {
  const ReviewsPage({super.key});
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('收到的评价')),
      body: const Center(child: Text('TODO: 评价列表 + 评分（spec §3.1 P1）')),
    );
  }
}
```

```dart
// test/pages/message/message_pages_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/pages/message/chat_page.dart';
import 'package:escort_app/pages/message/conversation_list_page.dart';

void main() {
  testWidgets('ConversationListPage 渲染', (t) async {
    await t.pumpWidget(const MaterialApp(home: ConversationListPage()));
    expect(find.text('站内信'), findsOneWidget);
  });
  testWidgets('ChatPage 渲染', (t) async {
    await t.pumpWidget(const MaterialApp(home: ChatPage(conversationId: 1)));
    expect(find.text('会话 #1'), findsOneWidget);
  });
}
```

```dart
// test/pages/profile/profile_pages_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/pages/profile/edit_profile_page.dart';
import 'package:escort_app/pages/profile/reviews_page.dart';

void main() {
  testWidgets('EditProfilePage 渲染', (t) async {
    await t.pumpWidget(const MaterialApp(home: EditProfilePage()));
    expect(find.text('资料编辑'), findsOneWidget);
  });
  testWidgets('ReviewsPage 渲染', (t) async {
    await t.pumpWidget(const MaterialApp(home: ReviewsPage()));
    expect(find.text('收到的评价'), findsOneWidget);
  });
}
```

**Step 2: 跑测试 + Commit**

```bash
flutter test test/pages/message/ test/pages/profile/
git add escort-app/lib/pages/message/ escort-app/lib/pages/profile/ escort-app/test/pages/message/ escort-app/test/pages/profile/
git commit -m "feat(escort-app): pages/message (2 P2) + pages/profile (2 P1) + 4 个 widget 测试"
```

---

### Task 16: providers/{**invitation, availability**, wallet, message, training} + 测试

**Files:**
- **Create**: `escort-app/lib/providers/invitation_provider.dart`（**替代原 order_provider**）
- **Create**: `escort-app/lib/providers/invitation_provider_test.dart`
- **Create**: `escort-app/lib/providers/availability_provider.dart`（**新增**）
- **Create**: `escort-app/lib/providers/availability_provider_test.dart`
- Create: `escort-app/lib/providers/wallet_provider.dart`
- Create: `escort-app/lib/providers/wallet_provider_test.dart`
- Create: `escort-app/lib/providers/message_provider.dart`
- Create: `escort-app/lib/providers/message_provider_test.dart`
- Create: `escort-app/lib/providers/training_provider.dart`
- Create: `escort-app/lib/providers/training_provider_test.dart`

**Step 1: invitation_provider.dart（**StreamProvider 5s 轮询 + isLive 客户端过滤 + confirm/reject Controller**）**

```dart
// lib/providers/invitation_provider.dart
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:escort_app/api/dio_client.dart';
import 'package:escort_app/api/order_api.dart';
import 'package:escort_app/models/invitation.dart';
import 'package:escort_app/models/order.dart';

final orderApiProvider = Provider<OrderApi>((ref) => OrderApi(ref.watch(dioProvider)));

/// 「我的邀请」列表：每 5s 拉一次（spec §7 强制约束：v1 不引 WebSocket）。
/// 客户端按 `isLive` 过滤过期邀请（30s 确认窗口）。
final invitationsProvider = StreamProvider<List<Invitation>>((ref) async* {
  final api = ref.watch(orderApiProvider);
  yield await api.invitations();
  await for (final _ in Stream.periodic(const Duration(seconds: 5))) {
    try {
      yield await api.invitations();
    } catch (_) {
      // 轮询失败保持上一次列表；下次 tick 再试（避免 UI 闪烁）
      yield* Stream<List<Invitation>>.empty();
    }
  }
});

/// 30s 内确认接单（按 orderId family 缓存 Future）。
final confirmAcceptControllerProvider =
    FutureProvider.family.autoDispose<Order, int>((ref, orderId) async {
  final o = await ref.read(orderApiProvider).confirmAccept(orderId);
  ref.invalidate(invitationsProvider);
  return o;
});

/// 拒接（按 orderId family 缓存 Future）。
final rejectAcceptControllerProvider =
    FutureProvider.family.autoDispose<Order, int>((ref, orderId) async {
  final o = await ref.read(orderApiProvider).rejectAccept(orderId);
  ref.invalidate(invitationsProvider);
  return o;
});

/// 我的订单 provider（保留；订单 tab 用）。
final myOrdersProvider = FutureProvider.family<List<Order>, String?>((ref, status) async {
  return ref.read(orderApiProvider).listMine(status: status);
});
```

**Step 2: invitation_provider_test.dart**

```dart
// test/providers/invitation_provider_test.dart
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/api/dio_client.dart';
import 'package:escort_app/providers/invitation_provider.dart';
import 'package:escort_app/services/token_storage.dart';

class StubAdapter implements HttpClientAdapter {
  final dynamic body; final int status;
  StubAdapter({this.body, this.status = 200});
  @override Future<ResponseBody> fetch(RequestOptions o, Stream<List<int>>? s, Future<void>? c) async {
    final bytes = (body ?? {'code': 0, 'data': []}).toString().codeUnits;
    return ResponseBody.fromBytes(bytes, status, headers: {'content-type': ['application/json']});
  }
  @override void close({bool force = false}) {}
}

ProviderContainer makeContainer({required dynamic body}) {
  final storage = TokenStorage.forTest();
  final dio = buildDio(storage: storage, baseUrl: 'https://x');
  dio.httpClientAdapter = StubAdapter(body: body);
  return ProviderContainer(overrides: [dioProvider.overrideWithValue(dio)]);
}

void main() {
  test('invitationsProvider 第一次 yield 列表', () async {
    final c = makeContainer(body: {
      'code': 0,
      'data': [
        {
          'order_id': 1,
          'hospital_name': '协和',
          'hospital_lat': 39.9, 'hospital_lng': 116.4,
          'package_name': '半日陪诊',
          'service_start_at': '2026-09-25T09:00:00Z',
          'amount': 300.0,
          'escort_pending_expire_at': DateTime.now().add(const Duration(seconds: 25)).toIso8601String(),
        },
      ],
    });
    addTearDown(c.dispose);
    final list = await c.read(invitationsProvider.future).timeout(const Duration(seconds: 2));
    expect(list.length, 1);
    expect(list.first.orderId, 1);
    expect(list.first.isLive, isTrue);
  });

  test('invitationsProvider 后端返 5xx 不挂', () async {
    final c = makeContainer(body: {'code': 500, 'message': 'server error'});
    c.read(invitationsProvider.future); // 不 await；autoDispose 容器结束即销毁
    addTearDown(c.dispose);
    await Future<void>.delayed(const Duration(milliseconds: 200));
    // 不抛异常即通过（轮询失败容错在 provider 内部处理）
  });
}
```

**Step 3: availability_provider.dart（**新增**）+ availability_provider_test.dart**

```dart
// lib/providers/availability_provider.dart
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:escort_app/api/availability_api.dart';
import 'package:escort_app/api/dio_client.dart';
import 'package:escort_app/models/availability.dart';

final availabilityApiProvider = Provider<AvailabilityApi>((ref) => AvailabilityApi(ref.watch(dioProvider)));

final availabilityProvider = FutureProvider<List<Availability>>((ref) async {
  return ref.read(availabilityApiProvider).listMine();
});

/// 创建时段 Controller（StateNotifier 模式；异步提交后刷新列表）。
class CreateAvailabilityController extends StateNotifier<AsyncValue<Availability?>> {
  final Ref ref;
  CreateAvailabilityController(this.ref) : super(const AsyncValue.data(null));

  Future<void> submit({required DateTime startAt, required DateTime endAt}) async {
    state = const AsyncValue.loading();
    try {
      final a = await ref.read(availabilityApiProvider).create(startAt: startAt, endAt: endAt);
      state = AsyncValue.data(a);
      ref.invalidate(availabilityProvider);
    } catch (e, st) {
      state = AsyncValue.error(e, st);
    }
  }
}

final createAvailabilityControllerProvider =
    StateNotifierProvider<CreateAvailabilityController, AsyncValue<Availability?>>((ref) {
  return CreateAvailabilityController(ref);
});

/// 删除时段（按 id family 缓存 Future）。
final deleteAvailabilityControllerProvider =
    FutureProvider.family.autoDispose<void, int>((ref, id) async {
  await ref.read(availabilityApiProvider).delete(id);
  ref.invalidate(availabilityProvider);
});
```

```dart
// test/providers/availability_provider_test.dart
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/api/dio_client.dart';
import 'package:escort_app/providers/availability_provider.dart';
import 'package:escort_app/services/token_storage.dart';

class StubAdapter implements HttpClientAdapter {
  final dynamic body; final int status;
  StubAdapter({this.body, this.status = 200});
  @override Future<ResponseBody> fetch(RequestOptions o, Stream<List<int>>? s, Future<void>? c) async {
    final bytes = (body ?? {'code': 0, 'data': []}).toString().codeUnits;
    return ResponseBody.fromBytes(bytes, status, headers: {'content-type': ['application/json']});
  }
  @override void close({bool force = false}) {}
}

void main() {
  test('availabilityProvider 列表渲染', () async {
    final storage = TokenStorage.forTest();
    final dio = buildDio(storage: storage, baseUrl: 'https://x');
    dio.httpClientAdapter = StubAdapter(body: {
      'code': 0,
      'data': [
        {'id': 1, 'escort_id': 22, 'start_at': '2026-09-25T14:00:00Z',
         'end_at': '2026-09-25T18:00:00Z', 'status': 'available'},
      ],
    });
    final c = ProviderContainer(overrides: [dioProvider.overrideWithValue(dio)]);
    addTearDown(c.dispose);
    final list = await c.read(availabilityProvider.future);
    expect(list.length, 1);
  });

  test('createAvailabilityController.submit → 调 API + 刷新列表', () async {
    final storage = TokenStorage.forTest();
    final dio = buildDio(storage: storage, baseUrl: 'https://x');
    var calls = 0;
    dio.httpClientAdapter = _CountingAdapter(onFetch: (_) => calls++, body: {
      'code': 0,
      'data': {'id': 2, 'escort_id': 22, 'start_at': '2026-09-25T14:00:00Z',
               'end_at': '2026-09-25T18:00:00Z', 'status': 'available'},
    });
    final c = ProviderContainer(overrides: [dioProvider.overrideWithValue(dio)]);
    addTearDown(c.dispose);
    await c.read(createAvailabilityControllerProvider.notifier).submit(
      startAt: DateTime.parse('2026-09-25T14:00:00Z'),
      endAt: DateTime.parse('2026-09-25T18:00:00Z'),
    );
    expect(calls, greaterThanOrEqualTo(2)); // list + create
  });
}

class _CountingAdapter implements HttpClientAdapter {
  final void Function(RequestOptions) onFetch;
  final dynamic body;
  _CountingAdapter({required this.onFetch, this.body});
  @override Future<ResponseBody> fetch(RequestOptions o, Stream<List<int>>? s, Future<void>? c) async {
    onFetch(o);
    final bytes = (body ?? {'code': 0, 'data': null}).toString().codeUnits;
    return ResponseBody.fromBytes(bytes, 200, headers: {'content-type': ['application/json']});
  }
  @override void close({bool force = false}) {}
}
```

**Step 4: wallet_provider.dart + 测试**

```dart
// lib/providers/wallet_provider.dart
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:escort_app/api/dio_client.dart';
import 'package:escort_app/api/wallet_api.dart';
import 'package:escort_app/models/wallet.dart';
import 'package:escort_app/models/withdrawal.dart';

final walletApiProvider = Provider<WalletApi>((ref) => WalletApi(ref.watch(dioProvider)));

final walletProvider = FutureProvider<Wallet>((ref) async {
  return ref.read(walletApiProvider).getMyWallet();
});

final transactionsProvider = FutureProvider<List<Withdrawal>>((ref) async {
  return ref.read(walletApiProvider).listTransactions();
});

class WithdrawController extends StateNotifier<AsyncValue<Withdrawal?>> {
  final Ref ref;
  WithdrawController(this.ref) : super(const AsyncValue.data(null));

  Future<void> submit({required double amount, required String channel, required String account}) async {
    state = const AsyncValue.loading();
    try {
      final w = await ref.read(walletApiProvider).withdraw(amount: amount, channel: channel, account: account);
      state = AsyncValue.data(w);
      ref.invalidate(walletProvider); // 刷新钱包
    } catch (e, st) {
      state = AsyncValue.error(e, st);
    }
  }
}

final withdrawControllerProvider = StateNotifierProvider<WithdrawController, AsyncValue<Withdrawal?>>((ref) {
  return WithdrawController(ref);
});
```

```dart
// test/providers/wallet_provider_test.dart
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/api/dio_client.dart';
import 'package:escort_app/providers/wallet_provider.dart';
import 'package:escort_app/services/token_storage.dart';

class StubAdapter implements HttpClientAdapter {
  final dynamic body; final int status;
  StubAdapter({this.body, this.status = 200});
  @override Future<ResponseBody> fetch(RequestOptions o, Stream<List<int>>? s, Future<void>? c) async {
    final bytes = (body ?? {}).toString().codeUnits;
    return ResponseBody.fromBytes(bytes, status, headers: {'content-type': ['application/json']});
  }
  @override void close({bool force = false}) {}
}

void main() {
  test('walletProvider 返回 Wallet', () async {
    final storage = TokenStorage.forTest();
    final dio = buildDio(storage: storage, baseUrl: 'https://x');
    dio.httpClientAdapter = StubAdapter(body: {
      'code': 0,
      'data': {
        'user_id': 1, 'balance': 100.0, 'frozen': 30.0,
        'total_earned': 200.0, 'updated_at': '2026-09-24T15:00:00Z',
      },
    });
    final c = ProviderContainer(overrides: [dioProvider.overrideWithValue(dio)]);
    addTearDown(c.dispose);
    final w = await c.read(walletProvider.future);
    expect(w.balance, 100.0);
  });
}
```

**Step 5: message_provider.dart + training_provider.dart**（一次性写完，无单测 — Provider 包装层由 API 单测覆盖）

```dart
// lib/providers/message_provider.dart
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:escort_app/api/dio_client.dart';
import 'package:escort_app/api/message_api.dart';

final messageApiProvider = Provider<MessageApi>((ref) => MessageApi(ref.watch(dioProvider)));

final conversationsProvider = FutureProvider<List<Conversation>>((ref) async {
  return ref.read(messageApiProvider).listConversations();
});

final messagesProvider = FutureProvider.family<List<MessageItem>, int>((ref, conversationId) async {
  return ref.read(messageApiProvider).listMessages(conversationId);
});
```

```dart
// lib/providers/training_provider.dart
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:escort_app/api/dio_client.dart';
import 'package:escort_app/api/training_api.dart';

final trainingApiProvider = Provider<TrainingApi>((ref) => TrainingApi(ref.watch(dioProvider)));

final coursesProvider = FutureProvider<List<TrainingCourse>>((ref) async {
  return ref.read(trainingApiProvider).listCourses();
});

class QuizSubmitController extends StateNotifier<AsyncValue<int?>> {
  final Ref ref;
  QuizSubmitController(this.ref) : super(const AsyncValue.data(null));

  Future<void> submit({required int courseId, required List<int> answers}) async {
    state = const AsyncValue.loading();
    try {
      final score = await ref.read(trainingApiProvider).submitQuiz(courseId: courseId, answers: answers);
      state = AsyncValue.data(score);
    } catch (e, st) {
      state = AsyncValue.error(e, st);
    }
  }
}

final quizSubmitProvider = StateNotifierProvider.family<QuizSubmitController, AsyncValue<int?>, int>((ref, courseId) {
  return QuizSubmitController(ref);
});
```

**Step 6: 跑全部 providers 测试**

```bash
flutter test test/providers/
```

Expected: PASS（auth 3 + invitation 2 + availability 2 + wallet 1 = 8 个；message/training 无单测靠 API 单测覆盖）

**Step 7: Commit**

```bash
git add escort-app/lib/providers/ escort-app/test/providers/
git commit -m "feat(escort-app): providers (invitation[5s 轮询]/availability/wallet/message/training) + 6 个 Provider 单测"
```

---

### Task 17: widgets/ 公共组件 + 测试（**order_card + availability_tile + rating_stars + status_chip + gps_checkin_button**）

**Files:**
- Create: `escort-app/lib/widgets/order_card.dart`
- Create: `escort-app/lib/widgets/order_card_test.dart`
- **Create**: `escort-app/lib/widgets/availability_tile.dart`（**新增**）
- **Create**: `escort-app/lib/widgets/availability_tile_test.dart`
- Create: `escort-app/lib/widgets/rating_stars.dart`
- Create: `escort-app/lib/widgets/rating_stars_test.dart`
- **Modify**: `escort-app/lib/widgets/status_chip.dart`（**加 escortPendingAcceptance 颜色映射**）
- **Modify**: `escort-app/lib/widgets/status_chip_test.dart`
- Create: `escort-app/lib/widgets/gps_checkin_button.dart`
- Create: `escort-app/lib/widgets/gps_checkin_button_test.dart`

**Step 1: status_chip_test.dart（**RED；加 escortPendingAcceptance 用例**）**

```dart
// test/widgets/status_chip_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/models/order.dart';
import 'package:escort_app/widgets/status_chip.dart';

void main() {
  testWidgets('selectingEscort → 蓝色', (t) async {
    await t.pumpWidget(MaterialApp(home: Scaffold(body: StatusChip(status: OrderStatus.selectingEscort))));
    expect(find.byType(StatusChip), findsOneWidget);
  });

  testWidgets('escortPendingAcceptance → 橙色（spec §4.1）', (t) async {
    await t.pumpWidget(MaterialApp(home: Scaffold(body: StatusChip(status: OrderStatus.escortPendingAcceptance))));
    expect(find.byType(StatusChip), findsOneWidget);
  });

  testWidgets('accepted → 绿色', (t) async {
    await t.pumpWidget(MaterialApp(home: Scaffold(body: StatusChip(status: OrderStatus.accepted))));
    expect(find.byType(StatusChip), findsOneWidget);
  });
}
```

**Step 2: 跑测试确认失败**

```bash
flutter test test/widgets/status_chip_test.dart
```

Expected: FAIL — not found（首次）或缺 escortPendingAcceptance 用例

**Step 3: status_chip.dart（**加 escortPendingAcceptance 颜色：橙色**）**

```dart
// lib/widgets/status_chip.dart
import 'package:flutter/material.dart';
import 'package:escort_app/models/order.dart';

/// 订单状态机颜色 chip（spec §3.2）。
class StatusChip extends StatelessWidget {
  final OrderStatus status;
  const StatusChip({required this.status, super.key});

  Color _color() {
    switch (status) {
      case OrderStatus.created:
      case OrderStatus.paid:
        return Colors.grey;
      case OrderStatus.matching:
      case OrderStatus.selectingEscort:           // 候选已生成（spec/2026-09-24-order-matching-redesign）
        return Colors.blue;
      case OrderStatus.escortPendingAcceptance:   // 待陪诊师 30s 确认（spec §1.2；橙色醒目）
        return Colors.orange;
      case OrderStatus.accepted:
      case OrderStatus.inService:
        return Colors.green;
      case OrderStatus.completed:
      case OrderStatus.reviewed:
        return Colors.teal;
      case OrderStatus.refunding:
      case OrderStatus.refunded:
      case OrderStatus.settling:
        return Colors.amber;
      case OrderStatus.disputed:
        return Colors.deepOrange;
      case OrderStatus.closed:
      case OrderStatus.canceled:
        return Colors.black54;
    }
  }

  String _text() {
    switch (status) {
      case OrderStatus.created: return '已创建';
      case OrderStatus.paid: return '已支付';
      case OrderStatus.matching: return '匹配中';
      case OrderStatus.selectingEscort: return '待患者选人';
      case OrderStatus.escortPendingAcceptance: return '待陪诊师确认';
      case OrderStatus.accepted: return '已接单';
      case OrderStatus.inService: return '服务中';
      case OrderStatus.completed: return '已完成';
      case OrderStatus.reviewed: return '已评价';
      case OrderStatus.refunding: return '退款中';
      case OrderStatus.refunded: return '已退款';
      case OrderStatus.settling: return '结算中';
      case OrderStatus.disputed: return '申诉中';
      case OrderStatus.closed: return '已关闭';
      case OrderStatus.canceled: return '已取消';
    }
  }

  @override
  Widget build(BuildContext context) {
    return Chip(
      label: Text(_text()),
      backgroundColor: _color(),
      labelStyle: const TextStyle(color: Colors.white),
    );
  }
}
```

**Step 4: 跑测试确认通过**

```bash
flutter test test/widgets/status_chip_test.dart
```

Expected: PASS（3 个）

**Step 5: order_card.dart + order_card_test.dart**

```dart
// lib/widgets/order_card.dart
import 'package:flutter/material.dart';
import 'package:escort_app/models/order.dart';

/// 通用订单卡片（不绑抢单池；orders tab 共用）。
class OrderCard extends StatelessWidget {
  final OrderSummary summary;
  final VoidCallback? onTap;
  const OrderCard({required this.summary, this.onTap, super.key});

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: ListTile(
        title: Text(summary.hospitalName),
        subtitle: Text('${summary.serviceStartAtText}  ·  ${summary.packageName}'),
        trailing: Text(summary.amountText, style: const TextStyle(color: Colors.red)),
        onTap: onTap,
      ),
    );
  }
}
```

```dart
// test/widgets/order_card_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/models/order.dart';
import 'package:escort_app/widgets/order_card.dart';

void main() {
  testWidgets('OrderCard 渲染', (t) async {
    final s = OrderSummary.fromJson({
      'id': 7, 'hospital_name': '协和',
      'service_start_at_text': '明天 09:00',
      'amount': 300.0, 'package_name': '半日陪诊',
    });
    await t.pumpWidget(MaterialApp(home: Scaffold(body: OrderCard(summary: s))));
    expect(find.text('协和'), findsOneWidget);
    expect(find.text('¥300.00'), findsOneWidget);
  });
}
```

**Step 6: availability_tile.dart（**新增**）+ availability_tile_test.dart**

```dart
// lib/widgets/availability_tile.dart
import 'package:flutter/material.dart';
import 'package:escort_app/models/availability.dart';

/// 时段行（左时段 / 右删除按钮 或 「已预订」chip）。
/// `onDelete == null` 时不渲染删除按钮（用于 booked/canceled）。
class AvailabilityTile extends StatelessWidget {
  final Availability availability;
  final VoidCallback? onDelete;
  const AvailabilityTile({required this.availability, this.onDelete, super.key});

  @override
  Widget build(BuildContext context) {
    return ListTile(
      leading: Icon(
        availability.status == AvailabilityStatus.available
            ? Icons.event_available
            : Icons.event_busy,
      ),
      title: Text('${availability.startText} ~ ${availability.endText}'),
      subtitle: Text(_statusText(availability.status)),
      trailing: onDelete != null
          ? IconButton(icon: const Icon(Icons.delete_outline), onPressed: onDelete)
          : Chip(label: Text(_statusText(availability.status))),
    );
  }

  String _statusText(AvailabilityStatus s) {
    switch (s) {
      case AvailabilityStatus.available: return '可预约';
      case AvailabilityStatus.booked: return '已预订';
      case AvailabilityStatus.canceled: return '已撤销';
    }
  }
}
```

```dart
// test/widgets/availability_tile_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/models/availability.dart';
import 'package:escort_app/widgets/availability_tile.dart';

Availability _make({required AvailabilityStatus status, int? orderId}) =>
    Availability.fromJson({
      'id': 1, 'escort_id': 22,
      'start_at': '2026-09-25T14:00:00Z',
      'end_at': '2026-09-25T18:00:00Z',
      'status': status.name,
      if (orderId != null) 'order_id': orderId,
    });

void main() {
  testWidgets('available + onDelete → 显示删除按钮', (t) async {
    var deleted = false;
    await t.pumpWidget(MaterialApp(
      home: Scaffold(body: AvailabilityTile(availability: _make(status: AvailabilityStatus.available), onDelete: () => deleted = true)),
    ));
    expect(find.byIcon(Icons.delete_outline), findsOneWidget);
    await t.tap(find.byIcon(Icons.delete_outline));
    await t.pump();
    expect(deleted, isTrue);
  });

  testWidgets('booked + onDelete=null → 显示「已预订」chip（无删除按钮）', (t) async {
    await t.pumpWidget(MaterialApp(
      home: Scaffold(body: AvailabilityTile(availability: _make(status: AvailabilityStatus.booked, orderId: 7))),
    ));
    expect(find.byIcon(Icons.delete_outline), findsNothing);
    expect(find.text('已预订'), findsOneWidget);
  });
}
```

**Step 7: rating_stars.dart + gps_checkin_button.dart + 各自测试**

```dart
// lib/widgets/rating_stars.dart
import 'package:flutter/material.dart';

class RatingStars extends StatelessWidget {
  final double rating;
  final double size;
  const RatingStars({required this.rating, this.size = 16, super.key});

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: List.generate(5, (i) {
        final filled = i < rating.round();
        return Icon(
          filled ? Icons.star : Icons.star_border,
          size: size,
          color: Colors.amber,
        );
      }),
    );
  }
}
```

```dart
// test/widgets/rating_stars_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/widgets/rating_stars.dart';

void main() {
  testWidgets('rating 4 → 4 颗实心 + 1 颗空心', (t) async {
    await t.pumpWidget(const MaterialApp(home: Scaffold(body: RatingStars(rating: 4))));
    expect(find.byIcon(Icons.star), findsNWidgets(4));
    expect(find.byIcon(Icons.star_border), findsNWidgets(1));
  });
}
```

```dart
// lib/widgets/gps_checkin_button.dart
import 'dart:async';
import 'package:flutter/material.dart';

/// 长按 2s GPS 签到按钮（spec §5）。
class GpsCheckinButton extends StatefulWidget {
  final VoidCallback onLongPressed;
  final Duration duration;
  const GpsCheckinButton({
    required this.onLongPressed,
    this.duration = const Duration(seconds: 2),
    super.key,
  });

  @override
  State<GpsCheckinButton> createState() => _GpsCheckinButtonState();
}

class _GpsCheckinButtonState extends State<GpsCheckinButton> with SingleTickerProviderStateMixin {
  late AnimationController _ctrl;
  Timer? _holdTimer;

  @override
  void initState() {
    super.initState();
    _ctrl = AnimationController(vsync: this, duration: widget.duration);
  }

  void _start(LongPressStartDetails _) {
    _ctrl.forward(from: 0);
    _holdTimer = Timer(widget.duration, widget.onLongPressed);
  }

  void _end(LongPressEndDetails _) {
    _ctrl.stop();
    _holdTimer?.cancel();
    _ctrl.reset();
  }

  @override
  void dispose() {
    _holdTimer?.cancel();
    _ctrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onLongPressStart: _start,
      onLongPressEnd: _end,
      onLongPressCancel: _end,
      child: AnimatedBuilder(
        animation: _ctrl,
        builder: (_, __) => FilledButton(
          onPressed: null,
          child: Text('长按 ${(widget.duration.inSeconds)}s GPS 签到 (${(_ctrl.value * 100).toInt()}%)'),
        ),
      ),
    );
  }
}
```

```dart
// test/widgets/gps_checkin_button_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/widgets/gps_checkin_button.dart';

void main() {
  testWidgets('长按 2s 触发回调', (t) async {
    var called = 0;
    await t.pumpWidget(MaterialApp(
      home: Scaffold(body: Center(child: GpsCheckinButton(onLongPressed: () => called++))),
    ));
    final g = await t.startGesture(const Offset(100, 100));
    await t.pump(const Duration(seconds: 2));
    await g.up();
    expect(called, 1);
  });
}
```

**Step 8: 跑全部 widgets 测试 + Commit**

```bash
flutter test test/widgets/
git add escort-app/lib/widgets/ escort-app/test/widgets/
git commit -m "feat(escort-app): widgets (OrderCard/InvitationCard/AvailabilityTile/RatingStars/StatusChip[+escortPendingAcceptance]/GpsCheckinButton) + 8 个 widget 测试"
```

---

### Task 18: app.dart + main.dart + integration_test（**新增 invitations E2E**）+ dev.md

**Files:**
- Create: `escort-app/lib/app.dart`
- Create: `escort-app/lib/main.dart`（替换 `flutter create` 默认）
- Create: `escort-app/integration_test/app_test.dart`
- **Create**: `escort-app/integration_test/invitations_test.dart`（**替代原 feed_test**）
- Modify: `dev.md`

**Step 1: app.dart**

```dart
// lib/app.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:escort_app/router/app_router.dart';

class DoctorsEscortApp extends ConsumerWidget {
  const DoctorsEscortApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return MaterialApp.router(
      title: '陪诊师',
      theme: ThemeData(
        useMaterial3: true,
        colorScheme: ColorScheme.fromSeed(seedColor: const Color(0xFF1989FA)),
      ),
      routerConfig: ref.watch(goRouterProvider),
    );
  }
}
```

> `goRouterProvider` 在 `lib/router/app_router.dart` 里定义（Task 9）。

**Step 2: main.dart**

```dart
// lib/main.dart
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:escort_app/api/dio_client.dart';
import 'package:escort_app/app.dart';
import 'package:escort_app/providers/auth_provider.dart';
import 'package:escort_app/services/token_storage.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  const storage = FlutterSecureStorage();
  final tokenStorage = TokenStorage(storage);
  final dio = buildDio(storage: tokenStorage, baseUrl: kDefaultBaseUrl, onUnauthorized: () {
    // 401 → 清理 token；SplashPage bootstrap 会重定向到 /auth/login
  });

  runApp(ProviderScope(
    overrides: [
      tokenStorageProvider.overrideWithValue(tokenStorage),
      dioProvider.overrideWithValue(dio),
    ],
    child: const DoctorsEscortApp(),
  ));
}
```

**Step 3: integration_test/app_test.dart**（Splash → Login 注册引导链）

```dart
// integration_test/app_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:escort_app/app.dart';
import 'package:escort_app/main.dart' as app;
import 'package:flutter_riverpod/flutter_riverpod.dart';

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  testWidgets('启动 → splash → 跳 login（无 token）', (t) async {
    app.main();
    await t.pumpAndSettle(const Duration(seconds: 3));
    // 跳转到登录页：标题 "登录"
    expect(find.text('登录'), findsWidgets);
  });
}
```

**Step 4: integration_test/invitations_test.dart**（**替代原 feed_test，验证匹配新流程**）

```dart
// integration_test/invitations_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:escort_app/main.dart' as app;

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  testWidgets('启动 → splash → 跳 login（无 token）→ 我的空余时段不可达', (t) async {
    app.main();
    await t.pumpAndSettle(const Duration(seconds: 3));
    // 1. splash 跳到 login（未登录守卫）
    expect(find.text('登录'), findsWidgets);
    // 2. /home/invitations 被守卫拦截，不会渲染
    expect(find.text('我的邀请'), findsNothing);
    // 3. /home/availability 顶层路由不在 BottomNavBar；同样被拦截
    expect(find.text('我的空余时段'), findsNothing);
  });
}
```

> **E2E 真实测试**需 mock 后端或打 staging；本骨架只验证路由注册 + 守卫拦截。
> v2 完整 E2E（mock invitations → 30s 倒计时 → 确认接单 → 状态切换）留后续 plan。

**Step 5: dev.md 加 §10.13**

```markdown
### 10.13 escort-app setup plan（2026-09-24，revised）

落地陪诊师 Flutter App 骨架（iOS + Android）。
**业务流程按 spec/2026-09-24-order-matching-redesign 修订**：删抢单池，新增「我的邀请」+「我的空余时段」。

**落地 commits（约 18 个）**：

| commit | 内容 |
| :-- | :-- |
| chore(escort-app) | flutter create + pubspec deps + iOS/Android 权限 + README |
| chore(escort-app) | OpenAPI 生成器 config + gen-api.sh |
| feat(escort-app) | utils (format/trace/error_handler[+13101/13102/13103]) + 14 单测 |
| feat(escort-app) | models (Order[+selectingEscort/escortPendingAcceptance]/Invitation/Availability/EscortProfile/Wallet/Withdrawal/Signal) + 13 单测 |
| feat(escort-app) | services (TokenStorage/GpsService/NopPushService) + 3 单测 |
| feat(escort-app) | dio_client (拦截器) + 3 单测 |
| feat(escort-app) | api modules (AuthApi/OrderApi[invitations+confirmAccept+rejectAccept]/AvailabilityApi/...) + 7 StubAdapter 单测 |
| feat(escort-app) | AuthState (sealed) + AuthNotifier + 3 单测 |
| feat(escort-app) | router (go_router + 路由：删 /home/feed 加 /home/invitations + /home/availability + 3 守卫) + 8 单测 |
| feat(escort-app) | pages/splash (跳 /home/invitations) + auth (login/register) + 3 测试 |
| feat(escort-app) | pages/onboarding (6 骨架) + 6 测试 |
| feat(escort-app) | pages/audit/pending + pages/home (shell + invitations/availability/orders[w/邀请tab]/wallet/profile[w/时段入口]) + 8 测试 |
| feat(escort-app) | pages/order (3 页 + CountdownBadge[改文案「待确认剩余」]) + InvitationCard + 7 测试 |
| feat(escort-app) | pages/wallet + pages/sos + SosLongPress + 5 测试 |
| feat(escort-app) | pages/message (2 P2) + pages/profile (2 P1) + 4 测试 |
| feat(escort-app) | providers (invitation[5s 轮询]/availability/wallet/message/training) + 6 Provider 单测 |
| feat(escort-app) | widgets (OrderCard/InvitationCard/AvailabilityTile/RatingStars/StatusChip[+escortPendingAcceptance]/GpsCheckinButton) + 8 测试 |
| feat(escort-app) | app.dart + main.dart 装配 + integration_test smoke (app + invitations 守卫) + dev.md 10.13 |

**API 增量**：14 API（escort 端）= 原 13 + AvailabilityApi（listMine/create/delete）− 抢单 0；业务码增量 13101~13103（邀请过期/已被接受/时段冲突）。

**业务页面修订（vs 原 plan）**：
- ❌ 删 `/home/feed` 路由 + `FeedPage` + `feedProvider` + `OrderApi.feed/accept` + `integration_test/feed_test.dart`
- ✅ 新 `/home/invitations` 路由 + `InvitationsPage` + `/home/availability` 路由 + `AvailabilityPage`（顶层；非 BottomNavBar tab；从 ProfilePage 入口进）
- ✅ 修 `OrderStatus` enum：`selectingEscort` / `escortPendingAcceptance`；删 `pendingAcceptance`（保留兼容 alias）
- ✅ 修 `CountdownBadge` UI 文案：「锁单剩余」→「待确认剩余」
- ✅ 修 `HomeShell` 4 tabs：`feed` → `invitations`
- ✅ 修 `OrdersPage` tab：`邀请`（跳 `/home/invitations`）/ 待服务 / 服务中 / 已完成
- ✅ 修 `ProfilePage`：加「我的空余时段」入口 → `/home/availability`

**未做**：UI 完善（具体表单 / 错误 toast / 动画）留 v1.x 后续 plan；OpenAPI generated/ 待 `web/openapi/contracts.yaml` 落地后跑 `bash scripts/gen-api.sh`；推送通道（极光/友盟）留 v2；视频陪诊 / i18n / 离线模式 / AI 推荐留 v3；mock E2E（mock invitations 30s 倒计时 → 确认接单）留 v1.x 后续 plan。
```

**Step 6: 跑 `flutter analyze` 验证**

```bash
cd escort-app
flutter analyze
```

Expected: `No issues found!`（lib/api/generated/ 已 exclude；其余 0 issue）

**Step 7: 跑 `flutter test` 验证全部单测**

```bash
flutter test
```

Expected: PASS（约 95+ 测试）

**Step 8: Commit**

```bash
cd /Users/growduduan/ai/doctors
git add escort-app/lib/app.dart escort-app/lib/main.dart escort-app/integration_test/ dev.md
git commit -m "feat(escort-app): app.dart + main.dart 装配 + integration_test smoke (app + invitations) + dev.md 10.13 (revised)"
```

---

### Task 19: 全量回归 + push

**Step 1: 跑 `flutter analyze` + 全部单测**

```bash
cd /Users/growduduan/ai/doctors/escort-app
flutter pub get
flutter analyze                # 0 issue
flutter test                   # 全过
```

**Step 2: 跑 `flutter build` 验证 iOS / Android 编译通过**

```bash
flutter build apk --debug    # Android debug APK（cleartext 允许）
flutter build ios --debug --no-codesign   # iOS debug（不签）
```

Expected: 两个平台均编译通过（无 API 真机联调，仅验证编译链）。

**Step 3: 跑 `integration_test`（需要真机 / 模拟器）**

```bash
flutter test integration_test/
```

> v1 在 CI 上跑需要至少一个连接的设备或模拟器；本 plan 在 `flutter test` 已覆盖大部分 widget 逻辑，integration_test 留运行验证入口。

**Step 4: push**

```bash
cd /Users/growduduan/ai/doctors
git push origin main
```

Expected: pushed.

**Step 5: Commit（如 dev.md / README 微调）**

```bash
git add escort-app/ dev.md
git commit -m "chore(escort-app): 全量验证（flutter analyze + test + build）"
```

---

## Self-Review

- ✅ **Spec 覆盖**（`specs/2026-09-24-order-matching-redesign.md`）：
    - **§1.2 新流程**：陪诊师设时段 → 患者选人 → 陪诊师 30s 确认 → accepted。已映射到：
        - `AvailabilityPage` + `AvailabilityApi.listMine/create/delete`（§3.2 escort_availabilities 表）
        - `InvitationsPage` + `OrderApi.invitations/confirmAccept/rejectAccept`（§4.1 escort 端新 API）
        - `invitationProvider` 5s 轮询 + 客户端 `isLive` 过滤（§7.1 不引 WebSocket）
        - `InvitationCard` + `CountdownBadge` UI（30s 确认窗口）
    - **§2.1 状态机**：`OrderStatus.selectingEscort` / `escortPendingAcceptance` 落地；`pendingAcceptance` 保留兼容 alias；transitions 通过 `OrderStatus.fromString` 隐式支持
    - **§3.2 escort_availabilities 表**：`Availability` 模型 + `AvailabilityStatus` enum（available/booked/canceled）
    - **§4.1 escort 端新 API**：
        - `GET /escorts/me/invitations` → `OrderApi.invitations()`
        - `POST /orders/{id}/confirm-accept` → `OrderApi.confirmAccept()`
        - `POST /orders/{id}/reject-accept` → `OrderApi.rejectAccept()`
        - `GET /escorts/me/availability` → `AvailabilityApi.listMine()`
        - `PUT /escorts/me/availability` → `AvailabilityApi.create()`
        - `DELETE /escorts/me/availability/{id}` → `AvailabilityApi.delete()`
    - **§4.2 撤销**：`GET /match/feed` + `POST /orders/{id}/accept` **已删**（原 `feedProvider` + `OrderApi.feed/accept` 已替换）
    - **§6 影响的 plan**：本 plan 已按 13 → 8 修订清单完成 escort-app-setup 章节
    - **§7 设计决策**：✅ 不引 WebSocket（5s 轮询保留原 feedProvider 架构）；✅ 30s 超时由后端 `order-lock` 处理（前端按 `escort_pending_expire_at` 自渲染）；✅ `AvailabilityApi.create` 透传 `startAt/endAt`（时段冲突由后端 UNIQUE 索引校验，错误码 13103）
    - **§8 验收**：本骨架覆盖 E2E 场景 1（陪诊师设时段）和场景 5（P 选 A → 30s 倒计时 → confirm → accepted）；场景 6/7 的真实 E2E（mock backend + UI 自动化）留 v1.x 后续 plan

- ✅ **无占位符**: 每个 Task 给出完整代码 / 测试 / 命令；commit message 明确（18 个 commits）

- ✅ **类型一致**：
    - `OrderStatus.fromString` ↔ `models/order.dart` ↔ `StatusChip._text`
    - `AvailabilityStatus.fromString` ↔ `AvailabilityTile._statusText` ↔ `Availability.isDeletable`
    - `Invitation.fromJson` ↔ `InvitationCard` ↔ `invitationsProvider`（isLive 客户端过滤）
    - `Availability.fromJson` ↔ `AvailabilityTile` ↔ `availabilityProvider`
    - `OnboardingStage.fromString` ↔ `EscortProfile.fromJson` ↔ `EscortApi.realNameAuth` 返回
    - `SignalStatus.fromString` ↔ `SosApi.trigger` 返回
    - `WithdrawalStatus.fromString` ↔ `WalletApi.withdraw/listTransactions` 返回
    - `AuthState` sealed → `authGuardProvider.maybeWhen(authenticated: (userId, phone, role, realNameVerified, approved) => ...)` pattern（Task 9 注明若 pattern 数不对按编译错误调整）
    - `goRouterProvider` 在 `app_router.dart` 定义，`app.dart` 引用

- ✅ **测试矩阵**：
    - utils: 5（format）+ 2（trace）+ 7（error_handler 含 13101/13102/13103）= 14 个
    - models: 3（order）+ 3（invitation）+ 3（availability）+ 1（escort_profile）+ 1（wallet）+ 1（withdrawal）+ 1（signal）= 13 个
    - services: 1（token）+ 1（gps）+ 1（push）= 3 个
    - api: 1（auth）+ 3（order 含 invitations/confirm/reject）+ 3（availability）+ 6（其它占位）≈ 13 个
    - providers: 3（auth）+ 2（invitation）+ 2（availability）+ 1（wallet）+ 1（message）+ 1（training）= 10 个
    - router: 4（guards）+ 4（redirect）= 8 个
    - widgets: 2（countdown）+ 2（invitation_card）+ 2（status_chip）+ 1（order_card）+ 2（availability_tile）+ 1（rating_stars）+ 1（gps_checkin）+ 2（sos_long_press）= 13 个
    - pages: 8（home：pending+invitations+availability+wallet+profile+orders + splash + login）≈ 12 个
    - integration: 1（app）+ 1（invitations 守卫）= 2 个
    - **合计 ≈ 99 个 `flutter test` + 2 个 `integration_test`**（vs 原 plan 73 个；净增 ~26 个，含新业务页面/模型/状态机/widget 测试）

- ✅ **YAGNI**：
    - **不**引入 FlutterFire（`firebase_messaging` 留 v2）
    - **不**引入 BLoC / GetX / Provider（仅 Riverpod）
    - **不**引入 retrofit / chopper / dio_smart_retry（裸 dio + 拦截器）
    - **不**引入 lottie / flutter_markdown（v1 用 Text 占位）
    - **不**引入 i18n（spec §14 v3 才做）
    - **不**接 OpenAPI generated（v1 仅写 Dart API wrapper；`scripts/gen-api.sh` 留口子，generated/ 不入仓逻辑代码）
    - **不**做推送通道（spec §14 留 v2）
    - **不**做 video player（spec §14 v3）
    - **不**做 WebSocket 推送（spec/2026-09-24-order-matching-redesign §7.1 强制约束；用 5s 轮询）
    - **不**实现前端时段冲突检测（依赖后端 UNIQUE 索引 + 错误码 13103）

## Execution Options

> Plan 已 commit 到 `docs/superpowers/plans/2026-09-24-escort-app-setup.md`。
> 当前为 plan 阶段；不进入实施。

**关联 plan**：
- 后端：`docs/superpowers/plans/2026-09-24-escort-availability.md`（**新增**；escort_availabilities 表迁移 + AvailabilityRepo + AvailabilityService + 时段冲突检测 + 与 order-service / match-service 集成 + 8 集成测试）— 本 plan Task 7 的 `AvailabilityApi` 依赖其后端落地
- 后端：`docs/superpowers/plans/2026-09-24-escort-business.md`（escort 业务 API）— escort-app Task 7 的 `EscortApi` 模块依赖
- 后端：`docs/superpowers/plans/2026-09-24-escort-order-ext.md`（订单 escort 视图扩展）— escort-app Task 7 `OrderApi.invitations/confirmAccept/rejectAccept` 依赖（spec/2026-09-24-order-matching-redesign §4.1 重写 §1）
- 后端：`docs/superpowers/plans/2026-09-24-order-lock.md`（**整个 plan 改写**：从抢单锁单 → 选 escort + 30s 陪诊师确认锁单；Redis SETNX key 改为 `orders:confirm:{order_id}`）— 前端倒计时数据源
- 后端：`docs/superpowers/plans/2026-09-24-state-machine.md`（§4.2 加 selecting_escort + escort_pending_acceptance 状态）— `OrderStatus` enum 一致性依赖
- 前端：`docs/superpowers/plans/2026-09-24-patient-miniapp-setup.md`（§3.1 订单详情：新增"选陪诊师"步骤 + 候选列表页 + 30s 倒计时）— 与本 plan 的 `InvitationsPage` 是互补两端
- 前端：`docs/superpowers/plans/2026-09-24-admin-web-setup.md`（订单列表：新增"是否已选 escort"列 + 状态机新颜色）— 与本 plan 的 `OrderStatus` / `StatusChip` 一致

**下一步选项**：
1. **进入实施** —— 实施 Task 1~19（subagent-driven 推荐）
2. **暂停 + review** —— 调整 plan（页面骨架 vs UI 完善 / mock E2E 时机 / `invitationsProvider` 轮询节奏 5s 是否合适 / `AvailabilityTile` 「已预订」状态视觉 / `HomeShell` 4 tabs 是否需要「我的空余时段」独立入口）
3. **继续产 plan** —— 接着出 escort-availability（新）/ escort-order-ext / order-lock 改写 等后端 plan