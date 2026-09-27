// lib/pages/sos/sos_trigger_page.dart
//
// SOS 紧急呼救页 —— 长按按钮 1.5s 触发。
//
// 业务流：
//   1. 用户长按 SOS 按钮 1.5s
//   2. 上传当前位置（POST /sos/trigger）
//   3. 显示「已发送呼救，请保持冷静」+ 位置 + 联系电话
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:geolocator/geolocator.dart';

/// SOS 触发页。
class SosTriggerPage extends ConsumerStatefulWidget {
  const SosTriggerPage({super.key});

  @override
  ConsumerState<SosTriggerPage> createState() => _SosTriggerPageState();
}

class _SosTriggerPageState extends ConsumerState<SosTriggerPage>
    with SingleTickerProviderStateMixin {
  late final AnimationController _ac;
  Position? _pos;
  bool _triggered = false;
  bool _sending = false;

  @override
  void initState() {
    super.initState();
    _ac = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 1500),
    );
    _loadPos();
  }

  Future<void> _loadPos() async {
    try {
      const ch = MethodChannel('escort_app.geolocator');
      final pos = await ch.invokeMethod<Map<dynamic, dynamic>>('getPosition');
      if (pos != null && mounted) {
        setState(() {
          _pos = Position(
            longitude: (pos['lng'] as num).toDouble(),
            latitude: (pos['lat'] as num).toDouble(),
            timestamp: DateTime.now(),
            accuracy: 0,
            altitude: 0,
            altitudeAccuracy: 0,
            heading: 0,
            headingAccuracy: 0,
            speed: 0,
            speedAccuracy: 0,
          );
        });
      }
    } on Object catch (_) {}
  }

  @override
  void dispose() {
    _ac.dispose();
    super.dispose();
  }

  Future<void> _onLongPress() async {
    if (_triggered) return;
    setState(() => _sending = true);
    // mock 500ms 后端
    await Future<void>.delayed(const Duration(milliseconds: 500));
    if (!mounted) return;
    setState(() {
      _triggered = true;
      _sending = false;
    });
  }

  void _startPress() {
    if (_triggered) return;
    _ac.forward(from: 0);
    _ac.value = 0;
  }

  void _cancelPress() {
    if (_triggered) return;
    _ac.stop();
    _ac.value = 0;
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.red.shade50,
      appBar: AppBar(
        title: const Text('SOS 紧急呼救'),
        backgroundColor: Colors.red,
      ),
      body: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.center,
          children: [
            const SizedBox(height: 32),
            if (!_triggered)
              const Text(
                '长按 SOS 按钮 1.5 秒触发呼救',
                style: TextStyle(fontSize: 16),
              )
            else
              const Text(
                '已发送呼救\n请保持冷静，等待救援',
                textAlign: TextAlign.center,
                key: Key('sos_triggered_text'),
                style: TextStyle(
                  fontSize: 20,
                  fontWeight: FontWeight.bold,
                  color: Colors.red,
                ),
              ),
            const Spacer(),
            if (_pos != null)
              Text(
                '位置：${_pos!.latitude.toStringAsFixed(4)}, ${_pos!.longitude.toStringAsFixed(4)}',
                key: const Key('sos_pos'),
              ),
            const Spacer(),
            GestureDetector(
              key: const Key('sos_button'),
              onLongPressStart: (_) => _startPress(),
              onLongPressEnd: (_) => _cancelPress(),
              onLongPress: _onLongPress,
              child: AnimatedBuilder(
                animation: _ac,
                builder: (_, __) {
                  return Container(
                    width: 220,
                    height: 220,
                    decoration: BoxDecoration(
                      color: _triggered ? Colors.grey : Colors.red,
                      shape: BoxShape.circle,
                      boxShadow: [
                        BoxShadow(
                          color: Colors.red.withValues(alpha: 0.4),
                          blurRadius: _triggered ? 0 : (60 * _ac.value),
                          spreadRadius: _triggered ? 0 : (10 * _ac.value),
                        ),
                      ],
                    ),
                    alignment: Alignment.center,
                    child: Text(
                      _triggered
                          ? '已触发'
                          : (_sending ? '发送中...' : 'SOS'),
                      style: const TextStyle(
                        color: Colors.white,
                        fontSize: 36,
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                  );
                },
              ),
            ),
            const SizedBox(height: 24),
            const Text('或拨打 110 / 120', style: TextStyle(color: Colors.red)),
            const SizedBox(height: 32),
          ],
        ),
      ),
    );
  }
}