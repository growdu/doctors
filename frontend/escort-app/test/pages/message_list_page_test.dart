// test/pages/message_list_page_test.dart
//
// MessageListPage widget test —— 验证：
//   - 渲染 3 条会话
//   - 未读数显示
//   - 点击会话 → 跳 /message/chat/:id
import 'package:escort_app/pages/message/message_list_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

GoRouter _router() => GoRouter(
      initialLocation: '/message',
      routes: [
        GoRoute(
          path: '/message',
          builder: (c, s) => const MessageListPage(),
        ),
        GoRoute(
          path: '/message/chat/:id',
          builder: (c, s) =>
              Scaffold(body: Text('CHAT_${s.pathParameters['id']}')),
        ),
      ],
    );

void main() {
  testWidgets('渲染 3 条会话', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(routerConfig: _router()),
      ),
    );
    expect(find.byKey(const Key('msg_item_cs')), findsOneWidget);
    expect(find.byKey(const Key('msg_item_sys')), findsOneWidget);
    expect(find.byKey(const Key('msg_item_order')), findsOneWidget);
  });

  testWidgets('未读数渲染（cs=2, order=1）', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(routerConfig: _router()),
      ),
    );
    expect(find.byKey(const Key('msg_unread_cs')), findsOneWidget);
    expect(find.byKey(const Key('msg_unread_order')), findsOneWidget);
    expect(find.byKey(const Key('msg_unread_sys')), findsNothing);
  });

  testWidgets('点击会话 → 跳 /message/chat/cs', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(routerConfig: _router()),
      ),
    );
    await tester.tap(find.byKey(const Key('msg_item_cs')));
    await tester.pumpAndSettle();
    expect(find.text('CHAT_cs'), findsOneWidget);
  });

  testWidgets('标题「消息」', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(routerConfig: _router()),
      ),
    );
    expect(find.text('消息'), findsOneWidget);
  });
}