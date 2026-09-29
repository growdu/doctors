// test/utils/trace_test.dart
//
// newTraceId 格式 + 唯一性 + 前缀校验。
import 'package:escort_app/core/constants.dart';
import 'package:escort_app/utils/trace.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('newTraceId', () {
    test('前缀为 kTracePrefix（escort）', () {
      final id = newTraceId();
      expect(id, startsWith('$kTracePrefix-'));
    });

    test('格式为 escort-{ms}-{rand6}（3 段，rand 段定长 6 位）', () {
      final id = newTraceId();
      final parts = id.split('-');
      expect(parts.length, 3);
      expect(parts[0], kTracePrefix);
      expect(int.tryParse(parts[1]), isNotNull,
          reason: 'ms 段应为可解析整数');
      expect(parts[2].length, 6, reason: 'rand 段应为 6 位');
      expect(int.tryParse(parts[2]), isNotNull,
          reason: 'rand 段应为可解析整数');
      expect(int.parse(parts[2]), greaterThanOrEqualTo(0));
      expect(int.parse(parts[2]), lessThan(1000000));
    });

    test('连续两次调用 → 不同（ms / rand 任意一项不同）', () {
      final a = newTraceId();
      final b = newTraceId();
      expect(a == b, isFalse, reason: 'trace id 必须唯一');
    });

    test('ms 段与 DateTime.now().millisecondsSinceEpoch 接近（差 < 100ms）', () {
      final before = DateTime.now().millisecondsSinceEpoch;
      final id = newTraceId();
      final after = DateTime.now().millisecondsSinceEpoch;
      final msInId = int.parse(id.split('-')[1]);
      expect(msInId, greaterThanOrEqualTo(before));
      expect(msInId, lessThanOrEqualTo(after));
    });
  });
}