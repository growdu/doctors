// test/services/token_storage_test.dart
//
// TokenStorage fake 单测 —— 验证 write/read/delete 三态正确，无 platform channel 依赖。
import 'package:escort_app/services/token_storage.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('TokenStorage.forTest', () {
    test('初始 read → null', () async {
      final s = TokenStorage.forTest();
      expect(await s.read(), isNull);
    });

    test('write 后 read 返回原值', () async {
      final s = TokenStorage.forTest();
      await s.write('tk-abc-123');
      expect(await s.read(), 'tk-abc-123');
    });

    test('write 覆盖前值', () async {
      final s = TokenStorage.forTest();
      await s.write('old');
      await s.write('new');
      expect(await s.read(), 'new');
    });

    test('delete → null', () async {
      final s = TokenStorage.forTest();
      await s.write('tk');
      await s.delete();
      expect(await s.read(), isNull);
    });

    test('多次 delete 不报错', () async {
      final s = TokenStorage.forTest();
      await s.delete();
      await s.delete();
      expect(await s.read(), isNull);
    });

    test('不同 instance 互不影响（隔离）', () async {
      final a = TokenStorage.forTest();
      final b = TokenStorage.forTest();
      await a.write('tk-a');
      expect(await b.read(), isNull);
      expect(await a.read(), 'tk-a');
    });
  });
}