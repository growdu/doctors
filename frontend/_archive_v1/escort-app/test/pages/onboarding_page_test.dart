// test/pages/onboarding_page_test.dart
//
// OnboardingPage widget test —— 验证：
//   - 渲染 5 步骤（身份 / 健康证 / 培训 / 协议 / 提交审核）
//   - PageView 切换 + 上一步/下一步按钮
//   - 步骤 0 校验：姓名 + 身份证号
//   - 最后一步「提交审核」→ /audit/pending
import 'package:escort_app/pages/onboarding/onboarding_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

GoRouter _router() => GoRouter(
      initialLocation: '/onboarding',
      routes: [
        GoRoute(
          path: '/onboarding',
          builder: (c, s) => const OnboardingPage(),
        ),
        GoRoute(
          path: '/audit/pending',
          builder: (c, s) =>
              const Scaffold(body: Text('AUDIT_PENDING_ROUTE')),
        ),
      ],
    );

Future<void> _fillStep0(WidgetTester t) async {
  await t.enterText(find.byKey(const Key('onb_name')), '张三');
  await t.enterText(find.byKey(const Key('onb_idno')), '110101199001011234');
  await t.pump();
}

void main() {
  testWidgets('渲染 Step 0（身份信息）', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(routerConfig: _router()),
      ),
    );
    expect(find.text('身份信息 (1/5)'), findsOneWidget);
    expect(find.byKey(const Key('onb_name')), findsOneWidget);
    expect(find.byKey(const Key('onb_idno')), findsOneWidget);
    expect(find.byKey(const Key('onb_next')), findsOneWidget);
  });

  testWidgets('上一步按钮在 Step 0 不显示', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(routerConfig: _router()),
      ),
    );
    expect(find.byKey(const Key('onb_prev')), findsNothing);
  });

  testWidgets('Step 0 未填 → 下一步按钮 disabled', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(routerConfig: _router()),
      ),
    );
    final btn = tester.widget<FilledButton>(
      find.byKey(const Key('onb_next')),
    );
    expect(btn.onPressed, isNull);
  });

  testWidgets('Step 0 填完 → 下一步 → Step 1（健康证）', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(routerConfig: _router()),
      ),
    );
    await _fillStep0(tester);
    await tester.tap(find.byKey(const Key('onb_next')));
    await tester.pumpAndSettle();
    expect(find.text('健康证 (2/5)'), findsOneWidget);
    expect(find.byKey(const Key('onb_upload_health')), findsOneWidget);
  });

  testWidgets('最后一步「提交审核」→ /audit/pending', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(routerConfig: _router()),
      ),
    );
    await _fillStep0(tester);
    await tester.tap(find.byKey(const Key('onb_next')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('onb_upload_health')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('onb_next')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('onb_finish_training')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('onb_next')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('onb_terms')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('onb_next')));
    await tester.pumpAndSettle();
    expect(find.text('提交审核 (5/5)'), findsOneWidget);
    await tester.tap(find.byKey(const Key('onb_submit')));
    await tester.pump(const Duration(milliseconds: 600));
    await tester.pumpAndSettle();
    expect(find.text('AUDIT_PENDING_ROUTE'), findsOneWidget);
  });

  testWidgets('上一步 → 返回 Step 0', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(routerConfig: _router()),
      ),
    );
    await _fillStep0(tester);
    await tester.tap(find.byKey(const Key('onb_next')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('onb_prev')), findsOneWidget);
    await tester.tap(find.byKey(const Key('onb_prev')));
    await tester.pumpAndSettle();
    expect(find.text('身份信息 (1/5)'), findsOneWidget);
  });
}