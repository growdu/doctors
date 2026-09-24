// lib/core/router.dart
//
// GoRouter Provider —— v1 骨架只占位路由，后续 Task 4+ 补全 24 个页面 + redirect 守卫。
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

/// 全局 GoRouter。
///
/// v1（2026-09-24）：仅注册 splash 占位 + 路由守卫 placeholder；后续 Task 替换：
///   - `/splash` → splashPage
///   - `/login` → loginPage
///   - `/home/invitations` → invitationsPage（替换原 `/home/feed`）
///   - `/home/availability` → availabilityPage
///   - `/home/orders` / `/home/wallet` / `/home/profile`
///   - 3 个 redirect 守卫：authGuardProvider / realNameGuardProvider / approvedGuardProvider
final goRouterProvider = Provider<GoRouter>((ref) {
  return GoRouter(
    initialLocation: '/splash',
    routes: [
      GoRoute(
        path: '/splash',
        builder: (context, state) => const _SplashPlaceholder(),
      ),
    ],
  );
});

class _SplashPlaceholder extends StatelessWidget {
  const _SplashPlaceholder();

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('escort-app v1 骨架')),
      body: const Center(
        child: Text('TODO: 后续 Task 替换为真实页面'),
      ),
    );
  }
}