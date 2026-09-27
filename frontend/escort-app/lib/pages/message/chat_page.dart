// lib/pages/message/chat_page.dart
//
// 聊天页 —— 气泡列表 + 输入框。
//
// 数据：mock 历史消息（v1 不接 WebSocket）
// 设计：ListView 倒序 + TextField 输入 + 发送按钮
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class _Msg {
  final bool fromMe;
  final String text;
  final DateTime at;
  const _Msg({required this.fromMe, required this.text, required this.at});
}

final _mockHistory = <_Msg>[
  _Msg(
    fromMe: false,
    text: '您好，请问有什么可以帮您？',
    at: DateTime.now().subtract(const Duration(minutes: 10)),
  ),
  _Msg(
    fromMe: true,
    text: '请问订单 #1024 几点开始？',
    at: DateTime.now().subtract(const Duration(minutes: 9)),
  ),
  _Msg(
    fromMe: false,
    text: '订单 #1024 上午 10 点开始，请提前 15 分钟签到。',
    at: DateTime.now().subtract(const Duration(minutes: 8)),
  ),
];

/// 聊天页。
class ChatPage extends ConsumerStatefulWidget {
  final String conversationId;
  const ChatPage({super.key, required this.conversationId});

  @override
  ConsumerState<ChatPage> createState() => _ChatPageState();
}

class _ChatPageState extends ConsumerState<ChatPage> {
  final _inputCtrl = TextEditingController();
  final _scrollCtrl = ScrollController();
  late List<_Msg> _msgs;

  @override
  void initState() {
    super.initState();
    _msgs = List.of(_mockHistory);
  }

  @override
  void dispose() {
    _inputCtrl.dispose();
    _scrollCtrl.dispose();
    super.dispose();
  }

  void _send() {
    final t = _inputCtrl.text.trim();
    if (t.isEmpty) return;
    setState(() {
      _msgs.add(_Msg(fromMe: true, text: t, at: DateTime.now()));
      _inputCtrl.clear();
    });
    // mock 自动回复
    Future<void>.delayed(const Duration(milliseconds: 600), () {
      if (!mounted) return;
      setState(() {
        _msgs.add(_Msg(
          fromMe: false,
          text: '已收到您的消息，请稍候。',
          at: DateTime.now(),
        ),
        );
      });
      _scrollToBottom();
    });
    _scrollToBottom();
  }

  void _scrollToBottom() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (_scrollCtrl.hasClients) {
        _scrollCtrl.animateTo(
          _scrollCtrl.position.maxScrollExtent,
          duration: const Duration(milliseconds: 200),
          curve: Curves.easeOut,
        );
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text('会话 ${widget.conversationId}')),
      body: Column(
        children: [
          Expanded(
            child: ListView.builder(
              key: const Key('chat_list'),
              controller: _scrollCtrl,
              padding: const EdgeInsets.all(8),
              itemCount: _msgs.length,
              itemBuilder: (_, i) {
                final m = _msgs[i];
                return _Bubble(msg: m);
              },
            ),
          ),
          SafeArea(
            child: Padding(
              padding: const EdgeInsets.all(8),
              child: Row(
                children: [
                  Expanded(
                    child: TextField(
                      key: const Key('chat_input'),
                      controller: _inputCtrl,
                      decoration: const InputDecoration(
                        hintText: '输入消息…',
                        border: OutlineInputBorder(),
                        isDense: true,
                      ),
                      onSubmitted: (_) => _send(),
                    ),
                  ),
                  const SizedBox(width: 8),
                  IconButton.filled(
                    key: const Key('chat_send'),
                    icon: const Icon(Icons.send),
                    onPressed: _send,
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _Bubble extends StatelessWidget {
  final _Msg msg;
  const _Bubble({required this.msg});

  @override
  Widget build(BuildContext context) {
    final isMe = msg.fromMe;
    return Align(
      alignment: isMe ? Alignment.centerRight : Alignment.centerLeft,
      child: Container(
        key: Key(isMe ? 'chat_bubble_me' : 'chat_bubble_other'),
        margin: const EdgeInsets.symmetric(vertical: 4),
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        constraints: BoxConstraints(
          maxWidth: MediaQuery.of(context).size.width * 0.7,
        ),
        decoration: BoxDecoration(
          color: isMe ? Colors.blue : Colors.grey.shade300,
          borderRadius: BorderRadius.circular(12),
        ),
        child: Text(
          msg.text,
          style: TextStyle(color: isMe ? Colors.white : Colors.black),
        ),
      ),
    );
  }
}