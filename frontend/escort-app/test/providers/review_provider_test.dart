// test/providers/review_provider_test.dart
//
// reviewsProvider 单测。
import 'package:dio/dio.dart';
import 'package:escort_app/providers/review_provider.dart';
import 'package:escort_app/services/api_client.dart';
import 'package:escort_app/services/token_storage.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

class _StubDio implements Dio {
  final Map<String, dynamic> Function(RequestOptions) handler;
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
  group('reviewsProvider', () {
    test('拉取 → 解析 List<Review>（含 rating + tags）', () async {
      final dio = _StubDio((opts) {
        if (opts.path == '/escorts/me/reviews') {
          return {
            'list': [
              {
                'id': 1,
                'order_id': 7,
                'patient_nickname': '张*',
                'rating': 5,
                'content': '很专业，准时到达',
                'created_at': '2026-09-20T10:00:00Z',
                'tags': ['专业', '耐心'],
              },
              {
                'id': 2,
                'order_id': 8,
                'patient_nickname': '李*',
                'rating': 4,
                'content': '服务好',
                'created_at': '2026-09-21T10:00:00Z',
                'tags': [],
              },
            ],
            'total': 2,
          };
        }
        return {};
      });
      final c = _container(dio);
      addTearDown(c.dispose);
      final list = await c.read(
        reviewsProvider(const ReviewListKey(page: 1, size: 20)).future,
      );
      expect(list.length, 2);
      expect(list[0].rating, 5);
      expect(list[0].tags, ['专业', '耐心']);
      expect(list[0].ratingText, '★★★★★');
      expect(list[1].rating, 4);
      expect(list[1].ratingText, '★★★★☆');
      expect(dio.getCalls.first['queryParameters']['page'], 1);
    });

    test('data.list 非 List → 空（兜底）', () async {
      final dio = _StubDio((opts) => {});
      final c = _container(dio);
      addTearDown(c.dispose);
      final list = await c.read(
        reviewsProvider(const ReviewListKey()).future,
      );
      expect(list, isEmpty);
    });
  });
}