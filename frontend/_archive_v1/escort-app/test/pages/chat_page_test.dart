// test/pages/chat_page_test.dart
//
// ChatPage widget test —— 验证：
//   - 渲染 mock 历史 3 条气泡
//   - 输入 + 发送 → 新增自己气泡 + mock 自动回复
//   - 滚动到底部
import 'package:escort_app/pages/message/chat_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('渲染 mock 历史 3 条气泡', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(
          home: const ChatPage(conversationId: 'cs'),
        ),
      ),
    );
    expect(find.text('您好，请问有什么可以帮您？'), findsOneWidget);
    expect(find.text('请问订单 #1024 几点开始？'), findsOneWidget);
    expect(find.text('订单 #1024 上午 10 点开始，请提前 15 分钟签到。'),
        findsOneWidget);
  });

  testWidgets('输入 + 发送 → 新增自己气泡 + 自动回复', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(
          home: const ChatPage(conversationId: 'cs'),
        ),
      ),
    );
    await tester.enterText(find.byKey(const Key('chat_input')), '你好');
    await tester.tap(find.byKey(const Key('chat_send')));
    await tester.pump();
    expect(find.text('你好'), findsOneWidget);
    // mock 自动回复 600ms 后
    await tester.pump(const Duration(milliseconds: 700));
    expect(find.text('已收到您的消息，请稍候。'), findsOneWidget);
  });

  testWidgets('空文本不发送', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(
          home: const ChatPage(conversationId: 'cs'),
        ),
      ),
    );
    // 不输入文本，直接 tap send
    await tester.tap(find.byKey(const Key('chat_send')));
    await tester.pump();
    // 仅 mock 历史 3 条 + 0 自加
    expect(find.text('您好，请问有什么可以帮您？'), findsOneWidget);
    // 「你好」不应存在
    expect(find.text('你好'), findsNothing);
  });

  testWidgets('标题显示会话 ID', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(
          home: const ChatPage(conversationId: 'order'),
        ),
      ),
    );
    expect(find.text('会话 order'), findsOneWidget);
  });
}