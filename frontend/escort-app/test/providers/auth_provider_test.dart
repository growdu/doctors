// test/providers/auth_provider_test.dart
//
// AuthState sealed + AuthNotifier bootstrap / logout / onUnauthorized 单测。
//
// 不依赖真实 dio：用 fake TokenStorage + override dioProvider 注入 _StubDio。
import 'package:dio/dio.dart';
import 'package:escort_app/providers/auth_provider.dart';
import 'package:escort_app/services/api_client.dart';
import 'package:escort_app/services/token_storage.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

/// 可控 dio —— 可在测试中预设 `/auth/me` / `/auth/login` 的响应。
class _StubDio implements Dio {
  final Map<String, dynamic> Function(RequestOptions) handler;

  _StubDio(this.handler);

  @override
  dynamic noSuchMethod(Invocation invocation) {
    final m = invocation.memberName;
    // 拦截 get/post/delete
    if (m == #get || m == #post) {
      final args = args as Map;
      final opts = args[#options] as RequestOptions;
      return Future.value(Response<dynamic>(
        requestOptions: opts,
        statusCode: 200,
        data: handler(opts),
      ));
    }
    if (m == #delete) {
      final args = args as Map;
      final opts = args[#options] as RequestOptions;
      return Future.value(Response<dynamic>(
        requestOptions: opts,
        statusCode: 204,
      ));
    }
    throw UnimplementedError('StubDio.${m.toString()} not stubbed');
  }
}

RequestOptions _opts(String path) => RequestOptions(path: path);

ProviderContainer _container({TokenStorage? storage, Dio? dio}) {
  return ProviderContainer(overrides: [
    tokenStorageProvider.overrideWithValue(storage ?? TokenStorage.forTest()),
    dioProvider.overrideWithValue(dio ?? _StubDio((opts) {
      // 默认 /auth/me 返回空 data → 走 unauthenticated 分支
      if (opts.path == '/auth/me') {
        return {'code': 0, 'data': null};
      }
      return {'code': 0, 'data': {}};
    })),
  ]);
}

void main() {
  group('AuthState 派生属性', () {
    test('AuthInitial.isAuthed → false', () {
      expect(const AuthInitial().isAuthed, isFalse);
      expect(const AuthInitial().isRealNameVerified, isFalse);
      expect(const AuthInitial().isApproved, isFalse);
    });

    test('AuthUnauthenticated.isAuthed → false', () {
      expect(const AuthUnauthenticated().isAuthed, isFalse);
    });

    test('AuthAuthenticated(isAuthed=true; verified/approved 各自独立)', () {
      const a1 = AuthAuthenticated(
        userId: 1,
        phone: '1',
        role: 'escort',
        realNameVerified: true,
        approved: false,
      );
      expect(a1.isAuthed, isTrue);
      expect(a1.isRealNameVerified, isTrue);
      expect(a1.isApproved, isFalse);

      const a2 = AuthAuthenticated(
        userId: 1,
        phone: '1',
        role: 'escort',
        realNameVerified: false,
        approved: true,
      );
      expect(a2.isRealNameVerified, isFalse);
      expect(a2.isApproved, isTrue);
    });

    test('copyWith 覆盖指定字段', () {
      const a = AuthAuthenticated(
        userId: 1,
        phone: '1',
        role: 'escort',
        realNameVerified: false,
        approved: false,
      );
      final b = a.copyWith(approved: true);
      expect(b.userId, 1);
      expect(b.approved, isTrue);
      expect(b.realNameVerified, isFalse);
    });
  });

  group('AuthNotifier.bootstrap', () {
    test('无 token → unauthenticated', () async {
      final c = _container();
      addTearDown(c.dispose);
      await c.read(authProvider.notifier).bootstrap();
      expect(c.read(authProvider), isA<AuthUnauthenticated>());
    });

    test('有 token + /auth/me 返回空 → 清 token + unauthenticated', () async {
      final storage = TokenStorage.forTest();
      await storage.write('stale-tk');
      final c = _container(storage: storage);
      addTearDown(c.dispose);
      await c.read(authProvider.notifier).bootstrap();
      expect(c.read(authProvider), isA<AuthUnauthenticated>());
      // token 应被清掉
      expect(await storage.read(), isNull);
    });

    test('有 token + /auth/me 返回用户 → authenticated', () async {
      final storage = TokenStorage.forTest();
      await storage.write('good-tk');
      final c = _container(
        storage: storage,
        dio: _StubDio((opts) {
          if (opts.path == '/auth/me') {
            return {
              'code': 0,
              'data': {
                'id': 42,
                'phone': '13800138000',
                'role': 'escort',
                'real_name_verified': true,
                'approved': true,
              },
            };
          }
          return {'code': 0, 'data': {}};
        }),
      );
      addTearDown(c.dispose);
      await c.read(authProvider.notifier).bootstrap();
      final s = c.read(authProvider);
      expect(s, isA<AuthAuthenticated>());
      final a = s as AuthAuthenticated;
      expect(a.userId, 42);
      expect(a.phone, '13800138000');
      expect(a.realNameVerified, isTrue);
      expect(a.approved, isTrue);
    });
  });

  group('AuthNotifier.logout', () {
    test('authenticated → unauthenticated + 清 token', () async {
      final storage = TokenStorage.forTest();
      await storage.write('good-tk');
      final c = _container(
        storage: storage,
        dio: _StubDio((opts) {
          if (opts.path == '/auth/me') {
            return {
              'code': 0,
              'data': {
                'id': 1, 'phone': '1', 'role': 'escort',
                'real_name_verified': false, 'approved': false,
              },
            };
          }
          return {'code': 0, 'data': {}};
        }),
      );
      addTearDown(c.dispose);
      await c.read(authProvider.notifier).bootstrap();
      expect(c.read(authProvider), isA<AuthAuthenticated>());

      await c.read(authProvider.notifier).logout();
      expect(c.read(authProvider), isA<AuthUnauthenticated>());
      expect(await storage.read(), isNull);
    });
  });

  group('AuthNotifier.onUnauthorized', () {
    test('触发 → 清 token + unauthenticated', () async {
      final storage = TokenStorage.forTest();
      await storage.write('good-tk');
      final c = _container(storage: storage);
      addTearDown(c.dispose);
      // 模拟已登录态
      const initial = AuthAuthenticated(
        userId: 1, phone: '1', role: 'escort',
        realNameVerified: false, approved: false,
      );
      c.read(authProvider.notifier).state = initial;

      await c.read(authProvider.notifier).onUnauthorized();
      expect(c.read(authProvider), isA<AuthUnauthenticated>());
      expect(await storage.read(), isNull);
    });
  });
}