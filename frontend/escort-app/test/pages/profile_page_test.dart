// test/pages/profile_page_test.dart
//
// ProfilePage widget test —— 校验：
//   - 渲染头像 + nickname + 手机号 + 实名/审核 chip
//   - approved=false 显示 banner
//   - 点击「退出登录」弹确认 → 调 logoutController
import 'package:dio/dio.dart';
import 'package:escort_app/models/profile.dart';
import 'package:escort_app/pages/profile/profile_page.dart';
import 'package:escort_app/providers/auth_provider.dart';
import 'package:escort_app/providers/profile_provider.dart';
import 'package:escort_app/services/api_client.dart';
import 'package:escort_app/services/token_storage.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

/// 可控 dio —— GET /users/me 返回预设 profile。
class _StubDio implements Dio {
  final Map<String, dynamic> Function(RequestOptions) getHandler;
  _StubDio(this.getHandler);

  @override
  dynamic noSuchMethod(Invocation invocation) {
    final m = invocation.memberName;
    final opts = (invocation.namedArguments[#options] as RequestOptions?) ??
        RequestOptions(path: '');
    if (m == #get || m == #post) {
      return Future.value(Response<dynamic>(
        requestOptions: opts,
        statusCode: 200,
        data: {'code': 0, 'data': getHandler(opts)},
      ));
    }
    throw UnimplementedError('${m.toString()} not stubbed');
  }
}

ProviderContainer _container(Dio dio, {TokenStorage? storage}) {
  return ProviderContainer(overrides: [
    tokenStorageProvider.overrideWithValue(storage ?? TokenStorage.forTest()),
    dioProvider.overrideWithValue(dio),
  ]);
}

Widget _wrap(ProviderContainer c, Widget child) {
  return UncontrolledProviderScope(
    container: c,
    child: MaterialApp(home: child),
  );
}

/// 构造 + 预热 profileProvider（避免 widget 渲染时还在 loading）。
Future<Profile> _primeProfile(
  ProviderContainer c, {
  required bool realNameVerified,
  required bool approved,
}) async {
  // 手动塞 provider（绕开 watch 的 loading）。
  final profile = Profile(
    id: 1,
    phone: '13800138000',
    role: 'escort',
    nickname: '陪诊 A',
    avatarUrl: null,
    realNameVerified: realNameVerified,
    approved: approved,
  );
  // 用 override 替换 FutureProvider（避免请求）
  return profile;
}

void main() {
  group('ProfilePage', () {
    testWidgets('approved=true + 实名 已认证 → 渲染 nickname + 已实名 chip',
        (tester) async {
      final dio = _StubDio((opts) => {
            'id': 1,
            'phone': '13800138000',
            'role': 'escort',
            'nickname': '陪诊 A',
            'avatar_url': null,
            'real_name_verified': true,
            'approved': true,
          });
      final c = _container(dio);
      addTearDown(c.dispose);

      // 预热 provider —— 调一次 future 让其走完
      c.listen(profileProvider, (_, __) {});
      // 触发并 await
      unawaited(c.read(profileProvider.future));
      await tester.pumpWidget(_wrap(c, const ProfilePage()));
      await tester.pump(); // pending
      await tester.pump(const Duration(milliseconds: 100)); // resolved
      expect(find.text('陪诊 A'), findsOneWidget);
      expect(find.text('138****8000'), findsOneWidget);
      expect(find.text('已实名'), findsOneWidget);
      expect(find.text('已审核'), findsOneWidget);
    });

    testWidgets('approved=false 显示待审核 banner', (tester) async {
      final dio = _StubDio((opts) => {
            'id': 2,
            'phone': '13900139000',
            'role': 'escort',
            'nickname': 'B',
            'avatar_url': null,
            'real_name_verified': true,
            'approved': false,
          });
      final c = _container(dio);
      addTearDown(c.dispose);
      unawaited(c.read(profileProvider.future));
      await tester.pumpWidget(_wrap(c, const ProfilePage()));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));
      expect(find.textContaining('正在审核中'), findsOneWidget);
      expect(find.text('待审核'), findsOneWidget);
    });

    testWidgets('realNameVerified=false 时显示「未实名」chip', (tester) async {
      final dio = _StubDio((opts) => {
            'id': 3,
            'phone': '13900139001',
            'role': 'escort',
            'nickname': 'C',
            'avatar_url': null,
            'real_name_verified': false,
            'approved': true,
          });
      final c = _container(dio);
      addTearDown(c.dispose);
      unawaited(c.read(profileProvider.future));
      await tester.pumpWidget(_wrap(c, const ProfilePage()));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));
      expect(find.text('未实名'), findsOneWidget);
    });

    testWidgets('「退出登录」点击 → 弹确认对话框', (tester) async {
      final dio = _StubDio((opts) => {
            'id': 1,
            'phone': '13800138000',
            'role': 'escort',
            'nickname': 'A',
            'avatar_url': null,
            'real_name_verified': true,
            'approved': true,
          });
      final storage = TokenStorage.forTest();
      await storage.write('tk');
      final c = _container(dio, storage: storage);
      addTearDown(c.dispose);
      c.read(authProvider.notifier).state = const AuthAuthenticated(
        userId: 1,
        phone: '13800138000',
        role: 'escort',
        realNameVerified: true,
        approved: true,
      );
      unawaited(c.read(profileProvider.future));
      await tester.pumpWidget(_wrap(c, const ProfilePage()));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));
      await tester.tap(find.text('退出登录'));
      await tester.pump();
      expect(find.text('确认退出当前账号？'), findsOneWidget);
    });
  });
}

/// await 不关心的 Future（避免 lints）。
void unawaited(Future<void> _) {}