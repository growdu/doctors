// test/pages/transactions_page_test.dart
//
// TransactionsPage widget test —— 验证：
//   - 渲染 3 个过滤 chip
//   - 默认显示全部 5 条
//   - 点「收入」→ 仅 3 条收入
//   - 点「支出」→ 仅 2 条支出
import 'package:escort_app/pages/wallet/transactions_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('渲染 3 个过滤 chip', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(home: const TransactionsPage()),
      ),
    );
    expect(find.byKey(const Key('tx_filter_all')), findsOneWidget);
    expect(find.byKey(const Key('tx_filter_income')), findsOneWidget);
    expect(find.byKey(const Key('tx_filter_expense')), findsOneWidget);
  });

  testWidgets('默认显示全部 5 条', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(home: const TransactionsPage()),
      ),
    );
    expect(find.byKey(const Key('tx_item_t1')), findsOneWidget);
    expect(find.byKey(const Key('tx_item_t2')), findsOneWidget);
    expect(find.byKey(const Key('tx_item_t3')), findsOneWidget);
    expect(find.byKey(const Key('tx_item_t4')), findsOneWidget);
    expect(find.byKey(const Key('tx_item_t5')), findsOneWidget);
  });

  testWidgets('点「收入」→ 只剩 3 条收入', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(home: const TransactionsPage()),
      ),
    );
    await tester.tap(find.byKey(const Key('tx_filter_income')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('tx_item_t1')), findsOneWidget);
    expect(find.byKey(const Key('tx_item_t2')), findsOneWidget);
    expect(find.byKey(const Key('tx_item_t5')), findsOneWidget);
    // 支出不应存在
    expect(find.byKey(const Key('tx_item_t3')), findsNothing);
    expect(find.byKey(const Key('tx_item_t4')), findsNothing);
  });

  testWidgets('点「支出」→ 只剩 2 条支出', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(home: const TransactionsPage()),
      ),
    );
    await tester.tap(find.byKey(const Key('tx_filter_expense')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('tx_item_t3')), findsOneWidget);
    expect(find.byKey(const Key('tx_item_t4')), findsOneWidget);
    expect(find.byKey(const Key('tx_item_t1')), findsNothing);
  });
}