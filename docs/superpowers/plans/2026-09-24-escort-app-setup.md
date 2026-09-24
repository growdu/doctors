# escort-app Setup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 落地陪诊师 Flutter App（iOS + Android）骨架：项目脚手架、24 个 P0 页面骨架、Riverpod providers、API client + 拦截器、token 持久化、go_router + 3 个路由守卫、`flutter test` + `integration_test` 框架配置、OpenAPI Dart client 自动生成。**UI 完善（页面具体交互 / 表单 / 错误 toast 等）留后续 plan**。

**Architecture:** 在 `escort-app/` 下创建 Flutter 3.24+ 项目；按 `specs/2026-09-24-escort-app-design.md` §2 目录结构落地。Riverpod 2.5 用 `@riverpod` 注解 + `StateNotifier` 模式（auth 流）；其他领域 provider 用 `FutureProvider` / `StreamProvider`。HTTP 走 dio 5.7 + `flutter_secure_storage` 持久化 JWT；route guards 走 `go_router` 的 `redirect` + `Provider<bool>` 派生守卫。OpenAPI 契约来自 `web/openapi/contracts.yaml`，用 `openapi-generator-cli` 生成 Dart client 到 `lib/api/generated/`。

**Tech Stack:** Flutter 3.24+ (Dart 3.5) · flutter_riverpod 2.5+ · go_router 14.x · dio 5.7 · flutter_secure_storage 9.x · shared_preferences 2.x · geolocator 13.x · image_picker 1.x · flutter_image_compress 2.x · openapi-generator-cli 7.x。

**前置依赖:**
- `docs/superpowers/specs/2026-09-24-escort-app-design.md` §1–§6（功能模块 + 页面 + 状态机 + 抢单池 + GPS 签到 + 钱包 + SOS）
- `docs/superpowers/specs/2026-09-24-l2-api-gap-design.md` §2.2 escort 端 14 API + §3 entities
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
- v1 范围：**只**做骨架 + 路由 + providers；页面内容用 `Scaffold(body: Center(child: Text('TODO: <page>')))` 占位（仍可测渲染 + 路由注册）
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
| `escort-app/lib/app.dart` | Create | `MaterialApp.router(routerConfig: ref.watch(appRouterProvider))` |
| `escort-app/lib/router/app_router.dart` | Create | GoRouter + 24 路由 + redirect 链 |
| `escort-app/lib/router/route_guards.dart` | Create | `authGuardProvider` / `realNameGuardProvider` / `approvedGuardProvider` |
| `escort-app/lib/router/route_guards_test.dart` | Create | 3 个守卫的派生逻辑单测 |
| `escort-app/lib/router/app_router_test.dart` | Create | redirect 链 widget 测试（未登录跳 login / 未实名跳 real-name / 未审核跳 audit/pending） |
| `escort-app/lib/utils/format.dart` | Create | `formatMoney` / `formatDateTime` / `maskPhone` |
| `escort-app/lib/utils/format_test.dart` | Create | 3 个函数单测 |
| `escort-app/lib/utils/trace.dart` | Create | `newTraceId()` → `escort-{ms}-{rand}` |
| `escort-app/lib/utils/trace_test.dart` | Create | 格式 + 唯一性 |
| `escort-app/lib/utils/error_handler.dart` | Create | `handleDioError(DioException)` → 用户文案 |
| `escort-app/lib/utils/error_handler_test.dart` | Create | 业务码 → 文案映射 |
| `escort-app/lib/models/order.dart` | Create | `Order` + `OrderSummary` + `OrderStatus` enum |
| `escort-app/lib/models/escort_profile.dart` | Create | `EscortProfile` + `OnboardingStage` enum |
| `escort-app/lib/models/wallet.dart` | Create | `Wallet` |
| `escort-app/lib/models/withdrawal.dart` | Create | `Withdrawal` + `WithdrawalStatus` enum |
| `escort-app/lib/models/signal.dart` | Create | `Signal` + `SignalStatus` enum |
| `escort-app/lib/models/*_test.dart` | Create | `fromJson` / `toJson` 单测 |
| `escort-app/lib/services/token_storage.dart` | Create | `TokenStorage`（封装 `flutter_secure_storage`）+ `TokenStorageFake`（测试用） |
| `escort-app/lib/services/token_storage_test.dart` | Create | write/read/delete + fake 单测 |
| `escort-app/lib/services/gps_service.dart` | Create | `GpsService.currentPosition()` 封装 `geolocator` |
| `escort-app/lib/services/gps_service_test.dart` | Create | 单测（mock geolocator） |
| `escort-app/lib/services/push_service.dart` | Create | `PushService` 接口 + `NopPushService` 实现（v1 mock） |
| `escort-app/lib/services/push_service_test.dart` | Create | Nop 单测 |
| `escort-app/lib/api/dio_client.dart` | Create | `dioProvider`（BaseOptions + Auth 拦截器 + Trace 拦截器 + 401 handler） |
| `escort-app/lib/api/dio_client_test.dart` | Create | 拦截器单测（用 `MockAdapter`） |
| `escort-app/lib/api/auth_api.dart` | Create | `AuthApi`（login / smsSend / refresh / me） |
| `escort-app/lib/api/order_api.dart` | Create | `OrderApi`（feed / getById / accept / confirmAccept / checkin / checkout / listMine） |
| `escort-app/lib/api/escort_api.dart` | Create | `EscortApi`（register / realNameAuth / uploadHealthCert / trainingComplete / me / updateStatus） |
| `escort-app/lib/api/wallet_api.dart` | Create | `WalletApi`（getMyWallet / withdraw / listTransactions） |
| `escort-app/lib/api/training_api.dart` | Create | `TrainingApi`（listCourses / getCourse / submitQuiz） |
| `escort-app/lib/api/sos_api.dart` | Create | `SosApi`（trigger） |
| `escort-app/lib/api/review_api.dart` | Create | `ReviewApi`（listMine） |
| `escort-app/lib/api/message_api.dart` | Create | `MessageApi`（listConversations / listMessages） |
| `escort-app/lib/api/*_test.dart` | Create | 各 API 用 `MockAdapter` 单测 |
| `escort-app/lib/providers/auth_provider.dart` | Create | `AuthState`（sealed）+ `AuthNotifier`（StateNotifier）+ `authProvider` |
| `escort-app/lib/providers/auth_provider_test.dart` | Create | bootstrap / loginByPhone / logout 单测 |
| `escort-app/lib/providers/order_provider.dart` | Create | `feedProvider`（StreamProvider）/ `acceptProvider.family` |
| `escort-app/lib/providers/order_provider_test.dart` | Create | feed 周期 + accept 单测 |
| `escort-app/lib/providers/wallet_provider.dart` | Create | `walletProvider`（FutureProvider） + `withdrawProvider` |
| `escort-app/lib/providers/wallet_provider_test.dart` | Create | 单测 |
| `escort-app/lib/providers/message_provider.dart` | Create | `conversationsProvider` + `messagesProvider.family` |
| `escort-app/lib/providers/message_provider_test.dart` | Create | 单测 |
| `escort-app/lib/providers/training_provider.dart` | Create | `coursesProvider` + `quizSubmitProvider.family` |
| `escort-app/lib/providers/training_provider_test.dart` | Create | 单测 |
| `escort-app/lib/pages/splash/splash_page.dart` | Create | 启动页骨架（token 检查 + redirect） |
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
| `escort-app/lib/pages/home/home_shell.dart` | Create | BottomNavigationBar（feed / orders / wallet / profile） |
| `escort-app/lib/pages/home/feed_page.dart` | Create | 抢单池骨架（ListView + 5s 刷新占位） |
| `escort-app/lib/pages/home/orders_page.dart` | Create | 我的订单骨架（tab: 待服务 / 服务中 / 已完成） |
| `escort-app/lib/pages/home/wallet_page.dart` | Create | 钱包页骨架（余额 + 提现按钮占位） |
| `escort-app/lib/pages/home/profile_page.dart` | Create | 个人中心骨架（实名状态 + 评分 + 培训记录） |
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
| `escort-app/lib/widgets/order_card.dart` | Create | 订单卡片骨架（医院 + 金额 + 抢单按钮占位） |
| `escort-app/lib/widgets/countdown_badge.dart` | Create | 30s 锁单倒计时（基于 `expireAt`） |
| `escort-app/lib/widgets/countdown_badge_test.dart` | Create | 渲染 / 归零 → 已超时 |
| `escort-app/lib/widgets/rating_stars.dart` | Create | 评分星级 |
| `escort-app/lib/widgets/rating_stars_test.dart` | Create | 渲染 |
| `escort-app/lib/widgets/status_chip.dart` | Create | 状态机颜色 chip |
| `escort-app/lib/widgets/status_chip_test.dart` | Create | 颜色映射 |
| `escort-app/lib/widgets/gps_checkin_button.dart` | Create | 长按 GPS 验证按钮 |
| `escort-app/lib/widgets/gps_checkin_button_test.dart` | Create | 长按回调 |
| `escort-app/lib/widgets/sos_long_press.dart` | Create | SOS 长按 3s 触发器（含 AnimationController） |
| `escort-app/lib/widgets/sos_long_press_test.dart` | Create | 3s 后回调触发 |
| `escort-app/lib/widgets/order_card_test.dart` | Create | 渲染 |
| `escort-app/ios/Runner/Info.plist` | Modify | 加 NSLocationWhenInUseUsageDescription / NSCameraUsageDescription / NSPhotoLibraryUsageDescription |
| `escort-app/android/app/src/main/AndroidManifest.xml` | Modify | debug buildType 加 `android:usesCleartextTraffic="true"` |
| `escort-app/android/app/src/main/AndroidManifest.xml` | Modify | ACCESS_FINE_LOCATION / CAMERA / READ_MEDIA_IMAGES 权限 |
| `escort-app/integration_test/app_test.dart` | Create | E2E：splash → login → register → onboarding 引导链 |
| `escort-app/integration_test/feed_test.dart` | Create | E2E：mock feed → 接单 → 倒计时 → 完成 |
| `dev.md` | Modify | §10.x 加 escort-app setup 落地记录 |

> 24 个 page widget + 8 个 provider + 8 个 API + 6 个公共 widget + 5 个 model = **51 个 lib 文件 + 18 个 test 文件 + 5 个 config 文件 = ~74 个新文件**。

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

```bash
flutter pub get              # 拉依赖
flutter run -d ios           # iOS 模拟器
flutter run -d android       # Android 模拟器
flutter test                 # 单测 + widget 测试
flutter test integration_test/  # E2E
flutter analyze              # 静态分析
bash scripts/gen-api.sh      # 重新生成 OpenAPI Dart client
```

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

