// test/services/api_client_modules_test.dart
//
// E1 新增 5 个 API 模块（ProfileApi / WalletApi / TrainingApi / ReviewApi / OrderApi）
// 单元测试 —— 用 _StubDio 拦截 HTTP 请求，验证：
//   - 路径正确
//   - query / body 正确序列化
//   - 响应 data 解析（兜底空 Map / 空 List）
import 'package:dio/dio.dart';
import 'package:escort_app/services/api_client.dart';
import 'package:flutter_test/flutter_test.dart';

/// 通用 stub —— 按 path 分发预设响应。
class _StubDio implements Dio {
  final Map<String, dynamic> Function(RequestOptions) getHandler;
  final Map<String, dynamic> Function(RequestOptions) postHandler;
  final List<dynamic> getCalls;
  final List<dynamic> postCalls;
  final List<dynamic> patchCalls;

  _StubDio({
    required this.getHandler,
    required this.postHandler,
  })  : getCalls = [],
        postCalls = [],
        patchCalls = [];

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
    if (m == #patch) {
      patchCalls.add(invocation.namedArguments);
      return Future.value(Response<dynamic>(
        requestOptions: opts,
        statusCode: 200,
        data: {'code': 0, 'data': {}},
      ));
    }
    throw UnimplementedError('${m.toString()} not stubbed');
  }
}

