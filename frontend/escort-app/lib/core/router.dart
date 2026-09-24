// lib/core/router.dart
//
// GoRouter Provider —— 注册 escort-app v1.1 核心路由。
//
// 路由清单（v1.1）：
//   - `/splash`（占位）
//   - `/home/invitations`（spec §4.1 选人模式核心）
//   - `/home/availability`（spec §3.2 空余时段管理）
//
// v1 骨架路径：`/home/feed`（抢单池）已**删除**；新增 `/home/invitations` + `/home/availability`。
// 后续 Task（24 个 P0 页面）按 plan 继续补全：login / register / onboarding / audit / orders /
// wallet / profile / order / sos / message 等。
import 'package:escort_app/pages/availability/availability_page.dart';
import 'package:escort_app/pages/invitations/invitations_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

/// 全局 GoRouter（v1.1 选人模式核心路由）。
final goRouterProvider = Provider<GoRouter>((ref) {
  return GoRouter(
    initialLocation: '/home/invitations',
    routes: [
      GoRoute(
        path: '/splash',
        builder: (context, state) => const _SplashPlaceholder(),
      ),
      // 选人模式核心路由（v1.1 新增，替换原 /home/feed）
      GoRoute(
        path: '/home/invitations',
        builder: (context, state) => const InvitationsPage(),
      ),
      GoRoute(
        path: '/home/availability',
        builder: (context, state) => const AvailabilityPage(),
      ),
    ],
  );
});

class _SplashPlaceholder extends StatelessWidget {
  const _SplashPlaceholder();

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('escort-app v1.1')),
      body: const Center(
        child: Text('escort-app v1.1 选人模式骨架'),
      ),
    );
  }
}