// lib/pages/order_checkout/checkout_page.dart
//
// 订单完成页 —— 服务完成 + 备注 + 提交。
//
// 业务流：
//   1. 默认标记服务已完成（开关）
//   2. 选填备注（最多 200 字）
//   3. 点「提交完成」→ POST /orders/:id/complete → 跳订单详情
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

/// 订单完成页。
class CheckoutPage extends ConsumerStatefulWidget {
  final int orderId;
  const CheckoutPage({super.key, required this.orderId});

  @override
  ConsumerState<CheckoutPage> createState() => _CheckoutPageState();
}

class _CheckoutPageState extends ConsumerState<CheckoutPage> {
  bool _serviceDone = true;
  final _noteCtrl = TextEditingController();
  bool _busy = false;

  @override
  void dispose() {
    _noteCtrl.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (!_serviceDone) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('请确认服务已完成')),
      );
      return;
    }
    setState(() => _busy = true);
    await Future<void>.delayed(const Duration(milliseconds: 500));
    if (!mounted) return;
    setState(() => _busy = false);
    context.go('/home/orders/${widget.orderId}');
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('完成订单')),
      body: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Card(
              child: Padding(
                padding: const EdgeInsets.all(12),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text('订单 #${widget.orderId}',
                        style: Theme.of(context).textTheme.titleMedium,
                    ),
                    const SizedBox(height: 8),
                    SwitchListTile(
                      key: const Key('checkout_done'),
                      value: _serviceDone,
                      onChanged: (v) => setState(() => _serviceDone = v),
                      title: const Text('服务已完成'),
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 16),
            TextField(
              key: const Key('checkout_note'),
              controller: _noteCtrl,
              decoration: const InputDecoration(
                labelText: '备注（可选，最多 200 字）',
                border: OutlineInputBorder(),
              ),
              maxLines: 4,
              maxLength: 200,
            ),
            const Spacer(),
            FilledButton(
              key: const Key('checkout_submit'),
              onPressed: _busy ? null : _submit,
              child: _busy
                  ? const SizedBox(
                      height: 16,
                      width: 16,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Text('提交完成'),
            ),
          ],
        ),
      ),
    );
  }
}