// test/pages/pending_page_test.dart
//
// PendingPage widget test —— 验证：
//   - 渲染审核中图标 + 倒计时
//   - 返回按钮 disabled
//   - PopScope canPop = false
import 'package:escort_app/pages/audit/pending_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

GoRouter _router() => GoRouter(
      initialLocation: '/audit/pending',
      routes: [
        GoRoute(
          path: '/audit/pending',
          builder: (c, s) => const PendingPage(),
        ),
      ],
    );

void main() {
  testWidgets('渲染审核中 + 倒计时', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(routerConfig: _router()),
      ),
    );
    expect(find.text('审核中'), findsWidgets);
    expect(find.byKey(const Key('pending_remaining')), findsOneWidget);
    expect(find.byIcon(Icons.hourglass_top), findsOneWidget);
  });

  testWidgets('返回按钮 onPressed = null（禁用）', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(routerConfig: _router()),
      ),
    );
    final iconBtn = tester.widget<IconButton>(
      find.byKey(const Key('pending_back')),
    );
    expect(iconBtn.onPressed, isNull);
  });

  testWidgets('PopScope canPop = false', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(routerConfig: _router()),
      ),
    );
    final pop = tester.widget<PopScope>(find.byType(PopScope));
    expect(pop.canPop, isFalse);
  });

  testWidgets('倒计时每秒递减', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(routerConfig: _router()),
      ),
    );
    final first = tester
        .widget<Text>(find.byKey(const Key('pending_remaining')))
        .data!;
    await tester.pump(const Duration(seconds: 2));
    final after = tester
        .widget<Text>(find.byKey(const Key('pending_remaining')))
        .data!;
    expect(first, isNot(after));
  });
}