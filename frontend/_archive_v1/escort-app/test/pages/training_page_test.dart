// test/pages/training_page_test.dart
//
// TrainingPage widget test —— 校验：
//   - 渲染统计卡片（总 / 已完成 / 进行中）+ 课程列表
//   - 进度条 + 状态 chip 渲染
//   - 空态显示「暂无培训课程」
import 'package:dio/dio.dart';
import 'package:escort_app/pages/training/training_page.dart';
import 'package:escort_app/providers/training_provider.dart';
import 'package:escort_app/services/api_client.dart';
import 'package:escort_app/services/token_storage.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

class _StubDio implements Dio {
  final dynamic Function(RequestOptions) handler;
  _StubDio(this.handler);

  @override
  dynamic noSuchMethod(Invocation invocation) {
    final m = invocation.memberName;
    final opts = (invocation.namedArguments[#options] as RequestOptions?) ??
        RequestOptions(path: '');
    if (m == #get) {
      return Future.value(Response<dynamic>(
        requestOptions: opts,
        statusCode: 200,
        data: {'code': 0, 'data': handler(opts)},
      ));
    }
    throw UnimplementedError('${m.toString()} not stubbed');
  }
}

ProviderContainer _container(Dio dio) => ProviderContainer(overrides: [
      tokenStorageProvider.overrideWithValue(TokenStorage.forTest()),
      dioProvider.overrideWithValue(dio),
    ]);

Widget _wrap(ProviderContainer c) => UncontrolledProviderScope(
      container: c,
      child: const MaterialApp(home: TrainingPage()),
    );

void main() {
  group('TrainingPage', () {
    testWidgets('渲染统计卡片 + 课程列表（含进度 + 状态 chip）',
        (tester) async {
      final dio = _StubDio((opts) => [
            {
              'id': 1,
              'title': '陪诊基础',
              'description': '服务规范',
              'progress': 80,
              'status': 'in_progress',
              'duration_minutes': 60,
              'due_at': '2026-10-01T00:00:00Z',
            },
            {
              'id': 2,
              'title': '急救培训',
              'progress': 100,
              'status': 'completed',
              'duration_minutes': 30,
            },
          ]);
      final c = _container(dio);
      addTearDown(c.dispose);
      unawaited(c.read(trainingsProvider.future));
      await tester.pumpWidget(_wrap(c));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));
      // 统计卡片
      expect(find.text('总课程'), findsOneWidget);
      expect(find.text('已完成'), findsOneWidget);
      expect(find.text('进行中'), findsOneWidget);
      expect(find.text('2'), findsOneWidget); // 总课程 = 2
      expect(find.text('1'), findsWidgets); // 已完成 = 1, 进行中 = 1
      // 课程标题
      expect(find.text('陪诊基础'), findsOneWidget);
      expect(find.text('急救培训'), findsOneWidget);
      // 状态 chip
      expect(find.text('进行中'), findsWidgets);
      expect(find.text('已完成'), findsWidgets);
      // 进度
      expect(find.text('进度 80%'), findsOneWidget);
      // 时长
      expect(find.text('时长 60 分钟'), findsOneWidget);
    });

    testWidgets('空态显示「暂无培训课程」', (tester) async {
      final dio = _StubDio((opts) => []);
      final c = _container(dio);
      addTearDown(c.dispose);
      unawaited(c.read(trainingsProvider.future));
      await tester.pumpWidget(_wrap(c));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));
      expect(find.text('暂无培训课程'), findsOneWidget);
    });

    testWidgets('error 态显示错误 + 重试按钮', (tester) async {
      final dio = _ErrDio();
      final c = _container(dio);
      addTearDown(c.dispose);
      unawaited(c.read(trainingsProvider.future));
      await tester.pumpWidget(_wrap(c));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));
      expect(find.textContaining('加载失败'), findsOneWidget);
      expect(find.text('重试'), findsOneWidget);
    });
  });
}

class _ErrDio implements Dio {
  @override
  dynamic noSuchMethod(Invocation invocation) {
    final m = invocation.memberName;
    if (m == #get) {
      return Future.error(DioException(
        requestOptions: RequestOptions(path: '/x'),
        type: DioExceptionType.badResponse,
        response: Response<dynamic>(
          requestOptions: RequestOptions(path: '/x'),
          statusCode: 500,
        ),
      ));
    }
    throw UnimplementedError('${m.toString()} not stubbed');
  }
}

void unawaited(Future<void> _) {}