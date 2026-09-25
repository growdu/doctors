// lib/pages/order_detail/order_detail_page.dart
//
// 订单详情页 —— plan §3 E8 / spec §2.1 订单状态机。
//
// 业务流：
//   1. URL 传入 orderId（/home/orders/:id）→ 调 orderDetailProvider(id) 拉详情
//   2. 6 节点状态机可视化：paid → matching → selectingEscort → escortPendingAcceptance → accepted → inService → completed
//   3. 状态 = escortPendingAcceptance → 显示 CountdownBadge + 「确认接单 / 拒接」按钮
//      「确认接单」→ POST /orders/{id}/confirm（走 v1.1 confirmAcceptControllerProvider）
//      「拒接」→ POST /orders/{id}/reject（走 v1.1 rejectAcceptControllerProvider）
//   4. 客户信息（脱敏） + 套餐 + 金额 + 医院地址（lat/lng → 「打开地图」按钮）
//   5. 操作完成 → invalidate orderDetailProvider 自动刷新
//
// 设计要点：
//   - 6 节点进度条：横向 Stepper（自绘版避免引第三方包）
//   - 倒计时复用 v1.1 CountdownBadge widget
//   - 拒接/确认与 Invitations 页用同一 controller（StateNotifier.autoDispose 共享）
import 'package:escort_app/models/order.dart';
import 'package:escort_app/providers/invitation_provider.dart';
import 'package:escort_app/providers/order_provider.dart';
import 'package:escort_app/utils/format.dart';
import 'package:escort_app/widgets/countdown_badge.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// 订单详情页。
class OrderDetailPage extends ConsumerWidget {
  final int orderId;
  const OrderDetailPage({super.key, required this.orderId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final asyncOrder = ref.watch(orderDetailProvider(orderId));

    return Scaffold(
      appBar: AppBar(
        title: const Text('订单详情'),
        actions: [
          IconButton(
            tooltip: '刷新',
            icon: const Icon(Icons.refresh),
            onPressed: () => ref.invalidate(orderDetailProvider(orderId)),
          ),
        ],
      ),
      body: asyncOrder.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (err, _) => Center(
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              const Icon(Icons.error_outline, size: 48, color: Colors.red),
              const SizedBox(height: 12),
              Text('加载失败：$err', textAlign: TextAlign.center),
              const SizedBox(height: 12),
              FilledButton(
                onPressed: () =>
                    ref.invalidate(orderDetailProvider(orderId)),
                child: const Text('重试'),
              ),
            ],
          ),
        ),
        data: (order) => _OrderDetailBody(order: order),
      ),
    );
  }
}

/// 订单详情主体。
class _OrderDetailBody extends ConsumerWidget {
  final Order order;
  const _OrderDetailBody({required this.order});

  Future<void> _onAccept(BuildContext context, WidgetRef ref) async {
    try {
      await ref
          .read(confirmAcceptControllerProvider.notifier)
          .invoke(order.id);
      if (!context.mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('已确认接单')),
      );
      ref.invalidate(orderDetailProvider(order.id));
    } catch (e) {
      if (!context.mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('确认失败：$e')),
      );
    }
  }

  Future<void> _onReject(BuildContext context, WidgetRef ref) async {
    final confirm = await showDialog<bool>(
      context: context,
      builder: (_) => AlertDialog(
        title: const Text('拒接订单'),
        content: const Text('拒接后患者可重新选择其他陪诊师'),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(false),
            child: const Text('取消'),
          ),
          FilledButton(
            onPressed: () => Navigator.of(context).pop(true),
            child: const Text('确认拒接'),
          ),
        ],
      ),
    );
    if (confirm != true) return;
    try {
      await ref
          .read(rejectAcceptControllerProvider.notifier)
          .invoke(order.id);
      if (!context.mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('已拒接')),
      );
      // 拒接后订单状态变化 → 触发详情刷新
      ref.invalidate(orderDetailProvider(order.id));
    } catch (e) {
      if (!context.mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('拒接失败：$e')),
      );
    }
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final o = order;
    final showAction = o.status == OrderStatus.escortPendingAcceptance;
    final isLive = o.escortPendingExpireAt != null &&
        DateTime.now().isBefore(o.escortPendingExpireAt!);

    return RefreshIndicator(
      onRefresh: () async {
        ref.invalidate(orderDetailProvider(o.id));
        await ref.read(orderDetailProvider(o.id).future);
      },
      child: ListView(
        padding: const EdgeInsets.all(12),
        children: [
          // 顶部：状态 + 倒计时
          _StatusHeader(order: o, isLive: isLive),
          const SizedBox(height: 12),
          // 6 节点进度
          _OrderProgress(currentStatus: o.status),
          const SizedBox(height: 16),
          // 客户信息
          _CustomerCard(order: o),
          const SizedBox(height: 12),
          // 订单详情
          _OrderInfoCard(order: o),
          const SizedBox(height: 16),
          // 操作按钮
          if (showAction)
            Row(
              children: [
                Expanded(
                  child: OutlinedButton(
                    onPressed:
                        isLive ? () => _onReject(context, ref) : null,
                    style: OutlinedButton.styleFrom(
                      foregroundColor: Colors.grey[700],
                    ),
                    child: const Text('拒接'),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: FilledButton(
                    onPressed:
                        isLive ? () => _onAccept(context, ref) : null,
                    child: const Text('确认接单'),
                  ),
                ),
              ],
            ),
        ],
      ),
    );
  }
}

