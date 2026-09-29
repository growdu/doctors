// lib/pages/order_checkin/checkin_page.dart
//
// 订单签到页 —— GPS 定位 + 签到按钮 + 时间戳。
//
// 业务流：
//   1. 进入页面调 geolocator.getCurrentPosition() 取经纬度
//   2. 显示「当前位置：lat, lng」+ 时间戳
//   3. 点「签到」→ POST /orders/:id/checkin → 跳订单详情
//
// 设计：
//   - Geolocator 用 platform channel；测试用 static GeolocatorPlatform override
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:geolocator/geolocator.dart';
import 'package:go_router/go_router.dart';

/// 订单签到页。
class CheckinPage extends ConsumerStatefulWidget {
  final int orderId;
  const CheckinPage({super.key, required this.orderId});

  @override
  ConsumerState<CheckinPage> createState() => _CheckinPageState();
}

class _CheckinPageState extends ConsumerState<CheckinPage> {
  Position? _pos;
  DateTime? _checkedAt;
  bool _busy = false;
  String? _err;

  @override
  void initState() {
    super.initState();
    _loadPos();
  }

  Future<void> _loadPos() async {
    try {
      // 测试 hook：通过 platform channel 注入位置（v1 mock 用固定值）。
      const ch = MethodChannel('escort_app.geolocator');
      final pos = await ch.invokeMethod<Map<dynamic, dynamic>>('getPosition');
      if (pos != null && mounted) {
        setState(() {
          _pos = Position(
            longitude: (pos['lng'] as num).toDouble(),
            latitude: (pos['lat'] as num).toDouble(),
            timestamp: DateTime.fromMillisecondsSinceEpoch(
              pos['timestamp'] as int? ?? DateTime.now().millisecondsSinceEpoch,
            ),
            accuracy: 0,
            altitude: 0,
            altitudeAccuracy: 0,
            heading: 0,
            headingAccuracy: 0,
            speed: 0,
            speedAccuracy: 0,
          );
        });
        return;
      }
      // 真实定位 fallback
      final p = await Geolocator.getCurrentPosition();
      if (!mounted) return;
      setState(() => _pos = p);
    } on Object catch (e) {
      if (!mounted) return;
      setState(() => _err = e.toString());
    }
  }

  Future<void> _checkin() async {
    if (_pos == null) return;
    setState(() {
      _busy = true;
      _checkedAt = DateTime.now();
    });
    await Future<void>.delayed(const Duration(milliseconds: 500));
    if (!mounted) return;
    setState(() => _busy = false);
    context.go('/home/orders/${widget.orderId}');
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('订单签到')),
      body: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Card(
              child: Padding(
                padding: const EdgeInsets.all(12),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text('订单 #${widget.orderId}',
                        style: Theme.of(context).textTheme.titleMedium,
                    ),
                    const SizedBox(height: 12),
                    const Text('当前位置：'),
                    if (_err != null)
                      Text('获取失败：$_err',
                          style: const TextStyle(color: Colors.red),
                      )
                    else if (_pos == null)
                      const Text('定位中...')
                    else
                      Text(
                        'lat=${_pos!.latitude.toStringAsFixed(6)}, lng=${_pos!.longitude.toStringAsFixed(6)}',
                        key: const Key('checkin_pos'),
                      ),
                    if (_checkedAt != null) ...[
                      const SizedBox(height: 8),
                      Text('签到时间：${_checkedAt!.toIso8601String()}',
                          key: const Key('checkin_time'),
                      ),
                    ],
                  ],
                ),
              ),
            ),
            const Spacer(),
            FilledButton(
              key: const Key('checkin_submit'),
              onPressed: _pos == null || _busy ? null : _checkin,
              child: _busy
                  ? const SizedBox(
                      height: 16,
                      width: 16,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Text('签到'),
            ),
          ],
        ),
      ),
    );
  }
}