// lib/models/availability.dart
//
// 「我的空余时段」领域模型（spec/2026-09-24-order-matching-redesign §3.2）。
//
// 业务规则：
//   - available 状态可被患者候选选中；可被 escort 删除
//   - booked 状态已被订单占用（confirmAccept 后自动转换）；不可删
//   - canceled 已撤销（不可删）
//
// UNIQUE 索引：escort_id + start_at（spec §3.2 表约束）；冲突时后端返 13103。
import '../utils/format.dart';

/// 时段状态机。
enum AvailabilityStatus {
  available,
  booked,
  canceled;

  static AvailabilityStatus fromString(String s) {
    switch (s) {
      case 'available':
        return AvailabilityStatus.available;
      case 'booked':
        return AvailabilityStatus.booked;
      case 'canceled':
        return AvailabilityStatus.canceled;
    }
    throw ArgumentError('unknown AvailabilityStatus: $s');
  }

  String get wireValue {
    switch (this) {
      case AvailabilityStatus.available:
        return 'available';
      case AvailabilityStatus.booked:
        return 'booked';
      case AvailabilityStatus.canceled:
        return 'canceled';
    }
  }
}

/// 时段（陪诊师挂出的「可被邀请」窗口）。
class Availability {
  final int id;
  final int escortId;
  final DateTime startAt;
  final DateTime endAt;
  final AvailabilityStatus status;

  /// 关联订单 ID（booked 时回填，便于跳转订单详情）。
  final int? orderId;

  const Availability({
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

  Map<String, dynamic> toJson() => {
        'id': id,
        'escort_id': escortId,
        'start_at': startAt.toIso8601String(),
        'end_at': endAt.toIso8601String(),
        'status': status.wireValue,
        if (orderId != null) 'order_id': orderId,
      };

  /// 时段是否可被 escort 删除（仅 available 状态可删）。
  bool get isDeletable => status == AvailabilityStatus.available;

  /// 是否已被患者选中（UI 显示「已预订」chip）。
  bool get isBooked => status == AvailabilityStatus.booked;

  String get startText => formatDateTime(startAt);
  String get endText => formatDateTime(endAt);

  /// 时段长度（小时，向下取整）；UI 副标题展示用。
  int get durationHours => endAt.difference(startAt).inHours;
}