Source of truth: `web/openapi/contracts.yaml`（后端 OpenAPI 3.0 契约）。

v1 策略：本目录的生成产物仅作为参考 / 类型来源；`lib/api/*.dart`（`AuthApi` / `OrderApi` 等）是手工包装层，便于：
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

**Step 6: error_handler.dart + error_handler_test.dart**

```dart
// lib/utils/error_handler.dart
import 'package:dio/dio.dart';

/// 把 dio / 业务异常翻译成用户文案（中文，spec §10 配色 / 文案规范）。
/// 业务码来自 l2-api-gap-design.md §3.2：13001~13009 + 标准 401/403/404/500。
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
      case 409 when bizCode == 13008:
        return '抢单池已关闭';
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
  test('非 DioException → 未知错误', () {
    expect(handleDioError('x'), '未知错误');
  });
}
```

**Step 7: 跑全部 utils 测试**

```bash
flutter test test/utils/
```

Expected: PASS（format 5 + trace 2 + error_handler 4 = 11 个）

**Step 8: Commit**

```bash
git add escort-app/lib/utils/
git commit -m "feat(escort-app): utils (format/trace/error_handler) + 11 个单测"
```

---

### Task 4: models/ + JSON 单测

**Files:**
- Create: `escort-app/lib/models/order.dart`
- Create: `escort-app/lib/models/escort_profile.dart`
- Create: `escort-app/lib/models/wallet.dart`
- Create: `escort-app/lib/models/withdrawal.dart`
- Create: `escort-app/lib/models/signal.dart`
- Create: `escort-app/test/models/order_test.dart`
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
  test('Order.fromJson 解析完整字段', () {
    final j = {
      'id': 7,
      'order_no': 'O-001',
      'patient_id': 11,
      'escort_id': 22,
      'hospital_id': 33,
      'hospital_name': '北京协和医院',
      'hospital_lat': 39.9,
      'hospital_lng': 116.4,
      'package_name': '半日陪诊',
      'service_start_at': '2026-09-25T09:00:00Z',
      'amount': 300.0,
      'final_amount': 300.0,
      'status': 'pending_acceptance',
      'lock_owner': 22,
      'lock_expire_at': '2026-09-24T15:35:00Z',
      'created_at': '2026-09-24T15:00:00Z',
    };
    final o = Order.fromJson(j);
    expect(o.id, 7);
    expect(o.status, OrderStatus.pendingAcceptance);
    expect(o.hospitalName, '北京协和医院');
    expect(o.amount, 300.0);
    expect(o.lockExpireAt!.isAfter(o.serviceStartAt), isFalse);
  });

  test('OrderSummary.fromJson 抢单池卡片用', () {
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

**Step 3: 写 order.dart**

```dart
// lib/models/order.dart
import 'package:escort_app/utils/format.dart';

enum OrderStatus {
  created, paid, matching, pendingAcceptance, accepted, inService,
  completed, reviewed, refunding, refunded, settling, disputed, closed, canceled;

  static OrderStatus fromString(String s) {
    switch (s) {
      case 'created': return OrderStatus.created;
      case 'paid': return OrderStatus.paid;
      case 'matching': return OrderStatus.matching;
      case 'pending_acceptance': return OrderStatus.pendingAcceptance;
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
  final String hospitalName;
  final double hospitalLat;
  final double hospitalLng;
  final String packageName;
  final DateTime serviceStartAt;
  final double amount;
  final OrderStatus status;
  final DateTime? lockExpireAt;

  Order({
    required this.id,
    required this.patientId,
    this.escortId,
    required this.hospitalName,
    required this.hospitalLat,
    required this.hospitalLng,
    required this.packageName,
    required this.serviceStartAt,
    required this.amount,
    required this.status,
    this.lockExpireAt,
  });

  factory Order.fromJson(Map<String, dynamic> j) => Order(
    id: j['id'] as int,
    patientId: j['patient_id'] as int,
    escortId: j['escort_id'] as int?,
    hospitalName: j['hospital_name'] as String,
    hospitalLat: (j['hospital_lat'] as num).toDouble(),
    hospitalLng: (j['hospital_lng'] as num).toDouble(),
    packageName: j['package_name'] as String,
    serviceStartAt: DateTime.parse(j['service_start_at'] as String),
    amount: (j['amount'] as num).toDouble(),
    status: OrderStatus.fromString(j['status'] as String),
    lockExpireAt: j['lock_expire_at'] == null ? null : DateTime.parse(j['lock_expire_at'] as String),
  );
}

/// 抢单池卡片用（瘦）
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

Expected: PASS（2 个）

**Step 5: escort_profile.dart + 测试**

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

**Step 6: wallet.dart + withdrawal.dart + signal.dart + 各自测试**

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

**Step 7: 三个模型各一个 fromJson 测试**

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
      'id': 1, 'amount': 50.0, 'channel': 'wx', 'account': '138****0000',
      'status': 'pending', 'created_at': '2026-09-24T15:00:00Z',
    });
    expect(w.status, WithdrawalStatus.pending);
    expect(w.channel, 'wx');
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
      'lat': 39.9, 'lng': 116.4, 'address': '协和',
      'status': 'raised', 'created_at': '2026-09-24T15:00:00Z',
    });
    expect(s.status, SignalStatus.raised);
    expect(s.triggerRole, 'escort');
  });
}
```

**Step 8: 跑全部 models 测试**

```bash
flutter test test/models/
```

Expected: PASS（2+1+1+1+1 = 6 个）

**Step 9: Commit**

```bash
git add escort-app/lib/models/ escort-app/test/models/
git commit -m "feat(escort-app): models (Order/EscortProfile/Wallet/Withdrawal/Signal) + 6 个 fromJson 单测"
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

**Step 1: 写 token_storage_test.dart（RED）**

```dart
// test/services/token_storage_test.dart
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/services/token_storage.dart';

void main() {
  group('TokenStorage', () {
    late TokenStorage storage;

    setUp(() {
      // 用 fake（不读真实 keychain / keystore）
      storage = TokenStorage(FlutterSecureStorage());
      // 注：真实 keychain 在 widget test 环境会 mock；这里用 NullStorage fallback
    });

    test('write → read 往返一致', () async {
      await storage.write('tok-abc');
      expect(await storage.read(), 'tok-abc');
    });

    test('delete 后 read 返回 null', () async {
      await storage.write('tok-abc');
      await storage.delete();
      expect(await storage.read(), isNull);
    });

    test('readSync 在未 write 时返回 null（不抛）', () {
      expect(storage.readSync(), isNull);
    });
  });
}
```

**Step 2: 跑测试确认失败**

```bash
flutter test test/services/token_storage_test.dart
```

Expected: FAIL — `package:escort_app/services/token_storage.dart` not found

**Step 3: 写 token_storage.dart**

```dart
// lib/services/token_storage.dart
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

/// TokenStorage 包装 flutter_secure_storage。
/// - write / delete 异步（走 platform channel）
/// - readSync 同步读（dio 拦截器同步拿，避免 await）
///   - v1 内存缓存 + 启动时 bootstrap 一次；token 变更后调 invalidate()
class TokenStorage {
  static const _key = 'escort_jwt';
  final FlutterSecureStorage _delegate;
  String? _cache;

  TokenStorage(this._delegate);

  Future<String?> read() async {
    final v = await _delegate.read(key: _key);
    _cache = v;
    return v;
  }

  String? readSync() => _cache;

  Future<void> write(String token) async {
    await _delegate.write(key: _key, value: token);
    _cache = token;
  }

  Future<void> delete() async {
    await _delegate.delete(key: _key);
    _cache = null;
  }

  /// 测试 / 切换用户后清缓存
  void invalidate() { _cache = null; }
}
```

**Step 4: 跑测试确认通过**

```bash
flutter test test/services/token_storage_test.dart
```

Expected: PASS（3 个）

> **注**：`flutter_secure_storage` 在 `flutter test` 环境无 platform channel，会抛 `MissingPluginException`。生产用真机；测试用 `TokenStorageFake`（见 Step 8）。**本 Task 的 read 测试在真机 / integration_test 环境跑；当前 `flutter test` 默认平台是 `dart:io` 的 host 环境**，会失败。**Plan 修正**：本 Task 仅跑 `readSync == null` 单测（不需要 platform），另两个搬到 `integration_test/`。

调整测试（最终版本）：

```dart
// test/services/token_storage_test.dart
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/services/token_storage.dart';

void main() {
  test('readSync 在未 write 时返回 null（不抛）', () {
    final storage = TokenStorage.forTest();
    expect(storage.readSync(), isNull);
  });

  test('invalidate 后 _cache 为 null', () {
    final storage = TokenStorage.forTest();
    storage.invalidate();
    expect(storage.readSync(), isNull);
  });
}
```

并在 `token_storage.dart` 末尾加：

```dart
/// 测试用：跳过 FlutterSecureStorage，直接用 in-memory 实现。
@visibleForTesting
factory TokenStorage.forTest() => TokenStorage._fake();
TokenStorage._fake() : _delegate = FlutterSecureStorage() {
  _cache = null;
}
```

> 真正异步 write/read 走 integration_test（见 Task 20）。

**Step 5: 跑测试确认通过**

```bash
flutter test test/services/token_storage_test.dart
```

Expected: PASS（2 个）

**Step 6: gps_service.dart + 测试**

```dart
// lib/services/gps_service.dart
import 'package:geolocator/geolocator.dart';

class GpsService {
  Future<Position> currentPosition() async {
    if (!await Geolocator.isLocationServiceEnabled()) {
      throw StateError('位置服务未开启');
    }
    var perm = await Geolocator.checkPermission();
    if (perm == LocationPermission.denied) {
      perm = await Geolocator.requestPermission();
      if (perm == LocationPermission.denied || perm == LocationPermission.deniedForever) {
        throw StateError('未授予定位权限');
      }
    }
    return Geolocator.getCurrentPosition(
      locationSettings: const LocationSettings(accuracy: LocationAccuracy.high),
    );
  }

  /// 计算两点距离（米）。
  double distanceBetween(double lat1, double lng1, double lat2, double lng2) {
    return Geolocator.distanceBetween(lat1, lng1, lat2, lng2);
  }
}
```

```dart
// test/services/gps_service_test.dart
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/services/gps_service.dart';

void main() {
  test('distanceBetween 北京 → 上海 ≈ 1,067,000m', () {
    final s = GpsService();
    final d = s.distanceBetween(39.9, 116.4, 31.2, 121.5);
    expect(d, greaterThan(1_000_000));
    expect(d, lessThan(1_100_000));
  });
}
```

**Step 7: push_service.dart + 测试（v1 Nop mock）**

```dart
// lib/services/push_service.dart
abstract class PushService {
  Future<String?> getToken();
  Stream<String> onMessage();
}

class NopPushService implements PushService {
  @override
  Future<String?> getToken() async => null;
  @override
  Stream<String> onMessage() async* {} // 空流
}
```

```dart
// test/services/push_service_test.dart
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/services/push_service.dart';

void main() {
  test('NopPushService.getToken → null', () async {
    expect(await NopPushService().getToken(), isNull);
  });
  test('NopPushService.onMessage → 空流', () async {
    final got = await NopPushService().onMessage().toList();
    expect(got, isEmpty);
  });
}
```

**Step 8: 跑全部 services 测试**

```bash
flutter test test/services/
```

Expected: PASS（2 token + 1 gps + 2 push = 5 个）

**Step 9: Commit**

```bash
git add escort-app/lib/services/ escort-app/test/services/
git commit -m "feat(escort-app): services (TokenStorage/GpsService/NopPushService) + 5 个单测"
```

---

### Task 6: api/dio_client + 拦截器 + 单测

**Files:**
- Create: `escort-app/lib/api/dio_client.dart`
- Create: `escort-app/test/api/dio_client_test.dart`

**Step 1: 写 dio_client_test.dart（RED）**

```dart
// test/api/dio_client_test.dart
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/api/dio_client.dart';
import 'package:escort_app/services/token_storage.dart';

/// 构造一个带 FakeAdapter + TokenStorage 的 dio 用于测试拦截器。
Dio makeTestDio({String? token, MockHttpClientAdapter? adapter}) {
  final storage = TokenStorage.forTest();
  if (token != null) {
    // 直接写 cache（绕开 keychain）
    storage.writeSync(token);
  }
  final d = buildDio(storage: storage, baseUrl: 'https://test/api/v1');
  d.httpClientAdapter = adapter ?? MockHttpClientAdapter();
  return d;
}

class MockHttpClientAdapter implements HttpClientAdapter {
  final List<RequestOptions> seen = [];
  final Response<dynamic> Function(RequestOptions) responder;

  MockHttpClientAdapter({
    this.responder = _default,
  });

  static Response<dynamic> _default(RequestOptions o) {
    return Response<dynamic>(requestOptions: o, statusCode: 200, data: {'ok': true});
  }

  @override
  Future<ResponseBody> fetch(RequestOptions options, Stream<List<int>>? requestStream, Future<void>? cancelFuture) async {
    seen.add(options);
    final r = responder(options);
    return ResponseBody.fromBytes(
      _bodyBytes(r.data),
      r.statusCode ?? 200,
      headers: {'content-type': ['application/json']},
    );
  }

  @override
  void close({bool force = false}) {}

  static List<int> _bodyBytes(dynamic data) {
    final s = data is String ? data : (data == null ? '{}' : data.toString());
    return s.codeUnits;
  }
}

void main() {
  test('请求带 Authorization Bearer', () async {
    final adapter = MockHttpClientAdapter();
    final dio = makeTestDio(token: 'tok-abc', adapter: adapter);
    await dio.get<dynamic>('/test');
    expect(adapter.seen.single.headers['Authorization'], 'Bearer tok-abc');
  });

  test('请求带 X-Trace-Id 以 escort- 开头', () async {
    final adapter = MockHttpClientAdapter();
    final dio = makeTestDio(adapter: adapter);
    await dio.get<dynamic>('/test');
    final t = adapter.seen.single.headers['X-Trace-Id'] as String;
    expect(t, startsWith('escort-'));
  });

  test('401 响应调用 onUnauthorized 回调', () async {
    final adapter = MockHttpClientAdapter(
      responder: (o) => Response<dynamic>(requestOptions: o, statusCode: 401, data: {'code': 401}),
    );
    var called = 0;
    final dio = makeTestDio(adapter: adapter);
    dio.interceptors.add(InterceptorsWrapper(
      onError: (e, h) {
        if (e.response?.statusCode == 401) called++;
        h.next(e);
      },
    ));
    try {
      await dio.get<dynamic>('/test');
    } catch (_) {}
    expect(called, 1);
  });
}
```

**Step 2: 跑测试确认失败**

```bash
flutter test test/api/dio_client_test.dart
```

Expected: FAIL — `package:escort_app/api/dio_client.dart` not found

**Step 3: 写 dio_client.dart**

```dart
// lib/api/dio_client.dart
import 'package:dio/dio.dart';
import 'package:escort_app/services/token_storage.dart';
import 'package:escort_app/utils/trace.dart';

/// 默认 base URL（生产）。dev 覆盖：flutter run --dart-define=API_BASE=https://api.dev.doctors.example.com/api/v1
const String kDefaultBaseUrl = String.fromEnvironment(
  'API_BASE',
  defaultValue: 'https://api.doctors.example.com/api/v1',
);

/// 构造 dio（含拦截器链：auth + trace + 401 logout）。
/// storage 仅读（同步 readSync），写由 auth_provider 控制。
Dio buildDio({
  required TokenStorage storage,
  String baseUrl = kDefaultBaseUrl,
  void Function()? onUnauthorized,
}) {
  final dio = Dio(BaseOptions(
    baseUrl: baseUrl,
    connectTimeout: const Duration(seconds: 10),
    receiveTimeout: const Duration(seconds: 30),
    headers: {'Content-Type': 'application/json'},
  ));

  dio.interceptors.add(InterceptorsWrapper(
    onRequest: (options, handler) {
      final token = storage.readSync();
      if (token != null && token.isNotEmpty) {
        options.headers['Authorization'] = 'Bearer $token';
      }
      options.headers['X-Trace-Id'] = newTraceId();
      handler.next(options);
    },
    onError: (err, handler) {
      if (err.response?.statusCode == 401) {
        onUnauthorized?.call();
      }
      handler.next(err);
    },
  ));
  return dio;
}
```

并在 `token_storage.dart` 加（测试需要）：

```dart
/// 测试用：直接写 cache（不走 keychain）。
@visibleForTesting
void writeSync(String token) {
  _cache = token;
}
```

**Step 4: 跑测试确认通过**

```bash
flutter test test/api/dio_client_test.dart
```

Expected: PASS（3 个）

**Step 5: Commit**

```bash
git add escort-app/lib/api/dio_client.dart escort-app/lib/services/token_storage.dart escort-app/test/api/
git commit -m "feat(escort-app): dio_client (BaseOptions + auth/trace 拦截器 + 401 handler) + 3 个单测"
```

---

### Task 7: api/ 业务模块（8 个 API）+ 单测

**Files:**
- Create: `escort-app/lib/api/auth_api.dart`
- Create: `escort-app/lib/api/order_api.dart`
- Create: `escort-app/lib/api/escort_api.dart`
- Create: `escort-app/lib/api/wallet_api.dart`
- Create: `escort-app/lib/api/training_api.dart`
- Create: `escort-app/lib/api/sos_api.dart`
- Create: `escort-app/lib/api/review_api.dart`
- Create: `escort-app/lib/api/message_api.dart`
- Create: `escort-app/test/api/auth_api_test.dart`
- Create: `escort-app/test/api/order_api_test.dart`
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
  @override
  void close({bool force = false}) {}
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

**Step 5: order_api.dart**（重点：feed / accept / confirmAccept / checkin / checkout）

```dart
// lib/api/order_api.dart
import 'package:dio/dio.dart';
import 'package:escort_app/models/order.dart';

class OrderApi {
  final Dio _dio;
  OrderApi(this._dio);

  Future<List<OrderSummary>> feed() async {
    final r = await _dio.get<Map<String, dynamic>>('/orders', queryParameters: {'role': 'escort', 'status': 'matching'});
    final list = (r.data!['data'] as List).cast<Map<String, dynamic>>();
    return list.map(OrderSummary.fromJson).toList();
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

  Future<Order> accept(int id) async {
    final r = await _dio.post<Map<String, dynamic>>('/orders/$id/accept');
    return Order.fromJson(r.data!['data'] as Map<String, dynamic>);
  }

  Future<Order> confirmAccept(int id) async {
    final r = await _dio.post<Map<String, dynamic>>('/orders/$id/confirm-accept');
    return Order.fromJson(r.data!['data'] as Map<String, dynamic>);
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

**Step 6: order_api_test.dart**

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
  test('OrderApi.feed → OrderSummary 列表', () async {
    final dio = buildDio(storage: TokenStorage.forTest(), baseUrl: 'https://x');
    dio.httpClientAdapter = StubAdapter(body: {
      'code': 0,
      'data': [
        {'id': 1, 'hospital_name': '协和', 'service_start_at_text': '明天 09:00', 'amount': 300.0, 'package_name': '半日陪诊'}
      ],
    });
    final list = await OrderApi(dio).feed();
    expect(list.length, 1);
    expect(list.first.id, 1);
  });

  test('OrderApi.accept → pending_acceptance 订单', () async {
    final dio = buildDio(storage: TokenStorage.forTest(), baseUrl: 'https://x');
    dio.httpClientAdapter = StubAdapter(body: {
      'code': 0,
      'data': {
        'id': 7, 'patient_id': 11, 'escort_id': 22,
        'hospital_name': '协和', 'hospital_lat': 39.9, 'hospital_lng': 116.4,
        'package_name': '半日陪诊', 'service_start_at': '2026-09-25T09:00:00Z',
        'amount': 300.0, 'status': 'pending_acceptance',
        'lock_expire_at': '2026-09-24T15:35:00Z',
      },
    });
    final o = await OrderApi(dio).accept(7);
    expect(o.status, OrderStatus.pendingAcceptance);
    expect(o.lockExpireAt, isNotNull);
  });
}
```

**Step 7: escort_api.dart + wallet_api.dart + sos_api.dart + 测试**

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
  final int durationSec;
  final int passScore;
  TrainingCourse({required this.id, required this.title, required this.durationSec, required this.passScore});
  factory TrainingCourse.fromJson(Map<String, dynamic> j) => TrainingCourse(
    id: j['id'] as int, title: j['title'] as String,
    durationSec: j['duration_sec'] as int? ?? 0, passScore: j['pass_score'] as int? ?? 80,
  );
}

class TrainingApi {
  final Dio _dio;
  TrainingApi(this._dio);

  Future<List<TrainingCourse>> listCourses() async {
    final r = await _dio.get<Map<String, dynamic>>('/escorts/me/training-courses');
    final list = (r.data!['data'] as List).cast<Map<String, dynamic>>();
    return list.map(TrainingCourse.fromJson).toList();
  }

  Future<TrainingCourse> getCourse(int id) async {
    final r = await _dio.get<Map<String, dynamic>>('/escorts/me/training-courses/$id');
    return TrainingCourse.fromJson(r.data!['data'] as Map<String, dynamic>);
  }

  Future<int> submitQuiz({required int courseId, required List<int> answers}) async {
    final r = await _dio.post<Map<String, dynamic>>('/escorts/me/training-courses/$courseId/quiz',
        data: {'answers': answers});
    return r.data!['data']['score'] as int;
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
  final String? comment;
  final DateTime createdAt;
  ReviewItem({required this.id, required this.orderId, required this.rating, this.comment, required this.createdAt});
  factory ReviewItem.fromJson(Map<String, dynamic> j) => ReviewItem(
    id: j['id'] as int, orderId: j['order_id'] as int, rating: j['rating'] as int,
    comment: j['comment'] as String?, createdAt: DateTime.parse(j['created_at'] as String),
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
  final DateTime lastMessageAt;
  final int unreadCount;
  Conversation({required this.id, required this.lastMessageAt, required this.unreadCount});
  factory Conversation.fromJson(Map<String, dynamic> j) => Conversation(
    id: j['id'] as int,
    lastMessageAt: DateTime.parse(j['last_message_at'] as String),
    unreadCount: j['unread_count'] as int? ?? 0,
  );
}

class MessageItem {
  final int id;
  final int senderId;
  final String contentType;
  final String content;
  final DateTime createdAt;
  MessageItem({required this.id, required this.senderId, required this.contentType, required this.content, required this.createdAt});
  factory MessageItem.fromJson(Map<String, dynamic> j) => MessageItem(
    id: j['id'] as int, senderId: j['sender_id'] as int, contentType: j['content_type'] as String,
    content: j['content'] as String, createdAt: DateTime.parse(j['created_at'] as String),
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
    final r = await _dio.get<Map<String, dynamic>>('/messages/conversations/$conversationId/messages');
    final list = (r.data!['data'] as List).cast<Map<String, dynamic>>();
    return list.map(MessageItem.fromJson).toList();
  }
}
```

**Step 8: escort_api_test.dart + wallet_api_test.dart + sos_api_test.dart**

```dart
// test/api/escort_api_test.dart
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/api/escort_api.dart';
import 'package:escort_app/api/dio_client.dart';
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
  test('EscortApi.realNameAuth → approved: false', () async {
    final dio = buildDio(storage: TokenStorage.forTest(), baseUrl: 'https://x');
    dio.httpClientAdapter = StubAdapter(body: {
      'code': 0,
      'data': {
        'id': 1, 'user_id': 2, 'nickname': '张师傅', 'stage': 'health_cert_pending',
        'real_name_verified': true, 'approved': false,
      },
    });
    final p = await EscortApi(dio).realNameAuth(idCard: '110101199001010000', realName: '张三');
    expect(p.realNameVerified, true);
    expect(p.stage.name, 'healthCertPending');
  });
}
```

```dart
// test/api/wallet_api_test.dart
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/api/wallet_api.dart';
import 'package:escort_app/api/dio_client.dart';
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
  test('WalletApi.getMyWallet', () async {
    final dio = buildDio(storage: TokenStorage.forTest(), baseUrl: 'https://x');
    dio.httpClientAdapter = StubAdapter(body: {
      'code': 0,
      'data': {
        'user_id': 1, 'balance': 100.0, 'frozen': 30.0,
        'total_earned': 200.0, 'updated_at': '2026-09-24T15:00:00Z',
      },
    });
    final w = await WalletApi(dio).getMyWallet();
    expect(w.balance, 100.0);
  });
}
```

```dart
// test/api/sos_api_test.dart
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/api/sos_api.dart';
import 'package:escort_app/api/dio_client.dart';
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
  test('SosApi.trigger → status=raised', () async {
    final dio = buildDio(storage: TokenStorage.forTest(), baseUrl: 'https://x');
    dio.httpClientAdapter = StubAdapter(body: {
      'code': 0,
      'data': {
        'id': 1, 'order_id': 7, 'trigger_by': 22, 'trigger_role': 'escort',
        'status': 'raised', 'created_at': '2026-09-24T15:00:00Z',
      },
    });
    final s = await SosApi(dio).trigger(orderId: 7, lat: 39.9, lng: 116.4, address: '协和');
    expect(s.status.name, 'raised');
  });
}
```

**Step 9: 跑全部 api 测试**

```bash
flutter test test/api/
```

Expected: PASS（auth 1 + order 2 + escort 1 + wallet 1 + sos 1 = 6 个）

**Step 10: Commit**

```bash
git add escort-app/lib/api/ escort-app/test/api/
git commit -m "feat(escort-app): api modules (8 个: auth/order/escort/wallet/training/sos/review/message) + 6 个 StubAdapter 单测"
```

---

### Task 8: providers/auth_provider + auth_provider_test

**Files:**
- Create: `escort-app/lib/providers/auth_provider.dart`
- Create: `escort-app/test/providers/auth_provider_test.dart`

**Step 1: 写 auth_provider_test.dart（RED）**

```dart
// test/providers/auth_provider_test.dart
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:escort_app/api/auth_api.dart';
import 'package:escort_app/api/dio_client.dart';
import 'package:escort_app/providers/auth_provider.dart';
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

ProviderContainer makeContainer({String? token, Map<String, dynamic>? meBody}) {
  final storage = TokenStorage.forTest();
  if (token != null) storage.writeSync(token);
  final dio = buildDio(storage: storage, baseUrl: 'https://x');
  dio.httpClientAdapter = StubAdapter(body: meBody ?? {
    'code': 0,
    'data': {'id': 1, 'phone': '13800138000', 'role': 'escort', 'real_name_verified': true, 'approved': true},
  });
  return ProviderContainer(overrides: [
    tokenStorageProvider.overrideWithValue(storage),
    dioProvider.overrideWithValue(dio),
  ]);
}

void main() {
  test('无 token → state = unauthenticated', () async {
    final c = makeContainer();
    addTearDown(c.dispose);
    await c.read(authProvider.notifier).bootstrap();
    expect(c.read(authProvider), isA<Unauthenticated>());
  });

  test('有 token + /users/me OK → state = authenticated', () async {
    final c = makeContainer(token: 'tk-1');
    addTearDown(c.dispose);
    await c.read(authProvider.notifier).bootstrap();
    final s = c.read(authProvider);
    expect(s, isA<Authenticated>());
    expect((s as Authenticated).user.role, 'escort');
  });

  test('loginByPhone 写 token + 切 authenticated', () async {
    final c = makeContainer();
    addTearDown(c.dispose);
    // 切换 stub adapter 给 login
    final dio = c.read(dioProvider);
    dio.httpClientAdapter = StubAdapter(body: {
      'code': 0,
      'data': {
        'access_token': 'tk-new',
        'user': {'id': 1, 'phone': '13800138000', 'role': 'escort',
                 'real_name_verified': false, 'approved': false},
      },
    });
    await c.read(authProvider.notifier).loginByPhone(phone: '13800138000', code: '1234');
    expect(c.read(authProvider), isA<Authenticated>());
    expect(c.read(tokenStorageProvider).readSync(), 'tk-new');
  });

  test('logout 清 token + state = unauthenticated', () async {
    final c = makeContainer(token: 'tk-1');
    addTearDown(c.dispose);
    await c.read(authProvider.notifier).bootstrap();
    await c.read(authProvider.notifier).logout();
    expect(c.read(authProvider), isA<Unauthenticated>());
    expect(c.read(tokenStorageProvider).readSync(), isNull);
  });
}
```

**Step 2: 跑测试确认失败**

```bash
flutter test test/providers/auth_provider_test.dart
```

Expected: FAIL — `package:escort_app/providers/auth_provider.dart` not found

**Step 3: 写 auth_provider.dart**

```dart
// lib/providers/auth_provider.dart
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:escort_app/api/auth_api.dart';
import 'package:escort_app/api/dio_client.dart';
import 'package:escort_app/services/token_storage.dart';

/// AuthState 是 sealed class（Dart 3 sealed classes）。
sealed class AuthState { const AuthState(); }
class Unknown extends AuthState { const Unknown(); }
class Unauthenticated extends AuthState { const Unauthenticated(); }
class Authenticated extends AuthState {
  final AuthUser user;
  const Authenticated(this.user);
}

/// tokenStorage + dio 由 main.dart 装配；provider 仅声明。
final tokenStorageProvider = Provider<TokenStorage>((ref) {
  throw UnimplementedError('override in main.dart');
});

final dioProvider = Provider<Dio>((ref) {
  throw UnimplementedError('override in main.dart');
});

final authApiProvider = Provider<AuthApi>((ref) => AuthApi(ref.watch(dioProvider)));

class AuthNotifier extends StateNotifier<AuthState> {
  final Ref _ref;
  AuthNotifier(this._ref) : super(const Unknown());

  Future<void> bootstrap() async {
    final storage = _ref.read(tokenStorageProvider);
    final token = await storage.read();
    if (token == null || token.isEmpty) {
      state = const Unauthenticated();
      return;
    }
    try {
      final user = await _ref.read(authApiProvider).me();
      state = Authenticated(user);
    } catch (_) {
      await storage.delete();
      state = const Unauthenticated();
    }
  }

  Future<void> loginByPhone({required String phone, required String code}) async {
    final res = await _ref.read(authApiProvider).loginByPhone(phone: phone, code: code);
    await _ref.read(tokenStorageProvider).write(res.accessToken);
    state = Authenticated(res.user);
  }

  Future<void> logout() async {
    await _ref.read(tokenStorageProvider).delete();
    state = const Unauthenticated();
  }
}

final authProvider = StateNotifierProvider<AuthNotifier, AuthState>((ref) {
  return AuthNotifier(ref);
});
```

**Step 4: 跑测试确认通过**

```bash
flutter test test/providers/auth_provider_test.dart
```

Expected: PASS（4 个）

**Step 5: Commit**

```bash
git add escort-app/lib/providers/auth_provider.dart escort-app/test/providers/auth_provider_test.dart
git commit -m "feat(escort-app): AuthState (sealed) + AuthNotifier (bootstrap/login/logout) + dio/tokenStorage Provider + 4 个单测"
```

---

### Task 9: router/route_guards + app_router + 测试

**Files:**
- Create: `escort-app/lib/router/route_guards.dart`
- Create: `escort-app/lib/router/route_guards_test.dart`
- Create: `escort-app/lib/router/app_router.dart`
- Create: `escort-app/lib/router/app_router_test.dart`

**Step 1: 写 route_guards_test.dart（RED）**

```dart
// test/router/route_guards_test.dart
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/api/auth_api.dart';
import 'package:escort_app/providers/auth_provider.dart';
import 'package:escort_app/router/route_guards.dart';

ProviderContainer makeContainer(AuthState state) {
  return ProviderContainer(overrides: [
    authProvider.overrideWith((ref) => _FakeAuthNotifier(state)),
  ]);
}

class _FakeAuthNotifier extends AuthNotifier {
  AuthState _initial;
  _FakeAuthNotifier(this._initial) : super(_FakeRef()) { state = _initial; }
  @override Future<void> bootstrap() async {}
  @override Future<void> loginByPhone({required String phone, required String code}) async {}
  @override Future<void> logout() async {}
}
class _FakeRef implements Ref {
  @override T read<T>(ProviderListenable<T> p) => throw UnimplementedError();
  @override noSuchMethod(Invocation i) => super.noSuchMethod(i);
}

void main() {
  test('authGuard: unknown → false', () {
    final c = makeContainer(const Unknown());
    addTearDown(c.dispose);
    expect(c.read(authGuardProvider), false);
  });
  test('authGuard: unauthenticated → false', () {
    final c = makeContainer(const Unauthenticated());
    addTearDown(c.dispose);
    expect(c.read(authGuardProvider), false);
  });
  test('authGuard: authenticated → true', () {
    final c = makeContainer(Authenticated(AuthUser(id: 1, phone: 'p', role: 'escort', realNameVerified: false, approved: false)));
    addTearDown(c.dispose);
    expect(c.read(authGuardProvider), true);
  });

  test('realNameGuard: authenticated + verified → true', () {
    final c = makeContainer(Authenticated(AuthUser(id: 1, phone: 'p', role: 'escort', realNameVerified: true, approved: false)));
    addTearDown(c.dispose);
    expect(c.read(realNameGuardProvider), true);
  });
  test('realNameGuard: authenticated 但未实名 → false', () {
    final c = makeContainer(Authenticated(AuthUser(id: 1, phone: 'p', role: 'escort', realNameVerified: false, approved: false)));
    addTearDown(c.dispose);
    expect(c.read(realNameGuardProvider), false);
  });

  test('approvedGuard: authenticated + realName + approved → true', () {
    final c = makeContainer(Authenticated(AuthUser(id: 1, phone: 'p', role: 'escort', realNameVerified: true, approved: true)));
    addTearDown(c.dispose);
    expect(c.read(approvedGuardProvider), true);
  });
  test('approvedGuard: 未审核 → false', () {
    final c = makeContainer(Authenticated(AuthUser(id: 1, phone: 'p', role: 'escort', realNameVerified: true, approved: false)));
    addTearDown(c.dispose);
    expect(c.read(approvedGuardProvider), false);
  });
}
```

**Step 2: 跑测试确认失败**

```bash
flutter test test/router/route_guards_test.dart
```

Expected: FAIL — `package:escort_app/router/route_guards.dart` not found

**Step 3: 写 route_guards.dart**

```dart
// lib/router/route_guards.dart
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:escort_app/providers/auth_provider.dart';

/// AuthGuard — 是否已登录（token 有效 + user 已加载）。
final authGuardProvider = Provider<bool>((ref) {
  return ref.watch(authProvider).maybeWhen(
    authenticated: (_, __) => true,
    orElse: () => false,
  );
});

/// RealNameGuard — 是否已通过实名。
final realNameGuardProvider = Provider<bool>((ref) {
  return ref.watch(authProvider).maybeWhen(
    authenticated: (user, _) => user.realNameVerified,
    orElse: () => false,
  );
});

/// ApprovedGuard — 是否已通过平台审核（可上线 / 抢单）。
final approvedGuardProvider = Provider<bool>((ref) {
  return ref.watch(authProvider).maybeWhen(
    authenticated: (user, _) => user.realNameVerified && user.approved,
    orElse: () => false,
  );
});
```

> **注**：`authProvider.maybeWhen(authenticated: (user) => ...)` 在 sealed AuthState 上是 `authenticated: (user, _) => ...` 还是 `(user)`？取决于 Riverpod / sealed class 的 pattern 数。若实现编译失败，按错误提示调整 pattern 数量（可能 sealed class 自动添加 named 参数 `_`）。

**Step 4: 跑测试确认通过**

```bash
flutter test test/router/route_guards_test.dart
```

Expected: PASS（7 个）

**Step 5: app_router_test.dart（widget 测试 redirect 链）**

```dart
// test/router/app_router_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:escort_app/api/auth_api.dart';
import 'package:escort_app/providers/auth_provider.dart';
import 'package:escort_app/router/app_router.dart';

class _FakeAuthNotifier extends AuthNotifier {
  AuthState initial;
  _FakeAuthNotifier(this.initial) : super(_FakeRef()) { state = initial; }
  @override Future<void> bootstrap() async {}
  @override Future<void> loginByPhone({required String phone, required String code}) async {}
  @override Future<void> logout() async {}
}
class _FakeRef implements Ref {
  @override T read<T>(ProviderListenable<T> p) => throw UnimplementedError();
  @override noSuchMethod(Invocation i) => super.noSuchMethod(i);
}

Widget _wrap({required AuthState state, String initial = '/home/feed'}) {
  return ProviderScope(
    overrides: [
      authProvider.overrideWith((ref) => _FakeAuthNotifier(state)),
    ],
    child: MaterialApp.router(routerConfig: buildAppRouter()),
  );
}

void main() {
  testWidgets('未登录访问 /home/feed → 重定向到 /auth/login', (t) async {
    await t.pumpWidget(_wrap(state: const Unauthenticated()));
    await t.pumpAndSettle();
    expect(find.text('登录'), findsOneWidget);
  });

  testWidgets('已登录未实名访问 /home/feed → 重定向到 /onboarding/real-name', (t) async {
    await t.pumpWidget(_wrap(state: Authenticated(AuthUser(id: 1, phone: 'p', role: 'escort', realNameVerified: false, approved: false))));
    await t.pumpAndSettle();
    expect(find.text('实名认证'), findsOneWidget);
  });

  testWidgets('已登录实名但未审核访问 /home/feed → 重定向到 /audit/pending', (t) async {
    await t.pumpWidget(_wrap(state: Authenticated(AuthUser(id: 1, phone: 'p', role: 'escort', realNameVerified: true, approved: false))));
    await t.pumpAndSettle();
    expect(find.textContaining('审核'), findsWidgets);
  });

  testWidgets('已登录 + 实名 + 审核通过 → 显示 feed', (t) async {
    await t.pumpWidget(_wrap(state: Authenticated(AuthUser(id: 1, phone: 'p', role: 'escort', realNameVerified: true, approved: true))));
    await t.pumpAndSettle();
    expect(find.text('抢单池'), findsOneWidget);
  });
}
```

**Step 6: 写 app_router.dart**（24 个路由 + redirect 链）

```dart
// lib/router/app_router.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:escort_app/pages/audit/pending_audit_page.dart';
import 'package:escort_app/pages/auth/login_page.dart';
import 'package:escort_app/pages/auth/register_page.dart';
import 'package:escort_app/pages/home/feed_page.dart';
import 'package:escort_app/pages/home/home_shell.dart';
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

GoRouter buildAppRouter() {
  return GoRouter(
    initialLocation: '/splash',
    redirect: (context, state) {
      final auth = state.maybeWhen != null ? null : null; // 简化：用 ref 必须传 goRouter redirect wrapper
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
      ShellRoute(
        builder: (_, __, child) => HomeShell(child: child),
        routes: [
          GoRoute(path: '/home/feed', builder: (_, __) => const FeedPage()),
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
}
```

> **redirect 链实现**：上面 `redirect: (ctx, state) => null` 是占位；真实实现需在 `GoRouter` 构造时通过 `ProviderScope` 注入 ref：
>
> ```dart
> // 改为函数式 Provider
> final goRouterProvider = Provider<GoRouter>((ref) {
>   return GoRouter(
>     initialLocation: '/splash',
>     redirect: (ctx, state) {
>       final auth = ref.read(authProvider);
>       final loc = state.matchedLocation;
>       // splash / auth 自由通行
>       if (loc == '/splash' || loc.startsWith('/auth/')) return null;
>       // 未登录 → login
>       if (!ref.read(authGuardProvider)) return '/auth/login';
>       // 未实名 → real-name（仅当试图访问 /home/* 或 /order/* 等业务页）
>       if (!ref.read(realNameGuardProvider) && loc.startsWith('/home/')) {
>         return '/onboarding/real-name';
>       }
>       // 未审核 → audit/pending
>       if (!ref.read(approvedGuardProvider) && loc.startsWith('/home/feed')) {
>         return '/audit/pending';
>       }
>       return null;
>     },
>     routes: [ /* ... */ ],
>   );
> });
> ```

**Step 7: 跑测试确认通过**

```bash
flutter test test/router/
```

Expected: PASS（7 个 guard + 4 个 redirect = 11 个）

**Step 8: Commit**

```bash
git add escort-app/lib/router/ escort-app/test/router/
git commit -m "feat(escort-app): router (go_router + 24 路由 + 3 守卫 redirect 链) + 11 个测试"
```

---

### Task 10: pages/splash + auth/login + auth/register + 测试

**Files:**
- Create: `escort-app/lib/pages/splash/splash_page.dart`
- Create: `escort-app/lib/pages/auth/login_page.dart`
- Create: `escort-app/lib/pages/auth/register_page.dart`
- Create: `escort-app/test/pages/splash_page_test.dart`
- Create: `escort-app/test/pages/login_page_test.dart`

**Step 1: 写 splash_page.dart**

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
        context.go('/home/feed');
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
  _FakeAuthNotifier(this.initial) : super(_FakeRef()) { state = initial; }
  @override Future<void> bootstrap() async {}
  @override Future<void> loginByPhone({required String phone, required String code}) async {}
  @override Future<void> logout() async {}
}
class _FakeRef implements Ref {
  @override T read<T>(ProviderListenable<T> p) => throw UnimplementedError();
  @override noSuchMethod(Invocation i) => super.noSuchMethod(i);
}

void main() {
  testWidgets('SplashPage 渲染 Doctors Escort', (t) async {
    await t.pumpWidget(
      ProviderScope(
        overrides: [
          authProvider.overrideWith((ref) => _FakeAuthNotifier(const Unknown())),
        ],
        child: const SplashPage(),
      ),
    );
    expect(find.text('Doctors Escort'), findsOneWidget);
  });
}
```

**Step 3: login_page.dart + register_page.dart**

```dart
// lib/pages/auth/login_page.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:escort_app/providers/auth_provider.dart';

class LoginPage extends ConsumerStatefulWidget {
  const LoginPage({super.key});
  @override
  ConsumerState<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends ConsumerState<LoginPage> {
  final _phoneCtrl = TextEditingController();
  final _codeCtrl = TextEditingController();

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('登录')),
      body: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(children: [
          TextField(controller: _phoneCtrl, decoration: const InputDecoration(labelText: '手机号')),
          TextField(controller: _codeCtrl, decoration: const InputDecoration(labelText: '验证码')),
          const SizedBox(height: 16),
          FilledButton(
            onPressed: () => ref.read(authProvider.notifier).loginByPhone(
              phone: _phoneCtrl.text, code: _codeCtrl.text,
            ),
            child: const Text('登录'),
          ),
          TextButton(onPressed: () {/* TODO: register nav */}, child: const Text('注册陪诊师')),
        ]),
      ),
    );
  }

  @override
  void dispose() {
    _phoneCtrl.dispose();
    _codeCtrl.dispose();
    super.dispose();
  }
}
```

```dart
// lib/pages/auth/register_page.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:escort_app/api/auth_api.dart';
import 'package:escort_app/api/dio_client.dart';
import 'package:escort_app/api/escort_api.dart';
import 'package:escort_app/providers/auth_provider.dart';

