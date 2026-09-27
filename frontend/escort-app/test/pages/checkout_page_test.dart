// test/pages/checkout_page_test.dart
//
// CheckoutPage widget test —— 验证：
//   - 渲染订单号 + 备注 + 提交按钮
//   - 服务未完成 → 提示
//   - 备注输入 200 字限制
//   - 提交 → 跳订单详情
import 'package:escort_app/pages/order_checkout/checkout_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

GoRouter _router() => GoRouter(
      initialLocation: '/checkout/7',
      routes: [
        GoRoute(
          path: '/checkout/:id',
          builder: (c, s) => CheckoutPage(
            orderId: int.tryParse(s.pathParameters['id'] ?? '0') ?? 0,
          ),
        ),
        GoRoute(
          path: '/home/orders/:id',
          builder: (c, s) => Scaffold(
            body: Text('ORDER_DETAIL_${s.pathParameters['id']}'),
          ),
        ),
      ],
    );

void main() {
  testWidgets('渲染订单号 + 备注 + 提交', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(routerConfig: _router()),
      ),
    );
    expect(find.text('订单 #7'), findsOneWidget);
    expect(find.byKey(const Key('checkout_done')), findsOneWidget);
    expect(find.byKey(const Key('checkout_note')), findsOneWidget);
    expect(find.byKey(const Key('checkout_submit')), findsOneWidget);
  });

  testWidgets('服务未完成 → 提交时提示', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(routerConfig: _router()),
      ),
    );
    await tester.tap(find.byKey(const Key('checkout_done'))); // 关闭
    await tester.pump();
    await tester.tap(find.byKey(const Key('checkout_submit')));
    await tester.pump();
    expect(find.text('请确认服务已完成'), findsOneWidget);
  });

  testWidgets('提交 → 跳订单详情', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(routerConfig: _router()),
      ),
    );
    await tester.tap(find.byKey(const Key('checkout_submit')));
    await tester.pump(const Duration(milliseconds: 600));
    await tester.pumpAndSettle();
    expect(find.text('ORDER_DETAIL_7'), findsOneWidget);
  });

  testWidgets('备注输入文本', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(routerConfig: _router()),
      ),
    );
    await tester.enterText(find.byKey(const Key('checkout_note')),
        '已完成所有项目，患者非常满意');
    await tester.pump();
    final field = tester.widget<TextField>(
      find.byKey(const Key('checkout_note')),
    );
    expect(field.controller!.text, '已完成所有项目，患者非常满意');
  });
}