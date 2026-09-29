// lib/providers/wallet_provider.dart
//
// 钱包 provider —— GET /wallet + GET /wallet/transactions + POST /escorts/me/wallet/withdraw。
//
// 设计要点：
//   - walletProvider：FutureProvider 拉余额 + 冻结
//   - walletTransactionsProvider：FutureProvider.family<...,({int page,int size})> 拉分页流水
//   - withdrawControllerProvider：StateNotifier，提交提现后 invalidate wallet + transactions
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/wallet.dart';
import '../services/api_client.dart';

/// 钱包余额 provider（FutureProvider）。
final walletProvider = FutureProvider<Wallet>((ref) async {
  final dio = ref.read(dioProvider);
  final data = await WalletApi.getWallet(dio);
  return Wallet.fromJson(data);
});

/// 流水分页参数（family key）。
class WalletTxKey {
  final int page;
  final int size;
  const WalletTxKey({this.page = 1, this.size = 20});

  @override
  bool operator ==(Object other) =>
      other is WalletTxKey && other.page == page && other.size == size;
  @override
  int get hashCode => Object.hash(page, size);
}

/// 流水分页 provider（FutureProvider.family）。
final walletTransactionsProvider =
    FutureProvider.family<List<WalletTransaction>, WalletTxKey>((ref, key) async {
  final dio = ref.read(dioProvider);
  final data = await WalletApi.getTransactions(dio, page: key.page, size: key.size);
  final raw = data['list'];
  if (raw is! List) return <WalletTransaction>[];
  return raw
      .map((e) => WalletTransaction.fromJson(e as Map<String, dynamic>))
      .toList();
});

/// 提现提交状态机（sealed）。
sealed class WithdrawState {
  const WithdrawState();
}

class WithdrawIdle extends WithdrawState {
  const WithdrawIdle();
}

class WithdrawSubmitting extends WithdrawState {
  const WithdrawSubmitting();
}

class WithdrawSuccess extends WithdrawState {
  final int withdrawId;
  const WithdrawSuccess(this.withdrawId);
}

class WithdrawFailure extends WithdrawState {
  final String message;
  const WithdrawFailure(this.message);
}

/// 提现 controller provider。
final withdrawControllerProvider =
    StateNotifierProvider.autoDispose<WithdrawController, WithdrawState>(
  (ref) => WithdrawController(ref),
);

/// 提现控制器。
class WithdrawController extends StateNotifier<WithdrawState> {
  final Ref ref;

  WithdrawController(this.ref) : super(const WithdrawIdle());

  Future<void> submit({
    required double amount,
    required String bankAccount,
  }) async {
    state = const WithdrawSubmitting();
    try {
      final dio = ref.read(dioProvider);
      final r = await WalletApi.withdraw(
        dio,
        amount: amount,
        bankAccount: bankAccount,
      );
      final id = (r['withdraw_id'] as num?)?.toInt() ?? 0;
      state = WithdrawSuccess(id);
      // 刷新余额 + 流水（提现成功后冻结金额会变）
      ref.invalidate(walletProvider);
      ref.invalidate(walletTransactionsProvider);
    } on DioException catch (e) {
      final msg = e.response?.data is Map
          ? ((e.response!.data as Map)['message']?.toString() ?? e.toString())
          : e.toString();
      state = WithdrawFailure(msg);
      rethrow;
    }
  }

  void reset() => state = const WithdrawIdle();
}