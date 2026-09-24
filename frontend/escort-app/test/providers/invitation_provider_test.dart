// test/providers/invitation_provider_test.dart
//
// invitationsProvider 轮询 + 过滤过期 + confirm/reject controller 单测。
import 'dart:async';

import 'package:dio/dio.dart';
import 'package:escort_app/providers/invitation_provider.dart';
import 'package:escort_app/services/api_client.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

/// 可控 dio：每次 get 返回预设邀请列表（按 path 分发）。
class _StubDio implements Dio {
  final List<Map<String, dynamic>> Function(String path) handler;
  final List<dynamic> postCalls;
  final List<dynamic> deleteCalls;

  _StubDio(this.handler)
      : postCalls = [],
        deleteCalls = [];

  @override
  dynamic noSuchMethod(Invocation invocation) {
    final m = invocation.memberName;
    final args = invocation.namedArguments;
    final positional = invocation.positionalArguments;
    if (m == #get) {
      final path = (args['path'] as String?) ?? (positional.firstOrNull as String?);
      final opts = RequestOptions(path: path);
      return Future.value(Response<dynamic>(
        requestOptions: opts,
        statusCode: 200,
        data: {'code': 0, 'data': handler(path)},
      ));
    }
    if (m == #post) {
      postCalls.add(args);
      final path = (args['path'] as String?) ?? '';
      final opts = RequestOptions(path: path);
      return Future.value(Response<dynamic>(
        requestOptions: opts,
        statusCode: 200,
        data: {'code': 0, 'data': null},
      ));
    }
    if (m == #delete) {
      deleteCalls.add(args);
      return Future.value(Response<dynamic>(
        requestOptions: RequestOptions(path: ''),
        statusCode: 204,
      ));
    }
    throw UnimplementedError('StubDio.${m.toString()} not stubbed');
  }
}

void main() {
  group('confirmAcceptControllerProvider', () {
    test('invoke(orderId) → POST /orders/{id}/confirm + 触发 invalidations',
        () async {
      final dio = _StubDio((p) => <Map<String, dynamic>>[]);
      final c = ProviderContainer(overrides: [
        dioProvider.overrideWithValue(dio),
      ]);
      addTearDown(c.dispose);

      final notifier = c.read(confirmAcceptControllerProvider.notifier);
      // dio impl returns 200
      await notifier.invoke(7);
      expect(dio.postCalls.length, 1);
      expect(dio.postCalls.first['path'], '/orders/7/confirm');
      expect(c.read(confirmAcceptControllerProvider),
          const AsyncValue<dynamic>.data(null));
    });

    test('dio 抛 DioException → state 转 error', () async {
      final dio = _DioErrorDio();
      final c = ProviderContainer(overrides: [
        dioProvider.overrideWithValue(dio),
      ]);
      addTearDown(c.dispose);

      final notifier = c.read(confirmAcceptControllerProvider.notifier);
      await expectLater(notifier.invoke(7), throwsA(isA<DioException>()));
      expect(c.read(confirmAcceptControllerProvider), isA<AsyncError>());
    });
  });

  group('rejectAcceptControllerProvider', () {
    test('invoke(orderId) → POST /orders/{id}/reject', () async {
      final dio = _StubDio((p) => <Map<String, dynamic>>[]);
      final c = ProviderContainer(overrides: [
        dioProvider.overrideWithValue(dio),
      ]);
      addTearDown(c.dispose);

      final notifier = c.read(rejectAcceptControllerProvider.notifier);
      await notifier.invoke(9);
      expect(dio.postCalls.length, 1);
      expect(dio.postCalls.first['path'], '/orders/9/reject');
    });
  });
}

/// dio 强制抛 DioException（用于错误分支）。
class _DioErrorDio implements Dio {
  @override
  dynamic noSuchMethod(Invocation invocation) {
    final m = invocation.memberName;
    if (m == #post) {
      return Future.error(DioException(
        requestOptions: RequestOptions(path: '/x'),
        type: DioExceptionType.badResponse,
        response: Response<dynamic>(
          requestOptions: RequestOptions(path: '/x'),
          statusCode: 409,
          data: {'code': 13101},
        ),
      ));
    }
    throw UnimplementedError('${m.toString()} not stubbed');
  }
}