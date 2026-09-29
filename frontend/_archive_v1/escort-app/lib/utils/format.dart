// lib/utils/format.dart
//
// 通用格式化工具 —— 金额 / 时间 / 手机号。
// 单一职责：无副作用、无 platform 依赖；可在 isolate / 单元测试中直接调用。
//
// 设计依据：
//   - formatMoney：spec §10 钱包展示统一 `¥xxx.xx`
//   - formatDateTime：陪诊时段卡片 / 订单详情统一 `yyyy-MM-dd HH:mm`
//   - maskPhone：敏感信息脱敏（仅展示前 3 + 后 4）
//
// 错误策略：
//   - formatMoney 负数抛 ArgumentError（资金不能为负，调用方应先校验）
//   - maskPhone 非 11 位原样返回（兜底，避免异常打断列表渲染）

/// 格式化金额：负数抛 ArgumentError；小数位四舍五入保留 2 位。
///
/// 例：`formatMoney(100) → '¥100.00'`、`formatMoney(99.999) → '¥100.00'`。
String formatMoney(num amount) {
  if (amount.isNegative) {
    throw ArgumentError.value(amount, 'amount', 'must be >= 0');
  }
  return '¥${amount.toStringAsFixed(2)}';
}

/// 格式化日期时间：`yyyy-MM-dd HH:mm`（本地时区）。
String formatDateTime(DateTime dt) {
  String two(int n) => n.toString().padLeft(2, '0');
  return '${dt.year}-${two(dt.month)}-${two(dt.day)} '
      '${two(dt.hour)}:${two(dt.minute)}';
}

/// 手机号脱敏：11 位 → `前3****后4`；非 11 位原样返回（兜底）。
String maskPhone(String phone) {
  if (phone.length != 11) return phone;
  return '${phone.substring(0, 3)}****${phone.substring(7)}';
}