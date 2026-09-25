// lib/models/review.dart
//
// 陪诊师收到的评价（spec §A7 个人中心评价列表）。
//
// 数据来源：GET /escorts/me/reviews。
// v1.2 仅展示用；写评价流程由患者端（patient-miniapp）发起。
import '../utils/format.dart';

/// 单条评价。
class Review {
  final int id;

  /// 关联订单 ID。
  final int orderId;

  /// 患者昵称（已脱敏为「张*」形式；后端处理）。
  final String patientNickname;

  /// 评分（1-5 整数）。
  final int rating;

  /// 评价内容。
  final String content;

  /// 评价时间。
  final DateTime createdAt;

  /// 标签数组（['专业', '耐心']）。
  final List<String> tags;

  const Review({
    required this.id,
    required this.orderId,
    required this.patientNickname,
    required this.rating,
    required this.content,
    required this.createdAt,
    this.tags = const <String>[],
  });

  factory Review.fromJson(Map<String, dynamic> j) => Review(
        id: j['id'] as int,
        orderId: (j['order_id'] as num).toInt(),
        patientNickname: j['patient_nickname'] as String,
        rating: (j['rating'] as num).toInt(),
        content: (j['content'] as String?) ?? '',
        createdAt: DateTime.parse(j['created_at'] as String),
        tags: (j['tags'] as List<dynamic>?)
                ?.map((e) => e as String)
                .toList() ??
            const <String>[],
      );

  String get createdAtText => formatDateTime(createdAt.toLocal());
  String get ratingText => '★' * rating + '☆' * (5 - rating);
}