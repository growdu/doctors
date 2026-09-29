// lib/widgets/countdown_badge.dart
//
// 倒计时徽章 —— spec/2026-09-24-order-matching-redesign §4.1 邀请卡片专用。
//
// 设计要点：
//   - 参数化 `expireAt: DateTime`（不绑定订单状态语义；UI 文案显示「待确认剩余」）
//   - < 60s 红色 `#ff4d4f`（紧急色）；> 60s 主题色（蓝色 #1989FA）
//   - 组件销毁时 `cancel()` Timer，避免泄漏
//   - 每秒重建（setState）；表盘粒度够用（spec 要求 30s 精度）
//
// 复用：本组件后续可扩展为通用倒计时（订单超时 / 优惠券到期等）。
import 'dart:async';

import 'package:flutter/material.dart';

/// 倒计时阈值（秒）；剩余 ≤ 此值时切红色。
const int kCountdownWarnSeconds = 60;

/// 倒计时徽章。
///
/// 父组件传入 `expireAt`，组件内部按秒自更新 UI，到期后展示「已超时」红色文案。
class CountdownBadge extends StatefulWidget {
  /// 倒计时到期时间（绝对本地时区）。
  final DateTime expireAt;

  /// 自定义前缀文案（默认「待确认剩余」）。
  final String label;

  /// 警告色（默认 spec 红 #ff4d4f）。
  final Color warnColor;

  /// 主色（默认 spec 主色蓝 #1989FA）。
  final Color baseColor;

  const CountdownBadge({
    super.key,
    required this.expireAt,
    this.label = '待确认剩余',
    this.warnColor = const Color(0xFFFF4D4F),
    this.baseColor = const Color(0xFF1989FA),
  });

  @override
  State<CountdownBadge> createState() => _CountdownBadgeState();
}

class _CountdownBadgeState extends State<CountdownBadge> {
  Timer? _timer;
  late int _secondsLeft;

  @override
  void initState() {
    super.initState();
    _secondsLeft = _compute();
    _timer = Timer.periodic(const Duration(seconds: 1), (_) {
      if (!mounted) return;
      setState(() => _secondsLeft = _compute());
    });
  }

  int _compute() {
    final diff = widget.expireAt.difference(DateTime.now()).inSeconds;
    return diff < 0 ? 0 : diff;
  }

  @override
  void didUpdateWidget(covariant CountdownBadge oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.expireAt != widget.expireAt) {
      setState(() => _secondsLeft = _compute());
    }
  }

  @override
  void dispose() {
    _timer?.cancel();
    super.dispose();
  }

  /// 是否处于紧急态（≤ 60s）。
  bool get _isWarn => _secondsLeft <= kCountdownWarnSeconds;

  /// 是否已超时。
  bool get _isExpired => _secondsLeft <= 0;

  String get _display {
    if (_isExpired) return '已超时';
    if (_secondsLeft < 60) return '${_secondsLeft}s';
    final m = _secondsLeft ~/ 60;
    final s = _secondsLeft % 60;
    return s == 0 ? '${m}m' : '${m}m${s}s';
  }

  @override
  Widget build(BuildContext context) {
    final color = _isExpired || _isWarn ? widget.warnColor : widget.baseColor;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: color.withOpacity(0.1),
        borderRadius: BorderRadius.circular(4),
        border: Border.all(color: color, width: 1),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(
            _isExpired ? Icons.timer_off_outlined : Icons.timer_outlined,
            size: 14,
            color: color,
          ),
          const SizedBox(width: 4),
          Text(
            _isExpired ? '已超时' : '${widget.label} • $_display',
            style: TextStyle(
              color: color,
              fontSize: 12,
              fontWeight: FontWeight.w500,
            ),
          ),
        ],
      ),
    );
  }
}