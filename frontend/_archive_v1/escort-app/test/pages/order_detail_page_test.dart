// test/pages/order_detail_page_test.dart
//
// OrderDetailPage widget test —— 校验：
//   - 渲染状态头部 + 6 节点进度 + 客户信息 + 订单信息
//   - status=escort_pending_acceptance 显示「确认接单 / 拒接」按钮
//   - 点击「确认接单」调 confirmAcceptController
//   - 点击「拒接」弹确认对话框
import 'package:dio/dio.dart';
import 'package:escort_app/models/order.dart';
import 'package:escort_app/pages/order_detail/order_detail_page.dart';
import 'package:escort_app/providers/order_provider.dart';
import 'package:escort_app/services/api_client.dart';
import 'package:escort_app/services/token_storage.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

class _StubDio implements Dio {
  final dynamic Function(RequestOptions) handler;
  final List<dynamic> postCalls;
  _StubDio(this.handler) : postCalls = [];

  @override
  dynamic noSuchMethod(Invocation invocation) {
    final m = invocation.memberName;
    final opts = (invocation.namedArguments[#options] as RequestOptions?) ??
        RequestOptions(path: '');
    if (m == #get || m == #post) {
      postCalls.add(invocation.namedArguments);
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

Widget _wrap(ProviderContainer c, int orderId) => UncontrolledProviderScope(
      container: c,
      child: MaterialApp(home: OrderDetailPage(orderId: orderId)),
    );

Map<String, dynamic> _orderJson(int id, String status) => {
      'id': id,
      'patient_id': 11,
      'hospital_name': '北京协和医院',
      'hospital_lat': 39.9,
      'hospital_lng': 116.4,
      'package_name': '半日陪诊',
      'service_start_at': '2026-09-25T09:00:00Z',
      'amount': 300.0,
      'status': status,
    };

void main() {
  group('OrderDetailPage', () {
    testWidgets('in_service 状态：渲染头部 + 进度 + 客户 + 订单信息',
        (tester) async {
      final dio = _StubDio((opts) {
        if (opts.path == '/orders/7') return _orderJson(7, 'in_service');
        return {};
      });
      final c = _container(dio);
      addTearDown(c.dispose);
      unawaited(c.read(orderDetailProvider(7).future));
      await tester.pumpWidget(_wrap(c, 7));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));
      expect(find.text('服务中'), findsWidgets);
      expect(find.text('订单进度'), findsOneWidget);
      expect(find.text('客户信息'), findsOneWidget);
      expect(find.text('订单信息'), findsOneWidget);
      expect(find.text('北京协和医院'), findsOneWidget);
      expect(find.text('半日陪诊'), findsOneWidget);
      expect(find.text('¥300.00'), findsOneWidget);
      // in_service 时不显示「确认接单 / 拒接」
      expect(find.text('确认接单'), findsNothing);
      expect(find.text('拒接'), findsNothing);
    });

    testWidgets('escort_pending_acceptance 状态：显示「确认接单 + 拒接」',
        (tester) async {
      final dio = _StubDio((opts) {
        if (opts.path == '/orders/7') {
          return {
            ..._orderJson(7, 'escort_pending_acceptance'),
            'escort_pending_expire_at':
                DateTime.now().add(const Duration(seconds: 30)).toIso8601String(),
          };
        }
        return {};
      });
      final c = _container(dio);
      addTearDown(c.dispose);
      unawaited(c.read(orderDetailProvider(7).future));
      await tester.pumpWidget(_wrap(c, 7));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));
      expect(find.text('确认接单'), findsOneWidget);
      expect(find.text('拒接'), findsOneWidget);
      // CountdownBadge 显示
      expect(find.textContaining('确认剩余'), findsOneWidget);
    });

    testWidgets('点击「确认接单」调 confirm controller', (tester) async {
      final dio = _StubDio((opts) {
        if (opts.path == '/orders/7') {
          return {
            ..._orderJson(7, 'escort_pending_acceptance'),
            'escort_pending_expire_at':
                DateTime.now().add(const Duration(seconds: 30)).toIso8601String(),
          };
        }
        return {};
      });
      final c = _container(dio);
      addTearDown(c.dispose);
      unawaited(c.read(orderDetailProvider(7).future));
      await tester.pumpWidget(_wrap(c, 7));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));
      await tester.tap(find.text('确认接单'));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));
      final confirmCalls = dio.postCalls
          .where((c) => (c as Map)['path'] == '/orders/7/confirm');
      expect(confirmCalls, isNotEmpty);
    });

    testWidgets('点击「拒接」弹确认对话框', (tester) async {
      final dio = _StubDio((opts) {
        if (opts.path == '/orders/7') {
          return {
            ..._orderJson(7, 'escort_pending_acceptance'),
            'escort_pending_expire_at':
                DateTime.now().add(const Duration(seconds: 30)).toIso8601String(),
          };
        }
        return {};
      });
      final c = _container(dio);
      addTearDown(c.dispose);
      unawaited(c.read(orderDetailProvider(7).future));
      await tester.pumpWidget(_wrap(c, 7));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));
      await tester.tap(find.text('拒接'));
      await tester.pump();
      expect(find.text('拒接订单'), findsOneWidget);
      expect(find.text('患者可重新选择其他陪诊师'), findsOneWidget);
    });

    testWidgets('6 节点进度：6 个步骤标签都渲染', (tester) async {
      final dio = _StubDio((opts) {
        if (opts.path == '/orders/7') return _orderJson(7, 'accepted');
        return {};
      });
      final c = _container(dio);
      addTearDown(c.dispose);
      unawaited(c.read(orderDetailProvider(7).future));
      await tester.pumpWidget(_wrap(c, 7));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));
      // 6 个节点 label
      expect(find.text('已支付'), findsOneWidget);
      expect(find.text('匹配中'), findsOneWidget);
      expect(find.text('选人中'), findsOneWidget);
      expect(find.text('待确认'), findsOneWidget);
      expect(find.text('已接单'), findsWidgets); // progress + status
      expect(find.text('服务中'), findsOneWidget);
    });

    testWidgets('error 态显示错误 + 重试按钮', (tester) async {
      final dio = _ErrDio();
      final c = _container(dio);
      addTearDown(c.dispose);
      unawaited(c.read(orderDetailProvider(7).future));
      await tester.pumpWidget(_wrap(c, 7));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));
      expect(find.textContaining('加载失败'), findsOneWidget);
      expect(find.text('重试'), findsOneWidget);
    });

    testWidgets('订单金额为 0 → 显示 ¥0.00', (tester) async {
      final dio = _StubDio((opts) {
        if (opts.path == '/orders/7') {
          return {
            ..._orderJson(7, 'paid'),
            'amount': 0.0,
          };
        }
        return {};
      });
      final c = _container(dio);
      addTearDown(c.dispose);
      unawaited(c.read(orderDetailProvider(7).future));
      await tester.pumpWidget(_wrap(c, 7));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));
      expect(find.text('¥0.00'), findsOneWidget);
    });
  });
}

class _ErrDio implements Dio {
  @override
  dynamic noSuchMethod(Invocation invocation) {
    final m = invocation.memberName;
    if (m == #get) {
      return Future.error(DioException(
        requestOptions: RequestOptions(path: '/x'),
        type: DioExceptionType.badResponse,
        response: Response<dynamic>(
          requestOptions: RequestOptions(path: '/x'),
          statusCode: 404,
        ),
      ));
    }
    throw UnimplementedError('${m.toString()} not stubbed');
  }
}

void unawaited(Future<void> _) {}