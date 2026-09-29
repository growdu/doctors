// lib/utils/trace.dart
//
// Trace ID 生成器 —— 注入 dio 拦截器，给每个请求打唯一 id 便于后端日志侧按端拆分。
//
// 格式：`escort-{ms}-{rand6}` —— 与 lib/core/constants.dart 的 kTracePrefix 一致。
//   - ms: DateTime.now().millisecondsSinceEpoch（人类可读时间序）
//   - rand6: 6 位随机数（0..999999）—— 提升同 ms 内多请求唯一性
//
// 设计：纯函数 + 静态 `Random`；无 platform channel；单测可直接调用。
import 'dart:math';

import '../core/constants.dart';

final Random _rand = Random();

/// 生成新 trace id。
///
/// 例：`escort-1727187600123-458721`。
String newTraceId() {
  final ms = DateTime.now().millisecondsSinceEpoch;
  // 6 位（0..999999），左补零保证定长，便于日志聚合
  final r = _rand.nextInt(1000000).toString().padLeft(6, '0');
  return '$kTracePrefix-$ms-$r';
}