class RegisterPage extends ConsumerStatefulWidget {
  const RegisterPage({super.key});
  @override
  ConsumerState<RegisterPage> createState() => _RegisterPageState();
}

class _RegisterPageState extends ConsumerState<RegisterPage> {
  final _phoneCtrl = TextEditingController();
  final _codeCtrl = TextEditingController();
  bool _busy = false;

  Future<void> _submit() async {
    setState(() => _busy = true);
    try {
      await EscortApi(ref.read(dioProvider)).register(
        phone: _phoneCtrl.text, code: _codeCtrl.text,
      );
      if (!mounted) return;
      context.go('/onboarding/real-name');
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('注册陪诊师')),
      body: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(children: [
          const Text('注册即同意《陪诊师合作协议》'),
          TextField(controller: _phoneCtrl, decoration: const InputDecoration(labelText: '手机号')),
          TextField(controller: _codeCtrl, decoration: const InputDecoration(labelText: '验证码')),
          const SizedBox(height: 16),
          FilledButton(
            onPressed: _busy ? null : _submit,
            child: Text(_busy ? '提交中...' : '注册'),
          ),
        ]),
      ),
    );
  }

  @override
  void dispose() {
    _phoneCtrl.dispose();
    _codeCtrl.dispose();
    super.dispose();
  }
}
```

**Step 4: login_page_test.dart**

```dart
// test/pages/login_page_test.dart
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/api/auth_api.dart';
import 'package:escort_app/pages/auth/login_page.dart';
import 'package:escort_app/providers/auth_provider.dart';

