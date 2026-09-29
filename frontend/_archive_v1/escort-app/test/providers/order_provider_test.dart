// test/providers/order_provider_test.dart
//
// ordersProvider (family) + orderDetailProvider (family) 单测。
import 'package:dio/dio.dart';
import 'package:escort_app/providers/order_provider.dart';
import 'package:escort_app/services/api_client.dart';
import 'package:escort_app/services/token_storage.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

class _StubDio implements Dio {
  final dynamic Function(RequestOptions) handler;
  final List<dynamic> getCalls;
  _StubDio(this.handler) : getCalls = [];

  @override
  dynamic noSuchMethod(Invocation invocation) {
    final m = invocation.memberName;
    final opts = (invocation.namedArguments[#options] as RequestOptions?) ??
        RequestOptions(path: '');
    if (m == #get) {
      getCalls.add(invocation.namedArguments);
      return Future.value(Response<dynamic>(
        requestOptions: opts,
        statusCode: 200,
        data: {'code': 0, 'data': handler(opts)},
      ));
    }
    throw UnimplementedError('${m.toString()} not stubbed');
  }
}

ProviderContainer _container(Dio dio) => ProviderContainer(overrides: [
      tokenStorageProvider.overrideWithValue(TokenStorage.forTest()),
      dioProvider.overrideWithValue(dio),
    ]);

void main() {
  group('ordersProvider', () {
    test('不带 status → 解析 List<OrderSummary>', () async {
      final dio = _StubDio((opts) {
        if (opts.path == '/escorts/me/orders') {
          return [
            {
              'id': 7,
              'hospital_name': '北京协和',
              'service_start_at_text': '明天 09:00',
              'amount': 300.0,
              'package_name': '半日陪诊',
              'status': 'in_service',
            }
          ];
        }
        return [];
      });
      final c = _container(dio);
      addTearDown(c.dispose);
      final list = await c.read(
        ordersProvider(const OrderListKey()).future,
      );
      expect(list.length, 1);
      expect(list.first.id, 7);
      expect(list.first.status, OrderStatus.inService);
      expect(dio.getCalls.first['path'], '/escorts/me/orders');
    });

    test('带 status=escort_pending_acceptance → query 注入', () async {
      final dio = _StubDio((opts) {
        if (opts.path == '/escorts/me/orders') return [];
        return [];
      });
      final c = _container(dio);
      addTearDown(c.dispose);
      await c.read(
        ordersProvider(const OrderListKey(
          status: 'escort_pending_acceptance',
        )).future,
      );
      expect(dio.getCalls.first['queryParameters']['status'],
          'escort_pending_acceptance');
    });

    test('响应 data 是 {list: [...]} 格式 → 解析 list', () async {
      final dio = _StubDio((opts) {
        if (opts.path == '/escorts/me/orders') {
          return {
            'list': [
              {
                'id': 1,
                'hospital_name': 'x',
                'service_start_at_text': 'x',
                'amount': 100.0,
                'package_name': 'x',
              }
            ],
            'total': 1,
          };
        }
        return [];
      });
      final c = _container(dio);
      addTearDown(c.dispose);
      final list = await c.read(
        ordersProvider(const OrderListKey(status: 'completed')).future,
      );
      expect(list.length, 1);
    });

    test('响应 data 非 List / Map → 空（兜底）', () async {
      final dio = _StubDio((opts) => null);
      final c = _container(dio);
      addTearDown(c.dispose);
      final list = await c.read(
        ordersProvider(const OrderListKey()).future,
      );
      expect(list, isEmpty);
    });
  });

  group('orderDetailProvider', () {
    test('按 orderId 拉 → 解析 Order', () async {
      final dio = _StubDio((opts) {
        if (opts.path == '/orders/7') {
          return {
            'id': 7,
            'patient_id': 11,
            'escort_id': 22,
            'selected_escort_id': 22,
            'hospital_name': '北京协和',
            'hospital_lat': 39.9,
            'hospital_lng': 116.4,
            'package_name': '半日陪诊',
            'service_start_at': '2026-09-25T09:00:00Z',
            'amount': 300.0,
            'status': 'in_service',
          };
        }
        return {};
      });
      final c = _container(dio);
      addTearDown(c.dispose);
      final o = await c.read(orderDetailProvider(7).future);
      expect(o.id, 7);
      expect(o.status, OrderStatus.inService);
      expect(o.hospitalName, '北京协和');
      expect(dio.getCalls.first['path'], '/orders/7');
    });

    test('orderId 变化 → 不同缓存 key', () async {
      final dio = _StubDio((opts) {
        final id = int.parse(opts.path.split('/').last);
        return {
          'id': id,
          'patient_id': 1,
          'hospital_name': 'x',
          'hospital_lat': 0,
          'hospital_lng': 0,
          'package_name': 'x',
          'service_start_at': '2026-09-25T09:00:00Z',
          'amount': 100.0,
          'status': 'in_service',
        };
      });
      final c = _container(dio);
      addTearDown(c.dispose);
      final o1 = await c.read(orderDetailProvider(1).future);
      final o2 = await c.read(orderDetailProvider(2).future);
      expect(o1.id, 1);
      expect(o2.id, 2);
    });
  });
}