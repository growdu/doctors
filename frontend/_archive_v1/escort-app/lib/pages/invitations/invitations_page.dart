// lib/pages/invitations/invitations_page.dart
//
// 「我的邀请」页 —— spec/2026-09-24-order-matching-redesign §4.1。
//
// 业务流：
//   1. 进入页面 → invitationsProvider 立即发请求（5s 间隔轮询）
//   2. 渲染 ListView（InvitationCard 列表）
//   3. 每张卡片含 CountdownBadge（参数化 expireAt；< 60s 红色）
//   4. 「确认接单」→ confirmAcceptControllerProvider.invoke(orderId)
//   5. 「拒接」→ rejectAcceptControllerProvider.invoke(orderId)
//   6. 操作成功 → 卡片消失（下一次 5s tick 自动刷新）
//
// 替换关系：本页替换 v1 plan 原生 `/home/feed` 抢单池页。
import 'package:escort_app/models/invitation.dart';
import 'package:escort_app/providers/invitation_provider.dart';
import 'package:escort_app/widgets/countdown_badge.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class InvitationsPage extends ConsumerWidget {
  const InvitationsPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final asyncInvitations = ref.watch(invitationsProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('我的邀请'),
        actions: [
          IconButton(
            tooltip: '刷新',
            icon: const Icon(Icons.refresh),
            onPressed: () => ref.invalidate(invitationsProvider),
          ),
        ],
      ),
      body: asyncInvitations.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (err, _) => _ErrorView(
          message: '加载失败：$err',
          onRetry: () => ref.invalidate(invitationsProvider),
        ),
        data: (list) {
          if (list.isEmpty) return const _EmptyView();
          return RefreshIndicator(
            onRefresh: () async {
              ref.invalidate(invitationsProvider);
              await ref.read(invitationsProvider.future);
            },
            child: ListView.separated(
              padding: const EdgeInsets.all(12),
              itemCount: list.length,
              separatorBuilder: (_, __) => const SizedBox(height: 8),
              itemBuilder: (_, i) => InvitationCard(invitation: list[i]),
            ),
          );
        },
      ),
    );
  }
}

/// 邀请卡片 —— 内部 widget（不导出；本页独占）。
class InvitationCard extends ConsumerWidget {
  final Invitation invitation;

  const InvitationCard({super.key, required this.invitation});

  Future<void> _confirm(BuildContext context, WidgetRef ref) async {
    try {
      await ref
          .read(confirmAcceptControllerProvider.notifier)
          .invoke(invitation.orderId);
      if (!context.mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('已确认接单')),
      );
    } catch (e) {
      if (!context.mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('确认失败：$e')),
      );
    }
  }

  Future<void> _reject(BuildContext context, WidgetRef ref) async {
    try {
      await ref
          .read(rejectAcceptControllerProvider.notifier)
          .invoke(invitation.orderId);
      if (!context.mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('已拒接')),
      );
    } catch (e) {
      if (!context.mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('拒接失败：$e')),
      );
    }
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final inv = invitation;
    final isLive = inv.isLive;

    return Card(
      elevation: 2,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Expanded(
                  child: Text(
                    inv.hospitalName,
                    style: Theme.of(context).textTheme.titleMedium,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
                CountdownBadge(expireAt: inv.escortPendingExpireAt),
              ],
            ),
            const SizedBox(height: 8),
            Text('套餐：${inv.packageName}'),
            const SizedBox(height: 4),
            Text('服务开始：${inv.serviceStartAt.toLocal()}'),
            const SizedBox(height: 4),
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Text(
                  '订单金额：${inv.amountText}',
                  style: TextStyle(
                    fontSize: 16,
                    fontWeight: FontWeight.bold,
                    color: Theme.of(context).colorScheme.primary,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 12),
            Row(
              children: [
                Expanded(
                  child: OutlinedButton(
                    onPressed: isLive ? () => _reject(context, ref) : null,
                    style: OutlinedButton.styleFrom(
                      foregroundColor: Colors.grey[700],
                    ),
                    child: const Text('拒接'),
                  ),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: FilledButton(
                    onPressed: isLive ? () => _confirm(context, ref) : null,
                    child: const Text('确认接单'),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _EmptyView extends StatelessWidget {
  const _EmptyView();

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(Icons.inbox_outlined, size: 64, color: Colors.grey[400]),
          const SizedBox(height: 12),
          Text('暂无邀请', style: Theme.of(context).textTheme.titleMedium),
          const SizedBox(height: 4),
          const Text(
            '患者选中您后会推送到这里',
            style: TextStyle(color: Colors.grey),
          ),
        ],
      ),
    );
  }
}

class _ErrorView extends StatelessWidget {
  final String message;
  final VoidCallback onRetry;

  const _ErrorView({required this.message, required this.onRetry});

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          const Icon(Icons.error_outline, size: 48, color: Colors.red),
          const SizedBox(height: 12),
          Text(message, textAlign: TextAlign.center),
          const SizedBox(height: 12),
          FilledButton(onPressed: onRetry, child: const Text('重试')),
        ],
      ),
    );
  }
}