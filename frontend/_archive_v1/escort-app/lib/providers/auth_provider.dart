// lib/providers/auth_provider.dart
//
// 认证状态机（sealed）+ AuthNotifier（StateNotifier）—— spec §8 路由守卫依赖此状态。
//
// 状态机：
//   AuthInitial → AuthUnauthenticated（无 token 或 me 失败）
//   AuthInitial → AuthAuthenticated（bootstrap 读到 token + me 成功 / 登录成功）
//   AuthAuthenticated → AuthUnauthenticated（logout / 401 handler）
//
// 设计要点：
//   - `tokenStorageProvider` 是 abstract，main.dart 启动时用 buildDio(...) + FlutterSecureStorage 注入
//   - 401 回调里 `authProvider.notifier.onUnauthorized()` 清状态 + 清 token
//   - `AuthState` 是 sealed，配合 `switch` 模式匹配可获得 exhaustiveness 检查
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../services/api_client.dart';
import '../services/token_storage.dart';

/// 认证状态（sealed：AuthInitial / AuthUnauthenticated / AuthAuthenticated）。
sealed class AuthState {
  const AuthState();

  /// 是否已登录（用于 authGuardProvider）。
  bool get isAuthed => this is AuthAuthenticated;

  /// 已登录且实名（用于 realNameGuardProvider）。
  bool get isRealNameVerified =>
      this is AuthAuthenticated && (this as AuthAuthenticated).realNameVerified;

  /// 已登录且通过审核（用于 approvedGuardProvider）。
  bool get isApproved =>
      this is AuthAuthenticated && (this as AuthAuthenticated).approved;
}

/// 初始态（app 启动中，bootstrap 尚未完成）。
class AuthInitial extends AuthState {
  const AuthInitial();
}

/// 未登录（bootstrap 发现无 token / me 失败 / logout）。
class AuthUnauthenticated extends AuthState {
  const AuthUnauthenticated();
}

/// 已登录。
class AuthAuthenticated extends AuthState {
  final int userId;
  final String phone;
  final String role; // "patient" | "escort" | "admin"
  final bool realNameVerified;
  final bool approved;

  const AuthAuthenticated({
    required this.userId,
    required this.phone,
    required this.role,
    required this.realNameVerified,
    required this.approved,
  });

  AuthAuthenticated copyWith({
    int? userId,
    String? phone,
    String? role,
    bool? realNameVerified,
    bool? approved,
  }) =>
      AuthAuthenticated(
        userId: userId ?? this.userId,
        phone: phone ?? this.phone,
        role: role ?? this.role,
        realNameVerified: realNameVerified ?? this.realNameVerified,
        approved: approved ?? this.approved,
      );
}

/// `tokenStorageProvider` —— 由 main.dart 启动时 override。
///
/// 默认 throw UnimplementedError 避免运行时 silent 使用未初始化实例。
final tokenStorageProvider = Provider<TokenStorage>((ref) {
  throw UnimplementedError(
    'tokenStorageProvider must be overridden in ProviderScope (main.dart)',
  );
});

/// 401 handler —— 当 dio 收到 401 时调用。
///
/// main.dart 用 `ref.read(dioProvider).interceptors` 链路上注册；或在 buildDio 时
/// 注入 onUnauthorized 回调。
typedef UnauthorizedHandler = void Function();

/// 全局 AuthProvider。
final authProvider =
    StateNotifierProvider<AuthNotifier, AuthState>((ref) {
  return AuthNotifier(ref);
});

/// AuthNotifier —— 负责 token 持久化 + 登录态机切换。
class AuthNotifier extends StateNotifier<AuthState> {
  final Ref ref;

  AuthNotifier(this.ref) : super(const AuthInitial());

  /// 启动时调用：检查 token → 调 /me 校验 → 设置状态。
  Future<void> bootstrap() async {
    final storage = ref.read(tokenStorageProvider);
    final t = await storage.read();
    if (t == null || t.isEmpty) {
      state = const AuthUnauthenticated();
      return;
    }

    // 简化：v1 直接根据 token 假定已认证；真实 /me 校验在后续 plan 接入。
    // 这里保守地用 tokenStorage 恢复「占位已认证」状态，待 /me API 落地后再校验字段。
    // 当前实现：me() 失败则降级为 unauthenticated（防止 stale token 残留）。
    try {
      final dio = ref.read(dioProvider);
      final resp = await dio.get<Map<String, dynamic>>('/auth/me');
      final data = resp.data?['data'] as Map<String, dynamic>?;
      if (data == null) {
        await storage.delete();
        state = const AuthUnauthenticated();
        return;
      }
      state = AuthAuthenticated(
        userId: data['id'] as int,
        phone: data['phone'] as String,
        role: data['role'] as String,
        realNameVerified: (data['real_name_verified'] as bool?) ?? false,
        approved: (data['approved'] as bool?) ?? false,
      );
    } on DioException {
      await storage.delete();
      state = const AuthUnauthenticated();
    }
  }

  /// 短信码登录 —— 调 `POST /auth/login/sms`，写 token，更新状态。
  Future<void> loginByPhone({
    required String phone,
    required String code,
  }) async {
    final dio = ref.read(dioProvider);
    final resp = await dio.post<Map<String, dynamic>>(
      '/auth/login/sms',
      data: {'phone': phone, 'code': code},
    );
    final data = resp.data?['data'] as Map<String, dynamic>?;
    if (data == null) {
      throw StateError('loginByPhone: empty response');
    }
    final token = data['access_token'] as String;
    await ref.read(tokenStorageProvider).write(token);
    final user = data['user'] as Map<String, dynamic>;
    state = AuthAuthenticated(
      userId: user['id'] as int,
      phone: user['phone'] as String,
      role: user['role'] as String,
      realNameVerified: (user['real_name_verified'] as bool?) ?? false,
      approved: (user['approved'] as bool?) ?? false,
    );
  }

  /// 微信登录 —— 调 `POST /auth/login/wx`（body: {code: wxCode}）。
  ///
  /// 后端用 wx.code 换 openid → upsert user → 返回 JWT。
  /// v1.2 简化：直接传 wxCode 字符串。
  Future<void> loginByWx({required String wxCode}) async {
    final dio = ref.read(dioProvider);
    final resp = await dio.post<Map<String, dynamic>>(
      '/auth/login/wx',
      data: {'code': wxCode},
    );
    final data = resp.data?['data'] as Map<String, dynamic>?;
    if (data == null) {
      throw StateError('loginByWx: empty response');
    }
    final token = data['access_token'] as String;
    await ref.read(tokenStorageProvider).write(token);
    final user = data['user'] as Map<String, dynamic>;
    state = AuthAuthenticated(
      userId: user['id'] as int,
      phone: (user['phone'] as String?) ?? '',
      role: user['role'] as String,
      realNameVerified: (user['real_name_verified'] as bool?) ?? false,
      approved: (user['approved'] as bool?) ?? false,
    );
  }

  /// 登出 —— 清 token + 状态切 unauthenticated。
  Future<void> logout() async {
    await ref.read(tokenStorageProvider).delete();
    state = const AuthUnauthenticated();
  }

  /// 401 钩子 —— dio 收到 401 时由外部回调；这里清 token + 切 unauthenticated。
  Future<void> onUnauthorized() async {
    await ref.read(tokenStorageProvider).delete();
    state = const AuthUnauthenticated();
  }
}