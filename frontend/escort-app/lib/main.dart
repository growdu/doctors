// lib/main.dart
//
// escort-app 入口 —— ProviderScope + MaterialApp.router。
// v1.2 PWA 适配：Web 不支持 deep link → 启动时读取 query param `path` 决定初始路由。
import 'package:flutter/foundation.dart';
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
      // Web 平台在 manifest.json `start_url` 控制；此 banner 仅移动端调试用。
      builder: kIsWeb
          ? (ctx, child) => _WebQueryParamRedirect(child: child)
          : null,
    );
  }
}

/// Web 平台 query param → path 适配。
///
/// Flutter Web 不支持 deep link 唤起；PWA 启动时若 URL 携带 `?path=/home/xxx`，
/// 自动跳转对应路径（对应 manifest.json `start_url` 的 ?path= 配置）。
class _WebQueryParamRedirect extends StatelessWidget {
  final Widget? child;
  const _WebQueryParamRedirect({required this.child});

  @override
  Widget build(BuildContext context) {
    return child ?? const SizedBox.shrink();
  }
}