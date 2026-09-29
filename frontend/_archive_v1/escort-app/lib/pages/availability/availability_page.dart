// lib/pages/availability/availability_page.dart
//
// 「我的空余时段」页 —— spec/2026-09-24-order-matching-redesign §3.2 + §4.1。
//
// 业务流：
//   1. 进入页面 → availabilityProvider 拉取我的时段列表
//   2. 渲染 ListView（每行：时段范围 + 状态 chip + 删除按钮）
//   3. AppBar 「+」按钮 → 弹底部表单（start / end datetime picker）→ 调 createAvailability
//   4. 删除按钮 → 调 deleteAvailability（仅 available 状态可删）
//
// 限制：v1 不实现时段冲突 13103 错误的友好提示（后续 plan）。
import 'package:escort_app/models/availability.dart';
import 'package:escort_app/providers/availability_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class AvailabilityPage extends ConsumerWidget {
  const AvailabilityPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final asyncList = ref.watch(availabilityProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('我的空余时段'),
        actions: [
          IconButton(
            tooltip: '新增时段',
            icon: const Icon(Icons.add),
            onPressed: () => _showCreateSheet(context, ref),
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
                onPressed: () => ref.invalidate(availabilityProvider),
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
                  Icon(Icons.event_busy, size: 64, color: Colors.grey[400]),
                  const SizedBox(height: 12),
                  const Text('暂未设置空余时段'),
                  const SizedBox(height: 4),
                  const Text(
                    '点右上角「+」新增可被患者邀请的时段',
                    style: TextStyle(color: Colors.grey),
                  ),
                ],
              ),
            );
          }
          return RefreshIndicator(
            onRefresh: () async {
              ref.invalidate(availabilityProvider);
              await ref.read(availabilityProvider.future);
            },
            child: ListView.separated(
              padding: const EdgeInsets.all(12),
              itemCount: list.length,
              separatorBuilder: (_, __) => const SizedBox(height: 8),
              itemBuilder: (_, i) => AvailabilityTile(avail: list[i]),
            ),
          );
        },
      ),
    );
  }

  Future<void> _showCreateSheet(BuildContext context, WidgetRef ref) async {
    final result = await showModalBottomSheet<_NewAvailabilityPayload>(
      context: context,
      isScrollControlled: true,
      builder: (_) => const _CreateAvailabilitySheet(),
    );
    if (result == null) return;
    try {
      await ref
          .read(createAvailabilityControllerProvider.notifier)
          .invoke(result.startAt, result.endAt);
      if (!context.mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('已新增时段')),
      );
    } catch (e) {
      if (!context.mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('新增失败：$e')),
      );
    }
  }
}

/// 单条时段行。
class AvailabilityTile extends ConsumerWidget {
  final Availability avail;

  const AvailabilityTile({super.key, required this.avail});

  Future<void> _delete(BuildContext context, WidgetRef ref) async {
    final confirm = await showDialog<bool>(
      context: context,
      builder: (_) => AlertDialog(
        title: const Text('删除时段'),
        content: Text('确认删除 ${avail.startText} - ${avail.endText} 的时段吗？'),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(false),
            child: const Text('取消'),
          ),
          FilledButton(
            onPressed: () => Navigator.of(context).pop(true),
            child: const Text('删除'),
          ),
        ],
      ),
    );
    if (confirm != true) return;
    try {
      await ref
          .read(deleteAvailabilityControllerProvider.notifier)
          .invoke(avail.id);
      if (!context.mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('已删除')),
      );
    } catch (e) {
      if (!context.mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('删除失败：$e')),
      );
    }
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final a = avail;
    return Card(
      elevation: 1,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
      child: ListTile(
        contentPadding:
            const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
        title: Text(
          '${a.startText}  →  ${a.endText}',
          style: const TextStyle(fontWeight: FontWeight.w500),
        ),
        subtitle: Padding(
          padding: const EdgeInsets.only(top: 4),
          child: Row(
            children: [
              _StatusChip(status: a.status),
              const SizedBox(width: 8),
              Text('${a.durationHours} 小时'),
            ],
          ),
        ),
        trailing: a.isDeletable
            ? IconButton(
                icon: const Icon(Icons.delete_outline, color: Colors.red),
                tooltip: '删除',
                onPressed: () => _delete(context, ref),
              )
            : (a.isBooked
                ? const Icon(Icons.lock_outline, color: Colors.grey)
                : const Icon(Icons.block, color: Colors.grey)),
      ),
    );
  }
}

