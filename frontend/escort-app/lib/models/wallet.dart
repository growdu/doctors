// lib/models/wallet.dart
//
// 陪诊师钱包（余额 + 冻结 + 流水）。
//
// 设计要点：
//   - Wallet 仅余额元数据；Transaction 独立类（流水含 type / amount / created_at）
//   - 金额一律以 double 解析（与 Order.amount 保持一致；后端 decimal 转 num）
//   - 状态字段用 enum（避免拼写错误）
import '../utils/format.dart';

/// 钱包元数据。
class Wallet {
  /// 可用余额。
  final double balance;

  /// 冻结金额（提现中 / 结算中）。
  final double frozen;

  /// 币种（默认 CNY）。
  final String currency;

  const Wallet({
    required this.balance,
    required this.frozen,
    this.currency = 'CNY',
  });

  factory Wallet.fromJson(Map<String, dynamic> j) => Wallet(
        balance: (j['balance'] as num).toDouble(),
        frozen: (j['frozen'] as num?)?.toDouble() ?? 0.0,
        currency: (j['currency'] as String?) ?? 'CNY',
      );

  /// 总额（余额 + 冻结）。
  double get total => balance + frozen;

  String get balanceText => formatMoney(balance);
  String get frozenText => formatMoney(frozen);
  String get totalText => formatMoney(total);
}

/// 钱包流水类型。
enum TransactionType {
  income, // 订单收入
  withdraw, // 提现
  refund, // 退款
  frozen, // 冻结
  unfrozen; // 解冻

  static TransactionType fromString(String s) {
    switch (s) {
      case 'income':
        return TransactionType.income;
      case 'withdraw':
        return TransactionType.withdraw;
      case 'refund':
        return TransactionType.refund;
      case 'frozen':
        return TransactionType.frozen;
      case 'unfrozen':
        return TransactionType.unfrozen;
    }
    throw ArgumentError('unknown TransactionType: $s');
  }

  String get wireValue {
    switch (this) {
      case TransactionType.income:
        return 'income';
      case TransactionType.withdraw:
        return 'withdraw';
      case TransactionType.refund:
        return 'refund';
      case TransactionType.frozen:
        return 'frozen';
      case TransactionType.unfrozen:
        return 'unfrozen';
    }
  }

  /// 中文展示名（用于 wallet 页流水 tab）。
  String get displayName {
    switch (this) {
      case TransactionType.income:
        return '收入';
      case TransactionType.withdraw:
        return '提现';
      case TransactionType.refund:
        return '退款';
      case TransactionType.frozen:
        return '冻结';
      case TransactionType.unfrozen:
        return '解冻';
    }
  }
}

/// 单条钱包流水。
class WalletTransaction {
  final int id;

  /// 金额（正数：收入/解冻；负数：提现/退款/冻结）。
  final double amount;

  /// 流水类型。
  final TransactionType type;

  /// 备注（订单 ID / 提现单 ID / 系统说明）。
  final String? memo;

  /// 流水时间。
  final DateTime createdAt;

  const WalletTransaction({
    required this.id,
    required this.amount,
    required this.type,
    this.memo,
    required this.createdAt,
  });

  factory WalletTransaction.fromJson(Map<String, dynamic> j) =>
      WalletTransaction(
        id: j['id'] as int,
        amount: (j['amount'] as num).toDouble(),
        type: TransactionType.fromString(j['type'] as String),
        memo: j['memo'] as String?,
        createdAt: DateTime.parse(j['created_at'] as String),
      );

  String get amountText => formatMoney(amount.abs());
  String get signedAmountText {
    final sign = amount >= 0 ? '+' : '-';
    return '$sign${formatMoney(amount.abs())}';
  }
  String get createdAtText => formatDateTime(createdAt.toLocal());
}