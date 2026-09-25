// lib/pages/login/login_page.dart
//
// 登录页 —— spec §8 路由守卫前置；支持两种登录方式：
//   - 短信码登录：POST /auth/login/sms（phone + code）
//   - 微信登录：POST /auth/login/wx（code）
//
// 业务流：
//   1. 输入手机号 → 点「获取验证码」→ 调 /auth/login/sms/send（v1 简化：直接 mock 60s 倒计时）
//   2. 输入 6 位验证码 → 点「登录」→ 调 AuthNotifier.loginByPhone(phone, code)
//   3. 点「微信登录」→ 调 AuthNotifier.loginByWx(wxCode)（v1 简化：固定占位 wx-code）
//   4. 登录成功 → 跳 /home/invitations
//
// 设计要点：
//   - 表单校验：手机号 11 位 / 验证码 6 位数字
//   - 验证码倒计时 60s（disabled 期间按钮文案 "重新获取 (Xs)"）
//   - 监听 authProvider → 已登录则自动跳转（路由守卫兜底）
import 'dart:async';

import 'package:escort_app/providers/auth_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

/// 登录页（短信 + 微信）。
class LoginPage extends ConsumerStatefulWidget {
  const LoginPage({super.key});

  @override
  ConsumerState<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends ConsumerState<LoginPage> {
  final _phoneCtrl = TextEditingController();
  final _codeCtrl = TextEditingController();
  final _formKey = GlobalKey<FormState>();

  bool _sendingCode = false;
  int _codeCountdown = 0;
  Timer? _countdownTimer;
  bool _submitting = false;

  @override
  void initState() {
    super.initState();
    // 监听已登录 → 跳首页
    ref.listenManual<AuthState>(authProvider, (prev, next) {
      if (next.isAuthed && mounted) {
        context.go('/home/invitations');
      }
    });
  }

  @override
  void dispose() {
    _phoneCtrl.dispose();
    _codeCtrl.dispose();
    _countdownTimer?.cancel();
    super.dispose();
  }

  void _startCountdown() {
    setState(() => _codeCountdown = 60);
    _countdownTimer?.cancel();
    _countdownTimer = Timer.periodic(const Duration(seconds: 1), (t) {
      if (!mounted) return;
      setState(() {
        _codeCountdown -= 1;
        if (_codeCountdown <= 0) {
          t.cancel();
        }
      });
    });
  }

  Future<void> _onSendCode() async {
    final form = _formKey.currentState;
    if (form == null || !form.validate()) return;
    setState(() => _sendingCode = true);
    try {
      // v1 简化：直接进入倒计时；真实应调 /auth/login/sms/send
      await Future<void>.delayed(const Duration(milliseconds: 300));
      if (!mounted) return;
      _startCountdown();
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('验证码已发送（开发模式：任意 6 位数字通过）')),
      );
    } finally {
      if (mounted) setState(() => _sendingCode = false);
    }
  }

  Future<void> _onSmsLogin() async {
    final form = _formKey.currentState;
    if (form == null || !form.validate()) return;
    setState(() => _submitting = true);
    try {
      await ref.read(authProvider.notifier).loginByPhone(
            phone: _phoneCtrl.text.trim(),
            code: _codeCtrl.text.trim(),
          );
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('登录失败：$e')),
      );
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  Future<void> _onWxLogin() async {
    setState(() => _submitting = true);
    try {
      // v1 简化：固定占位 wxCode。真实应通过 wechat_kit / fluwx 拿 wx.login code。
      await ref.read(authProvider.notifier).loginByWx(wxCode: 'wx-placeholder-code');
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('微信登录失败：$e')),
      );
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  String? _validatePhone(String? v) {
    final s = v?.trim() ?? '';
    if (s.isEmpty) return '请输入手机号';
    if (s.length != 11) return '手机号应为 11 位';
    if (!RegExp(r'^\d{11}$').hasMatch(s)) return '手机号格式错误';
    return null;
  }

  String? _validateCode(String? v) {
    final s = v?.trim() ?? '';
    if (s.isEmpty) return '请输入验证码';
    if (s.length != 6) return '验证码应为 6 位';
    return null;
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('陪诊师登录')),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(20),
          child: Form(
            key: _formKey,
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                const SizedBox(height: 24),
                Center(
                  child: Column(
                    children: [
                      Icon(
                        Icons.medical_services_outlined,
                        size: 64,
                        color: Theme.of(context).colorScheme.primary,
                      ),
                      const SizedBox(height: 12),
                      Text(
                        '陪诊师端',
                        style: Theme.of(context).textTheme.headlineSmall,
                      ),
                      const SizedBox(height: 4),
                      const Text(
                        '欢迎登录',
                        style: TextStyle(color: Colors.grey),
                      ),
                    ],
                  ),
                ),
                const SizedBox(height: 32),
                TextFormField(
                  controller: _phoneCtrl,
                  decoration: const InputDecoration(
                    labelText: '手机号',
                    prefixIcon: Icon(Icons.phone_outlined),
                    border: OutlineInputBorder(),
                  ),
                  keyboardType: TextInputType.phone,
                  inputFormatters: [
                    FilteringTextInputFormatter.digitsOnly,
                    LengthLimitingTextInputFormatter(11),
                  ],
                  validator: _validatePhone,
                ),
                const SizedBox(height: 12),
                Row(
                  children: [
                    Expanded(
                      child: TextFormField(
                        controller: _codeCtrl,
                        decoration: const InputDecoration(
                          labelText: '验证码',
                          prefixIcon: Icon(Icons.message_outlined),
                          border: OutlineInputBorder(),
                        ),
                        keyboardType: TextInputType.number,
                        inputFormatters: [
                          FilteringTextInputFormatter.digitsOnly,
                          LengthLimitingTextInputFormatter(6),
                        ],
                        validator: _validateCode,
                      ),
                    ),
                    const SizedBox(width: 8),
                    SizedBox(
                      width: 120,
                      child: OutlinedButton(
                        onPressed: (_sendingCode || _codeCountdown > 0)
                            ? null
                            : _onSendCode,
                        child: Text(
                          _codeCountdown > 0
                              ? '重新获取 (${_codeCountdown}s)'
                              : '获取验证码',
                        ),
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 24),
                FilledButton(
                  onPressed: _submitting ? null : _onSmsLogin,
                  child: _submitting
                      ? const SizedBox(
                          width: 18,
                          height: 18,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        )
                      : const Text('登录'),
                ),
                const SizedBox(height: 16),
                Row(
                  children: const [
                    Expanded(child: Divider()),
                    Padding(
                      padding: EdgeInsets.symmetric(horizontal: 12),
                      child: Text('其他登录方式',
                          style: TextStyle(color: Colors.grey)),
                    ),
                    Expanded(child: Divider()),
                  ],
                ),
                const SizedBox(height: 16),
                OutlinedButton.icon(
                  onPressed: _submitting ? null : _onWxLogin,
                  icon: const Icon(Icons.chat_outlined, color: Colors.green),
                  label: const Text('微信登录'),
                ),
                const SizedBox(height: 24),
                const Center(
                  child: Text(
                    '登录即代表同意《用户协议》与《隐私政策》',
                    style: TextStyle(color: Colors.grey, fontSize: 12),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}