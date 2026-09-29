// test/models/invitation_test.dart
//
// Invitation fromJson + isLive + remainingSeconds 单测。
import 'package:escort_app/models/invitation.dart';
import 'package:flutter_test/flutter_test.dart';

Map<String, dynamic> _json({required DateTime expireAt, int orderId = 7}) => {
      'order_id': orderId,
      'hospital_name': '北京协和医院',
      'hospital_lat': 39.9,
      'hospital_lng': 116.4,
      'package_name': '半日陪诊',
      'service_start_at': '2026-09-25T09:00:00Z',
      'amount': 300.0,
      'escort_pending_expire_at': expireAt.toIso8601String(),
    };

void main() {
  group('Invitation.fromJson', () {
    test('完整字段解析', () {
      final j = _json(
        expireAt: DateTime.utc(2026, 9, 24, 15, 0, 30),
        orderId: 42,
      );
      final inv = Invitation.fromJson(j);
      expect(inv.orderId, 42);
      expect(inv.hospitalName, '北京协和医院');
      expect(inv.amount, 300.0);
      expect(inv.escortPendingExpireAt.year, 2026);
      expect(inv.serviceStartAt.year, 2026);
      expect(inv.amountText, '¥300.00');
    });

    test('amountText 走 formatMoney（两位小数）', () {
      final inv = Invitation.fromJson(_json(
        expireAt: DateTime.now().add(const Duration(seconds: 10)),
      ));
      expect(inv.amountText, '¥300.00');
    });
  });

  group('isLive', () {
    test('未到期 → true', () {
      final inv = Invitation.fromJson(_json(
        expireAt: DateTime.now().add(const Duration(seconds: 10)),
      ));
      expect(inv.isLive, isTrue);
    });

    test('已过期 → false', () {
      final inv = Invitation.fromJson(_json(
        expireAt: DateTime.now().subtract(const Duration(seconds: 5)),
      ));
      expect(inv.isLive, isFalse);
    });

    test('刚好到期（误差 1ms）→ false', () {
      final inv = Invitation.fromJson(_json(
        expireAt: DateTime.now().subtract(const Duration(milliseconds: 1)),
      ));
      expect(inv.isLive, isFalse);
    });
  });

  group('remainingSeconds', () {
    test('剩余 10s → 10', () {
      final inv = Invitation.fromJson(_json(
        expireAt: DateTime.now().add(const Duration(seconds: 10)),
      ));
      expect(inv.remainingSeconds, lessThanOrEqualTo(10));
      expect(inv.remainingSeconds, greaterThanOrEqualTo(8));
    });

    test('已过期 → 0', () {
      final inv = Invitation.fromJson(_json(
        expireAt: DateTime.now().subtract(const Duration(seconds: 30)),
      ));
      expect(inv.remainingSeconds, 0);
    });
  });
}