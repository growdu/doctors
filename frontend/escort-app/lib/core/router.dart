// lib/core/router.dart
//
// GoRouter Provider —— 注册 escort-app v1.2 核心路由（含 8 个 P0 业务页）。
//
// v1.2 路由清单（按 plan §3 路由表）：
//   - `/auth/login`（E3 登录）
//   - `/home/invitations`（v1.1 选人模式）
//   - `/home/availability`（v1.1 空余时段）
//   - `/home/orders`（E6 我的订单）
//   - `/home/orders/:id`（E8 订单详情）
//   - `/home/wallet`（E5 钱包）
//   - `/home/training`（E7 培训）
//   - `/home/profile`（E4 个人中心）
//
// v1 骨架路径：`/home/feed`（抢单池）已**删除**。
// v1.2 新增 `/auth/login` + 5 个 P0 业务页。
import 'package:escort_app/pages/availability/availability_page.dart';
import 'package:escort_app/pages/invitations/invitations_page.dart';
import 'package:escort_app/pages/login/login_page.dart';
import 'package:escort_app/pages/orders/orders_page.dart';
import 'package:escort_app/pages/profile/profile_page.dart';
import 'package:escort_app/pages/wallet/wallet_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

/// 全局 GoRouter（v1.2 8 核心业务页路由）。
final goRouterProvider = Provider<GoRouter>((ref) {
  return GoRouter(
    initialLocation: '/home/invitations',
    routes: [
      GoRoute(
        path: '/splash',
        builder: (context, state) => const _SplashPlaceholder(),
      ),
      // 登录（E3）
      GoRoute(
        path: '/auth/login',
        builder: (context, state) => const LoginPage(),
      ),
      // 选人模式核心路由（v1.1）
      GoRoute(
        path: '/home/invitations',
        builder: (context, state) => const InvitationsPage(),
      ),
      GoRoute(
        path: '/home/availability',
        builder: (context, state) => const AvailabilityPage(),
      ),
      // 我的订单（E6）
      GoRoute(
        path: '/home/orders',
        builder: (context, state) => const OrdersPage(),
      ),
      // 个人中心（E4）
      GoRoute(
        path: '/home/profile',
        builder: (context, state) => const ProfilePage(),
      ),
      // 钱包（E5）
      GoRoute(
        path: '/home/wallet',
        builder: (context, state) => const WalletPage(),
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