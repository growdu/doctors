// test/providers/wallet_provider_test.dart
//
// walletProvider + walletTransactionsProvider + withdrawController 单测。
import 'package:dio/dio.dart';
import 'package:escort_app/providers/wallet_provider.dart';
import 'package:escort_app/services/api_client.dart';
import 'package:escort_app/services/token_storage.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

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

ProviderContainer _container(Dio dio) => ProviderContainer(overrides: [
      tokenStorageProvider.overrideWithValue(TokenStorage.forTest()),
      dioProvider.overrideWithValue(dio),
    ]);

void main() {
  group('walletProvider', () {
    test('拉取 → 解析 Wallet（balance + frozen + currency）', () async {
      final dio = _StubDio(
        getHandler: (opts) {
          if (opts.path == '/wallet') {
            return {'balance': 300.5, 'frozen': 50.0, 'currency': 'CNY'};
          }
          return {'list': <dynamic>[], 'total': 0};
        },
        postHandler: (_) => {},
      );
      final c = _container(dio);
      addTearDown(c.dispose);
      final w = await c.read(walletProvider.future);
      expect(w.balance, 300.5);
      expect(w.frozen, 50.0);
      expect(w.total, 350.5);
      expect(w.balanceText, '¥300.50');
    });
  });

  group('walletTransactionsProvider', () {
    test('拉取 → 解析 List<WalletTransaction>', () async {
      final dio = _StubDio(
        getHandler: (opts) {
          if (opts.path == '/wallet/transactions') {
            return {
              'list': [
                {
                  'id': 1,
                  'amount': 100.0,
                  'type': 'income',
                  'memo': '订单 7',
                  'created_at': '2026-09-25T09:00:00Z',
                },
                {
                  'id': 2,
                  'amount': -50.0,
                  'type': 'withdraw',
                  'memo': '提现',
                  'created_at': '2026-09-26T09:00:00Z',
                },
              ],
              'total': 2,
            };
          }
          return {};
        },
        postHandler: (_) => {},
      );
      final c = _container(dio);
      addTearDown(c.dispose);
      final list = await c.read(
        walletTransactionsProvider(const WalletTxKey(page: 1, size: 20)).future,
      );
      expect(list.length, 2);
      expect(list[0].type, TransactionType.income);
      expect(list[1].type, TransactionType.withdraw);
      expect(list[1].signedAmountText, '-¥50.00');
      expect(dio.getCalls.first['queryParameters']['page'], 1);
    });

    test('data.list 非 List → 返回空（兜底）', () async {
      final dio = _StubDio(
        getHandler: (_) => {},
        postHandler: (_) => {},
      );
      final c = _container(dio);
      addTearDown(c.dispose);
      final list = await c.read(
        walletTransactionsProvider(const WalletTxKey()).future,
      );
      expect(list, isEmpty);
    });
  });

  group('withdrawControllerProvider', () {
    test('submit → POST + state Success + invalidate wallet/tx',
        () async {
      final dio = _StubDio(
        getHandler: (_) => {'balance': 100.0, 'frozen': 0.0},
        postHandler: (opts) {
          if (opts.path == '/escorts/me/wallet/withdraw') {
            return {'withdraw_id': 99, 'status': 'pending'};
          }
          return {};
        },
      );
      final c = _container(dio);
      addTearDown(c.dispose);
      final ctrl = c.read(withdrawControllerProvider.notifier);
      await ctrl.submit(amount: 50.0, bankAccount: '6225****1234');
      expect(c.read(withdrawControllerProvider), isA<WithdrawSuccess>());
      expect(dio.postCalls.first['path'], '/escorts/me/wallet/withdraw');
      final body = dio.postCalls.first['data'] as Map<String, dynamic>;
      expect(body['amount'], 50.0);
    });

    test('DioException → state Failure', () async {
      final dio = _ErrDio();
      final c = _container(dio);
      addTearDown(c.dispose);
      final ctrl = c.read(withdrawControllerProvider.notifier);
      await expectLater(
        ctrl.submit(amount: 1.0, bankAccount: 'x'),
        throwsA(isA<DioException>()),
      );
      expect(c.read(withdrawControllerProvider), isA<WithdrawFailure>());
    });
  });
}

class _ErrDio implements Dio {
  @override
  dynamic noSuchMethod(Invocation invocation) {
    final m = invocation.memberName;
    if (m == #post) {
      return Future.error(DioException(
        requestOptions: RequestOptions(path: '/x'),
        type: DioExceptionType.badResponse,
        response: Response<dynamic>(
          requestOptions: RequestOptions(path: '/x'),
          statusCode: 400,
          data: {'code': 13103, 'message': '余额不足'},
        ),
      ));
    }
    throw UnimplementedError('${m.toString()} not stubbed');
  }
}