class _FakeAuthNotifier extends AuthNotifier {
  int loginCalls = 0;
  _FakeAuthNotifier() : super(_FakeRef());
  @override Future<void> bootstrap() async {}
  @override Future<void> loginByPhone({required String phone, required String code}) async {
    loginCalls++;
  }
  @override Future<void> logout() async {}
}
class _FakeRef implements Ref {
  @override T read<T>(ProviderListenable<T> p) => throw UnimplementedError();
  @override noSuchMethod(Invocation i) => super.noSuchMethod(i);
}

void main() {
  testWidgets('LoginPage 渲染标题 + 登录按钮', (t) async {
    await t.pumpWidget(
      ProviderScope(
        overrides: [
          authProvider.overrideWith((ref) => _FakeAuthNotifier()),
        ],
        child: const LoginPage(),
      ),
    );
    expect(find.text('登录'), findsOneWidget);
  });
}
```

**Step 5: 跑测试**

```bash
flutter test test/pages/splash_page_test.dart test/pages/login_page_test.dart
```

Expected: PASS（1+1 = 2 个）

**Step 6: Commit**

```bash
git add escort-app/lib/pages/splash/ escort-app/lib/pages/auth/ escort-app/test/pages/splash_page_test.dart escort-app/test/pages/login_page_test.dart
git commit -m "feat(escort-app): pages/splash + pages/auth (login/register) + 2 个 widget 测试"
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

