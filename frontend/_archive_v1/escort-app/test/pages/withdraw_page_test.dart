// test/pages/withdraw_page_test.dart
//
// WithdrawPage widget test —— 验证：
//   - 渲染余额 + 3 张银行卡 + 提现按钮
//   - 校验金额（< 1 / > 余额）
//   - 切换银行卡
//   - 提交 → 跳钱包页 + snackbar
import 'package:escort_app/pages/wallet/withdraw_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

GoRouter _router() => GoRouter(
      initialLocation: '/wallet/withdraw',
      routes: [
        GoRoute(
          path: '/wallet/withdraw',
          builder: (c, s) => const WithdrawPage(),
        ),
        GoRoute(
          path: '/home/wallet',
          builder: (c, s) => const Scaffold(body: Text('WALLET_ROUTE')),
        ),
      ],
    );

void main() {
  testWidgets('渲染余额 + 3 张银行卡 + 提现按钮', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(routerConfig: _router()),
      ),
    );
    expect(find.byKey(const Key('withdraw_balance')), findsOneWidget);
    expect(find.byKey(const Key('withdraw_amount')), findsOneWidget);
    expect(find.byKey(const Key('withdraw_card_c1')), findsOneWidget);
    expect(find.byKey(const Key('withdraw_card_c2')), findsOneWidget);
    expect(find.byKey(const Key('withdraw_card_c3')), findsOneWidget);
    expect(find.byKey(const Key('withdraw_submit')), findsOneWidget);
  });

  testWidgets('金额 0 → 提交时报错', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(routerConfig: _router()),
      ),
    );
    await tester.enterText(find.byKey(const Key('withdraw_amount')), '0');
    await tester.pump();
    await tester.tap(find.byKey(const Key('withdraw_submit')));
    await tester.pump();
    expect(find.textContaining('请输入有效金额'), findsOneWidget);
  });

  testWidgets('切换银行卡（点击 c2 → 选中）', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(routerConfig: _router()),
      ),
    );
    await tester.tap(find.byKey(const Key('withdraw_card_c2')));
    await tester.pumpAndSettle();
    // c2 的图标应为 checked
    expect(find.descendant(
      of: find.byKey(const Key('withdraw_card_c2')),
      matching: find.byIcon(Icons.radio_button_checked),
    ), findsOneWidget);
    // c1 的图标应为 unchecked
    expect(find.descendant(
      of: find.byKey(const Key('withdraw_card_c1')),
      matching: find.byIcon(Icons.radio_button_unchecked),
    ), findsOneWidget);
  });

  testWidgets('合法金额 → 提交跳钱包', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(routerConfig: _router()),
      ),
    );
    await tester.enterText(find.byKey(const Key('withdraw_amount')), '100');
    await tester.pump();
    await tester.tap(find.byKey(const Key('withdraw_submit')));
    await tester.pump(const Duration(milliseconds: 600));
    await tester.pumpAndSettle();
    expect(find.text('WALLET_ROUTE'), findsOneWidget);
  });

  testWidgets('提现 snackbar 显示金额', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(routerConfig: _router()),
      ),
    );
    await tester.enterText(find.byKey(const Key('withdraw_amount')), '50');
    await tester.pump();
    await tester.tap(find.byKey(const Key('withdraw_submit')));
    // mock 500ms 后弹出 snackbar
    await tester.pump(const Duration(milliseconds: 600));
    expect(find.text('已申请提现 ¥50.00'), findsOneWidget);
    // 等 snackbar 退出，避免 pending timer
    await tester.pumpAndSettle(const Duration(seconds: 5));
  });
}