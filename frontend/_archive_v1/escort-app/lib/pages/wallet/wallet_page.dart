// lib/pages/wallet/wallet_page.dart
//
// 钱包页 —— spec §10 / plan §3 E5。
//
// 业务流：
//   1. 加载 walletProvider（GET /wallet）显示余额 + 冻结
//   2. 流水 tab：walletTransactionsProvider（按 type 分 tab 或全部展示）
//   3. 「提现」按钮 → 弹底部表单（金额 + 银行卡号）→ withdrawController.submit
//
// 设计要点：
//   - 顶部卡片突出余额（Material 3 Card + 渐变背景）
//   - 流水按 type 过滤：全部 / 收入 / 提现 / 退款
//   - 提现金额校验：> 0 且 ≤ balance（前端校验；后端再校验）
import 'package:escort_app/models/wallet.dart';
import 'package:escort_app/providers/wallet_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// 钱包页。
class WalletPage extends ConsumerStatefulWidget {
  const WalletPage({super.key});

  @override
  ConsumerState<WalletPage> createState() => _WalletPageState();
}

class _WalletPageState extends ConsumerState<WalletPage>
    with SingleTickerProviderStateMixin {
  late TabController _tabController;

  static const _tabs = <_TxFilterTab>[
    _TxFilterTab(label: '全部', type: null),
    _TxFilterTab(label: '收入', type: TransactionType.income),
    _TxFilterTab(label: '提现', type: TransactionType.withdraw),
    _TxFilterTab(label: '退款', type: TransactionType.refund),
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

  Future<void> _onWithdraw() async {
    final result = await showModalBottomSheet<_WithdrawPayload>(
      context: context,
      isScrollControlled: true,
      builder: (_) => const _WithdrawSheet(),
    );
    if (result == null) return;
    try {
      await ref.read(withdrawControllerProvider.notifier).submit(
            amount: result.amount,
            bankAccount: result.bankAccount,
          );
      if (!mounted) return;
      final state = ref.read(withdrawControllerProvider);
      if (state is WithdrawSuccess) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('提现申请已提交')),
        );
      }
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('提现失败：$e')),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final asyncWallet = ref.watch(walletProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('我的钱包'),
        actions: [
          IconButton(
            tooltip: '刷新',
            icon: const Icon(Icons.refresh),
            onPressed: () {
              ref.invalidate(walletProvider);
              ref.invalidate(walletTransactionsProvider);
            },
          ),
        ],
        bottom: TabBar(
          controller: _tabController,
          tabs: _tabs.map((t) => Tab(text: t.label)).toList(),
        ),
      ),
      body: Column(
        children: [
          asyncWallet.when(
            loading: () => const Padding(
              padding: EdgeInsets.all(24),
              child: Center(child: CircularProgressIndicator()),
            ),
            error: (err, _) => Padding(
              padding: const EdgeInsets.all(24),
              child: Column(
                children: [
                  const Icon(Icons.error_outline, size: 48, color: Colors.red),
                  const SizedBox(height: 12),
                  Text('加载失败：$err', textAlign: TextAlign.center),
                  const SizedBox(height: 12),
                  FilledButton(
                    onPressed: () => ref.invalidate(walletProvider),
                    child: const Text('重试'),
                  ),
                ],
              ),
            ),
            data: (w) => _WalletCard(
              wallet: w,
              onWithdraw: _onWithdraw,
            ),
          ),
          const Divider(height: 1),
          Expanded(
            child: TabBarView(
              controller: _tabController,
              children: _tabs
                  .map((t) => _TxList(filterType: t.type))
                  .toList(),
            ),
          ),
        ],
      ),
    );
  }
}

/// 流水过滤 tab 配置。
class _TxFilterTab {
  final String label;
  final TransactionType? type;
  const _TxFilterTab({required this.label, required this.type});
}

/// 钱包卡片（余额 + 冻结 + 提现按钮）。
class _WalletCard extends StatelessWidget {
  final Wallet wallet;
  final VoidCallback onWithdraw;