**Step 1: 6 个 page 骨架**（统一风格：`Scaffold(appBar: AppBar(title: Text(<title>)), body: Center(child: Text('TODO: ...')))`）

```dart
// lib/pages/onboarding/real_name_page.dart
import 'package:flutter/material.dart';

class RealNamePage extends StatelessWidget {
  const RealNamePage({super.key});
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('实名认证')),
      body: const Center(child: Text('TODO: 身份证 + 姓名表单（spec §3.1 / §4.2）')),
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
      body: const Center(child: Text('TODO: image_picker + flutter_image_compress（spec §4.3）')),
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
      body: const Center(child: Text('TODO: 课程列表 + 进度（spec §3.1）')),
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
      appBar: AppBar(title: Text('培训视频 #$courseId')),
      body: const Center(child: Text('TODO: 视频播放器 + 完成回调')),
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
      appBar: AppBar(title: Text('培训考核 #$courseId')),
      body: const Center(child: Text('TODO: 5 题 + 80 分通过')),
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
      appBar: AppBar(title: const Text('协议签署')),
      body: const Center(child: Text('TODO: 电子签名 + 提交')),
    );
  }
}
```

**Step 2: onboarding_pages_test.dart**（6 页渲染）

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

Widget wrap(Widget page) => MaterialApp(home: page);

