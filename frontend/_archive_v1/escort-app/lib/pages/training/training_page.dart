// lib/pages/training/training_page.dart
//
// 培训页 —— plan §3 E7。
//
// 业务流：
//   1. 加载 trainingsProvider（GET /escorts/me/trainings）
//   2. 渲染 ListView（每行：标题 + 进度条 + 状态 chip + 截止日期）
//   3. 顶部统计：总课程数 / 已完成 / 进行中
//   4. 点击课程 → 占位（v1 简化：弹 SnackBar 提示后续 plan 接入播放）
//
// 设计要点：
//   - 顶部统计卡片：3 个数字（总 / 完成 / 进行中）
//   - 每条课程：LinearProgressIndicator（0-100%）
//   - 状态 chip 复用 _StatusChip widget
import 'package:escort_app/models/training.dart';
import 'package:escort_app/providers/training_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// 培训页。
class TrainingPage extends ConsumerWidget {
  const TrainingPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final asyncList = ref.watch(trainingsProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('培训课程'),
        actions: [
          IconButton(
            tooltip: '刷新',
            icon: const Icon(Icons.refresh),
            onPressed: () => ref.invalidate(trainingsProvider),
          ),
        ],
      ),
      body: asyncList.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (err, _) => Center(
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              const Icon(Icons.error_outline, size: 48, color: Colors.red),
              const SizedBox(height: 12),
              Text('加载失败：$err', textAlign: TextAlign.center),
              const SizedBox(height: 12),
              FilledButton(
                onPressed: () => ref.invalidate(trainingsProvider),
                child: const Text('重试'),
              ),
            ],
          ),
        ),
        data: (list) {
          if (list.isEmpty) {
            return Center(
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Icon(Icons.school_outlined,
                      size: 64, color: Colors.grey[400]),
                  const SizedBox(height: 12),
                  const Text('暂无培训课程'),
                  const SizedBox(height: 4),
                  const Text(
                    '平台暂未发布新培训',
                    style: TextStyle(color: Colors.grey),
                  ),
                ],
              ),
            );
          }
          return RefreshIndicator(
            onRefresh: () async {
              ref.invalidate(trainingsProvider);
              await ref.read(trainingsProvider.future);
            },
            child: ListView(
              padding: const EdgeInsets.all(12),
              children: [
                _StatsCard(trainings: list),
                const SizedBox(height: 12),
                ...list.map((t) => Padding(
                      padding: const EdgeInsets.only(bottom: 8),
                      child: TrainingCard(training: t),
                    )),
              ],
            ),
          );
        },
      ),
    );
  }
}

/// 顶部统计卡片。
class _StatsCard extends StatelessWidget {
  final List<Training> trainings;
  const _StatsCard({required this.trainings});

  @override
  Widget build(BuildContext context) {
    final total = trainings.length;
    final completed = trainings
        .where((t) => t.status == TrainingStatus.completed)
        .length;
    final inProgress = trainings
        .where((t) => t.status == TrainingStatus.inProgress)
        .length;

    return Card(
      elevation: 2,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Row(
          mainAxisAlignment: MainAxisAlignment.spaceAround,
          children: [
            _StatItem(label: '总课程', value: '$total', color: Colors.blue),
            Container(
                height: 32, width: 1, color: Colors.grey.withOpacity(0.3)),
            _StatItem(label: '已完成', value: '$completed', color: Colors.green),
            Container(
                height: 32, width: 1, color: Colors.grey.withOpacity(0.3)),
            _StatItem(label: '进行中', value: '$inProgress', color: Colors.orange),
          ],
        ),
      ),
    );
  }
}

/// 单个统计数字。
class _StatItem extends StatelessWidget {
  final String label;
  final String value;
  final Color color;
  const _StatItem({
    required this.label,
    required this.value,
    required this.color,
  });

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Text(
          value,
          style: TextStyle(
            fontSize: 24,
            fontWeight: FontWeight.bold,
            color: color,
          ),
        ),
        const SizedBox(height: 4),
        Text(label, style: const TextStyle(color: Colors.grey, fontSize: 12)),
      ],
    );
  }
}

/// 单条培训课程卡片。
class TrainingCard extends ConsumerWidget {
  final Training training;
  const TrainingCard({super.key, required this.training});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final t = training;
    return Card(
      elevation: 1,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
      child: InkWell(
        borderRadius: BorderRadius.circular(8),
        onTap: () {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('开始学习：${t.title}（占位）')),
          );
        },
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Expanded(
                    child: Text(
                      t.title,
                      style: Theme.of(context).textTheme.titleMedium,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                  _StatusChip(status: t.status),
                ],
              ),
              if (t.description != null) ...[
                const SizedBox(height: 4),
                Text(
                  t.description!,
                  style: const TextStyle(color: Colors.grey, fontSize: 12),
                ),
              ],
              const SizedBox(height: 12),
              ClipRRect(
                borderRadius: BorderRadius.circular(4),
                child: LinearProgressIndicator(
                  value: t.progress / 100.0,
                  minHeight: 6,
                  backgroundColor: Colors.grey.withOpacity(0.2),
                ),
              ),
              const SizedBox(height: 6),
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Text(
                    '进度 ${t.progressText}',
                    style: const TextStyle(fontSize: 12, color: Colors.grey),
                  ),
                  if (t.dueAtText != null)
                    Text(
                      '截止 ${t.dueAtText}',
                      style: TextStyle(
                        fontSize: 12,
                        color: t.status == TrainingStatus.expired
                            ? Colors.red
                            : Colors.grey,
                      ),
                    ),
                ],
              ),
              if (t.durationMinutes != null) ...[
                const SizedBox(height: 4),
                Text(
                  '时长 ${t.durationMinutes} 分钟',
                  style: const TextStyle(fontSize: 12, color: Colors.grey),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

/// 状态 chip。
class _StatusChip extends StatelessWidget {
  final TrainingStatus status;
  const _StatusChip({required this.status});

  @override
  Widget build(BuildContext context) {
    final (text, color) = switch (status) {
      TrainingStatus.notStarted => ('未开始', Colors.grey),
      TrainingStatus.inProgress => ('进行中', Colors.orange),
      TrainingStatus.completed => ('已完成', Colors.green),
      TrainingStatus.expired => ('已过期', Colors.red),
    };
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: color.withOpacity(0.1),
        borderRadius: BorderRadius.circular(4),
        border: Border.all(color: color),
      ),
      child: Text(
        text,
        style: TextStyle(
          color: color,
          fontSize: 12,
          fontWeight: FontWeight.w500,
        ),
      ),
    );
  }
}