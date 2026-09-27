// lib/pages/wallet/withdraw_page.dart
//
// 提现页 —— 金额 + 银行卡选择 + 提现申请。
//
// 业务流：
//   1. 输入提现金额（最小 1 元，最大余额）
//   2. 选择银行卡（mock 列表）
//   3. 点「申请提现」→ POST /wallet/withdraw → 跳 /wallet
//
// 设计：mock 银行卡列表 + 余额
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

class _BankCard {
  final String id;
  final String label;
  const _BankCard(this.id, this.label);
}

const _mockCards = <_BankCard>[
  _BankCard('c1', '工商银行 **** 1234'),
  _BankCard('c2', '建设银行 **** 5678'),
  _BankCard('c3', '招商银行 **** 9012'),
];

const _mockBalance = 1234.56;

/// 提现页。
class WithdrawPage extends ConsumerStatefulWidget {
  const WithdrawPage({super.key});

  @override
  ConsumerState<WithdrawPage> createState() => _WithdrawPageState();
}

class _WithdrawPageState extends ConsumerState<WithdrawPage> {
  final _amountCtrl = TextEditingController();
  String _cardId = _mockCards.first.id;
  bool _busy = false;
  String? _err;

  @override
  void dispose() {
    _amountCtrl.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    setState(() => _err = null);
    final n = double.tryParse(_amountCtrl.text.trim());
    if (n == null || n < 1 || n > _mockBalance) {
      setState(() => _err = '请输入有效金额（1 ~ $_mockBalance）');
      return;
    }
    setState(() => _busy = true);
    await Future<void>.delayed(const Duration(milliseconds: 500));
    if (!mounted) return;
    setState(() => _busy = false);
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text('已申请提现 ¥${n.toStringAsFixed(2)}')),
    );
    context.go('/home/wallet');
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('提现')),
      body: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Card(
              child: Padding(
                padding: const EdgeInsets.all(12),
                child: Row(
                  children: [
                    const Text('可提现余额'),
                    const Spacer(),
                    Text(
                      '¥${_mockBalance.toStringAsFixed(2)}',
                      key: const Key('withdraw_balance'),
                      style: const TextStyle(
                          fontSize: 20, fontWeight: FontWeight.bold,
                      ),
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 16),
            TextField(
              key: const Key('withdraw_amount'),
              controller: _amountCtrl,
              decoration: InputDecoration(
                labelText: '提现金额',
                prefixText: '¥ ',
                errorText: _err,
              ),
              keyboardType:
                  const TextInputType.numberWithOptions(decimal: true),
              inputFormatters: [
                FilteringTextInputFormatter.allow(RegExp(r'^\d*\.?\d{0,2}')),
              ],
              onChanged: (_) => setState(() => _err = null),
            ),
            const SizedBox(height: 16),
            const Text('收款银行卡'),
            const SizedBox(height: 8),
            ..._mockCards.map(
              (c) => InkWell(
                key: Key('withdraw_card_${c.id}'),
                onTap: () => setState(() => _cardId = c.id),
                child: ListTile(
                  leading: Icon(
                    _cardId == c.id
                        ? Icons.radio_button_checked
                        : Icons.radio_button_unchecked,
                  ),
                  title: Text(c.label),
                ),
              ),
            ),
            const Spacer(),
            FilledButton(
              key: const Key('withdraw_submit'),
              onPressed: _busy ? null : _submit,
              child: _busy
                  ? const SizedBox(
                      height: 16,
                      width: 16,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Text('申请提现'),
            ),
          ],
        ),
      ),
    );
  }
}