// lib/pages/home/home_shell.dart
//
// Home Shell —— BottomNavigationBar 4 tabs + IndexedStack 保持 state。
//   tabs[0]: Invitations（抢单池 / 选人模式）
//   tabs[1]: Orders（我的订单）
//   tabs[2]: Wallet（钱包）
//   tabs[3]: Profile（个人中心）
//
// 设计：
//   - IndexedStack 保证每个 tab 内的滚动位置、表单状态等不被重建
//   - shell 自身不持有业务状态（业务状态在各自 provider）
import 'package:escort_app/pages/invitations/invitations_page.dart';
import 'package:escort_app/pages/orders/orders_page.dart';
import 'package:escort_app/pages/profile/profile_page.dart';
import 'package:escort_app/pages/wallet/wallet_page.dart';
import 'package:flutter/material.dart';

/// Home Shell（4 tabs BottomNavBar）。
class HomeShell extends StatefulWidget {
  const HomeShell({super.key});

  @override
  State<HomeShell> createState() => _HomeShellState();
}

class _HomeShellState extends State<HomeShell> {
  int _index = 0;

  static const _pages = <Widget>[
    InvitationsPage(),
    OrdersPage(),
    WalletPage(),
    ProfilePage(),
  ];

  void _onTap(int i) => setState(() => _index = i);

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: IndexedStack(
        key: const Key('home_indexed_stack'),
        index: _index,
        children: _pages,
      ),
      bottomNavigationBar: NavigationBar(
        key: const Key('home_navbar'),
        selectedIndex: _index,
        onDestinationSelected: _onTap,
        destinations: const [
          NavigationDestination(
            key: Key('home_tab_invitations'),
            icon: Icon(Icons.inbox),
            label: '邀请',
          ),
          NavigationDestination(
            key: Key('home_tab_orders'),
            icon: Icon(Icons.assignment),
            label: '订单',
          ),
          NavigationDestination(
            key: Key('home_tab_wallet'),
            icon: Icon(Icons.account_balance_wallet),
            label: '钱包',
          ),
          NavigationDestination(
            key: Key('home_tab_profile'),
            icon: Icon(Icons.person),
            label: '我的',
          ),
        ],
      ),
    );
  }
}