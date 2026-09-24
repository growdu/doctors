// test/utils/format_test.dart
//
// formatMoney / formatDateTime / maskPhone 三个工具函数全覆盖。
import 'package:escort_app/utils/format.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('formatMoney', () {
    test('整数 → 加 .00', () {
      expect(formatMoney(100), '¥100.00');
      expect(formatMoney(0), '¥0.00');
    });

    test('两位小数 → 原样', () {
      expect(formatMoney(99.99), '¥99.99');
    });

    test('超过两位小数 → 四舍五入到两位', () {
      expect(formatMoney(99.999), '¥100.00');
      expect(formatMoney(99.994), '¥99.99');
    });

    test('负数 → 抛 ArgumentError', () {
      expect(() => formatMoney(-1), throwsArgumentError);
      expect(() => formatMoney(-0.01), throwsArgumentError);
    });

    test('整数类型 → 自动转字符串', () {
      expect(formatMoney(300), '¥300.00');
    });
  });

  group('formatDateTime', () {
    test('标准时间 → yyyy-MM-dd HH:mm', () {
      final dt = DateTime(2026, 9, 24, 15, 30);
      expect(formatDateTime(dt), '2026-09-24 15:30');
    });

    test('月/日/时/分 < 10 → 自动补 0', () {
      final dt = DateTime(2026, 1, 5, 9, 3);
      expect(formatDateTime(dt), '2026-01-05 09:03');
    });

    test('跨年边界', () {
      final dt = DateTime(2025, 12, 31, 23, 59);
      expect(formatDateTime(dt), '2025-12-31 23:59');
    });
  });

  group('maskPhone', () {
    test('11 位手机号 → 前3****后4', () {
      expect(maskPhone('13800138000'), '138****8000');
      expect(maskPhone('19912345678'), '199****5678');
    });

    test('非 11 位原样返回', () {
      expect(maskPhone('12345'), '12345');
      expect(maskPhone('123456789012'), '123456789012');
    });

    test('空字符串 → 空字符串', () {
      expect(maskPhone(''), '');
    });
  });
}