/// 状态头部（status 文案 + 倒计时）。
class _StatusHeader extends StatelessWidget {
  final Order order;
  final bool isLive;
  const _StatusHeader({required this.order, required this.isLive});

  @override
  Widget build(BuildContext context) {
    final statusText = _statusText(order.status);
    final color = _statusColor(order.status);

    return Card(
      elevation: 2,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      color: color.withOpacity(0.05),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Row(
          children: [
            Icon(_statusIcon(order.status), color: color, size: 32),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    statusText,
                    style: TextStyle(
                      fontSize: 18,
                      fontWeight: FontWeight.bold,
                      color: color,
                    ),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    '订单 #${order.id}',
                    style: const TextStyle(color: Colors.grey),
                  ),
                ],
              ),
            ),
            if (order.status == OrderStatus.escortPendingAcceptance &&
                order.escortPendingExpireAt != null)
              CountdownBadge(
                expireAt: order.escortPendingExpireAt!,
                label: '确认剩余',
              ),
          ],
        ),
      ),
    );
  }
}

/// 6 节点进度条（自绘 Stepper）。
class _OrderProgress extends StatelessWidget {
  final OrderStatus currentStatus;
  const _OrderProgress({required this.currentStatus});

  /// 6 个主流程节点（spec §2.1）。
  static const _flow = <OrderStatus>[
    OrderStatus.paid,
    OrderStatus.matching,
    OrderStatus.selectingEscort,
    OrderStatus.escortPendingAcceptance,
    OrderStatus.accepted,
    OrderStatus.inService,
  ];

  int get _currentIndex {
    final i = _flow.indexOf(currentStatus);
    return i < 0 ? 0 : i;
  }

