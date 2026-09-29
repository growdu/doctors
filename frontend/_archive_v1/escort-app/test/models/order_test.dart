// test/models/order_test.dart
//
// Order + OrderStatus + OrderSummary 单测 —— 覆盖选人模式新字段 + 新状态。
import 'package:escort_app/models/order.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('OrderStatus.fromString', () {
    test('selecting_escort → selectingEscort', () {
      expect(
        OrderStatus.fromString('selecting_escort'),
        OrderStatus.selectingEscort,
      );
    });

    test('escort_pending_acceptance → escortPendingAcceptance', () {
      expect(
        OrderStatus.fromString('escort_pending_acceptance'),
        OrderStatus.escortPendingAcceptance,
      );
    });

    test('pending_acceptance 旧值兼容 → escortPendingAcceptance', () {
      expect(
        OrderStatus.fromString('pending_acceptance'),
        OrderStatus.escortPendingAcceptance,
      );
    });

    test('未知值 → 抛 ArgumentError', () {
      expect(() => OrderStatus.fromString('foo_bar'), throwsArgumentError);
    });
  });

  group('OrderStatus.wireValue', () {
    test('selectingEscort → selecting_escort', () {
      expect(OrderStatus.selectingEscort.wireValue, 'selecting_escort');
    });

    test('escortPendingAcceptance → escort_pending_acceptance', () {
      expect(
        OrderStatus.escortPendingAcceptance.wireValue,
        'escort_pending_acceptance',
      );
    });
  });

  group('Order.fromJson', () {
    test('完整字段（含 escort_pending_acceptance + selected_escort_id）', () {
      final j = {
        'id': 7,
        'order_no': 'O-001',
        'patient_id': 11,
        'escort_id': 22,
        'selected_escort_id': 22,
        'hospital_id': 33,
        'hospital_name': '北京协和医院',
        'hospital_lat': 39.9,
        'hospital_lng': 116.4,
        'package_name': '半日陪诊',
        'service_start_at': '2026-09-25T09:00:00Z',
        'amount': 300.0,
        'final_amount': 300.0,
        'status': 'escort_pending_acceptance',
        'escort_pending_expire_at': '2026-09-24T15:00:30Z',
        'created_at': '2026-09-24T15:00:00Z',
      };
      final o = Order.fromJson(j);
      expect(o.id, 7);
      expect(o.status, OrderStatus.escortPendingAcceptance);
      expect(o.selectedEscortId, 22);
      expect(o.escortId, 22);
      expect(o.hospitalName, '北京协和医院');
      expect(o.amount, 300.0);
      expect(o.escortPendingExpireAt, isNotNull);
      expect(o.createdAt, isNotNull);
    });

    test('selecting_escort 状态 + selectedEscortId 必填语义', () {
      final j = {
        'id': 8,
        'patient_id': 11,
        'selected_escort_id': 99,
        'hospital_name': '上海瑞金',
        'hospital_lat': 31.2,
        'hospital_lng': 121.5,
        'package_name': '全日陪诊',
        'service_start_at': '2026-09-26T09:00:00Z',
        'amount': 500.0,
        'status': 'selecting_escort',
      };
      final o = Order.fromJson(j);
      expect(o.status, OrderStatus.selectingEscort);
      expect(o.selectedEscortId, 99);
      expect(o.escortPendingExpireAt, isNull); // selecting_escort 时无 30s 倒计时
    });

    test('escortPendingExpireAt / completedAt / createdAt 为 null → 安全', () {
      final j = {
        'id': 9,
        'patient_id': 1,
        'hospital_name': 'x',
        'hospital_lat': 0.0,
        'hospital_lng': 0.0,
        'package_name': 'x',
        'service_start_at': '2026-09-25T09:00:00Z',
        'amount': 0.0,
        'status': 'created',
      };
      final o = Order.fromJson(j);
      expect(o.escortPendingExpireAt, isNull);
      expect(o.completedAt, isNull);
      expect(o.createdAt, isNull);
    });

    test('amountText 走 formatMoney', () {
      final o = Order.fromJson({
        'id': 1,
        'patient_id': 1,
        'hospital_name': 'x',
        'hospital_lat': 0.0,
        'hospital_lng': 0.0,
        'package_name': 'x',
        'service_start_at': '2026-09-25T09:00:00Z',
        'amount': 300.0,
        'status': 'paid',
      });
      expect(o.amountText, '¥300.00');
    });
  });

  group('Order.toJson', () {
    test('round-trip：fromJson → toJson → fromJson 一致', () {
      final j = {
        'id': 7,
        'patient_id': 11,
        'selected_escort_id': 22,
        'hospital_name': '北京协和',
        'hospital_lat': 39.9,
        'hospital_lng': 116.4,
        'package_name': '半日陪诊',
        'service_start_at': '2026-09-25T09:00:00.000Z',
        'amount': 300.0,
        'status': 'escort_pending_acceptance',
        'escort_pending_expire_at': '2026-09-24T15:00:30.000Z',
        'created_at': '2026-09-24T15:00:00.000Z',
      };
      final o1 = Order.fromJson(j);
      final j2 = o1.toJson();
      final o2 = Order.fromJson(j2);
      expect(o2.id, o1.id);
      expect(o2.status, o1.status);
      expect(o2.selectedEscortId, o1.selectedEscortId);
      expect(o2.amount, o1.amount);
      expect(o2.escortPendingExpireAt, o1.escortPendingExpireAt);
    });

    test('可选字段为 null → toJson 不输出对应 key', () {
      final o = Order(
        id: 1,
        patientId: 1,
        hospitalName: 'x',
        hospitalLat: 0.0,
        hospitalLng: 0.0,
        packageName: 'x',
        serviceStartAt: DateTime.utc(2026, 9, 25, 9),
        amount: 100.0,
        status: OrderStatus.paid,
      );
      final j = o.toJson();
      expect(j.containsKey('escort_pending_expire_at'), isFalse);
      expect(j.containsKey('completed_at'), isFalse);
      expect(j.containsKey('created_at'), isFalse);
    });
  });

  group('OrderSummary.fromJson', () {
    test('邀请卡片瘦字段', () {
      final j = {
        'id': 7,
        'hospital_name': '北京协和医院',
        'service_start_at_text': '明天 09:00',
        'amount': 300.0,
        'package_name': '半日陪诊',
      };
      final s = OrderSummary.fromJson(j);
      expect(s.id, 7);
      expect(s.amount, 300.0);
      expect(s.serviceStartAtText, '明天 09:00');
      expect(s.amountText, '¥300.00');
      expect(s.status, isNull);
      expect(s.escortPendingExpireAt, isNull);
    });

    test('带 status + escortPendingExpireAt', () {
      final j = {
        'id': 7,
        'hospital_name': 'x',
        'service_start_at_text': 'x',
        'amount': 1.0,
        'package_name': 'x',
        'status': 'escort_pending_acceptance',
        'escort_pending_expire_at': '2026-09-24T15:00:30Z',
      };
      final s = OrderSummary.fromJson(j);
      expect(s.status, OrderStatus.escortPendingAcceptance);
      expect(s.escortPendingExpireAt, isNotNull);
    });
  });
}