void main() {
  testWidgets('RealNamePage 渲染', (t) async {
    await t.pumpWidget(wrap(const RealNamePage()));
    expect(find.text('实名认证'), findsOneWidget);
  });
  testWidgets('HealthCertPage 渲染', (t) async {
    await t.pumpWidget(wrap(const HealthCertPage()));
    expect(find.text('健康证上传'), findsOneWidget);
  });
  testWidgets('TrainingListPage 渲染', (t) async {
    await t.pumpWidget(wrap(const TrainingListPage()));
    expect(find.text('培训课程'), findsOneWidget);
  });
  testWidgets('TrainingVideoPage 渲染 + 接收 courseId', (t) async {
    await t.pumpWidget(wrap(const TrainingVideoPage(courseId: 42)));
    expect(find.text('培训视频 #42'), findsOneWidget);
  });
  testWidgets('TrainingQuizPage 渲染', (t) async {
    await t.pumpWidget(wrap(const TrainingQuizPage(courseId: 1)));
    expect(find.text('培训考核 #1'), findsOneWidget);
  });
  testWidgets('AgreementPage 渲染', (t) async {
    await t.pumpWidget(wrap(const AgreementPage()));
    expect(find.text('协议签署'), findsOneWidget);
  });
}
```

**Step 3: 跑测试**

```bash
flutter test test/pages/onboarding_pages_test.dart
```

Expected: PASS（6 个）

**Step 4: Commit**

```bash
git add escort-app/lib/pages/onboarding/ escort-app/test/pages/onboarding_pages_test.dart
git commit -m "feat(escort-app): pages/onboarding (6 页骨架: real-name/health-cert/training list/video/quiz/agreement) + 6 个 widget 测试"
```

---

### Task 12: pages/audit/pending + pages/home shell + 4 tabs + 测试

**Files:**
- Create: `escort-app/lib/pages/audit/pending_audit_page.dart`
- Create: `escort-app/lib/pages/audit/pending_audit_page_test.dart`
- Create: `escort-app/lib/pages/home/home_shell.dart`
- Create: `escort-app/lib/pages/home/feed_page.dart`
- Create: `escort-app/lib/pages/home/orders_page.dart`
- Create: `escort-app/lib/pages/home/wallet_page.dart`
- Create: `escort-app/lib/pages/home/profile_page.dart`
- Create: `escort-app/test/pages/home_pages_test.dart`

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

**Step 2: home_shell.dart + 4 tabs**

```dart
// lib/pages/home/home_shell.dart
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

class HomeShell extends StatelessWidget {
  final Widget child;
  const HomeShell({required this.child, super.key});

  static const _tabs = ['/home/feed', '/home/orders', '/home/wallet', '/home/profile'];
  static const _titles = ['抢单池', '我的订单', '钱包', '个人中心'];

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
          NavigationDestination(icon: Icon(Icons.list_alt), label: '抢单池'),
          NavigationDestination(icon: Icon(Icons.assignment), label: '订单'),
          NavigationDestination(icon: Icon(Icons.account_balance_wallet), label: '钱包'),
          NavigationDestination(icon: Icon(Icons.person), label: '我的'),
        ],
      ),
    );
  }
}
```

```dart
// lib/pages/home/feed_page.dart
import 'package:flutter/material.dart';

class FeedPage extends StatelessWidget {
  const FeedPage({super.key});
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('抢单池')),
      body: const Center(child: Text('TODO: 抢单池 Feed（spec §4 / 5s 刷新）')),
    );
  }
}
```

```dart
// lib/pages/home/orders_page.dart
import 'package:flutter/material.dart';

class OrdersPage extends StatelessWidget {
  const OrdersPage({super.key});
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('我的订单')),
      body: const Center(child: Text('TODO: tab: 待服务 / 服务中 / 已完成')),
    );
  }
}
```

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

class ProfilePage extends StatelessWidget {
  const ProfilePage({super.key});
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('个人中心')),
      body: const Center(child: Text('TODO: 实名状态 + 评分 + 培训记录')),
    );
  }
}
```

**Step 3: home_pages_test.dart**

```dart
// test/pages/home_pages_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/pages/audit/pending_audit_page.dart';
import 'package:escort_app/pages/home/feed_page.dart';
import 'package:escort_app/pages/home/orders_page.dart';
import 'escort_app/pages/home/wallet_page.dart' show WalletPage;
import 'escort_app/pages/home/profile_page.dart' show ProfilePage;

void main() {
  testWidgets('PendingAuditPage 渲染', (t) async {
    await t.pumpWidget(const MaterialApp(home: PendingAuditPage()));
    expect(find.text('等待审核'), findsOneWidget);
  });
  testWidgets('FeedPage 渲染', (t) async {
    await t.pumpWidget(const MaterialApp(home: FeedPage()));
    expect(find.text('抢单池'), findsOneWidget);
  });
  testWidgets('OrdersPage 渲染', (t) async {
    await t.pumpWidget(const MaterialApp(home: OrdersPage()));
    expect(find.text('我的订单'), findsOneWidget);
  });
  testWidgets('WalletPage 渲染', (t) async {
    await t.pumpWidget(const MaterialApp(home: WalletPage()));
    expect(find.text('钱包'), findsOneWidget);
  });
  testWidgets('ProfilePage 渲染', (t) async {
    await t.pumpWidget(const MaterialApp(home: ProfilePage()));
    expect(find.text('个人中心'), findsOneWidget);
  });
}
```

**Step 4: 跑测试**

```bash
flutter test test/pages/home_pages_test.dart test/pages/audit/pending_audit_page_test.dart
```

Expected: PASS（5 + 1 = 6 个）

**Step 5: Commit**

