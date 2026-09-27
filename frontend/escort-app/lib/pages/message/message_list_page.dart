// lib/pages/message/message_list_page.dart
//
// 消息列表 —— 客服 / 系统 / 订单 三类会话。
//
// 数据：mock 会话列表（v1 不接 /messages）
// 设计：ListView + 头像 + 最近消息 + 时间戳 + 未读红点
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

class _MockConv {
  final String id;
  final String title;
  final String preview;
  final DateTime at;
  final int unread;
  final IconData icon;
  const _MockConv({
    required this.id,
    required this.title,
    required this.preview,
    required this.at,
    required this.unread,
    required this.icon,
  });
}

final _mockConvs = <_MockConv>[
  _MockConv(
    id: 'cs',
    title: '在线客服',
    preview: '请问有什么可以帮您？',
    at: DateTime.now().subtract(const Duration(minutes: 5)),
    unread: 2,
    icon: Icons.support_agent,
  ),
  _MockConv(
    id: 'sys',
    title: '系统通知',
    preview: '您的审核已通过，欢迎加入陪诊服务',
    at: DateTime.now().subtract(const Duration(hours: 2)),
    unread: 0,
    icon: Icons.notifications,
  ),
  _MockConv(
    id: 'order',
    title: '订单 #1024',
    preview: '患者已签到，请尽快到达',
    at: DateTime.now().subtract(const Duration(hours: 5)),
    unread: 1,
    icon: Icons.local_hospital,
  ),
];

/// 消息列表页。
class MessageListPage extends ConsumerWidget {
  const MessageListPage({super.key});

  String _ago(DateTime t) {
    final d = DateTime.now().difference(t);
    if (d.inMinutes < 60) return '${d.inMinutes}分钟前';
    if (d.inHours < 24) return '${d.inHours}小时前';
    return '${d.inDays}天前';
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Scaffold(
      appBar: AppBar(title: const Text('消息')),
      body: ListView.separated(
        key: const Key('msg_list'),
        itemCount: _mockConvs.length,
        separatorBuilder: (_, __) => const Divider(height: 1),
        itemBuilder: (_, i) {
          final c = _mockConvs[i];
          return ListTile(
            key: Key('msg_item_${c.id}'),
            leading: CircleAvatar(
              child: Icon(c.icon),
            ),
            title: Text(c.title),
            subtitle: Text(
              c.preview,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
            trailing: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              crossAxisAlignment: CrossAxisAlignment.end,
              children: [
                Text(_ago(c.at), style: const TextStyle(fontSize: 12)),
                if (c.unread > 0)
                  Container(
                    key: Key('msg_unread_${c.id}'),
                    margin: const EdgeInsets.only(top: 4),
                    padding:
                        const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                    decoration: BoxDecoration(
                      color: Colors.red,
                      borderRadius: BorderRadius.circular(8),
                    ),
                    child: Text(
                      '${c.unread}',
                      style: const TextStyle(
                          color: Colors.white, fontSize: 10,
                      ),
                    ),
                  ),
              ],
            ),
            onTap: () => context.go('/message/chat/${c.id}'),
          );
        },
      ),
    );
  }
}