// lib/pages/onboarding/onboarding_page.dart
//
// 6 步引导（PageView）—— 实名 / 健康证 / 培训 / 协议 / 提交审核。
//
// 业务流：
//   1. 身份证信息（姓名 + 身份证号 + 上传身份证正反面）
//   2. 健康证（上传体检报告）
//   3. 培训（观看培训视频 + 完成 5 题测试）
//   4. 服务协议（阅读 + 同意）
//   5. 提交审核（mock POST /audit/submit → /audit/pending）
//
// 设计：
//   - PageController + 6 steps
//   - 每步独立 validator
//   - 最后一步点「提交审核」→ 跳 /audit/pending
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

/// 6 步引导 PageView。
class OnboardingPage extends ConsumerStatefulWidget {
  const OnboardingPage({super.key});

  @override
  ConsumerState<OnboardingPage> createState() => _OnboardingPageState();
}

class _OnboardingPageState extends ConsumerState<OnboardingPage> {
  final _pageCtrl = PageController();
  int _step = 0;

  final _nameCtrl = TextEditingController();
  final _idNoCtrl = TextEditingController();
  bool _healthCertUploaded = false;
  bool _trainingDone = false;
  bool _termsAgreed = false;

  static const _stepTitles = [
    '身份信息',
    '健康证',
    '培训学习',
    '服务协议',
    '提交审核',
  ];

  @override
  void dispose() {
    _pageCtrl.dispose();
    _nameCtrl.dispose();
    _idNoCtrl.dispose();
    super.dispose();
  }

  void _next() {
    if (_step < _stepTitles.length - 1) {
      _pageCtrl.nextPage(
        duration: const Duration(milliseconds: 250),
        curve: Curves.easeOut,
      );
    }
  }

  void _prev() {
    if (_step > 0) {
      _pageCtrl.previousPage(
        duration: const Duration(milliseconds: 250),
        curve: Curves.easeOut,
      );
    }
  }

  Future<void> _submit() async {
    await Future<void>.delayed(const Duration(milliseconds: 500));
    if (!mounted) return;
    context.go('/audit/pending');
  }

  bool _canNext() {
    switch (_step) {
      case 0:
        return _nameCtrl.text.trim().length >= 2 &&
            _idNoCtrl.text.trim().length == 18;
      case 1:
        return _healthCertUploaded;
      case 2:
        return _trainingDone;
      case 3:
        return _termsAgreed;
      default:
        return true;
    }
  }

  Widget _step0Identity() {
    return ListView(
      key: const Key('onb_step0'),
      padding: const EdgeInsets.all(16),
      children: [
        Text('身份证信息', style: Theme.of(context).textTheme.titleLarge),
        const SizedBox(height: 16),
        TextField(
          key: const Key('onb_name'),
          controller: _nameCtrl,
          decoration: const InputDecoration(labelText: '真实姓名'),
          onChanged: (_) => setState(() {}),
        ),
        const SizedBox(height: 12),
        TextField(
          key: const Key('onb_idno'),
          controller: _idNoCtrl,
          decoration: const InputDecoration(labelText: '身份证号'),
          maxLength: 18,
          onChanged: (_) => setState(() {}),
        ),
      ],
    );
  }

  Widget _step1HealthCert() {
    return ListView(
      key: const Key('onb_step1'),
      padding: const EdgeInsets.all(16),
      children: [
        Text('上传健康证', style: Theme.of(context).textTheme.titleLarge),
        const SizedBox(height: 16),
        Card(
          child: ListTile(
            leading: Icon(_healthCertUploaded
                ? Icons.check_circle
                : Icons.upload_file,
            ),
            title: Text(_healthCertUploaded ? '健康证已上传' : '上传体检报告'),
            subtitle: const Text('JPG / PNG，< 5MB'),
            trailing: TextButton(
              key: const Key('onb_upload_health'),
              onPressed: () => setState(() => _healthCertUploaded = true),
              child: const Text('选择文件'),
            ),
          ),
        ),
      ],
    );
  }