  @override
  Widget build(BuildContext context) {
    return Card(
      elevation: 1,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text('订单进度',
                style: TextStyle(fontWeight: FontWeight.bold)),
            const SizedBox(height: 12),
            SingleChildScrollView(
              scrollDirection: Axis.horizontal,
              child: Row(
                children: List.generate(_flow.length, (i) {
                  final reached = i <= _currentIndex;
                  final isCurrent = i == _currentIndex;
                  return Row(
                    children: [
                      Column(
                        children: [
                          CircleAvatar(
                            radius: 14,
                            backgroundColor: reached
                                ? Theme.of(context).colorScheme.primary
                                : Colors.grey[300],
                            child: reached
                                ? Icon(
                                    isCurrent
                                        ? Icons.access_time
                                        : Icons.check,
                                    size: 14,
                                    color: Colors.white,
                                  )
                                : Text(
                                    '${i + 1}',
                                    style: const TextStyle(
                                        color: Colors.white, fontSize: 11),
                                  ),
                          ),
                          const SizedBox(height: 4),
                          SizedBox(
                            width: 60,
                            child: Text(
                              _nodeLabel(_flow[i]),
                              textAlign: TextAlign.center,
                              style: TextStyle(
                                fontSize: 11,
                                color: reached ? Colors.black : Colors.grey,
                                fontWeight: isCurrent
                                    ? FontWeight.bold
                                    : FontWeight.normal,
                              ),
                            ),
                          ),
                        ],
                      ),
                      if (i < _flow.length - 1)
                        Container(
                          width: 24,
                          height: 2,
                          margin: const EdgeInsets.symmetric(horizontal: 2),
                          color: i < _currentIndex
                              ? Theme.of(context).colorScheme.primary
                              : Colors.grey[300],
                        ),
                    ],
                  );
                }),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// 客户信息卡。
class _CustomerCard extends StatelessWidget {
  final Order order;
  const _CustomerCard({required this.order});

  @override
  Widget build(BuildContext context) {
    // v1 简化：仅显示 patient_id + 「查看完整信息需通过虚拟号」（占位）
    return Card(
      elevation: 1,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text('客户信息',
                style: TextStyle(fontWeight: FontWeight.bold)),
            const SizedBox(height: 12),
            Row(
              children: [
                CircleAvatar(
                  backgroundColor:
                      Theme.of(context).colorScheme.primary.withOpacity(0.1),
                  child: Icon(Icons.person_outline,
                      color: Theme.of(context).colorScheme.primary),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text('患者 #${order.patientId}'),
                      const SizedBox(height: 4),
                      Text(
                        '联系方式将通过虚拟号保护',
                        style: const TextStyle(
                            color: Colors.grey, fontSize: 12),
                      ),
                    ],
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

/// 订单详情卡（医院 + 套餐 + 时间 + 金额）。
class _OrderInfoCard extends StatelessWidget {
  final Order order;
  const _OrderInfoCard({required this.order});

  @override
  Widget build(BuildContext context) {
    return Card(
      elevation: 1,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text('订单信息',
                style: TextStyle(fontWeight: FontWeight.bold)),
            const SizedBox(height: 12),
            _InfoRow(label: '医院', value: order.hospitalName),
            const Divider(),
            _InfoRow(label: '套餐', value: order.packageName),
            const Divider(),
            _InfoRow(label: '服务时间', value: formatDateTime(order.serviceStartAt.toLocal())),
            const Divider(),
            _InfoRow(label: '医院位置', value: '${order.hospitalLat.toStringAsFixed(4)}, ${order.hospitalLng.toStringAsFixed(4)}'),
            const Divider(),
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                const Text('订单金额',
                    style: TextStyle(fontWeight: FontWeight.bold)),
                Text(
                  order.amountText,
                  style: TextStyle(
                    fontSize: 18,
                    fontWeight: FontWeight.bold,
                    color: Theme.of(context).colorScheme.primary,
                  ),
                ),
              ],
            ),
            if (order.completedAt != null) ...[
              const Divider(),
              _InfoRow(label: '完成时间', value: formatDateTime(order.completedAt!.toLocal())),
            ],
          ],
        ),
      ),
    );
  }
}

/// 信息行。
class _InfoRow extends StatelessWidget {
  final String label;
  final String value;
  const _InfoRow({required this.label, required this.value});
  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 80,
            child: Text(label, style: const TextStyle(color: Colors.grey)),
          ),
          Expanded(
            child: Text(value, style: const TextStyle(fontWeight: FontWeight.w500)),
          ),
        ],
      ),
    );
  }
}

/// 状态文案。
String _statusText(OrderStatus s) => switch (s) {
      OrderStatus.created => '已创建',
      OrderStatus.paid => '已支付',
      OrderStatus.matching => '匹配中',
      OrderStatus.selectingEscort => '患者选人中',
      OrderStatus.escortPendingAcceptance => '待您确认',
      OrderStatus.accepted => '已接单',
      OrderStatus.inService => '服务中',
      OrderStatus.completed => '已完成',
      OrderStatus.reviewed => '已评价',
      OrderStatus.refunding => '退款中',
      OrderStatus.refunded => '已退款',
      OrderStatus.settling => '结算中',
      OrderStatus.disputed => '争议处理',
      OrderStatus.closed => '已关闭',
      OrderStatus.canceled => '已取消',
    };

/// 状态颜色。
Color _statusColor(OrderStatus s) => switch (s) {
      OrderStatus.completed || OrderStatus.reviewed => Colors.green,
      OrderStatus.canceled ||
      OrderStatus.refunded ||
      OrderStatus.closed =>
        Colors.grey,
      OrderStatus.disputed || OrderStatus.refunding => Colors.red,
      OrderStatus.escortPendingAcceptance => Colors.orange,
      _ => Colors.blue,
    };

/// 状态图标。
IconData _statusIcon(OrderStatus s) => switch (s) {
      OrderStatus.completed => Icons.check_circle,
      OrderStatus.canceled => Icons.cancel,
      OrderStatus.refunded => Icons.undo,
      OrderStatus.escortPendingAcceptance => Icons.access_time,
      OrderStatus.accepted || OrderStatus.inService => Icons.medical_services,
      _ => Icons.info_outline,
    };

/// 节点标签。
String _nodeLabel(OrderStatus s) => switch (s) {
      OrderStatus.paid => '已支付',
      OrderStatus.matching => '匹配中',
      OrderStatus.selectingEscort => '选人中',
      OrderStatus.escortPendingAcceptance => '待确认',
      OrderStatus.accepted => '已接单',
      OrderStatus.inService => '服务中',
      _ => '',
    };