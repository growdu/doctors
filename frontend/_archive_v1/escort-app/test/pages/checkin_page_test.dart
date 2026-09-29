// test/pages/checkin_page_test.dart
//
// CheckinPage widget test —— 验证：
//   - 渲染订单号 + 定位显示（mock geolocator via platform channel）
//   - 签到按钮 → 跳订单详情
//   - 时间戳记录
import 'package:escort_app/pages/order_checkin/checkin_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

GoRouter _router() => GoRouter(
      initialLocation: '/checkin/42',
      routes: [
        GoRoute(
          path: '/checkin/:id',
          builder: (c, s) => CheckinPage(orderId: int.tryParse(s.pathParameters['id'] ?? '0') ?? 0),
        ),
        GoRoute(
          path: '/home/orders/:id',
          builder: (c, s) =>
              Scaffold(body: Text('ORDER_DETAIL_${s.pathParameters['id']}')),
        ),
      ],
    );

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() {
    const ch = MethodChannel('escort_app.geolocator');
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(ch, (call) async {
      if (call.method == 'getPosition') {
        return {
          'lat': 39.9042,
          'lng': 116.4074,
          'timestamp': DateTime.now().millisecondsSinceEpoch,
        };
      }
      return null;
    });
  });

  tearDown(() {
    const ch = MethodChannel('escort_app.geolocator');
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(ch, null);
  });

  testWidgets('渲染订单号 + 定位', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(routerConfig: _router()),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('订单 #42'), findsOneWidget);
    expect(find.byKey(const Key('checkin_pos')), findsOneWidget);
  });

  testWidgets('点击签到 → 跳订单详情', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(routerConfig: _router()),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('checkin_submit')));
    await tester.pump(const Duration(milliseconds: 600));
    await tester.pumpAndSettle();
    expect(find.text('ORDER_DETAIL_42'), findsOneWidget);
  });

  testWidgets('签到后显示时间戳', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(routerConfig: _router()),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('checkin_submit')));
    // mock 500ms 后已经设置 _checkedAt
    await tester.pump(const Duration(milliseconds: 100));
    expect(find.byKey(const Key('checkin_time')), findsOneWidget);
    // 避免 pending timer 阻塞测试结束
    await tester.pumpAndSettle(const Duration(milliseconds: 600));
  });
}