// lib/pages/splash/splash_page.dart
//
// 启动页 —— Auth Gate：读 tokenStorage 决定路由。
//   - 有 token → /home/invitations
//   - 无 token → /login
//
// 设计：纯 ConsumerStatefulWidget，initState 异步 bootstrap，
//       用 WidgetsBinding.addPostFrameCallback 触发跳转（避免 build 中调 context.go）。
import 'package:escort_app/providers/auth_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

/// 启动页 —— Auth Gate。
class SplashPage extends ConsumerStatefulWidget {
  const SplashPage({super.key});

  @override
  ConsumerState<SplashPage> createState() => _SplashPageState();
}

class _SplashPageState extends ConsumerState<SplashPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      final authed = ref.read(authProvider).isAuthed;
      context.go(authed ? '/home/invitations' : '/login');
    });
  }

  @override
  Widget build(BuildContext context) { // ignore: override_on_non_overriding_member
    return const Scaffold(
      body: Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            FlutterLogo(size: 96),
            SizedBox(height: 16),
            Text('escort-app v1.2', style: TextStyle(fontSize: 16)),
            SizedBox(height: 24),
            CircularProgressIndicator(),
          ],
        ),
      ),
    );
  }
}