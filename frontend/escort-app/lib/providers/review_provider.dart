// lib/providers/review_provider.dart
//
// 评价列表 provider —— GET /escorts/me/reviews。
//
// 设计：FutureProvider.family 按 (page, size) 分页拉取。
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/review.dart';
import '../services/api_client.dart';

/// 评价分页参数（family key）。
class ReviewListKey {
  final int page;
  final int size;
  const ReviewListKey({this.page = 1, this.size = 20});

  @override
  bool operator ==(Object other) =>
      other is ReviewListKey && other.page == page && other.size == size;
  @override
  int get hashCode => Object.hash(page, size);
}

/// 评价列表 provider（FutureProvider.family）。
final reviewsProvider =
    FutureProvider.family<List<Review>, ReviewListKey>((ref, key) async {
  final dio = ref.read(dioProvider);
  final data = await ReviewApi.listReviews(dio, page: key.page, size: key.size);
  final raw = data['list'];
  if (raw is! List) return <Review>[];
  return raw
      .map((e) => Review.fromJson(e as Map<String, dynamic>))
      .toList();
});