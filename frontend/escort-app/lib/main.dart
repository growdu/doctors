// lib/main.dart
//
// escort-app 入口 —— ProviderScope + MaterialApp.router。
// v1 骨架（Task 1~3）：仅占位路由；后续 Task 4+ 补全 24 个页面与守卫。
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'core/router.dart';
import 'core/theme.dart';

void main() {
  runApp(const ProviderScope(child: DoctorsEscortApp()));
}

/// 顶层 Widget —— 监听 [goRouterProvider]，把 GoRouter 注入 MaterialApp。
class DoctorsEscortApp extends ConsumerWidget {
  const DoctorsEscortApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final router = ref.watch(goRouterProvider);
    return MaterialApp.router(
      title: '陪诊师端',
      debugShowCheckedModeBanner: false,
      theme: appTheme,
      routerConfig: router,
    );
  }
}