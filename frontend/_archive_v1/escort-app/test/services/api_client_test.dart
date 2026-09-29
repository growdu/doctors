// test/services/api_client_test.dart
//
// api_client (buildDio) 单测 —— 用自定义 HttpClientAdapter 拦截请求验证：
//   - Authorization 头自动注入
//   - X-Trace-Id 头自动注入
//   - 401 触发 onUnauthorized 回调
import 'dart:async';
import 'dart:convert';

import 'package:dio/dio.dart';
import 'package:escort_app/services/api_client.dart';
import 'package:escort_app/services/token_storage.dart';
import 'package:escort_app/utils/trace.dart';
import 'package:flutter_test/flutter_test.dart';

/// 简单的 RequestOptions 捕获器（不依赖 dio_http_mock 等三方包）。
class _StubAdapter implements HttpClientAdapter {
  final void Function(RequestOptions options)? onFetch;
  final int statusCode;
  final dynamic body;

  RequestOptions? captured;

  _StubAdapter({
    this.onFetch,
    this.statusCode = 200,
    this.body,
  });

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<List<int>>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    captured = options;
    onFetch?.call(options);
    final bytes = utf8.encode(jsonEncode(body ?? {'code': 0, 'data': {}}));
    return ResponseBody.fromBytes(
      bytes,
      statusCode,
      headers: {
        'content-type': ['application/json'],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}

/// 401 响应专用 adapter。
class _AuthFailAdapter implements HttpClientAdapter {
  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<List<int>>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    final bytes = utf8.encode(jsonEncode({'code': 401}));
    return ResponseBody.fromBytes(
      bytes,
      401,
      headers: {'content-type': ['application/json']},
    );
  }

  @override
  void close({bool force = false}) {}
}

void main() {
  group('buildDio', () {
    test('baseUrl + 超时参数生效', () {
      final dio = buildDio(
        storage: TokenStorage.forTest(),
        baseUrl: 'https://x.example.com/api/v1',
      );
      expect(dio.options.baseUrl, 'https://x.example.com/api/v1');
      expect(dio.options.connectTimeout, const Duration(seconds: 10));
      expect(dio.options.receiveTimeout, const Duration(seconds: 15));
      expect(dio.options.responseType, ResponseType.json);
    });

    test('无 token → 不注入 Authorization 头', () async {
      final adapter = _StubAdapter();
      final dio = buildDio(
        storage: TokenStorage.forTest(),
        baseUrl: 'https://x',
      );
      dio.httpClientAdapter = adapter;
      await dio.get<dynamic>('/test');
      expect(adapter.captured?.headers.containsKey('Authorization'), isFalse);
    });

    test('有 token → 注入 Authorization: Bearer <token>', () async {
      final storage = TokenStorage.forTest();
      await storage.write('tk-abc-123');
      final adapter = _StubAdapter();
      final dio = buildDio(
        storage: storage,
        baseUrl: 'https://x',
      );
      dio.httpClientAdapter = adapter;
      await dio.get<dynamic>('/test');
      expect(adapter.captured?.headers['Authorization'], 'Bearer tk-abc-123');
    });

    test('自动注入 X-Trace-Id（格式 escort-{ms}-{rand}）', () async {
      final adapter = _StubAdapter();
      final dio = buildDio(
        storage: TokenStorage.forTest(),
        baseUrl: 'https://x',
      );
      dio.httpClientAdapter = adapter;
      await dio.get<dynamic>('/test');
      final tid = adapter.captured?.headers['X-Trace-Id'] as String?;
      expect(tid, isNotNull);
      expect(tid, startsWith('${apiBaseDefault.isEmpty ? '' : ''}')); // noop
      // 格式：escort-<ms>-<rand>
      expect(tid!.startsWith('escort-'), isTrue);
      final parts = tid.split('-');
      expect(parts.length, 3);
      expect(int.tryParse(parts[1]), isNotNull);
      expect(parts[2].length, 6);
    });

    test('trace 与 newTraceId 一致（前缀 escort- + 6 位 rand）', () async {
      final adapter = _StubAdapter();
      final dio = buildDio(
        storage: TokenStorage.forTest(),
        baseUrl: 'https://x',
      );
      dio.httpClientAdapter = adapter;
      await dio.get<dynamic>('/test');
      final tid = adapter.captured?.headers['X-Trace-Id'] as String;
      // 与 utils/trace.dart 的 newTraceId 格式一致
      expect(tid, newTraceId().substring(0, 7)); // 'escort' 同前缀
      expect(tid.split('-').last.length, 6); // rand6
    });

    test('401 响应 → onUnauthorized 回调触发 1 次', () async {
      var unauthCount = 0;
      final dio = buildDio(
        storage: TokenStorage.forTest(),
        baseUrl: 'https://x',
        onUnauthorized: () => unauthCount++,
      );
      dio.httpClientAdapter = _AuthFailAdapter();

      // dio 401 抛 DioException；用 try/catch 断言回调已触发
      try {
        await dio.get<dynamic>('/test');
      } on DioException {
        // 期望异常（401），但回调应已触发
      }
      expect(unauthCount, 1);
    });

    test('200 响应 → 不触发 onUnauthorized', () async {
      var unauthCount = 0;
      final dio = buildDio(
        storage: TokenStorage.forTest(),
        baseUrl: 'https://x',
        onUnauthorized: () => unauthCount++,
      );
      dio.httpClientAdapter = _StubAdapter(statusCode: 200);
      await dio.get<dynamic>('/test');
      expect(unauthCount, 0);
    });

    test('403 响应 → 不触发 onUnauthorized（仅 401）', () async {
      var unauthCount = 0;
      final dio = buildDio(
        storage: TokenStorage.forTest(),
        baseUrl: 'https://x',
        onUnauthorized: () => unauthCount++,
      );
      dio.httpClientAdapter = _StubAdapter(
        statusCode: 403,
        body: {'code': 13005, 'message': '请先实名'},
      );
      try {
        await dio.get<dynamic>('/test');
      } on DioException {
        // ignore
      }
      expect(unauthCount, 0);
    });

    test('delete token → 后续请求无 Authorization 头', () async {
      final storage = TokenStorage.forTest();
      await storage.write('tk-1');
      final adapter = _StubAdapter();
      final dio = buildDio(
        storage: storage,
        baseUrl: 'https://x',
      );
      dio.httpClientAdapter = adapter;
      await dio.get<dynamic>('/a');
      expect(adapter.captured?.headers['Authorization'], 'Bearer tk-1');

      await storage.delete();
      final adapter2 = _StubAdapter();
      dio.httpClientAdapter = adapter2;
      await dio.get<dynamic>('/b');
      expect(adapter2.captured?.headers.containsKey('Authorization'), isFalse);
    });

    test('POST 请求也注入 trace + auth', () async {
      final storage = TokenStorage.forTest();
      await storage.write('tk-post');
      final adapter = _StubAdapter();
      final dio = buildDio(
        storage: storage,
        baseUrl: 'https://x',
      );
      dio.httpClientAdapter = adapter;
      await dio.post<dynamic>('/p', data: {'k': 'v'});
      expect(adapter.captured?.headers['Authorization'], 'Bearer tk-post');
      expect(adapter.captured?.headers['X-Trace-Id'], isNotNull);
    });
  });
}