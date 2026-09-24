// lib/services/api_client.dart
//
// dio 封装 —— BaseOptions + 3 个拦截器：Auth (Bearer) / Trace (X-Trace-Id) / 401 handler。
//
// 设计：
//   - 单一 `buildDio` 函数（free function），无全局副作用；测试可多次构建
//   - `tokenStorage` 通过参数注入（避免硬 import singleton）
//   - 401 回调（onUnauthorized）由调用方订阅：典型用法是通知 AuthNotifier 清状态 + 跳登录
//   - 不引 dio_cache / dio_smart_retry 等三方插件（spec 边界）
//
// Riverpod 集成：
//   - `dioProvider` 注入 TokenStorage；上层（auth_provider / invitation_provider）通过
//     `ref.read(dioProvider)` 拿到已配置好的 dio 实例
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../core/constants.dart';
import '../utils/trace.dart';
import 'token_storage.dart';

/// 构建 dio（BaseOptions + Auth + Trace + 401 handler 拦截器）。
///
/// 参数：
///   - [storage]：JWT 读取来源（生产：FlutterSecureStorage；测试：fake）
///   - [baseUrl]：API base；通常来自 `resolveApiBase()`
///   - [onUnauthorized]：401 响应时回调（清 token + 跳登录的钩子）
Dio buildDio({
  required TokenStorage storage,
  required String baseUrl,
  void Function()? onUnauthorized,
}) {
  final dio = Dio(BaseOptions(
    baseUrl: baseUrl,
    connectTimeout: const Duration(seconds: 10),
    receiveTimeout: const Duration(seconds: 15),
    contentType: Headers.jsonContentType,
    responseType: ResponseType.json,
  ));

  // Auth 拦截器：注入 Authorization: Bearer <token>
  dio.interceptors.add(InterceptorsWrapper(
    onRequest: (options, handler) async {
      final t = await storage.read();
      if (t != null && t.isNotEmpty) {
        options.headers['Authorization'] = 'Bearer $t';
      }
      handler.next(options);
    },
  ));

  // Trace 拦截器：每个请求打 escort-{ms}-{rand6}
  dio.interceptors.add(InterceptorsWrapper(
    onRequest: (options, handler) {
      options.headers['X-Trace-Id'] = newTraceId();
      handler.next(options);
    },
  ));

  // 401 handler：onError 钩子；不拦截异常向上传播（业务侧自行决定是否重试）。
  dio.interceptors.add(InterceptorsWrapper(
    onResponse: (r, handler) => handler.next(r),
    onError: (e, handler) {
      if (e.response?.statusCode == 401) {
        onUnauthorized?.call();
      }
      handler.next(e);
    },
  ));

  return dio;
}

/// Riverpod provider —— 持有全局 dio 实例（单例）。
///
/// 依赖 `tokenStorageProvider`（在 auth_provider 中定义并 override）。
/// `main.dart` 启动时覆盖 `tokenStorageProvider` 注入真实 FlutterSecureStorage。
final dioProvider = Provider<Dio>((ref) {
  throw UnimplementedError(
    'dioProvider must be overridden in ProviderScope (with buildDio + tokenStorage)',
  );
});

/// 默认 base URL —— 从 `--dart-define=API_BASE=...` 解析（空则用 dev 默认）。
String defaultApiBase() => resolveApiBase();

/// kApiBaseDefault / kApiBaseProd 暴露给 main.dart（用于 main 中 override）。
const String apiBaseDefault = kApiBaseDefault;
const String apiBaseProd = kApiBaseProd;