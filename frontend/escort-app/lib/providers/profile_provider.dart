// lib/providers/profile_provider.dart
//
// 个人中心 / 实名认证 provider —— GET /users/me + POST /users/real-name/auth。
//
// 设计要点：
//   - profileProvider：FutureProvider，一次性拉取（用户主动下拉刷新才重新拉）
//   - realNameAuthControllerProvider：StateNotifier，提交后刷新 profileProvider
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/profile.dart';
import '../services/api_client.dart';

/// `profileProvider` —— 当前用户资料（FutureProvider）。
///
/// 由「个人中心」页消费；下拉刷新走 `ref.invalidate(profileProvider)`。
final profileProvider = FutureProvider<Profile>((ref) async {
  final dio = ref.read(dioProvider);
  final data = await ProfileApi.me(dio);
  return Profile.fromJson(data);
});

/// 实名认证状态机（sealed）：空 / 提交中 / 成功 / 失败。
sealed class RealNameAuthState {
  const RealNameAuthState();
}

class RealNameAuthIdle extends RealNameAuthState {
  const RealNameAuthIdle();
}

class RealNameAuthSubmitting extends RealNameAuthState {
  const RealNameAuthSubmitting();
}

class RealNameAuthSuccess extends RealNameAuthState {
  final bool verified;
  const RealNameAuthSuccess(this.verified);
}

class RealNameAuthFailure extends RealNameAuthState {
  final String message;
  const RealNameAuthFailure(this.message);
}

/// 实名认证 controller provider。
final realNameAuthControllerProvider = StateNotifierProvider.autoDispose<
    RealNameAuthController, RealNameAuthState>(
  (ref) => RealNameAuthController(ref),
);

/// 实名认证控制器。
class RealNameAuthController extends StateNotifier<RealNameAuthState> {
  final Ref ref;

  RealNameAuthController(this.ref) : super(const RealNameAuthIdle());

  /// 提交实名认证。
  Future<bool> submit({
    required String realName,
    required String idCard,
    String? idCardPhotoUrl,
  }) async {
    state = const RealNameAuthSubmitting();
    try {
      final dio = ref.read(dioProvider);
      final r = await ProfileApi.submitRealName(
        dio,
        realName: realName,
        idCard: idCard,
        idCardPhotoUrl: idCardPhotoUrl,
      );
      final verified = (r['verified'] as bool?) ?? true;
      state = RealNameAuthSuccess(verified);
      // 触发 profileProvider 刷新，下一次 watch 拿到最新 real_name_verified
      ref.invalidate(profileProvider);
      return verified;
    } on DioException catch (e) {
      final msg = e.response?.data is Map
          ? ((e.response!.data as Map)['message']?.toString() ?? e.toString())
          : e.toString();
      state = RealNameAuthFailure(msg);
      rethrow;
    }
  }

  /// 重置为空（关闭弹窗时调用）。
  void reset() => state = const RealNameAuthIdle();
}

/// 退出登录 controller provider（个人中心页用）。
///
/// 行为：调 AuthNotifier.logout() 清 token + 状态。
/// 这里不复用 logout 直接调；保留 controller 抽象便于 UI 层订阅状态变化。
final logoutControllerProvider =
    StateNotifierProvider.autoDispose<LogoutController, AsyncValue<void>>(
  (ref) => LogoutController(ref),
);

class LogoutController extends StateNotifier<AsyncValue<void>> {
  final Ref ref;

  LogoutController(this.ref) : super(const AsyncValue.data(null));

  Future<void> logout() async {
    state = const AsyncValue.loading();
    try {
      // 走 authProvider.notifier.logout() —— 包含清 token + 切 unauthenticated
      // 通过 dynamic 调用避免循环依赖（auth_provider 不依赖本文件）。
      final auth = ref.read(authProvider);
      // 触发 logout；不强类型 import，依赖全局 authProvider
      final container = ref.container;
      await container.read(authProvider.notifier).logout();
      state = const AsyncValue.data(null);
    } on DioException catch (e) {
      state = AsyncValue.error(e, StackTrace.current);
      rethrow;
    }
  }
}