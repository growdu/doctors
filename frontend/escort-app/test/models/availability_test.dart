// test/models/availability_test.dart
//
// Availability + AvailabilityStatus 单测 —— isDeletable / toJson 覆盖。
import 'package:escort_app/models/availability.dart';
import 'package:flutter_test/flutter_test.dart';

Map<String, dynamic> _json({
  int id = 1,
  int escortId = 22,
  required String status,
  int? orderId,
  DateTime? startAt,
  DateTime? endAt,
}) =>
    {
      'id': id,
      'escort_id': escortId,
      'start_at': (startAt ?? DateTime.utc(2026, 9, 25, 14)).toIso8601String(),
      'end_at': (endAt ?? DateTime.utc(2026, 9, 25, 18)).toIso8601String(),
      'status': status,
      if (orderId != null) 'order_id': orderId,
    };

void main() {
  group('AvailabilityStatus.fromString', () {
    test('available', () {
      expect(
        AvailabilityStatus.fromString('available'),
        AvailabilityStatus.available,
      );
    });

    test('booked', () {
      expect(
        AvailabilityStatus.fromString('booked'),
        AvailabilityStatus.booked,
      );
    });

    test('canceled', () {
      expect(
        AvailabilityStatus.fromString('canceled'),
        AvailabilityStatus.canceled,
      );
    });

    test('未知值抛 ArgumentError', () {
      expect(() => AvailabilityStatus.fromString('foo'), throwsArgumentError);
    });
  });

  group('AvailabilityStatus.wireValue', () {
    test('round-trip 一致', () {
      for (final s in AvailabilityStatus.values) {
        expect(AvailabilityStatus.fromString(s.wireValue), s);
      }
    });
  });

  group('Availability.fromJson', () {
    test('available 状态 + 完整字段', () {
      final a = Availability.fromJson(_json(status: 'available'));
      expect(a.id, 1);
      expect(a.escortId, 22);
      expect(a.status, AvailabilityStatus.available);
      expect(a.orderId, isNull);
      expect(a.isDeletable, isTrue);
      expect(a.isBooked, isFalse);
    });

    test('booked 状态 + orderId 回填', () {
      final a = Availability.fromJson(
          _json(status: 'booked', orderId: 7));
      expect(a.status, AvailabilityStatus.booked);
      expect(a.orderId, 7);
      expect(a.isDeletable, isFalse);
      expect(a.isBooked, isTrue);
    });

    test('canceled 状态不可删', () {
      final a = Availability.fromJson(_json(status: 'canceled'));
      expect(a.isDeletable, isFalse);
      expect(a.isBooked, isFalse);
    });
  });

  group('Availability.toJson', () {
    test('round-trip 一致', () {
      final a1 = Availability.fromJson(_json(status: 'available'));
      final j = a1.toJson();
      final a2 = Availability.fromJson(j);
      expect(a2.id, a1.id);
      expect(a2.status, a1.status);
      expect(a2.startAt, a1.startAt);
      expect(a2.endAt, a1.endAt);
    });

    test('orderId null → toJson 不输出 order_id', () {
      final a = Availability.fromJson(_json(status: 'available'));
      final j = a.toJson();
      expect(j.containsKey('order_id'), isFalse);
    });

    test('orderId 非 null → toJson 输出', () {
      final a = Availability.fromJson(_json(status: 'booked', orderId: 9));
      final j = a.toJson();
      expect(j['order_id'], 9);
    });
  });

  group('display helpers', () {
    test('startText / endText 走 formatDateTime', () {
      final a = Availability.fromJson(_json(status: 'available'));
      // start: 2026-09-25 14:00; end: 2026-09-25 18:00
      expect(a.startText, '2026-09-25 14:00');
      expect(a.endText, '2026-09-25 18:00');
    });

    test('durationHours = 4（14:00→18:00）', () {
      final a = Availability.fromJson(_json(status: 'available'));
      expect(a.durationHours, 4);
    });
  });
}