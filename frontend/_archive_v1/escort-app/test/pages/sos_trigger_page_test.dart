// test/pages/sos_trigger_page_test.dart
//
// SosTriggerPage widget test —— 验证：
//   - 渲染 SOS 按钮 + 提示文本
//   - 长按 → 触发「已发送呼救」+ 位置显示
import 'package:escort_app/pages/sos/sos_trigger_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() {
    const ch = MethodChannel('escort_app.geolocator');
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(ch, (call) async {
      if (call.method == 'getPosition') {
        return {
          'lat': 39.9042,
          'lng': 116.4074,
          'timestamp': DateTime.now().millisecondsSinceEpoch,
        };
      }
      return null;
    });
  });

  tearDown(() {
    const ch = MethodChannel('escort_app.geolocator');
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(ch, null);
  });

  testWidgets('渲染 SOS 按钮 + 提示', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(home: const SosTriggerPage()),
      ),
    );
    expect(find.text('SOS 紧急呼救'), findsWidgets);
    expect(find.byKey(const Key('sos_button')), findsOneWidget);
    expect(find.text('长按 SOS 按钮 1.5 秒触发呼救'), findsOneWidget);
  });

  testWidgets('位置加载后显示坐标', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(home: const SosTriggerPage()),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('sos_pos')), findsOneWidget);
  });

  testWidgets('长按按钮 → 触发呼救状态', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(home: const SosTriggerPage()),
      ),
    );
    await tester.pumpAndSettle();
    // 模拟 long press
    await tester.longPress(find.byKey(const Key('sos_button')));
    // mock 500ms 后完成
    await tester.pump(const Duration(milliseconds: 600));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('sos_triggered_text')), findsOneWidget);
    expect(find.text('已触发'), findsOneWidget);
  });

  testWidgets('触发后按钮文案变为「已触发」', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(home: const SosTriggerPage()),
      ),
    );
    await tester.pumpAndSettle();
    await tester.longPress(find.byKey(const Key('sos_button')));
    await tester.pump(const Duration(milliseconds: 600));
    await tester.pumpAndSettle();
    expect(find.text('已触发'), findsOneWidget);
  });
}