  const _WalletCard({required this.wallet, required this.onWithdraw});

  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.all(12),
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        gradient: LinearGradient(
          colors: [
            Theme.of(context).colorScheme.primary,
            Theme.of(context).colorScheme.primary.withOpacity(0.7),
          ],
        ),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text('可用余额',
              style: TextStyle(color: Colors.white70, fontSize: 13)),
          const SizedBox(height: 8),
          Text(
            wallet.balanceText,
            style: const TextStyle(
              color: Colors.white,
              fontSize: 32,
              fontWeight: FontWeight.bold,
            ),
          ),
          const SizedBox(height: 16),
          Row(
            children: [
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const Text('冻结金额',
                        style: TextStyle(color: Colors.white70, fontSize: 12)),
                    const SizedBox(height: 4),
                    Text(wallet.frozenText,
                        style: const TextStyle(
                            color: Colors.white,
                            fontSize: 16,
                            fontWeight: FontWeight.w500)),
                  ],
                ),
              ),
              FilledButton.tonal(
                onPressed: wallet.balance > 0 ? onWithdraw : null,
                style: FilledButton.styleFrom(
                  backgroundColor: Colors.white,
                  foregroundColor: Theme.of(context).colorScheme.primary,
                ),
                child: const Text('提现'),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

/// 流水列表。
class _TxList extends ConsumerWidget {
  final TransactionType? filterType;
  const _TxList({required this.filterType});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final asyncList = ref.watch(
      walletTransactionsProvider(const WalletTxKey(page: 1, size: 50)),
    );

    return asyncList.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (err, _) => Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const Icon(Icons.error_outline, size: 36, color: Colors.red),
            const SizedBox(height: 8),
            Text('加载失败：$err'),
            const SizedBox(height: 8),
            FilledButton(
              onPressed: () =>
                  ref.invalidate(walletTransactionsProvider),
              child: const Text('重试'),
            ),
          ],
        ),
      ),
      data: (all) {
        final list = filterType == null
            ? all
            : all.where((t) => t.type == filterType).toList();
        if (list.isEmpty) {
          return Center(
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Icon(Icons.receipt_long_outlined,
                    size: 56, color: Colors.grey[400]),
                const SizedBox(height: 8),
                const Text('暂无流水'),
              ],
            ),
          );
        }
        return RefreshIndicator(
          onRefresh: () async {
            ref.invalidate(walletTransactionsProvider);
          },
          child: ListView.separated(
            padding: const EdgeInsets.all(8),
            itemCount: list.length,
            separatorBuilder: (_, __) => const Divider(height: 1),
            itemBuilder: (_, i) => _TxTile(tx: list[i]),
          ),
        );
      },
    );
  }
}

/// 单条流水。
class _TxTile extends StatelessWidget {
  final WalletTransaction tx;
  const _TxTile({required this.tx});

  @override
  Widget build(BuildContext context) {
    final isPositive = tx.amount >= 0;
    return ListTile(
      leading: CircleAvatar(
        backgroundColor:
            (isPositive ? Colors.green : Colors.red).withOpacity(0.1),
        child: Icon(
          isPositive ? Icons.arrow_downward : Icons.arrow_upward,
          color: isPositive ? Colors.green : Colors.red,
        ),
      ),
      title: Text(tx.type.displayName),
      subtitle: Text('${tx.createdAtText}${tx.memo != null ? ' · ${tx.memo}' : ''}'),
      trailing: Text(
        tx.signedAmountText,
        style: TextStyle(
          color: isPositive ? Colors.green : Colors.red,
          fontWeight: FontWeight.bold,
          fontSize: 16,
        ),
      ),
    );
  }
}

/// 提现表单 payload。
class _WithdrawPayload {
  final double amount;
  final String bankAccount;
  const _WithdrawPayload(this.amount, this.bankAccount);
}

/// 提现底部表单。
class _WithdrawSheet extends StatefulWidget {
  const _WithdrawSheet();
  @override
  State<_WithdrawSheet> createState() => _WithdrawSheetState();
}

class _WithdrawSheetState extends State<_WithdrawSheet> {
  final _amountCtrl = TextEditingController();
  final _bankCtrl = TextEditingController();
  final _formKey = GlobalKey<FormState>();

  @override
  void dispose() {
    _amountCtrl.dispose();
    _bankCtrl.dispose();
    super.dispose();
  }

  String? _validateAmount(String? v) {
    final s = v?.trim() ?? '';
    if (s.isEmpty) return '请输入金额';
    final d = double.tryParse(s);
    if (d == null) return '金额格式错误';
    if (d <= 0) return '金额必须大于 0';
    return null;
  }

  String? _validateBank(String? v) {
    final s = v?.trim() ?? '';
    if (s.isEmpty) return '请输入银行卡号';
    if (s.length < 4) return '银行卡号格式错误';
    return null;
  }

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: EdgeInsets.only(
        bottom: MediaQuery.of(context).viewInsets.bottom,
        left: 16,
        right: 16,
        top: 16,
      ),
      child: Form(
        key: _formKey,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            const Text('提现申请',
                style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
            const SizedBox(height: 16),
            TextFormField(
              controller: _amountCtrl,
              decoration: const InputDecoration(
                labelText: '金额',
                prefixText: '¥ ',
                border: OutlineInputBorder(),
              ),
              keyboardType: const TextInputType.numberWithOptions(decimal: true),
              inputFormatters: [
                FilteringTextInputFormatter.allow(RegExp(r'^\d+\.?\d{0,2}')),
              ],
              validator: _validateAmount,
            ),
            const SizedBox(height: 12),
            TextFormField(
              controller: _bankCtrl,
              decoration: const InputDecoration(
                labelText: '银行卡号',
                border: OutlineInputBorder(),
                helperText: '提现到该银行卡',
              ),
              keyboardType: TextInputType.number,
              validator: _validateBank,
            ),
            const SizedBox(height: 16),
            FilledButton(
              onPressed: () {
                if (_formKey.currentState?.validate() != true) return;
                Navigator.of(context).pop(_WithdrawPayload(
                  double.parse(_amountCtrl.text.trim()),
                  _bankCtrl.text.trim(),
                ));
              },
              child: const Text('提交'),
            ),
            const SizedBox(height: 16),
          ],
        ),
      ),
    );
  }
}