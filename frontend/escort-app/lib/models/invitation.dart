// lib/models/invitation.dart
//
// 「我的邀请」页用 —— 后端 GET /escorts/me/invitations 返回每条 = 一个处于
// `escort_pending_acceptance` 状态的订单，已附带 `escort_pending_expire_at`。
//
// 选人模式（spec/2026-09-24-order-matching-redesign §4.1）：
//   - 患者从候选列表选了 escort → 后端推送 30s 邀请
//   - 陪诊师收到 → 在「我的邀请」倒计时内点「确认接单」或「拒接」
//   - 超时未操作 → 后端 30s 后订单回退 selecting_escort，escort_pending_expire_at < now
//
// 客户端责任：
//   - `isLive` 过滤已过期（不展示「确认」按钮）
//   - 5s 轮询拉新列表（StreamProvider）
import '../utils/format.dart';

/// 邀请（瘦订单 + 倒计时）。
class Invitation {
  /// 关联订单 ID（用于 confirmAccept(id) / rejectAccept(id)）。
  final int orderId;

  /// 医院名 + 经纬度（用于详情跳转）。
  final String hospitalName;
  final double hospitalLat;
  final double hospitalLng;

  /// 服务套餐 + 开始时间 + 金额。
  final String packageName;
  final DateTime serviceStartAt;
  final double amount;

  /// 30s 确认窗口到期时间（后端生成，客户端按此自渲染倒计时）。
  final DateTime escortPendingExpireAt;

  const Invitation({
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
        escortPendingExpireAt:
            DateTime.parse(j['escort_pending_expire_at'] as String),
      );

  /// 是否仍在 30s 确认窗口内（客户端按此过滤过期邀请）。
  bool get isLive => DateTime.now().isBefore(escortPendingExpireAt);

  /// 剩余秒数（向下取整；已过期返回 0）。
  int get remainingSeconds {
    final diff = escortPendingExpireAt.difference(DateTime.now()).inSeconds;
    return diff < 0 ? 0 : diff;
  }

  String get amountText => formatMoney(amount);
}