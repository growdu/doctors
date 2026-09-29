// lib/pages/auth/register_page.dart
//
// 注册页 —— 手机 + 验证码 + 密码（v1 mock 实名）。
//
// 业务流：
//   1. 输入手机号 + 验证码 + 密码
//   2. 点「注册」→ v1.1 mock（v1 不接 /auth/register，等待后端）
//   3. 跳转 /onboarding（实名 / 健康证 / 培训）
//
// v1.2 简化：注册成功直接入栈 onboarding（无后端校验）。
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

/// 注册页（手机 + 验证码 + 密码）。
class RegisterPage extends ConsumerStatefulWidget {
  const RegisterPage({super.key});

  @override
  ConsumerState<RegisterPage> createState() => _RegisterPageState();
}

class _RegisterPageState extends ConsumerState<RegisterPage> {
  final _formKey = GlobalKey<FormState>();
  final _phoneCtrl = TextEditingController();
  final _codeCtrl = TextEditingController();
  final _pwdCtrl = TextEditingController();
  bool _submitting = false;
  bool _agreedTerms = false;

  @override
  void dispose() {
    _phoneCtrl.dispose();
    _codeCtrl.dispose();
    _pwdCtrl.dispose();
    super.dispose();
  }

  String? _validatePhone(String? v) {
    final s = v?.trim() ?? '';
    if (s.length != 11 || !RegExp(r'^\d{11}$').hasMatch(s)) {
      return '请输入 11 位手机号';
    }
    return null;
  }

  String? _validateCode(String? v) {
    final s = v?.trim() ?? '';
    if (s.length != 6 || !RegExp(r'^\d{6}$').hasMatch(s)) {
      return '请输入 6 位验证码';
    }
    return null;
  }

  String? _validatePwd(String? v) {
    final s = v ?? '';
    if (s.length < 6 || s.length > 32) {
      return '密码长度 6-32 位';
    }
    return null;
  }

  Future<void> _onSubmit() async {
    final form = _formKey.currentState;
    if (form == null || !form.validate()) return;
    if (!_agreedTerms) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('请先同意《陪诊师服务协议》')),
      );
      return;
    }
    setState(() => _submitting = true);
    // v1 mock：等待 500ms 后跳转 onboarding
    await Future<void>.delayed(const Duration(milliseconds: 500));
    if (!mounted) return;
    setState(() => _submitting = false);
    context.go('/onboarding');
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('注册陪诊师')),
      body: Padding(
        padding: const EdgeInsets.all(16),
        child: Form(
          key: _formKey,
          child: ListView(
            children: [
              TextFormField(
                key: const Key('register_phone'),
                controller: _phoneCtrl,
                decoration: const InputDecoration(labelText: '手机号'),
                keyboardType: TextInputType.phone,
                validator: _validatePhone,
              ),
              const SizedBox(height: 12),
              TextFormField(
                key: const Key('register_code'),
                controller: _codeCtrl,
                decoration: const InputDecoration(labelText: '验证码'),
                keyboardType: TextInputType.number,
                validator: _validateCode,
              ),
              const SizedBox(height: 12),
              TextFormField(
                key: const Key('register_pwd'),
                controller: _pwdCtrl,
                decoration: const InputDecoration(labelText: '密码（6-32 位）'),
                obscureText: true,
                validator: _validatePwd,
              ),
              const SizedBox(height: 12),
              CheckboxListTile(
                key: const Key('register_terms'),
                value: _agreedTerms,
                onChanged: (v) => setState(() => _agreedTerms = v ?? false),
                title: const Text('我已阅读并同意《陪诊师服务协议》'),
                controlAffinity: ListTileControlAffinity.leading,
              ),
              const SizedBox(height: 16),
              FilledButton(
                key: const Key('register_submit'),
                onPressed: _submitting ? null : _onSubmit,
                child: _submitting
                    ? const SizedBox(
                        height: 16, width: 16,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Text('注册'),
              ),
              const SizedBox(height: 8),
              TextButton(
                onPressed: () => context.go('/login'),
                child: const Text('已有账号？去登录'),
              ),
            ],
          ),
        ),
      ),
    );
  }
}