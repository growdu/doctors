// lib/pages/orders/orders_page.dart
//
// 我的订单页 —— plan §3 E6。
//
// 业务流：
//   1. 加载 ordersProvider（按 status tab 拉：邀请中 / 已接单 / 服务中 / 已完成）
//   2. 渲染 ListView（OrderSummary 瘦字段：医院 + 时间 + 金额 + 套餐）
//   3. 点击订单 → 跳 /home/orders/:id（详情页 E8）
//
// 设计要点：
//   - 4 个 tab 对应 4 个 status：
//     - 邀请中：escort_pending_acceptance（含 30s 倒计时）
//     - 已接单：accepted
//     - 服务中：in_service
//     - 已完成：completed
//   - 「邀请中」tab 的订单卡片复用 CountdownBadge（v1.1 widget）
//   - 空态：每个 tab 独立空态文案
import 'package:escort_app/models/order.dart';
import 'package:escort_app/providers/order_provider.dart';
import 'package:escort_app/widgets/countdown_badge.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

/// 我的订单页。
class OrdersPage extends ConsumerStatefulWidget {
  const OrdersPage({super.key});

  @override
  ConsumerState<OrdersPage> createState() => _OrdersPageState();
}

class _OrdersPageState extends ConsumerState<OrdersPage>
    with SingleTickerProviderStateMixin {
  late TabController _tabController;

  /// Tab 配置：label + 状态过滤。
  static const _tabs = <_OrderTab>[
    _OrderTab(label: '邀请中', status: 'escort_pending_acceptance'),
    _OrderTab(label: '已接单', status: 'accepted'),
    _OrderTab(label: '服务中', status: 'in_service'),
    _OrderTab(label: '已完成', status: 'completed'),
  ];

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: _tabs.length, vsync: this);
  }

  @override
  void dispose() {
    _tabController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('我的订单'),
        actions: [
          IconButton(
            tooltip: '刷新',
            icon: const Icon(Icons.refresh),
            onPressed: () => ref.invalidate(ordersProvider),
          ),
        ],
        bottom: TabBar(
          controller: _tabController,
          isScrollable: true,
          tabs: _tabs.map((t) => Tab(text: t.label)).toList(),
        ),
      ),
      body: TabBarView(
        controller: _tabController,
        children: _tabs
            .map((t) => _OrderList(status: t.status))
            .toList(),
      ),
    );
  }
}

/// Tab 配置。
class _OrderTab {
  final String label;
  final String status; // 后端 snake_case
  const _OrderTab({required this.label, required this.status});
}

/// 单 tab 的订单列表。
class _OrderList extends ConsumerWidget {
  final String status;
  const _OrderList({required this.status});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final asyncList = ref.watch(
      ordersProvider(OrderListKey(status: status)),
    );

    return asyncList.when(
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
              onPressed: () => ref.invalidate(ordersProvider),
              child: const Text('重试'),
            ),
          ],
        ),
      ),
      data: (list) {
        if (list.isEmpty) return _EmptyView(status: status);
        return RefreshIndicator(
          onRefresh: () async {
            ref.invalidate(ordersProvider);
          },
          child: ListView.separated(
            padding: const EdgeInsets.all(12),
            itemCount: list.length,
            separatorBuilder: (_, __) => const SizedBox(height: 8),
            itemBuilder: (_, i) => OrderCard(order: list[i]),
          ),
        );
      },
    );
  }
}

/// 单条订单卡片。
class OrderCard extends ConsumerWidget {
  final OrderSummary order;
  const OrderCard({super.key, required this.order});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final isPending = order.status == OrderStatus.escortPendingAcceptance;
    return Card(
      elevation: 2,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
      child: InkWell(
        borderRadius: BorderRadius.circular(8),
        onTap: () => context.push('/home/orders/${order.id}'),
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
                      order.hospitalName,
                      style: Theme.of(context).textTheme.titleMedium,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                  if (isPending && order.escortPendingExpireAt != null)
                    CountdownBadge(
                      expireAt: order.escortPendingExpireAt!,
                      label: '确认剩余',
                    ),
                ],
              ),
              const SizedBox(height: 8),
              Text('套餐：${order.packageName}'),
              const SizedBox(height: 4),
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Text(
                    '服务时间：${order.serviceStartAtText}',
                    style: const TextStyle(color: Colors.grey),
                  ),
                  Text(
                    order.amountText,
                    style: TextStyle(
                      fontSize: 16,
                      fontWeight: FontWeight.bold,
                      color: Theme.of(context).colorScheme.primary,
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// 空态（按 tab 不同文案）。
class _EmptyView extends StatelessWidget {
  final String status;
  const _EmptyView({required this.status});

  @override
  Widget build(BuildContext context) {
    final (icon, title, hint) = switch (status) {
      'escort_pending_acceptance' => (
        Icons.notifications_none,
        '暂无邀请',
        '患者选中您后会推送到这里'
      ),
      'accepted' => (
        Icons.check_circle_outline,
        '暂无已接单订单',
        '接单后会显示在这里'
      ),
      'in_service' => (
        Icons.medical_services_outlined,
        '暂无服务中订单',
        '开始服务后会显示在这里'
      ),
      'completed' => (
        Icons.task_alt,
        '暂无已完成订单',
        '完成订单后会显示在这里'
      ),
      _ => (Icons.inbox_outlined, '暂无订单', '刷新试试'),
    };

    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(icon, size: 64, color: Colors.grey[400]),
          const SizedBox(height: 12),
          Text(title, style: Theme.of(context).textTheme.titleMedium),
          const SizedBox(height: 4),
          Text(hint, style: const TextStyle(color: Colors.grey)),
        ],
      ),
    );
  }
}