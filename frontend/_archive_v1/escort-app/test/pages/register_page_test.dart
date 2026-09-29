// test/pages/register_page_test.dart
//
// RegisterPage widget test —— 验证：
//   - 渲染手机 / 验证码 / 密码输入框 + 注册按钮
//   - 校验逻辑（手机 11 位、验证码 6 位、密码 6-32 位）
//   - 同意协议后才允许提交
//   - 提交成功后跳 /onboarding
import 'package:escort_app/pages/auth/register_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

GoRouter _router() => GoRouter(
      initialLocation: '/register',
      routes: [
        GoRoute(
          path: '/register',
          builder: (c, s) => const RegisterPage(),
        ),
        GoRoute(
          path: '/login',
          builder: (c, s) => const Scaffold(body: Text('LOGIN_ROUTE')),
        ),
        GoRoute(
          path: '/onboarding',
          builder: (c, s) => const Scaffold(body: Text('ONBOARDING_ROUTE')),
        ),
      ],
    );

void main() {
  testWidgets('渲染 4 个输入字段 + 协议 + 提交按钮', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(routerConfig: _router()),
      ),
    );
    expect(find.text('注册陪诊师'), findsOneWidget);
    expect(find.byKey(const Key('register_phone')), findsOneWidget);
    expect(find.byKey(const Key('register_code')), findsOneWidget);
    expect(find.byKey(const Key('register_pwd')), findsOneWidget);
    expect(find.byKey(const Key('register_terms')), findsOneWidget);
    expect(find.byKey(const Key('register_submit')), findsOneWidget);
  });

  testWidgets('校验：手机号必须 11 位', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(routerConfig: _router()),
      ),
    );
    await tester.enterText(find.byKey(const Key('register_phone')), '123');
    await tester.enterText(find.byKey(const Key('register_code')), '123456');
    await tester.enterText(find.byKey(const Key('register_pwd')), 'pass1234');
    await tester.tap(find.byKey(const Key('register_terms')));
    await tester.pump();
    await tester.tap(find.byKey(const Key('register_submit')));
    await tester.pump();
    expect(find.text('请输入 11 位手机号'), findsOneWidget);
  });

  testWidgets('未同意协议 → 提交提示「请先同意」', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(routerConfig: _router()),
      ),
    );
    await tester.enterText(find.byKey(const Key('register_phone')), '13800000000');
    await tester.enterText(find.byKey(const Key('register_code')), '123456');
    await tester.enterText(find.byKey(const Key('register_pwd')), 'pass1234');
    await tester.tap(find.byKey(const Key('register_submit')));
    await tester.pump();
    expect(find.text('请先同意《陪诊师服务协议》'), findsOneWidget);
  });

  testWidgets('同意协议 + 合法输入 → 跳转 /onboarding', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(routerConfig: _router()),
      ),
    );
    await tester.enterText(find.byKey(const Key('register_phone')), '13800000000');
    await tester.enterText(find.byKey(const Key('register_code')), '123456');
    await tester.enterText(find.byKey(const Key('register_pwd')), 'pass1234');
    await tester.tap(find.byKey(const Key('register_terms')));
    await tester.pump();
    await tester.tap(find.byKey(const Key('register_submit')));
    // mock 500ms 后跳转
    await tester.pump(const Duration(milliseconds: 600));
    await tester.pumpAndSettle();
    expect(find.text('ONBOARDING_ROUTE'), findsOneWidget);
  });

  testWidgets('「已有账号」链接 → /login', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(routerConfig: _router()),
      ),
    );
    await tester.tap(find.text('已有账号？去登录'));
    await tester.pumpAndSettle();
    expect(find.text('LOGIN_ROUTE'), findsOneWidget);
  });
}