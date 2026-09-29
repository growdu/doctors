// lib/providers/invitation_provider.dart
//
// 「我的邀请」provider —— spec/2026-09-24-order-matching-redesign §4.1。
//
// 设计要点：
//   - StreamProvider + Stream.periodic(5s) 轮询 `/escorts/me/invitations`
//     （spec 明确：不实现 WebSocket / push；客户端按 5s 间隔拉新）
//   - 客户端过滤 `isLive`（按 escort_pending_expire_at）；过期的不渲染「确认/拒接」按钮
//   - confirmAccept / rejectAccept 走 Notifier 触发 dio post，乐观更新 + 失败回滚
//
// 替换关系：v1 的 `feedProvider`（抢单池）已被本 provider 完全取代。
import 'dart:async';

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/invitation.dart';
import '../services/api_client.dart';

/// 轮询间隔（spec §4.1：5s）。
const Duration kInvitationPollInterval = Duration(seconds: 5);

/// `invitationsProvider` —— StreamProvider<List<Invitation>>。
///
/// 行为：
///   - 立即发一次 fetch
///   - 每 5s 重新 fetch
///   - 自动按 `isLive` 过滤（客户端侧 30s 过期判断）
final invitationsProvider = StreamProvider<List<Invitation>>((ref) {
  final controller = StreamController<List<Invitation>>.broadcast();

  Future<void> tick() async {
    try {
      final dio = ref.read(dioProvider);
      final resp = await dio.get<Map<String, dynamic>>(
        '/escorts/me/invitations',
      );
      final data = resp.data?['data'];
      final raw = (data is List) ? data : <dynamic>[];
      final list = raw
          .map((e) => Invitation.fromJson(e as Map<String, dynamic>))
          .where((inv) => inv.isLive)
          .toList();
      if (!controller.isClosed) controller.add(list);
    } on DioException catch (e) {
      if (!controller.isClosed) controller.addError(e);
    }
  }

  // 立即跑一次
  tick();

  // 5s 轮询；不依赖 WebSocket
  final timer = Timer.periodic(kInvitationPollInterval, (_) => tick());

  ref.onDispose(() {
    timer.cancel();
    controller.close();
  });

  return controller.stream;
});

/// 邀请操作结果 —— 用于 UI 层 SnackBar。
sealed class InvitationActionResult {
  const InvitationActionResult();
}

class InvitationActionSuccess extends InvitationActionResult {
  final int orderId;
  final String message;
  const InvitationActionSuccess(this.orderId, this.message);
}

class InvitationActionFailure extends InvitationActionResult {
  final int orderId;
  final String message;
  const InvitationActionFailure(this.orderId, this.message);
}

/// `confirmAcceptControllerProvider` —— 接受邀请（点击「确认接单」按钮）。
///
/// 后端契约：POST `/orders/{id}/confirm`（spec/2026-09-24-order-matching-redesign §4.1）。
final confirmAcceptControllerProvider =
    StateNotifierProvider.autoDispose<InvitationActionNotifier, AsyncValue<void>>(
  (ref) => InvitationActionNotifier(ref, '/orders/', 'confirm'),
);

/// `rejectAcceptControllerProvider` —— 拒接邀请。
///
/// 后端契约：POST `/orders/{id}/reject`（spec §4.1）。
final rejectAcceptControllerProvider =
    StateNotifierProvider.autoDispose<InvitationActionNotifier, AsyncValue<void>>(
  (ref) => InvitationActionNotifier(ref, '/orders/', 'reject'),
);

class InvitationActionNotifier
    extends StateNotifier<AsyncValue<void>> {
  final Ref ref;
  final String basePath; // e.g. '/orders/'
  final String action; // 'confirm' | 'reject'

  InvitationActionNotifier(this.ref, this.basePath, this.action)
      : super(const AsyncValue.data(null));

  Future<int> invoke(int orderId) async {
    state = const AsyncValue.loading();
    try {
      final dio = ref.read(dioProvider);
      await dio.post<dynamic>('$basePath$orderId/$action');
      state = const AsyncValue.data(null);
      // 触发邀请列表刷新（下次 5s tick 自动；这里手动 invalidate 立即触发）
      ref.invalidate(invitationsProvider);
      return orderId;
    } on DioException catch (e) {
      state = AsyncValue.error(e, StackTrace.current);
      rethrow;
    }
  }
}