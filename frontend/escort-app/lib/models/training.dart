// lib/models/training.dart
//
// 陪诊师培训课程（spec §A6 培训 + 考核）。
//
// 状态机：
//   - notStarted：未开始（progress=0）
//   - inProgress：进行中（0 < progress < 100）
//   - completed：已完成（progress=100 且考核通过）
//   - expired：已过期（需重学）
import '../utils/format.dart';

/// 培训状态。
enum TrainingStatus {
  notStarted,
  inProgress,
  completed,
  expired;

  static TrainingStatus fromString(String s) {
    switch (s) {
      case 'not_started':
        return TrainingStatus.notStarted;
      case 'in_progress':
        return TrainingStatus.inProgress;
      case 'completed':
        return TrainingStatus.completed;
      case 'expired':
        return TrainingStatus.expired;
    }
    throw ArgumentError('unknown TrainingStatus: $s');
  }

  String get wireValue {
    switch (this) {
      case TrainingStatus.notStarted:
        return 'not_started';
      case TrainingStatus.inProgress:
        return 'in_progress';
      case TrainingStatus.completed:
        return 'completed';
      case TrainingStatus.expired:
        return 'expired';
    }
  }

  String get displayName {
    switch (this) {
      case TrainingStatus.notStarted:
        return '未开始';
      case TrainingStatus.inProgress:
        return '进行中';
      case TrainingStatus.completed:
        return '已完成';
      case TrainingStatus.expired:
        return '已过期';
    }
  }
}

/// 单条培训课程。
class Training {
  final int id;
  final String title;
  final String? description;

  /// 进度（0-100）。
  final int progress;
  final TrainingStatus status;

  /// 课程时长（分钟）。
  final int? durationMinutes;

  /// 考核截止时间（可空；过期未考核则转 expired）。
  final DateTime? dueAt;

  /// 完成时间（status=completed 时回填）。
  final DateTime? completedAt;

  const Training({
    required this.id,
    required this.title,
    this.description,
    required this.progress,
    required this.status,
    this.durationMinutes,
    this.dueAt,
    this.completedAt,
  });

  factory Training.fromJson(Map<String, dynamic> j) => Training(
        id: j['id'] as int,
        title: j['title'] as String,
        description: j['description'] as String?,
        progress: (j['progress'] as num).toInt(),
        status: TrainingStatus.fromString(j['status'] as String),
        durationMinutes: (j['duration_minutes'] as num?)?.toInt(),
        dueAt: j['due_at'] == null ? null : DateTime.parse(j['due_at'] as String),
        completedAt: j['completed_at'] == null
            ? null
            : DateTime.parse(j['completed_at'] as String),
      );

  String get progressText => '$progress%';
  String? get dueAtText => dueAt == null ? null : formatDateTime(dueAt!);
}