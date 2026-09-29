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
//
// v1.2 扩展（E1）：
//   - 在同文件追加 5 个领域 API 模块类（ProfileApi / WalletApi / TrainingApi / ReviewApi / OrderApi）
//   - 每个模块提供 free-function 风格的 `XxxApi.get/post(...)`，接受 dio 作为参数（保持纯函数 + 易测）
//   - 与 patient-miniapp 的 `src/api/*.js` 模块化思路一致，便于按领域拆分
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

// ============================================================================
// E1: 5 个领域 API 模块（profile / wallet / training / review / order）
// ============================================================================
//
// 设计：每个模块以 `abstract class XxxApi + free functions` 形式暴露。
//   - 函数接受 `Dio` 作为第一参数（与 provider 解耦，方便单测注入 fake dio）
//   - 返回 `Map<String, dynamic>`（原始响应 data 字段）；DTO 解析在 caller 端用模型类
//   - 错误以 DioException 抛出（dio 已带 401 hook）
//   - 路径遵循后端 REST 规范：单数资源（`/users/me`、`/wallet`）+ 复数列表（`/wallet/transactions`、`/escorts/me/orders`）

/// Profile 模块 —— `GET /users/me` / `POST /users/real-name/auth`。
///
/// 用途：个人中心页 + 实名认证入口。
abstract class ProfileApi {
  /// 当前用户信息。
  static const String mePath = '/users/me';

  /// 提交实名认证（姓名 + 身份证号 + 证件照 URL）。
  static const String realNameAuthPath = '/users/real-name/auth';

  /// GET /users/me → {id, phone, nickname, avatar_url, real_name_verified, approved, ...}
  static Future<Map<String, dynamic>> me(Dio dio) async {
    final r = await dio.get<Map<String, dynamic>>(mePath);
    return r.data?['data'] as Map<String, dynamic>? ?? <String, dynamic>{};
  }

  /// POST /users/real-name/auth → {verified: bool}
  static Future<Map<String, dynamic>> submitRealName(
    Dio dio, {
    required String realName,
    required String idCard,
    String? idCardPhotoUrl,
  }) async {
    final r = await dio.post<Map<String, dynamic>>(
      realNameAuthPath,
      data: {
        'real_name': realName,
        'id_card': idCard,
        if (idCardPhotoUrl != null) 'id_card_photo_url': idCardPhotoUrl,
      },
    );
    return r.data?['data'] as Map<String, dynamic>? ?? <String, dynamic>{};
  }

  /// PATCH /users/me/nickname → 更新昵称（v1.2 暂不实现 UI，仅保留契约）。
  static Future<Map<String, dynamic>> updateNickname(
    Dio dio, {
    required String nickname,
  }) async {
    final r = await dio.patch<Map<String, dynamic>>(
      '/users/me/nickname',
      data: {'nickname': nickname},
    );
    return r.data?['data'] as Map<String, dynamic>? ?? <String, dynamic>{};
  }
}

/// Wallet 模块 —— `GET /wallet` / `GET /wallet/transactions` / `POST /escorts/me/wallet/withdraw`。
///
/// 用途：钱包页 + 提现流。
abstract class WalletApi {
  /// 钱包余额 + 冻结金额。
  static const String walletPath = '/wallet';

  /// 钱包流水（分页由 caller 传 page / size）。
  static const String transactionsPath = '/wallet/transactions';

  /// 陪诊师提现（escort 端）。
  static const String withdrawPath = '/escorts/me/wallet/withdraw';

  /// GET /wallet → {balance: num, frozen: num, currency: str}
  static Future<Map<String, dynamic>> getWallet(Dio dio) async {
    final r = await dio.get<Map<String, dynamic>>(walletPath);
    return r.data?['data'] as Map<String, dynamic>? ?? <String, dynamic>{};
  }

  /// GET /wallet/transactions?page=N&size=M → {list: [...], total: int}
  static Future<Map<String, dynamic>> getTransactions(
    Dio dio, {
    int page = 1,
    int size = 20,
  }) async {
    final r = await dio.get<Map<String, dynamic>>(
      transactionsPath,
      queryParameters: {'page': page, 'size': size},
    );
    return r.data?['data'] as Map<String, dynamic>? ?? <String, dynamic>{};
  }