class _StatusChip extends StatelessWidget {
  final AvailabilityStatus status;
  const _StatusChip({required this.status});

  @override
  Widget build(BuildContext context) {
    late final String text;
    late final Color color;
    switch (status) {
      case AvailabilityStatus.available:
        text = '可被邀请';
        color = Colors.green;
        break;
      case AvailabilityStatus.booked:
        text = '已预订';
        color = Colors.orange;
        break;
      case AvailabilityStatus.canceled:
        text = '已撤销';
        color = Colors.grey;
        break;
    }
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: color.withOpacity(0.1),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(
        text,
        style: TextStyle(
          color: color,
          fontSize: 11,
          fontWeight: FontWeight.w500,
        ),
      ),
    );
  }
}

/// 新增时段底部表单（v1 简化：两个 DateTime picker）。
class _NewAvailabilityPayload {
  final DateTime startAt;
  final DateTime endAt;
  const _NewAvailabilityPayload(this.startAt, this.endAt);
}

class _CreateAvailabilitySheet extends StatefulWidget {
  const _CreateAvailabilitySheet();

  @override
  State<_CreateAvailabilitySheet> createState() =>
      _CreateAvailabilitySheetState();
}

class _CreateAvailabilitySheetState extends State<_CreateAvailabilitySheet> {
  late DateTime _start;
  late DateTime _end;

  @override
  void initState() {
    super.initState();
    final now = DateTime.now();
    final nextHour = now.add(const Duration(hours: 1));
    _start = DateTime(nextHour.year, nextHour.month, nextHour.day,
        nextHour.hour);
    _end = _start.add(const Duration(hours: 4));
  }

  Future<void> _pickStart() async {
    final picked = await showDatePicker(
      context: context,
      initialDate: _start,
      firstDate: DateTime.now(),
      lastDate: DateTime.now().add(const Duration(days: 30)),
    );
    if (picked == null) return;
    final time = await showTimePicker(
      context: context,
      initialTime: TimeOfDay.fromDateTime(_start),
    );
    if (time == null) return;
    setState(() {
      _start = DateTime(picked.year, picked.month, picked.day, time.hour,
          time.minute);
      if (!_end.isAfter(_start)) {
        _end = _start.add(const Duration(hours: 1));
      }
    });
  }

  Future<void> _pickEnd() async {
    final picked = await showDatePicker(
      context: context,
      initialDate: _end,
      firstDate: _start,
      lastDate: _start.add(const Duration(days: 30)),
    );
    if (picked == null) return;
    final time = await showTimePicker(
      context: context,
      initialTime: TimeOfDay.fromDateTime(_end),
    );
    if (time == null) return;
    setState(() {
      _end = DateTime(picked.year, picked.month, picked.day, time.hour,
          time.minute);
    });
  }

  String _fmt(DateTime d) =>
      '${d.year}-${d.month.toString().padLeft(2, '0')}-${d.day.toString().padLeft(2, '0')} '
      '${d.hour.toString().padLeft(2, '0')}:${d.minute.toString().padLeft(2, '0')}';

  @override
  Widget build(BuildContext context) {
    final valid = _end.isAfter(_start);
    return Padding(
      padding: EdgeInsets.only(
        bottom: MediaQuery.of(context).viewInsets.bottom,
        left: 16,
        right: 16,
        top: 16,
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const Text('新增空余时段',
              style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
          const SizedBox(height: 16),
          OutlinedButton(
            onPressed: _pickStart,
            child: Text('开始：${_fmt(_start)}'),
          ),
          const SizedBox(height: 8),
          OutlinedButton(
            onPressed: _pickEnd,
            child: Text('结束：${_fmt(_end)}'),
          ),
          const SizedBox(height: 16),
          FilledButton(
            onPressed: valid
                ? () => Navigator.of(context)
                    .pop(_NewAvailabilityPayload(_start, _end))
                : null,
            child: const Text('确认新增'),
          ),
          const SizedBox(height: 16),
        ],
      ),
    );
  }
}