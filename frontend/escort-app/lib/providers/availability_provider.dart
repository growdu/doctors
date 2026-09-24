// lib/providers/availability_provider.dart
//
// 「我的空余时段」provider —— spec/2026-09-24-order-matching-redesign §3.2 + §4.1。
//
// 设计要点：
//   - FutureProvider 一次性拉 /escorts/me/availabilities
//   - createAvailabilityControllerProvider：POST 新时段（start_at + end_at）
//   - deleteAvailabilityControllerProvider：DELETE 单条时段
//   - 操作成功后 invalidate 主 provider 触发重新拉取
import 'dart:async';

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/availability.dart';
import '../services/api_client.dart';

/// `availabilityProvider` —— FutureProvider 拉我的时段列表。
final availabilityProvider = FutureProvider<List<Availability>>((ref) async {
  final dio = ref.read(dioProvider);
  final resp = await dio.get<Map<String, dynamic>>(
    '/escorts/me/availabilities',
  );
  final data = resp.data?['data'];
  final raw = (data is List) ? data : <dynamic>[];
  return raw
      .map((e) => Availability.fromJson(e as Map<String, dynamic>))
      .toList();
});

/// 增 / 删 controller provider。
final createAvailabilityControllerProvider = StateNotifierProvider.autoDispose<
    CreateAvailabilityController,
    AsyncValue<Availability?>>(
  (ref) => CreateAvailabilityController(ref),
);

final deleteAvailabilityControllerProvider = StateNotifierProvider.autoDispose<
    DeleteAvailabilityController,
    AsyncValue<void>>(
  (ref) => DeleteAvailabilityController(ref),
);

/// 创建时段。
class CreateAvailabilityController
    extends StateNotifier<AsyncValue<Availability?>> {
  final Ref ref;

  CreateAvailabilityController(this.ref)
      : super(const AsyncValue<Availability?>.data(null));

  Future<Availability> invoke(DateTime startAt, DateTime endAt) async {
    state = const AsyncValue<Availability?>.loading();
    try {
      final dio = ref.read(dioProvider);
      final resp = await dio.post<Map<String, dynamic>>(
        '/escorts/me/availabilities',
        data: {
          'start_at': startAt.toIso8601String(),
          'end_at': endAt.toIso8601String(),
        },
      );
      final data = resp.data?['data'] as Map<String, dynamic>?;
      if (data == null) {
        throw StateError('createAvailability: empty response');
      }
      final created = Availability.fromJson(data);
      state = AsyncValue<Availability?>.data(created);
      ref.invalidate(availabilityProvider);
      return created;
    } on DioException catch (e) {
      state = AsyncValue<Availability?>.error(e, StackTrace.current);
      rethrow;
    }
  }
}

/// 删除时段。
class DeleteAvailabilityController extends StateNotifier<AsyncValue<void>> {
  final Ref ref;

  DeleteAvailabilityController(this.ref) : super(const AsyncValue.data(null));

  Future<void> invoke(int availabilityId) async {
    state = const AsyncValue.loading();
    try {
      final dio = ref.read(dioProvider);
      await dio.delete<dynamic>('/escorts/me/availabilities/$availabilityId');
      state = const AsyncValue.data(null);
      ref.invalidate(availabilityProvider);
    } on DioException catch (e) {
      state = AsyncValue.error(e, StackTrace.current);
      rethrow;
    }
  }
}