  /// POST /escorts/me/wallet/withdraw → {withdraw_id: int, status: str}
  static Future<Map<String, dynamic>> withdraw(
    Dio dio, {
    required double amount,
    required String bankAccount,
  }) async {
    final r = await dio.post<Map<String, dynamic>>(
      withdrawPath,
      data: {'amount': amount, 'bank_account': bankAccount},
    );
    return r.data?['data'] as Map<String, dynamic>? ?? <String, dynamic>{};
  }
}

/// Training 模块 —— `GET /escorts/me/trainings`。
///
/// 用途：培训页（课程列表 + 进度 + 考核）。
abstract class TrainingApi {
  /// 陪诊师培训课程 + 进度 + 考核状态。
  static const String trainingsPath = '/escorts/me/trainings';

  /// GET /escorts/me/trainings → [{id, title, progress, status, ...}]
  static Future<List<dynamic>> listTrainings(Dio dio) async {
    final r = await dio.get<Map<String, dynamic>>(trainingsPath);
    final data = r.data?['data'];
    return (data is List) ? data : <dynamic>[];
  }

  /// POST /escorts/me/trainings/{id}/complete → 标记完成（v1.2 暂未上 UI，保留契约）。
  static Future<Map<String, dynamic>> completeTraining(
    Dio dio,
    int trainingId,
  ) async {
    final r = await dio.post<Map<String, dynamic>>(
      '/escorts/me/trainings/$trainingId/complete',
    );
    return r.data?['data'] as Map<String, dynamic>? ?? <String, dynamic>{};
  }
}

/// Review 模块 —— `GET /escorts/me/reviews`。
///
/// 用途：评价列表（个人中心子页 / Tab）。
abstract class ReviewApi {
  /// 陪诊师收到的评价。
  static const String reviewsPath = '/escorts/me/reviews';

  /// GET /escorts/me/reviews?page=N&size=M → {list: [...], total: int}
  static Future<Map<String, dynamic>> listReviews(
    Dio dio, {
    int page = 1,
    int size = 20,
  }) async {
    final r = await dio.get<Map<String, dynamic>>(
      reviewsPath,
      queryParameters: {'page': page, 'size': size},
    );
    return r.data?['data'] as Map<String, dynamic>? ?? <String, dynamic>{};
  }
}

/// Order 模块 —— `GET /escorts/me/orders` / `GET /orders/{id}`。
///
/// 用途：「我的订单」列表（按 status tab 过滤）+ 订单详情。
abstract class OrderApi {
  /// 陪诊师的订单列表（status 过滤）。
  static const String myOrdersPath = '/escorts/me/orders';

  /// 单个订单详情（含客户信息、6 节点状态、倒计时）。
  static String orderDetailPath(int id) => '/orders/$id';

  /// GET /escorts/me/orders?status=... → 瘦订单数组。
  static Future<List<dynamic>> listMyOrders(
    Dio dio, {
    String? status, // e.g. 'escort_pending_acceptance' / 'in_service' / 'completed'
    int page = 1,
    int size = 20,
  }) async {
    final r = await dio.get<Map<String, dynamic>>(
      myOrdersPath,
      queryParameters: {
        if (status != null) 'status': status,
        'page': page,
        'size': size,
      },
    );
    final data = r.data?['data'];
    if (data is List) return data;
    // 兼容后端返回 {list: [...], total: int} 格式
    if (data is Map<String, dynamic>) {
      final list = data['list'];
      if (list is List) return list;
    }
    return <dynamic>[];
  }

  /// GET /orders/{id} → 完整 Order DTO。
  static Future<Map<String, dynamic>> getOrder(Dio dio, int orderId) async {
    final r = await dio.get<Map<String, dynamic>>(orderDetailPath(orderId));
    return r.data?['data'] as Map<String, dynamic>? ?? <String, dynamic>{};
  }
}