  Widget _step2Training() {
    return ListView(
      key: const Key('onb_step2'),
      padding: const EdgeInsets.all(16),
      children: [
        Text('培训学习', style: Theme.of(context).textTheme.titleLarge),
        const SizedBox(height: 16),
        const Card(
          child: ListTile(
            leading: Icon(Icons.play_circle),
            title: Text('陪诊师岗前培训（视频）'),
            subtitle: Text('约 15 分钟'),
          ),
        ),
        const SizedBox(height: 12),
        Card(
          child: ListTile(
            leading: Icon(_trainingDone ? Icons.check : Icons.quiz),
            title: Text(_trainingDone ? '已通过 5 题测试' : '完成 5 题测试'),
            trailing: TextButton(
              key: const Key('onb_finish_training'),
              onPressed: () => setState(() => _trainingDone = true),
              child: const Text('完成'),
            ),
          ),
        ),
      ],
    );
  }

  Widget _step3Terms() {
    return ListView(
      key: const Key('onb_step3'),
      padding: const EdgeInsets.all(16),
      children: [
        Text('服务协议', style: Theme.of(context).textTheme.titleLarge),
        const SizedBox(height: 16),
        const Card(
          child: Padding(
            padding: EdgeInsets.all(12),
            child: Text(
              '陪诊师应遵守医院规章制度、保护患者隐私、按时按质完成服务……',
              style: TextStyle(height: 1.5),
            ),
          ),
        ),
        CheckboxListTile(
          key: const Key('onb_terms'),
          value: _termsAgreed,
          onChanged: (v) => setState(() => _termsAgreed = v ?? false),
          title: const Text('我已阅读并同意《陪诊师服务协议》'),
        ),
      ],
    );
  }

  Widget _step4Submit() {
    return ListView(
      key: const Key('onb_step4'),
      padding: const EdgeInsets.all(16),
      children: [
        Text('提交审核', style: Theme.of(context).textTheme.titleLarge),
        const SizedBox(height: 16),
        Card(
          child: Padding(
            padding: const EdgeInsets.all(12),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                _kv('姓名',
                    _nameCtrl.text.trim().isEmpty ? '—' : _nameCtrl.text.trim(),
                ),
                _kv('身份证号',
                    _idNoCtrl.text.trim().isEmpty ? '—' : _idNoCtrl.text.trim(),
                ),
                _kv('健康证', _healthCertUploaded ? '已上传' : '未上传'),
                _kv('培训', _trainingDone ? '已完成' : '未完成'),
                _kv('协议', _termsAgreed ? '已同意' : '未同意'),
              ],
            ),
          ),
        ),
        const SizedBox(height: 24),
        FilledButton(
          key: const Key('onb_submit'),
          onPressed: _submit,
          child: const Text('提交审核'),
        ),
      ],
    );
  }

  Widget _kv(String k, String v) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        children: [
          SizedBox(width: 80, child: Text(k)),
          Expanded(child: Text(v)),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final isLast = _step == _stepTitles.length - 1;
    return Scaffold(
      appBar: AppBar(
        title: Text('${_stepTitles[_step]} (${_step + 1}/${_stepTitles.length})'),
      ),
      body: PageView(
        controller: _pageCtrl,
        onPageChanged: (i) => setState(() => _step = i),
        children: [
          _step0Identity(),
          _step1HealthCert(),
          _step2Training(),
          _step3Terms(),
          _step4Submit(),
        ],
      ),
      bottomNavigationBar: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: Row(
            children: [
              if (_step > 0)
                OutlinedButton(
                  key: const Key('onb_prev'),
                  onPressed: _prev,
                  child: const Text('上一步'),
                ),
              const Spacer(),
              if (!isLast)
                FilledButton(
                  key: const Key('onb_next'),
                  onPressed: _canNext() ? _next : null,
                  child: const Text('下一步'),
                ),
            ],
          ),
        ),
      ),
    );
  }
}