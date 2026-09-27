// lib/pages/audit/pending_page.dart
//
// 审核中页 —— 显示「审核中」+ 预计时长 + 禁用返回。
//
// 业务流：
//   - 进入即展示审核状态
//   - 返回按钮 disabled（避免误退）
//   - 倒计时显示预计剩余时长（mock 24h）
import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

/// 审核中页。
class PendingPage extends ConsumerStatefulWidget {
  const PendingPage({super.key});

  @override
  ConsumerState<PendingPage> createState() => _PendingPageState();
}

class _PendingPageState extends ConsumerState<PendingPage> {
  late Duration _remaining;
  Timer? _timer;

  @override
  void initState() {
    super.initState();
    _remaining = const Duration(hours: 24);
    _timer = Timer.periodic(const Duration(seconds: 1), (_) {
      if (!mounted) return;
      setState(() {
        _remaining -= const Duration(seconds: 1);
        if (_remaining.isNegative) _remaining = Duration.zero;
      });
    });
  }

  @override
  void dispose() {
    _timer?.cancel();
    super.dispose();
  }

  String get _hms {
    final h = _remaining.inHours;
    final m = _remaining.inMinutes.remainder(60);
    final s = _remaining.inSeconds.remainder(60);
    return '${h.toString().padLeft(2, '0')}:${m.toString().padLeft(2, '0')}:${s.toString().padLeft(2, '0')}';
  }

  @override
  Widget build(BuildContext context) {
    return PopScope(
      canPop: false, // 禁用返回
      child: Scaffold(
        appBar: AppBar(
          title: const Text('审核中'),
          // ignore: prefer_const_constructors
          leading: IconButton(
            key: const Key('pending_back'),
            icon: const Icon(Icons.arrow_back),
            onPressed: null, // 禁用
          ),
        ),
        body: Center(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                const Icon(Icons.hourglass_top, size: 96, color: Colors.amber),
                const SizedBox(height: 24),
                Text('审核中',
                    style: Theme.of(context).textTheme.headlineMedium,
                ),
                const SizedBox(height: 12),
                const Text('我们正在审核您的资料，通常 24 小时内完成。'),
                const SizedBox(height: 24),
                Card(
                  child: Padding(
                    padding: const EdgeInsets.all(16),
                    child: Column(
                      children: [
                        const Text('预计剩余'),
                        const SizedBox(height: 8),
                        Text(_hms,
                            key: const Key('pending_remaining'),
                            style: const TextStyle(
                                fontSize: 36,
                                fontWeight: FontWeight.bold,
                                fontFeatures: [FontFeature.tabularFigures()],
                            ),),
                      ],
                    ),
                  ),
                ),
                const SizedBox(height: 24),
                TextButton(
                  onPressed: () => context.go('/home/invitations'),
                  child: const Text('临时返回首页（演示）'),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}