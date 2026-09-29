// lib/models/order.dart
//
// 订单领域模型（spec/2026-09-24-order-matching-redesign §2.1 状态机 + §3 实体）。
//
// v1.1 修订（相对 spec/2026-09-24-escort-app-design）：
//   - 移除 `pendingAcceptance` 旧枚举值（被 `escortPendingAcceptance` 取代）
//   - 新增 `selectingEscort` 状态（患者已选 escort，等待 30s 确认）
//   - 新增 `selectedEscortId` / `escortPendingExpireAt` 字段
//   - 移除 `lockOwner` / `lockExpireAt` 旧字段（v1 抢单池语义）
//
// 与 `OrderSummary` 区别：Order 完整用于详情页；OrderSummary 用于列表/邀请卡片瘦字段。
import '../utils/format.dart';

/// 订单状态机（spec §2.1 选人模式 12 态）。
///
/// 关键路径：`paid → matching → selectingEscort → escortPendingAcceptance
///          → accepted → inService → completed → reviewed`。
enum OrderStatus {
  created,
  paid,
  matching,
  selectingEscort,
  escortPendingAcceptance,
  accepted,
  inService,
  completed,
  reviewed,
  refunding,
  refunded,
  settling,
  disputed,
  closed,
  canceled;

  /// 后端 snake_case 字符串 → 枚举。未知值抛 ArgumentError（fail-fast，避免静默）。
  static OrderStatus fromString(String s) {
    switch (s) {
      case 'created':
        return OrderStatus.created;
      case 'paid':
        return OrderStatus.paid;
      case 'matching':
        return OrderStatus.matching;
      case 'selecting_escort':
        return OrderStatus.selectingEscort;
      case 'escort_pending_acceptance':
      case 'pending_acceptance': // 兼容 v1 老数据
        return OrderStatus.escortPendingAcceptance;
      case 'accepted':
        return OrderStatus.accepted;
      case 'in_service':
        return OrderStatus.inService;
      case 'completed':
        return OrderStatus.completed;
      case 'reviewed':
        return OrderStatus.reviewed;
      case 'refunding':
        return OrderStatus.refunding;
      case 'refunded':
        return OrderStatus.refunded;
      case 'settling':
        return OrderStatus.settling;
      case 'disputed':
        return OrderStatus.disputed;
      case 'closed':
        return OrderStatus.closed;
      case 'canceled':
        return OrderStatus.canceled;
    }
    throw ArgumentError('unknown OrderStatus: $s');
  }

  /// 序列化为后端 snake_case。
  String get wireValue {
    switch (this) {
      case OrderStatus.created:
        return 'created';
      case OrderStatus.paid:
        return 'paid';
      case OrderStatus.matching:
        return 'matching';
      case OrderStatus.selectingEscort:
        return 'selecting_escort';
      case OrderStatus.escortPendingAcceptance:
        return 'escort_pending_acceptance';
      case OrderStatus.accepted:
        return 'accepted';
      case OrderStatus.inService:
        return 'in_service';
      case OrderStatus.completed:
        return 'completed';
      case OrderStatus.reviewed:
        return 'reviewed';
      case OrderStatus.refunding:
        return 'refunding';
      case OrderStatus.refunded:
        return 'refunded';
      case OrderStatus.settling:
        return 'settling';
      case OrderStatus.disputed:
        return 'disputed';
      case OrderStatus.closed:
        return 'closed';
      case OrderStatus.canceled:
        return 'canceled';
    }
  }
}

/// 订单完整实体（详情页 / 服务中 / 结算页用）。
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

  /// 30s 确认窗口到期时间；status=escortPendingAcceptance 时必填。
  final DateTime? escortPendingExpireAt;

  /// 服务完成时间（可选；completed / reviewed 时回填）。
  final DateTime? completedAt;

  /// 订单创建时间。
  final DateTime? createdAt;

  const Order({
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
    this.completedAt,
    this.createdAt,
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
        completedAt: j['completed_at'] == null
            ? null
            : DateTime.parse(j['completed_at'] as String),
        createdAt: j['created_at'] == null
            ? null
            : DateTime.parse(j['created_at'] as String),
      );

  Map<String, dynamic> toJson() => {
        'id': id,
        'patient_id': patientId,
        'escort_id': escortId,
        'selected_escort_id': selectedEscortId,
        'hospital_name': hospitalName,
        'hospital_lat': hospitalLat,
        'hospital_lng': hospitalLng,
        'package_name': packageName,
        'service_start_at': serviceStartAt.toIso8601String(),
        'amount': amount,
        'status': status.wireValue,
        if (escortPendingExpireAt != null)
          'escort_pending_expire_at': escortPendingExpireAt!.toIso8601String(),
        if (completedAt != null) 'completed_at': completedAt!.toIso8601String(),
        if (createdAt != null) 'created_at': createdAt!.toIso8601String(),
      };

  /// 金额展示（spec §10 钱包规范）。
  String get amountText => formatMoney(amount);
}

/// 订单瘦字段（邀请卡片 / 订单列表用）。
///
/// 后端 `service_start_at_text` 字段已预格式化为可展示文案（避免客户端再调 formatDateTime）。
class OrderSummary {
  final int id;
  final String hospitalName;
  final String serviceStartAtText;
  final double amount;
  final String packageName;
  final OrderStatus? status;
  final DateTime? escortPendingExpireAt;

  const OrderSummary({
    required this.id,
    required this.hospitalName,
    required this.serviceStartAtText,
    required this.amount,
    required this.packageName,
    this.status,
    this.escortPendingExpireAt,
  });

  factory OrderSummary.fromJson(Map<String, dynamic> j) => OrderSummary(
        id: j['id'] as int,
        hospitalName: j['hospital_name'] as String,
        serviceStartAtText: j['service_start_at_text'] as String,
        amount: (j['amount'] as num).toDouble(),
        packageName: j['package_name'] as String,
        status: j['status'] == null
            ? null
            : OrderStatus.fromString(j['status'] as String),
        escortPendingExpireAt: j['escort_pending_expire_at'] == null
            ? null
            : DateTime.parse(j['escort_pending_expire_at'] as String),
      );

  String get amountText => formatMoney(amount);
}