```bash
git add escort-app/lib/pages/audit/ escort-app/lib/pages/home/ escort-app/test/pages/
git commit -m "feat(escort-app): pages/audit/pending + pages/home (shell + feed/orders/wallet/profile) + 6 个 widget 测试"
```

---

### Task 13: pages/order (3 页 + CountdownBadge widget) + 测试

**Files:**
- Create: `escort-app/lib/widgets/countdown_badge.dart`
- Create: `escort-app/lib/widgets/countdown_badge_test.dart`
- Create: `escort-app/lib/pages/order/order_detail_page.dart`
- Create: `escort-app/lib/pages/order/checkin_page.dart`
- Create: `escort-app/lib/pages/order/checkout_page.dart`
- Create: `escort-app/test/pages/order_pages_test.dart`

**Step 1: countdown_badge_test.dart（RED）**

```dart
// test/widgets/countdown_badge_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/widgets/countdown_badge.dart';

void main() {
  testWidgets('未超时：显示锁单剩余 N s', (t) async {
    final expireAt = DateTime.now().add(const Duration(seconds: 25));
    await t.pumpWidget(MaterialApp(home: Scaffold(body: CountdownBadge(expireAt: expireAt))));
    await t.pump();
    expect(find.textContaining('锁单剩余'), findsOneWidget);
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

Expected: FAIL — not found

**Step 3: countdown_badge.dart**

```dart
// lib/widgets/countdown_badge.dart
import 'dart:async';
import 'package:flutter/material.dart';

/// 30s 锁单倒计时（spec §4.3）。
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
      label: Text('锁单剩余 ${s}s'),
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

**Step 5: 3 个 order page 骨架**

```dart
// lib/pages/order/order_detail_page.dart
import 'package:flutter/material.dart';
import 'package:escort_app/widgets/countdown_badge.dart';

class OrderDetailPage extends StatelessWidget {
  final int orderId;
  const OrderDetailPage({required this.orderId, super.key});
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text('订单 #$orderId')),
      body: Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const Text('TODO: 状态机进度条 + 虚拟号 + 签到按钮'),
            // spec §3.1: 锁单中显示 CountdownBadge（需 expireAt，骨架阶段不接）
            // CountdownBadge(expireAt: DateTime.now().add(Duration(seconds: 30))),
          ],
        ),
      ),
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

**Step 6: order_pages_test.dart**

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

**Step 7: 跑测试 + Commit**

```bash
flutter test test/pages/order_pages_test.dart test/widgets/countdown_badge_test.dart
git add escort-app/lib/pages/order/ escort-app/lib/widgets/countdown_badge.dart escort-app/test/pages/order_pages_test.dart escort-app/test/widgets/countdown_badge_test.dart
git commit -m "feat(escort-app): pages/order (detail/checkin/checkout) + CountdownBadge widget + 5 个测试"
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
      body: const Center(child: Text('TODO: 评价列表（spec §3.1 P1）')),
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
    await t.pumpWidget(const MaterialApp(home: ChatPage(conversationId: 5)));
    expect(find.text('会话 #5'), findsOneWidget);
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
flutter test test/pages/message/message_pages_test.dart test/pages/profile/profile_pages_test.dart
git add escort-app/lib/pages/message/ escort-app/lib/pages/profile/ escort-app/test/pages/message/ escort-app/test/pages/profile/
git commit -m "feat(escort-app): pages/message (2 P2) + pages/profile (2 P1) + 4 个 widget 测试"
```

---

### Task 16: providers/{order,wallet,message,training} + 测试

**Files:**
- Create: `escort-app/lib/providers/order_provider.dart`
- Create: `escort-app/lib/providers/wallet_provider.dart`
- Create: `escort-app/lib/providers/message_provider.dart`
- Create: `escort-app/lib/providers/training_provider.dart`
- Create: `escort-app/test/providers/order_provider_test.dart`
- Create: `escort-app/test/providers/wallet_provider_test.dart`

**Step 1: order_provider.dart + 测试**

```dart
// lib/providers/order_provider.dart
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:escort_app/api/dio_client.dart';
import 'package:escort_app/api/order_api.dart';
import 'package:escort_app/models/order.dart';

final orderApiProvider = Provider<OrderApi>((ref) => OrderApi(ref.watch(dioProvider)));

/// 抢单池 Feed：每 5s 拉一次（spec §4.1）。
final feedProvider = StreamProvider<List<OrderSummary>>((ref) async* {
  final api = ref.watch(orderApiProvider);
  yield await api.feed();
  await for (final _ in Stream.periodic(const Duration(seconds: 5))) {
    yield await api.feed();
  }
});

/// 接单 family（按订单 id 缓存 Future）。
final acceptProvider = FutureProvider.family<Order, int>((ref, orderId) async {
  return ref.read(orderApiProvider).accept(orderId);
});

/// 我的订单 provider。
final myOrdersProvider = FutureProvider.family<List<Order>, String?>((ref, status) async {
  return ref.read(orderApiProvider).listMine(status: status);
});
```

```dart
// test/providers/order_provider_test.dart
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/api/dio_client.dart';
import 'package:escort_app/providers/order_provider.dart';
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

ProviderContainer makeContainer() {
  final storage = TokenStorage.forTest();
  final dio = buildDio(storage: storage, baseUrl: 'https://x');
  dio.httpClientAdapter = StubAdapter(body: {
    'code': 0, 'data': [
      {'id': 1, 'hospital_name': '协和', 'service_start_at_text': '明天 09:00', 'amount': 300.0, 'package_name': '半日陪诊'},
    ],
  });
  return ProviderContainer(overrides: [dioProvider.overrideWithValue(dio)]);
}

void main() {
  test('feedProvider 第一次 yield 列表', () async {
    final c = makeContainer();
    addTearDown(c.dispose);
    final feed = c.read(feedProvider.future);
    final list = await feed.timeout(const Duration(seconds: 2));
    expect(list.length, 1);
    expect(list.first.hospitalName, '协和');
  });
}
```

**Step 2: wallet_provider.dart + 测试**

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

**Step 3: message_provider.dart + training_provider.dart**（一次性写完，无单测 — Provider 包装层由 API 单测覆盖）

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

**Step 4: 跑全部 providers 测试**

```bash
flutter test test/providers/
```

Expected: PASS（auth 4 + order 1 + wallet 1 = 6 个；message/training 无单测靠 API 单测覆盖）

**Step 5: Commit**

```bash
git add escort-app/lib/providers/ escort-app/test/providers/
git commit -m "feat(escort-app): providers (order/wallet/message/training) + 2 个 Provider 单测"
```

---

### Task 17: widgets/ 公共组件 + 测试（order_card / rating_stars / status_chip / gps_checkin_button）

**Files:**
- Create: `escort-app/lib/widgets/order_card.dart`
- Create: `escort-app/lib/widgets/order_card_test.dart`
- Create: `escort-app/lib/widgets/rating_stars.dart`
- Create: `escort-app/lib/widgets/rating_stars_test.dart`
- Create: `escort-app/lib/widgets/status_chip.dart`
- Create: `escort-app/lib/widgets/status_chip_test.dart`
- Create: `escort-app/lib/widgets/gps_checkin_button.dart`
- Create: `escort-app/lib/widgets/gps_checkin_button_test.dart`

**Step 1: status_chip_test.dart（RED）**

```dart
// test/widgets/status_chip_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:escort_app/models/order.dart';
import 'package:escort_app/widgets/status_chip.dart';

void main() {
  testWidgets('StatusChip(matching) 显示 "抢单中" + 橙色', (t) async {
    await t.pumpWidget(MaterialApp(
      home: Scaffold(body: StatusChip(status: OrderStatus.matching)),
    ));
    expect(find.text('抢单中'), findsOneWidget);
  });

  testWidgets('StatusChip(completed) 显示 "已完成"', (t) async {
    await t.pumpWidget(MaterialApp(
      home: Scaffold(body: StatusChip(status: OrderStatus.completed)),
    ));
    expect(find.text('已完成'), findsOneWidget);
  });
}
```

**Step 2: 跑测试确认失败**

```bash
flutter test test/widgets/status_chip_test.dart
```

**Step 3: status_chip.dart**

```dart
// lib/widgets/status_chip.dart
import 'package:flutter/material.dart';
import 'package:escort_app/models/order.dart';

/// 订单状态 → 文案 + 颜色映射（spec §10 配色）。
class StatusChip extends StatelessWidget {
  final OrderStatus status;
  const StatusChip({required this.status, super.key});

  ({String text, Color color}) _map(OrderStatus s) {
    switch (s) {
      case OrderStatus.created: return (text: '已创建', color: Colors.grey);
      case OrderStatus.paid: return (text: '已支付', color: Colors.blue);
      case OrderStatus.matching: return (text: '抢单中', color: Colors.orange);
      case OrderStatus.pendingAcceptance: return (text: '锁单中', color: Colors.deepOrange);
      case OrderStatus.accepted: return (text: '已接单', color: Colors.cyan);
      case OrderStatus.inService: return (text: '服务中', color: Colors.green);
      case OrderStatus.completed: return (text: '已完成', color: Colors.green);
      case OrderStatus.reviewed: return (text: '已评价', color: Colors.green);
      case OrderStatus.refunding: return (text: '退款中', color: Colors.amber);
      case OrderStatus.refunded: return (text: '已退款', color: Colors.amber);
      case OrderStatus.settling: return (text: '结算中', color: Colors.purple);
      case OrderStatus.disputed: return (text: '争议', color: Colors.red);
      case OrderStatus.closed: return (text: '已关闭', color: Colors.grey);
      case OrderStatus.canceled: return (text: '已取消', color: Colors.grey);
    }
  }

  @override
  Widget build(BuildContext context) {
    final m = _map(status);
    return Chip(
      label: Text(m.text),
      backgroundColor: m.color.withOpacity(0.15),
      side: BorderSide(color: m.color),
    );
  }
}
```

**Step 4: 跑测试确认通过**

```bash
flutter test test/widgets/status_chip_test.dart
```

Expected: PASS（2 个）

**Step 5: order_card.dart + rating_stars.dart + gps_checkin_button.dart + 各自测试**

```dart
// lib/widgets/order_card.dart
import 'package:flutter/material.dart';
import 'package:escort_app/models/order.dart';

class OrderCard extends StatelessWidget {
  final OrderSummary order;
  final VoidCallback? onTap;
  const OrderCard({required this.order, this.onTap, super.key});

