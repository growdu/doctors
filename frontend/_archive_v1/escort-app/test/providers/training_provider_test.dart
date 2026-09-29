// test/providers/training_provider_test.dart
//
// trainingsProvider 单测。
import 'package:dio/dio.dart';
import 'package:escort_app/providers/training_provider.dart';
import 'package:escort_app/services/api_client.dart';
import 'package:escort_app/services/token_storage.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

class _StubDio implements Dio {
  final List<dynamic> Function(RequestOptions) handler;
  final List<dynamic> getCalls;
  _StubDio(this.handler) : getCalls = [];

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

void main() {
  group('trainingsProvider', () {
    test('拉取 → 解析 List<Training>（含进度 + 状态）', () async {
      final dio = _StubDio((opts) {
        if (opts.path == '/escorts/me/trainings') {
          return [
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
              'completed_at': '2026-09-20T00:00:00Z',
            },
          ];
        }
        return [];
      });
      final c = _container(dio);
      addTearDown(c.dispose);

      final list = await c.read(trainingsProvider.future);
      expect(list.length, 2);
      expect(list[0].status, TrainingStatus.inProgress);
      expect(list[0].progress, 80);
      expect(list[0].dueAtText, isNotNull);
      expect(list[1].status, TrainingStatus.completed);
      expect(list[1].completedAt, isNotNull);
    });

    test('data 非 List → 空（兜底）', () async {
      final dio = _StubDio((opts) => <dynamic>[]);
      final c = _container(dio);
      addTearDown(c.dispose);
      final list = await c.read(trainingsProvider.future);
      expect(list, isEmpty);
    });
  });
}