# lib/shared/

**跨域复用层** —— 跨多个 feature 共享的 widget / 工具。

按 spec §2 设计，后续 Task 4+ 落地：

- `lib/widgets/order_card.dart` / `invitation_card.dart` / `availability_tile.dart`
- `lib/widgets/countdown_badge.dart`（30s 倒计时）
- `lib/widgets/rating_stars.dart` / `status_chip.dart` / `gps_checkin_button.dart`
- `lib/widgets/sos_long_press.dart`
- `lib/utils/format.dart`（formatMoney / formatDateTime / maskPhone）
- `lib/utils/trace.dart`（newTraceId → `escort-{ms}-{rand}`）
- `lib/utils/error_handler.dart`（dio 异常 → 用户文案，含 13101/13102 邀请过期/已被接受）

> Task 1~3 不实现具体工具，仅保留目录结构。