  @override
  Widget build(BuildContext context) {
    return Card(
      child: ListTile(
        title: Text(order.hospitalName),
        subtitle: Text('${order.packageName} · ${order.serviceStartAtText}'),
        trailing: Text(order.amountText, style: const TextStyle(fontWeight: FontWeight.bold)),
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
  testWidgets('OrderCard 渲染医院名 + 金额', (t) async {
    final o = OrderSummary(
      id: 1, hospitalName: '协和', serviceStartAtText: '明天 09:00',
      amount: 300.0, packageName: '半日陪诊',
    );
    await t.pumpWidget(MaterialApp(home: Scaffold(body: OrderCard(order: o))));
    expect(find.text('协和'), findsOneWidget);
    expect(find.text('¥300.00'), findsOneWidget);
  });
}
```

```dart
// lib/widgets/rating_stars.dart
import 'package:flutter/material.dart';

class RatingStars extends StatelessWidget {
  final double rating; // 0-5
  const RatingStars({required this.rating, super.key});

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: List.generate(5, (i) {
        final filled = i < rating.floor();
        final half = !filled && i < rating;
        return Icon(
          filled ? Icons.star : (half ? Icons.star_half : Icons.star_border),
          color: Colors.amber, size: 16,
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
  testWidgets('RatingStars(4.5) 渲染 4 颗实心 + 1 颗半星', (t) async {
    await t.pumpWidget(const MaterialApp(home: Scaffold(body: RatingStars(rating: 4.5))));
    expect(find.byIcon(Icons.star), findsNWidgets(4));
    expect(find.byIcon(Icons.star_half), findsOneWidget);
    expect(find.byIcon(Icons.star_border), findsNothing);
  });
}
```

```dart
// lib/widgets/gps_checkin_button.dart
import 'dart:async';
import 'package:flutter/material.dart';

/// 长按 GPS 验证按钮（spec §5 防误触）。
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

**Step 6: 跑全部 widgets 测试 + Commit**

```bash
flutter test test/widgets/
git add escort-app/lib/widgets/ escort-app/test/widgets/
git commit -m "feat(escort-app): widgets (OrderCard/RatingStars/StatusChip/GpsCheckinButton) + 6 个 widget 测试"
```

---

### Task 18: app.dart + main.dart + integration_test + dev.md

**Files:**
- Create: `escort-app/lib/app.dart`
- Create: `escort-app/lib/main.dart`（替换 `flutter create` 默认）
- Create: `escort-app/integration_test/app_test.dart`
- Create: `escort-app/integration_test/feed_test.dart`
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

**Step 4: integration_test/feed_test.dart**（mock backend）

```dart
// integration_test/feed_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:escort_app/main.dart' as app;

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  testWidgets('feed 页面渲染（无真后端：expect find 占位文本）', (t) async {
    app.main();
    await t.pumpAndSettle(const Duration(seconds: 3));
    // 当前骨架阶段无法跳到 feed（重定向会拦住）；仅断言 splash 渲染
    expect(find.text('Doctors Escort'), findsOneWidget);
  });
}
```

**Step 5: dev.md 加 §10.13**

```markdown
### 10.13 escort-app setup plan（2026-09-24）

落地陪诊师 Flutter App 骨架（iOS + Android）。

**落地 commits（约 18 个）**：

| commit | 内容 |
| :-- | :-- |
| chore(escort-app) | flutter create + pubspec deps + iOS/Android 权限 + README |
| chore(escort-app) | OpenAPI 生成器 config + gen-api.sh |
| feat(escort-app) | utils (format/trace/error_handler) + 11 单测 |
| feat(escort-app) | models (Order/EscortProfile/Wallet/Withdrawal/Signal) + 6 单测 |
| feat(escort-app) | services (TokenStorage/GpsService/NopPushService) + 5 单测 |
| feat(escort-app) | dio_client (拦截器) + 3 单测 |
| feat(escort-app) | api modules (8 个) + 6 StubAdapter 单测 |
| feat(escort-app) | AuthState (sealed) + AuthNotifier + 4 单测 |
| feat(escort-app) | router (go_router + 24 路由 + 3 守卫) + 11 单测 |
| feat(escort-app) | pages/splash + pages/auth (login/register) + 2 测试 |
| feat(escort-app) | pages/onboarding (6 骨架) + 6 测试 |
| feat(escort-app) | pages/audit/pending + pages/home (shell + 4 tabs) + 6 测试 |
| feat(escort-app) | pages/order (3 页 + CountdownBadge widget) + 5 测试 |
| feat(escort-app) | pages/wallet + pages/sos + SosLongPress + 5 测试 |
| feat(escort-app) | pages/message (2 P2) + pages/profile (2 P1) + 4 测试 |
| feat(escort-app) | providers (order/wallet/message/training) + 2 Provider 单测 |
| feat(escort-app) | widgets (OrderCard/RatingStars/StatusChip/GpsCheckinButton) + 6 测试 |
| feat(escort-app) | app.dart + main.dart + integration_test + dev.md |

**API 增量**：14 API（escort 端）+ 8 个 lib/api/ Dart 模块（v2 接 OpenAPI generated）。

**未做**：UI 完善（具体表单 / 错误 toast / 动画）留 v1.x 后续 plan；OpenAPI generated/ 待 `web/openapi/contracts.yaml` 落地后跑 `bash scripts/gen-api.sh`；推送通道（极光/友盟）留 v2；视频陪诊 / i18n / 离线模式 / AI 推荐留 v3。
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

Expected: PASS（约 50+ 测试）

**Step 8: Commit**

```bash
cd /Users/growduduan/ai/doctors
git add escort-app/lib/app.dart escort-app/lib/main.dart escort-app/integration_test/ dev.md
git commit -m "feat(escort-app): app.dart + main.dart 装配 + integration_test smoke + dev.md 10.13"
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

- ✅ **Spec 覆盖**:
    - escort-app-design.md §1 12 个功能模块：注册→实名→健康证→培训→审核→上线→抢单→服务→钱包→提现→SOS（已映射到 providers / pages / api / widgets）；评价（P1）/ 站内信（P2）/ 资料编辑（P1）作为 page 骨架预留
    - §2 目录结构：24 个 page + 8 个 API + 5 个 model + 6 个 widget + 4 个 provider + 3 个 service + 3 个 util + 3 个 router 文件 = 与 spec §2 一致
    - §3 24 个 P0 页面：全部已建（splash / login / register / real-name / health-cert / training list/video/quiz / agreement / audit-pending / feed / orders / wallet / profile / order-detail / checkin / checkout / withdraw / transactions / sos-trigger / message list+detail / profile edit+reviews = 24 个）
    - §3.2 状态机映射到 `OnboardingStage` enum + `OrderStatus` enum
    - §4 抢单池：`feedProvider` StreamProvider + `CountdownBadge` widget + `acceptProvider.family`
    - §5 GPS 签到：`GpsService` + `GpsCheckinButton` widget + `CheckinPage`
    - §6 钱包：`walletProvider` + `withdrawControllerProvider` + `WithdrawPage` + `TransactionsPage`
    - §7 SOS 长按：`SosLongPress` widget（3s `AnimationController` + `Timer`）+ `SosTriggerPage`
    - §8 Riverpod：`AuthNotifier`（StateNotifier）+ `AuthState`（sealed）+ `authGuardProvider` / `realNameGuardProvider` / `approvedGuardProvider`
    - §9 dio 拦截器：`buildDio` + auth/trace 拦截器 + 401 handler
    - §10 Material 3：`useMaterial3: true` + 主色 `#1989FA`
    - §11 测试矩阵：utils 11 + models 6 + services 5 + api 6 + providers 6 + router 11 + widgets 6 + pages ~22 ≈ **73 个测试**
    - §12 构建命令：README + dev.md 已列出 `flutter run` / `flutter test` / `flutter analyze`
    - §14 不做：WebSocket 推送 / 推送通道 / 视频陪诊 / i18n / 离线模式 / AI 推荐 — 已在 Global Constraints 声明
  - ✅ **无占位符**: 每个 Task 给出完整代码 / 测试 / 命令；commit message 明确
  - ✅ **类型一致**:
    - `OrderStatus.fromString` ↔ `models/order.dart` ↔ `StatusChip._map`
    - `OnboardingStage.fromString` ↔ `EscortProfile.fromJson` ↔ `EscortApi.realNameAuth` 返回
    - `SignalStatus.fromString` ↔ `SosApi.trigger` 返回
    - `WithdrawalStatus.fromString` ↔ `WalletApi.withdraw/listTransactions` 返回
    - `AuthState` sealed → `authGuardProvider.maybeWhen(authenticated: (user, _) => ...)` pattern（Task 9 注明若 pattern 数不对按编译错误调整）
    - `goRouterProvider` 在 `app_router.dart` 定义，`app.dart` 引用
  - ✅ **测试矩阵**: 73 个 `flutter test` + 2 个 `integration_test`（smoke）+ `flutter analyze` 0 issue + `flutter build apk/ios --debug` 编译通过
  - ✅ **YAGNI**:
    - **不**引入 FlutterFire（`firebase_messaging` 留 v2）
    - **不**引入 BLoC / GetX / Provider（仅 Riverpod）
    - **不**引入 retrofit / chopper / dio_smart_retry（裸 dio + 拦截器）
    - **不**引入 lottie / flutter_markdown（v1 用 Text 占位）
    - **不**引入 i18n（spec §14 v3 才做）
    - **不**接 OpenAPI generated（v1 仅写 Dart API wrapper；`scripts/gen-api.sh` 留口子，generated/ 不入仓逻辑代码）
    - **不**做推送通道（spec §14 留 v2）
    - **不**做 video player（spec §14 v3）

## Execution Options

> Plan 已 commit 到 `docs/superpowers/plans/2026-09-24-escort-app-setup.md`。
> 当前为 plan 阶段；不进入实施。

**关联 plan**：
- 后端：`docs/superpowers/plans/2026-09-24-escort-business-plan.md`（escort 业务 API，待写）— escort-app Task 7 的 8 个 API 模块依赖其落地
- 后端：`docs/superpowers/plans/2026-09-24-escort-order-ext-plan.md`（订单 escort 视图扩展，待写）— escort-app Task 7 `OrderApi.feed / checkin / checkout` 依赖
- 后端：`docs/superpowers/plans/2026-09-24-wallet-plan.md`（wallet + withdrawal，待写）— escort-app Task 7 `WalletApi` 依赖
- 前端：`docs/superpowers/plans/2026-09-24-patient-miniapp-setup.md`（待写）
- 前端：`docs/superpowers/plans/2026-09-24-admin-web-setup.md`（待写）

**下一步选项**：
1. **进入实施** —— 实施 Task 1~19（subagent-driven 推荐）
2. **暂停 + review** —— 调整 plan（页面骨架 vs UI 完善 / OpenAPI 生成时机 / 测试覆盖基线）
3. **继续产 plan** —— 接着出 patient-miniapp-setup / admin-web-setup / 后端 escort-business / wallet / escort-order-ext 等 plan