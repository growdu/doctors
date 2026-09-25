// lib/providers/order_provider.dart
//
// 订单 provider —— GET /escorts/me/orders（按 status tab 过滤）+ GET /orders/{id}（详情）。
//
// 设计要点：
//   - ordersProvider：FutureProvider.family 按 status 过滤
//   - orderDetailProvider：FutureProvider.family 按 orderId 拉详情
//   - 两个都支持 invalidate 刷新（详情页拒接 / 订单状态变化）
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/order.dart';
import '../services/api_client.dart';

/// 订单状态 tab key（family 缓存粒度）。
class OrderListKey {
  /// 后端 snake_case（'escort_pending_acceptance' / 'in_service' / 'completed'）。
  /// null 表示「全部」。
  final String? status;
  final int page;
  final int size;

  const OrderListKey({this.status, this.page = 1, this.size = 20});

  @override
  bool operator ==(Object other) =>
      other is OrderListKey &&
      other.status == status &&
      other.page == page &&
      other.size == size;

  @override
  int get hashCode => Object.hash(status, page, size);
}

/// 订单列表 provider（按 status tab 拉）。
final ordersProvider =
    FutureProvider.family<List<OrderSummary>, OrderListKey>((ref, key) async {
  final dio = ref.read(dioProvider);
  final raw = await OrderApi.listMyOrders(
    dio,
    status: key.status,
    page: key.page,
    size: key.size,
  );
  return raw
      .map((e) => OrderSummary.fromJson(e as Map<String, dynamic>))
      .toList();
});

/// 订单详情 provider（按 orderId 拉完整 Order）。
final orderDetailProvider =
    FutureProvider.family<Order, int>((ref, orderId) async {
  final dio = ref.read(dioProvider);
  final data = await OrderApi.getOrder(dio, orderId);
  return Order.fromJson(data);
});