// test/pages/splash_page_test.dart
//
// SplashPage widget test —— 验证：
//   - 已登录（mock authProvider）→ /home/invitations
//   - 未登录 → /login
//   - 渲染 Logo + 进度条
import 'package:escort_app/providers/auth_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

import 'package:escort_app/pages/splash/splash_page.dart';

GoRouter _stubRouter() {
  return GoRouter(
    initialLocation: '/splash',
    routes: [
      GoRoute(
        path: '/splash',
        builder: (c, s) => const SplashPage(),
      ),
      GoRoute(
        path: '/login',
        builder: (c, s) => const Scaffold(body: Text('LOGIN_ROUTE')),
      ),
      GoRoute(
        path: '/home/invitations',
        builder: (c, s) => const Scaffold(body: Text('HOME_ROUTE')),
      ),
    ],
  );
}

class _StubAuth extends AuthNotifier {
  _StubAuth(Ref ref) : super(ref);

  @override
  Future<void> bootstrap() async {} // no-op
}

void main() {
  testWidgets('未登录 → 跳 /login', (tester) async {
    final container = ProviderContainer(overrides: [
      authProvider.overrideWith((ref) => _StubAuth(ref)),
    ]);
    addTearDown(container.dispose);
    // 预先把状态设为 unauthenticated
    container.read(authProvider.notifier).state = const AuthUnauthenticated();
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp.router(routerConfig: _stubRouter()),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('LOGIN_ROUTE'), findsOneWidget);
  });

  testWidgets('已登录 → 跳 /home/invitations', (tester) async {
    final container = ProviderContainer(overrides: [
      authProvider.overrideWith((ref) => _StubAuth(ref)),
    ]);
    addTearDown(container.dispose);
    container.read(authProvider.notifier).state = const AuthAuthenticated(
      userId: 1,
      phone: '13800000000',
      role: 'escort',
      realNameVerified: true,
      approved: true,
    );
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp.router(routerConfig: _stubRouter()),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('HOME_ROUTE'), findsOneWidget);
  });

  testWidgets('渲染 Logo + CircularProgressIndicator', (tester) async {
    final container = ProviderContainer(overrides: [
      authProvider.overrideWith((ref) => _StubAuth(ref)),
    ]);
    addTearDown(container.dispose);
    container.read(authProvider.notifier).state = const AuthUnauthenticated();
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp.router(routerConfig: _stubRouter()),
      ),
    );
    await tester.pump();
    expect(find.byType(CircularProgressIndicator), findsOneWidget);
    expect(find.text('escort-app v1.2'), findsOneWidget);
  });
}