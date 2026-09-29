// test/providers/availability_provider_test.dart
//
// availabilityProvider + create / delete controller 单测。
import 'dart:async';

import 'package:dio/dio.dart';
import 'package:escort_app/models/availability.dart';
import 'package:escort_app/providers/availability_provider.dart';
import 'package:escort_app/services/api_client.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

/// 可控 dio：根据 path 分发返回。
class _StubDio implements Dio {
  final List<dynamic> postCalls;
  final List<dynamic> deleteCalls;
  final List<Map<String, dynamic>> Function(String path) handler;

  _StubDio(this.handler)
      : postCalls = [],
        deleteCalls = [];

  @override
  dynamic noSuchMethod(Invocation invocation) {
    final m = invocation.memberName;
    if (m == #get) {
      final args = invocation.namedArguments;
      final path = (args['path'] as String?) ?? '';
      final opts = RequestOptions(path: path);
      return Future.value(Response<dynamic>(
        requestOptions: opts,
        statusCode: 200,
        data: {'code': 0, 'data': handler(path)},
      ));
    }
    if (m == #post) {
      postCalls.add(invocation.namedArguments);
      final args = invocation.namedArguments;
      final path = (args['path'] as String?) ?? '';
      final opts = RequestOptions(path: path);
      // 返回构造的 Availability
      final created = Availability.fromJson({
        'id': 99,
        'escort_id': 22,
        'start_at': '2026-09-25T14:00:00Z',
        'end_at': '2026-09-25T18:00:00Z',
        'status': 'available',
      });
      return Future.value(Response<dynamic>(
        requestOptions: opts,
        statusCode: 201,
        data: {'code': 0, 'data': created.toJson()},
      ));
    }
    if (m == #delete) {
      deleteCalls.add(invocation.namedArguments);
      return Future.value(Response<dynamic>(
        requestOptions: RequestOptions(path: ''),
        statusCode: 204,
      ));
    }
    throw UnimplementedError('StubDio.${m.toString()} not stubbed');
  }
}

void main() {
  group('availabilityProvider', () {
    test('拉取列表 → 解析 Availability[]', () async {
      final dio = _StubDio((p) {
        if (p == '/escorts/me/availabilities') {
          return [
            {
              'id': 1,
              'escort_id': 22,
              'start_at': '2026-09-25T14:00:00Z',
              'end_at': '2026-09-25T18:00:00Z',
              'status': 'available',
            },
            {
              'id': 2,
              'escort_id': 22,
              'start_at': '2026-09-26T10:00:00Z',
              'end_at': '2026-09-26T14:00:00Z',
              'status': 'booked',
              'order_id': 7,
            },
          ];
        }
        return [];
      });
      final c = ProviderContainer(overrides: [
        dioProvider.overrideWithValue(dio),
      ]);
      addTearDown(c.dispose);

      final list = await c.read(availabilityProvider.future);
      expect(list.length, 2);
      expect(list[0].status, AvailabilityStatus.available);
      expect(list[1].status, AvailabilityStatus.booked);
      expect(list[1].orderId, 7);
    });

    test('data 字段非 list → 空列表（兜底）', () async {
      final dio = _StubDio((p) => <Map<String, dynamic>>[]);
      final c = ProviderContainer(overrides: [
        dioProvider.overrideWithValue(dio),
      ]);
      addTearDown(c.dispose);
      final list = await c.read(availabilityProvider.future);
      expect(list, isEmpty);
    });
  });

  group('createAvailabilityControllerProvider', () {
    test('invoke(start, end) → POST + state 携带 Availability', () async {
      final dio = _StubDio((p) => <Map<String, dynamic>>[]);
      final c = ProviderContainer(overrides: [
        dioProvider.overrideWithValue(dio),
      ]);
      addTearDown(c.dispose);

      final notifier = c.read(createAvailabilityControllerProvider.notifier);
      final result = await notifier.invoke(
        DateTime.utc(2026, 9, 25, 14),
        DateTime.utc(2026, 9, 25, 18),
      );
      expect(result.id, 99);
      expect(result.status, AvailabilityStatus.available);
      expect(dio.postCalls.length, 1);
      expect(dio.postCalls.first['path'], '/escorts/me/availabilities');
    });
  });

  group('deleteAvailabilityControllerProvider', () {
    test('invoke(id) → DELETE /escorts/me/availabilities/{id}', () async {
      final dio = _StubDio((p) => <Map<String, dynamic>>[]);
      final c = ProviderContainer(overrides: [
        dioProvider.overrideWithValue(dio),
      ]);
      addTearDown(c.dispose);

      final notifier = c.read(deleteAvailabilityControllerProvider.notifier);
      await notifier.invoke(5);
      expect(dio.deleteCalls.length, 1);
      expect(dio.deleteCalls.first['path'], '/escorts/me/availabilities/5');
    });
  });
}