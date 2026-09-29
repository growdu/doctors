// lib/pages/wallet/transactions_page.dart
//
// 钱包流水页 —— 按类型过滤（全部 / 收入 / 支出）。
//
// 数据源：mock 流水（v1 不接 /wallet/transactions）。
// 设计：顶部 TabBar + ListView。
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

enum _TxKind { all, income, expense }

extension _TxKindLabel on _TxKind {
  String get label => switch (this) {
        _TxKind.all => '全部',
        _TxKind.income => '收入',
        _TxKind.expense => '支出',
      };
}

class _MockTx {
  final String id;
  final _TxKind kind;
  final String title;
  final double amount;
  final DateTime at;
  const _MockTx(this.id, this.kind, this.title, this.amount, this.at);
}

final _mockTx = <_MockTx>[
  _MockTx(
    't1', _TxKind.income, '订单 #1024 收入', 280.0, _dt(2026, 9, 26, 14, 30),
  ),
  _MockTx(
    't2', _TxKind.income, '订单 #1023 收入', 180.0, _dt(2026, 9, 25, 10, 15),
  ),
  _MockTx(
    't3', _TxKind.expense, '提现到 工商银行 **** 1234', -500.0, _dt(2026, 9, 24, 9, 0),
  ),
  _MockTx(
    't4', _TxKind.expense, '提现到 招商银行 **** 9012', -200.0, _dt(2026, 9, 22, 18, 45),
  ),
  _MockTx(
    't5', _TxKind.income, '订单 #1019 收入', 360.0, _dt(2026, 9, 20, 11, 0),
  ),
];

DateTime _dt(int y, int m, int d, int h, int mi) => DateTime(y, m, d, h, mi);

/// 钱包流水页。
class TransactionsPage extends ConsumerStatefulWidget {
  const TransactionsPage({super.key});

  @override
  ConsumerState<TransactionsPage> createState() => _TransactionsPageState();
}

class _TransactionsPageState extends ConsumerState<TransactionsPage>
    with SingleTickerProviderStateMixin {
  _TxKind _kind = _TxKind.all;

  List<_MockTx> get _filtered {
    if (_kind == _TxKind.all) return _mockTx;
    return _mockTx.where((t) => t.kind == _kind).toList();
  }

  @override
  Widget build(BuildContext context) {
    final list = _filtered;
    return Scaffold(
      appBar: AppBar(
        title: const Text('钱包流水'),
        bottom: PreferredSize(
          preferredSize: const Size.fromHeight(48),
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 8),
            child: Row(
              children: _TxKind.values.map((k) {
                final selected = _kind == k;
                return Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 4),
                  child: ChoiceChip(
                    key: Key('tx_filter_${k.name}'),
                    label: Text(k.label),
                    selected: selected,
                    onSelected: (_) => setState(() => _kind = k),
                  ),
                );
              }).toList(),
            ),
          ),
        ),
      ),
      body: ListView.separated(
        key: const Key('tx_list'),
        itemCount: list.length,
        separatorBuilder: (_, __) => const Divider(height: 1),
        itemBuilder: (_, i) {
          final t = list[i];
          final isIncome = t.amount > 0;
          return ListTile(
            key: Key('tx_item_${t.id}'),
            leading: Icon(
              isIncome ? Icons.arrow_downward : Icons.arrow_upward,
              color: isIncome ? Colors.green : Colors.red,
            ),
            title: Text(t.title),
            subtitle: Text(t.at.toIso8601String()),
            trailing: Text(
              (isIncome ? '+' : '') + t.amount.toStringAsFixed(2),
              style: TextStyle(
                color: isIncome ? Colors.green : Colors.red,
                fontWeight: FontWeight.bold,
              ),
            ),
          );
        },
      ),
    );
  }
}