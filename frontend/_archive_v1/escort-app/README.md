# escort-app

陪诊师端 Flutter App（iOS + Android）。v1 骨架：脚手架 + 路由 + Riverpod。

## 命令

```bash
flutter pub get              # 拉依赖
flutter run -d ios           # iOS 模拟器
flutter run -d android       # Android 模拟器
flutter test                 # 单测 + widget 测试
flutter test integration_test/  # E2E
flutter analyze              # 静态分析
bash scripts/gen-api.sh      # 重新生成 OpenAPI Dart client（Task 2）
```

## 业务流程（按 order-matching-redesign §1.2 + §4.1）

陪诊师**不再抢单**。流程：
1. 上线前在「我的空余时段」(`/home/availability`) 设置 `available` 时段
2. 患者从候选列表选了你 → 推送到「我的邀请」(`/home/invitations`)
3. 30s 倒计时内点「确认接单」或「拒接」；超时未操作 → 订单回退到 `selecting_escort`，患者可重选

## API base URL

dev: `https://api.dev.doctors.example.com/api/v1`
prod: `https://api.doctors.example.com/api/v1`
（由 `lib/api/dio_client.dart` 的 `baseUrl` 常量 + `--dart-define=API_BASE=...` 覆盖。）

## 目录结构（v1 骨架）

```
lib/
├── main.dart                    # ProviderScope + DoctorsEscortApp 入口
├── app.dart                     # MaterialApp.router
├── core/                        # 全局基础设施
│   ├── router.dart              # GoRouter Provider
│   ├── theme.dart               # Material 3 主题（主色 #1989FA）
│   └── constants.dart           # API baseUrl / trace prefix / 常量
├── api/                         # dio 客户端 + 各领域 API（Task 后续）
├── models/                      # 数据模型
├── providers/                   # Riverpod providers
├── pages/                       # 24 个 P0 页面
├── widgets/                     # 公共 widget
├── services/                    # token / GPS / push 等服务
├── router/                      # 路由守卫
└── utils/                       # format / trace / error_handler
```