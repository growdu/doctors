// test/providers/profile_provider_test.dart
//
// profileProvider + realNameAuthController + logoutController 单测。
import 'package:dio/dio.dart';
import 'package:escort_app/providers/auth_provider.dart';
import 'package:escort_app/providers/profile_provider.dart';
import 'package:escort_app/services/api_client.dart';
import 'package:escort_app/services/token_storage.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

/// 通用 stub dio —— get / post 按 path 分发。
class _StubDio implements Dio {
  final Map<String, dynamic> Function(RequestOptions) getHandler;
  final Map<String, dynamic> Function(RequestOptions) postHandler;
  final List<dynamic> getCalls;
  final List<dynamic> postCalls;

  _StubDio({required this.getHandler, required this.postHandler})
      : getCalls = [],
        postCalls = [];

  @override
  dynamic noSuchMethod(Invocation invocation) {
    final m = invocation.memberName;
    final opts = (invocation.namedArguments[#options] as RequestOptions?) ??
        RequestOptions(path: '');
    if (m == #get) {
      getCalls.add(invocation.namedArguments);
      return Future.value(Response<dynamic>(
        requestOptions: opts,
        statusCode: 200,
        data: {'code': 0, 'data': getHandler(opts)},
      ));
    }
    if (m == #post) {
      postCalls.add(invocation.namedArguments);
      return Future.value(Response<dynamic>(
        requestOptions: opts,
        statusCode: 200,
        data: {'code': 0, 'data': postHandler(opts)},
      ));
    }
    throw UnimplementedError('${m.toString()} not stubbed');
  }
}

ProviderContainer _container(Dio dio) {
  return ProviderContainer(overrides: [
    tokenStorageProvider.overrideWithValue(TokenStorage.forTest()),
    dioProvider.overrideWithValue(dio),
  ]);
}

void main() {
  group('profileProvider', () {
    test('拉取 → 解析 Profile', () async {
      final dio = _StubDio(
        getHandler: (opts) {
          if (opts.path == '/users/me') {
            return {
              'id': 1,
              'phone': '13800138000',
              'role': 'escort',
              'nickname': '陪诊 A',
              'avatar_url': null,
              'real_name_verified': true,
              'approved': true,
            };
          }
          return {};
        },
        postHandler: (_) => {},
      );
      final c = _container(dio);
      addTearDown(c.dispose);
      final p = await c.read(profileProvider.future);
      expect(p.id, 1);
      expect(p.nickname, '陪诊 A');
      expect(p.realNameVerified, isTrue);
      expect(p.maskedPhone, '138****8000');
    });
  });

  group('realNameAuthControllerProvider', () {
    test('submit → POST + state 转 Success + invalidate profileProvider',
        () async {
      final dio = _StubDio(
        getHandler: (_) => {},
        postHandler: (opts) {
          if (opts.path == '/users/real-name/auth') {
            return {'verified': true};
          }
          return {};
        },
      );
      final c = _container(dio);
      addTearDown(c.dispose);
      final ctrl = c.read(realNameAuthControllerProvider.notifier);
      final ok = await ctrl.submit(
        realName: '张三',
        idCard: '110101199001011234',
      );
      expect(ok, isTrue);
      expect(c.read(realNameAuthControllerProvider),
          isA<RealNameAuthSuccess>());
      expect(dio.postCalls.first['path'], '/users/real-name/auth');
    });

    test('DioException → state 转 Failure', () async {
      final dio = _ErrorDio();
      final c = _container(dio);
      addTearDown(c.dispose);
      final ctrl = c.read(realNameAuthControllerProvider.notifier);
      await expectLater(
        ctrl.submit(realName: 'x', idCard: 'y'),
        throwsA(isA<DioException>()),
      );
      expect(c.read(realNameAuthControllerProvider),
          isA<RealNameAuthFailure>());
    });

    test('reset() → state 转 Idle', () {
      final c = _container(_StubDio(
        getHandler: (_) => {},
        postHandler: (_) => {},
      ));
      addTearDown(c.dispose);
      final ctrl = c.read(realNameAuthControllerProvider.notifier);
      ctrl.reset();
      expect(c.read(realNameAuthControllerProvider), isA<RealNameAuthIdle>());
    });
  });

  group('logoutControllerProvider', () {
    test('logout → 触发 AuthNotifier.logout 清状态', () async {
      final dio = _StubDio(
        getHandler: (_) => {},
        postHandler: (_) => {},
      );
      final storage = TokenStorage.forTest();
      await storage.write('tk');
      final c = ProviderContainer(overrides: [
        tokenStorageProvider.overrideWithValue(storage),
        dioProvider.overrideWithValue(dio),
      ]);
      addTearDown(c.dispose);
      // 模拟已登录态
      c.read(authProvider.notifier).state = const AuthAuthenticated(
        userId: 1,
        phone: '13800138000',
        role: 'escort',
        realNameVerified: false,
        approved: false,
      );
      await c.read(logoutControllerProvider.notifier).logout();
      expect(c.read(authProvider), isA<AuthUnauthenticated>());
      expect(await storage.read(), isNull);
    });
  });
}

class _ErrorDio implements Dio {
  @override
  dynamic noSuchMethod(Invocation invocation) {
    final m = invocation.memberName;
    if (m == #get || m == #post || m == #patch) {
      return Future.error(DioException(
        requestOptions: RequestOptions(path: '/x'),
        type: DioExceptionType.badResponse,
        response: Response<dynamic>(
          requestOptions: RequestOptions(path: '/x'),
          statusCode: 400,
          data: {'code': 13001, 'message': '参数错误'},
        ),
      ));
    }
    throw UnimplementedError('${m.toString()} not stubbed');
  }
}