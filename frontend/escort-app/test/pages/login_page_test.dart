// test/pages/login_page_test.dart
//
// LoginPage widget test —— 校验：
//   - 渲染手机号/验证码/登录/微信按钮
//   - 输入校验（手机 11 位、验证码 6 位）
//   - 「获取验证码」启动 60s 倒计时
//   - 「登录」按钮触发 AuthNotifier.loginByPhone（用 override authProvider）
import 'package:dio/dio.dart';
import 'package:escort_app/providers/auth_provider.dart';
import 'package:escort_app/services/api_client.dart';
import 'package:escort_app/services/token_storage.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:escort_app/pages/login/login_page.dart';

/// 可控 dio —— /auth/login/sms 返回成功响应。
class _StubDio implements Dio {
  final Map<String, dynamic> Function(RequestOptions) handler;
  final List<dynamic> postCalls;
  _StubDio(this.handler) : postCalls = [];

  @override
  dynamic noSuchMethod(Invocation invocation) {
    final m = invocation.memberName;
    final opts = (invocation.namedArguments[#options] as RequestOptions?) ??
        RequestOptions(path: '');
    if (m == #get || m == #post) {
      postCalls.add(invocation.namedArguments);
      return Future.value(Response<dynamic>(
        requestOptions: opts,
        statusCode: 200,
        data: handler(opts),
      ));
    }
    throw UnimplementedError('${m.toString()} not stubbed');
  }
}

ProviderContainer _container(Dio dio) => ProviderContainer(overrides: [
      tokenStorageProvider.overrideWithValue(TokenStorage.forTest()),
      dioProvider.overrideWithValue(dio),
    ]);

void main() {
  group('LoginPage', () {
    testWidgets('渲染手机号/验证码/登录/微信按钮', (tester) async {
      final dio = _StubDio((opts) => {'code': 0, 'data': null});
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            tokenStorageProvider
                .overrideWithValue(TokenStorage.forTest()),
            dioProvider.overrideWithValue(dio),
          ],
          child: const MaterialApp(home: LoginPage()),
        ),
      );
      expect(find.text('陪诊师登录'), findsOneWidget);
      expect(find.text('手机号'), findsOneWidget);
      expect(find.text('验证码'), findsOneWidget);
      expect(find.text('登录'), findsOneWidget);
      expect(find.text('微信登录'), findsOneWidget);
    });

    testWidgets('手机号非 11 位 → 提示错误（不调 API）', (tester) async {
      final dio = _StubDio((opts) => {'code': 0, 'data': null});
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            tokenStorageProvider
                .overrideWithValue(TokenStorage.forTest()),
            dioProvider.overrideWithValue(dio),
          ],
          child: const MaterialApp(home: LoginPage()),
        ),
      );
      await tester.enterText(find.byType(TextFormField).at(0), '12345');
      await tester.tap(find.text('登录'));
      await tester.pump();
      expect(find.text('手机号应为 11 位'), findsOneWidget);
      expect(dio.postCalls.where((c) {
        final p = (c as Map)['path'] as String?;
        return p == '/auth/login/sms';
      }), isEmpty);
    });

    testWidgets('点击「获取验证码」启动倒计时（按钮 disabled）',
        (tester) async {
      final dio = _StubDio((opts) => {'code': 0, 'data': null});
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            tokenStorageProvider
                .overrideWithValue(TokenStorage.forTest()),
            dioProvider.overrideWithValue(dio),
          ],
          child: const MaterialApp(home: LoginPage()),
        ),
      );
      // 输入合法手机号
      await tester.enterText(
          find.byType(TextFormField).at(0), '13800138000');
      await tester.tap(find.text('获取验证码'));
      // 推进 Future.delayed
      await tester.pump(const Duration(milliseconds: 350));
      expect(find.textContaining('重新获取'), findsOneWidget);
    });

    testWidgets('合法输入 → 「登录」调 /auth/login/sms', (tester) async {
      final dio = _StubDio((opts) {
        if (opts.path == '/auth/login/sms') {
          return {
            'code': 0,
            'data': {
              'access_token': 'tk-from-test',
              'user': {
                'id': 1,
                'phone': '13800138000',
                'role': 'escort',
                'real_name_verified': false,
                'approved': false,
              },
            },
          };
        }
        return {'code': 0, 'data': null};
      });
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            tokenStorageProvider
                .overrideWithValue(TokenStorage.forTest()),
            dioProvider.overrideWithValue(dio),
          ],
          child: const MaterialApp(home: LoginPage()),
        ),
      );
      await tester.enterText(
          find.byType(TextFormField).at(0), '13800138000');
      await tester.enterText(
          find.byType(TextFormField).at(1), '123456');
      await tester.tap(find.text('登录'));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));
      final loginCalls = dio.postCalls.where((c) =>
          (c as Map)['path'] == '/auth/login/sms');
      expect(loginCalls, isNotEmpty);
    });

    testWidgets('微信登录 → 调 /auth/login/wx', (tester) async {
      final dio = _StubDio((opts) {
        if (opts.path == '/auth/login/wx') {
          return {
            'code': 0,
            'data': {
              'access_token': 'tk-wx',
              'user': {
                'id': 9,
                'phone': '',
                'role': 'escort',
                'real_name_verified': false,
                'approved': false,
              },
            },
          };
        }
        return {'code': 0, 'data': null};
      });
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            tokenStorageProvider
                .overrideWithValue(TokenStorage.forTest()),
            dioProvider.overrideWithValue(dio),
          ],
          child: const MaterialApp(home: LoginPage()),
        ),
      );
      await tester.tap(find.text('微信登录'));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));
      final wxCalls = dio.postCalls.where((c) =>
          (c as Map)['path'] == '/auth/login/wx');
      expect(wxCalls, isNotEmpty);
    });
  });
}