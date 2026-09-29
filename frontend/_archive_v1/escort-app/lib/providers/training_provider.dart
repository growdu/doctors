// lib/providers/training_provider.dart
//
// 培训 provider —— GET /escorts/me/trainings（培训列表 + 进度 + 考核）。
//
// 设计：FutureProvider 一次性拉；后续按 plan 接入具体培训内容播放 / 考核答题。
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/training.dart';
import '../services/api_client.dart';

/// 培训列表 provider。
final trainingsProvider = FutureProvider<List<Training>>((ref) async {
  final dio = ref.read(dioProvider);
  final raw = await TrainingApi.listTrainings(dio);
  return raw
      .map((e) => Training.fromJson(e as Map<String, dynamic>))
      .toList();
});