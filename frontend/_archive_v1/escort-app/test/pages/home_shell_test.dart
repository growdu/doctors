// test/pages/home_shell_test.dart
//
// HomeShell widget test —— 验证：
//   - 渲染 4 个 BottomNavBar tab
//   - IndexedStack 默认显示第一个 tab
//   - 点击 tab 切换（state 保持）
//
// 注：HomeShell 包含 ProfilePage / WalletPage / OrdersPage / InvitationsPage，
//     它们各自需要 dio / tokenStorage / authProvider 等依赖。
import 'package:dio/dio.dart';
import 'package:escort_app/pages/home/home_shell.dart';
import 'package:escort_app/providers/auth_provider.dart';
import 'package:escort_app/services/api_client.dart';
import 'package:escort_app/services/token_storage.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

class _StubDio implements Dio {
  @override
  dynamic noSuchMethod(Invocation invocation) {
    final m = invocation.memberName;
    final opts = (invocation.namedArguments[#options] as RequestOptions?) ??
        RequestOptions(path: '');
    if (m == #get) {
      return Future.value(Response<Map<String, dynamic>>(
        requestOptions: opts,
        statusCode: 200,
        data: {'code': 0, 'data': <String, dynamic>{}},
      ));
    }
    if (m == #post) {
      return Future.value(Response<Map<String, dynamic>>(
        requestOptions: opts,
        statusCode: 200,
        data: {'code': 0, 'data': <String, dynamic>{}},
      ));
    }
    return Future.value(null);
  }
}

class _StubAuth extends AuthNotifier {
  _StubAuth(Ref ref) : super(ref);
  @override
  Future<void> bootstrap() async {}
}

ProviderScope _wrap({required Widget child}) {
  return ProviderScope(
    overrides: [
      tokenStorageProvider.overrideWithValue(TokenStorage.forTest()),
      dioProvider.overrideWithValue(_StubDio()),
      authProvider.overrideWith((ref) => _StubAuth(ref)),
    ],
    child: child,
  );
}

void main() {
  testWidgets('渲染 4 个 BottomNavBar tab + IndexedStack', (tester) async {
    await tester.pumpWidget(
      _wrap(child: const MaterialApp(home: HomeShell())),
    );
    expect(find.byKey(const Key('home_navbar')), findsOneWidget);
    expect(find.byKey(const Key('home_indexed_stack')), findsOneWidget);
    expect(find.byKey(const Key('home_tab_invitations')), findsOneWidget);
    expect(find.byKey(const Key('home_tab_orders')), findsOneWidget);
    expect(find.byKey(const Key('home_tab_wallet')), findsOneWidget);
    expect(find.byKey(const Key('home_tab_profile')), findsOneWidget);
  });

  testWidgets('点击 Orders tab → 切到第二个', (tester) async {
    await tester.pumpWidget(
      _wrap(child: const MaterialApp(home: HomeShell())),
    );
    await tester.tap(find.byKey(const Key('home_tab_orders')));
    await tester.pumpAndSettle();
    final nav = tester.widget<NavigationBar>(
      find.byKey(const Key('home_navbar')),
    );
    expect(nav.selectedIndex, 1);
  });

  testWidgets('点击 Wallet tab → 切到第三个', (tester) async {
    await tester.pumpWidget(
      _wrap(child: const MaterialApp(home: HomeShell())),
    );
    await tester.tap(find.byKey(const Key('home_tab_wallet')));
    await tester.pumpAndSettle();
    final nav = tester.widget<NavigationBar>(
      find.byKey(const Key('home_navbar')),
    );
    expect(nav.selectedIndex, 2);
  });

  testWidgets('点击 Profile tab → 切到第四个', (tester) async {
    await tester.pumpWidget(
      _wrap(child: const MaterialApp(home: HomeShell())),
    );
    await tester.tap(find.byKey(const Key('home_tab_profile')));
    await tester.pumpAndSettle();
    final nav = tester.widget<NavigationBar>(
      find.byKey(const Key('home_navbar')),
    );
    expect(nav.selectedIndex, 3);
  });
}