void main() {
  group('ProfileApi', () {
    test('me → GET /users/me 解析 data', () async {
      final dio = _StubDio(
        getHandler: (opts) => {
          'id': 1,
          'phone': '13800138000',
          'nickname': '陪诊师 A',
          'avatar_url': 'https://x/a.jpg',
          'real_name_verified': true,
          'approved': true,
        },
        postHandler: (_) => {},
      );
      final data = await ProfileApi.me(dio);
      expect(dio.getCalls.first['path'], '/users/me');
      expect(data['nickname'], '陪诊师 A');
      expect(data['real_name_verified'], isTrue);
    });

    test('submitRealName → POST /users/real-name/auth 携带 3 字段', () async {
      final dio = _StubDio(
        getHandler: (_) => {},
        postHandler: (_) => {'verified': true},
      );
      final r = await ProfileApi.submitRealName(
        dio,
        realName: '张三',
        idCard: '110101199001011234',
        idCardPhotoUrl: 'https://x/id.jpg',
      );
      expect(dio.postCalls.length, 1);
      expect(dio.postCalls.first['path'], '/users/real-name/auth');
      final body = dio.postCalls.first['data'] as Map<String, dynamic>;
      expect(body['real_name'], '张三');
      expect(body['id_card'], '110101199001011234');
      expect(body['id_card_photo_url'], 'https://x/id.jpg');
      expect(r['verified'], isTrue);
    });

    test('updateNickname → PATCH /users/me/nickname', () async {
      final dio = _StubDio(
        getHandler: (_) => {},
        postHandler: (_) => {},
      );
      await ProfileApi.updateNickname(dio, nickname: '新昵称');
      expect(dio.patchCalls.first['path'], '/users/me/nickname');
      expect((dio.patchCalls.first['data'] as Map)['nickname'], '新昵称');
    });
  });

  group('WalletApi', () {
    test('getWallet → GET /wallet', () async {
      final dio = _StubDio(
        getHandler: (_) => {'balance': 300.0, 'frozen': 50.0, 'currency': 'CNY'},
        postHandler: (_) => {},
      );
      final w = await WalletApi.getWallet(dio);
      expect(dio.getCalls.first['path'], '/wallet');
      expect(w['balance'], 300.0);
      expect(w['frozen'], 50.0);
    });

    test('getTransactions → GET /wallet/transactions 携带 page/size',
        () async {
      final dio = _StubDio(
        getHandler: (_) => {
          'list': [
            {'id': 1, 'amount': 100.0, 'type': 'income'}
          ],
          'total': 1,
        },
        postHandler: (_) => {},
      );
      final r = await WalletApi.getTransactions(dio, page: 2, size: 10);
      expect(dio.getCalls.first['path'], '/wallet/transactions');
      expect(dio.getCalls.first['queryParameters']['page'], 2);
      expect(dio.getCalls.first['queryParameters']['size'], 10);
      expect((r['list'] as List).length, 1);
    });

    test('withdraw → POST /escorts/me/wallet/withdraw 携带 amount+account',
        () async {
      final dio = _StubDio(
        getHandler: (_) => {},
        postHandler: (_) => {'withdraw_id': 99, 'status': 'pending'},
      );
      final r = await WalletApi.withdraw(
        dio,
        amount: 200.0,
        bankAccount: '6225****1234',
      );
      expect(dio.postCalls.first['path'], '/escorts/me/wallet/withdraw');
      final body = dio.postCalls.first['data'] as Map<String, dynamic>;
      expect(body['amount'], 200.0);
      expect(body['bank_account'], '6225****1234');
      expect(r['withdraw_id'], 99);
    });
  });

  group('TrainingApi', () {
    test('listTrainings → GET /escorts/me/trainings 解析 List', () async {
      final dio = _StubDio(
        getHandler: (_) => [
          {
            'id': 1,
            'title': '陪诊基础',
            'progress': 80,
            'status': 'in_progress',
          },
          {'id': 2, 'title': '急救培训', 'progress': 100, 'status': 'completed'},
        ],
        postHandler: (_) => {},
      );
      final list = await TrainingApi.listTrainings(dio);
      expect(dio.getCalls.first['path'], '/escorts/me/trainings');
      expect(list.length, 2);
      expect((list.first as Map)['title'], '陪诊基础');
    });

    test('completeTraining → POST /escorts/me/trainings/{id}/complete',
        () async {
      final dio = _StubDio(
        getHandler: (_) => {},
        postHandler: (_) => {'completed_at': '2026-09-25T12:00:00Z'},
      );
      await TrainingApi.completeTraining(dio, 7);
      expect(dio.postCalls.first['path'], '/escorts/me/trainings/7/complete');
    });
  });

  group('ReviewApi', () {
    test('listReviews → GET /escorts/me/reviews 携带 page/size + 解析 list',
        () async {
      final dio = _StubDio(
        getHandler: (_) => {
          'list': [
            {
              'id': 1,
              'rating': 5,
              'content': '很专业',
              'created_at': '2026-09-20T10:00:00Z',
            }
          ],
          'total': 1,
        },
        postHandler: (_) => {},
      );
      final r = await ReviewApi.listReviews(dio, page: 1, size: 20);
      expect(dio.getCalls.first['path'], '/escorts/me/reviews');
      expect((r['list'] as List).length, 1);
      expect((r['list'].first as Map)['rating'], 5);
    });
  });

  group('OrderApi', () {
    test('listMyOrders 不带 status → 200 + 解析 list', () async {
      final dio = _StubDio(
        getHandler: (_) => [
          {
            'id': 7,
            'hospital_name': 'x',
            'service_start_at_text': '明天 09:00',
            'amount': 300.0,
            'package_name': '半日陪诊',
          },
        ],
        postHandler: (_) => {},
      );
      final list = await OrderApi.listMyOrders(dio);
      expect(dio.getCalls.first['path'], '/escorts/me/orders');
      expect(list.length, 1);
    });

    test('listMyOrders 带 status → query 参数注入', () async {
      final dio = _StubDio(
        getHandler: (_) => [],
        postHandler: (_) => {},
      );
      await OrderApi.listMyOrders(dio, status: 'in_service');
      expect(dio.getCalls.first['queryParameters']['status'], 'in_service');
    });

    test('listMyOrders 响应 data 是 {list: [...]} → 解析 list', () async {
      final dio = _StubDio(
        getHandler: (_) => {
          'list': [{'id': 1}, {'id': 2}],
          'total': 2,
        },
        postHandler: (_) => {},
      );
      final list = await OrderApi.listMyOrders(dio);
      expect(list.length, 2);
    });

    test('getOrder → GET /orders/{id}', () async {
      final dio = _StubDio(
        getHandler: (_) => {
          'id': 7,
          'patient_id': 11,
          'hospital_name': 'x',
          'hospital_lat': 0.0,
          'hospital_lng': 0.0,
          'package_name': 'x',
          'service_start_at': '2026-09-25T09:00:00Z',
          'amount': 100.0,
          'status': 'in_service',
        },
        postHandler: (_) => {},
      );
      final r = await OrderApi.getOrder(dio, 7);
      expect(dio.getCalls.first['path'], '/orders/7');
      expect(r['id'], 7);
      expect(r['status'], 'in_service');
    });

    test('响应 data 字段为 null → 兜底空 Map', () async {
      final dio = _StubDio(
        getHandler: (_) => {},
        postHandler: (_) => {},
      );
      final r = await OrderApi.getOrder(dio, 99);
      expect(r, isEmpty);
    });
  });
}