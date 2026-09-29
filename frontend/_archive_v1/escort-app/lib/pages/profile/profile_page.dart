// lib/pages/profile/profile_page.dart
//
// 个人中心页 —— spec §A5 / plan §3 E4。
//
// 业务流：
//   1. 加载 profileProvider（GET /users/me）
//   2. 渲染：头像 + nickname + 手机号（脱敏）+ 实名状态 chip + 平台审核 chip
//   3. 点击「实名认证」→ 弹底部表单（姓名 + 身份证号 + 证件照 URL 可选）→ 调 realNameAuthController
//   4. 点击「退出登录」→ 弹确认 → 调 logoutController → 跳登录页
//
// 设计要点：
//   - 实名未完成时显示「去认证」按钮（红色 hint）；已认证显示绿色 chip
//   - approved=false 时显示「待平台审核」警告 banner（影响能否接单）
//   - 头像：网络图（avatar_url）+ 默认 Icon 占位（null 时）
import 'package:escort_app/providers/profile_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

/// 个人中心页。
class ProfilePage extends ConsumerWidget {
  const ProfilePage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final asyncProfile = ref.watch(profileProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('个人中心'),
        actions: [
          IconButton(
            tooltip: '刷新',
            icon: const Icon(Icons.refresh),
            onPressed: () => ref.invalidate(profileProvider),
          ),
        ],
      ),
      body: asyncProfile.when(
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
                onPressed: () => ref.invalidate(profileProvider),
                child: const Text('重试'),
              ),
            ],
          ),
        ),
        data: (profile) => RefreshIndicator(
          onRefresh: () async {
            ref.invalidate(profileProvider);
            await ref.read(profileProvider.future);
          },
          child: ListView(
            padding: const EdgeInsets.all(12),
            children: [
              if (!profile.approved) const _ApprovalPendingBanner(),
              _ProfileHeader(profile: profile),
              const SizedBox(height: 16),
              _ProfileActions(profile: profile),
              const SizedBox(height: 24),
              _LogoutButton(),
            ],
          ),
        ),
      ),
    );
  }
}

/// 待审核 banner（approved=false 时显示）。
class _ApprovalPendingBanner extends StatelessWidget {
  const _ApprovalPendingBanner();
  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.only(bottom: 12),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: Colors.orange.withOpacity(0.1),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: Colors.orange),
      ),
      child: Row(
        children: const [
          Icon(Icons.warning_amber_outlined, color: Colors.orange),
          SizedBox(width: 8),
          Expanded(
            child: Text(
              '您的资料正在审核中，审核通过后才能正常接单',
              style: TextStyle(color: Colors.orange),
            ),
          ),
        ],
      ),
    );
  }
}

/// 头部：头像 + nickname + 手机号 + 实名状态 + 审核状态。
class _ProfileHeader extends StatelessWidget {
  final dynamic profile; // Profile
  const _ProfileHeader({required this.profile});

  @override
  Widget build(BuildContext context) {
    final p = profile;
    return Card(
      elevation: 2,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          children: [
            CircleAvatar(
              radius: 40,
              backgroundColor: Theme.of(context).colorScheme.primary,
              backgroundImage:
                  p.avatarUrl != null ? NetworkImage(p.avatarUrl!) : null,
              child: p.avatarUrl == null
                  ? const Icon(Icons.person, size: 40, color: Colors.white)
                  : null,
            ),
            const SizedBox(height: 12),
            Text(
              p.nickname.isEmpty ? '陪诊师' : p.nickname,
              style: Theme.of(context).textTheme.titleLarge,
            ),
            const SizedBox(height: 4),
            Text(
              p.maskedPhone,
              style: const TextStyle(color: Colors.grey),
            ),
            const SizedBox(height: 12),
            Wrap(
              spacing: 8,
              runSpacing: 4,
              alignment: WrapAlignment.center,
              children: [
                _StatusChip(
                  label: p.realNameVerified ? '已实名' : '未实名',
                  color: p.realNameVerified ? Colors.green : Colors.red,
                ),
                _StatusChip(
                  label: p.approved ? '已审核' : '待审核',
                  color: p.approved ? Colors.green : Colors.orange,
                ),
                _StatusChip(label: 'ID: ${p.id}', color: Colors.blueGrey),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

/// 状态 chip。
class _StatusChip extends StatelessWidget {
  final String label;
  final Color color;
  const _StatusChip({required this.label, required this.color});
  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: color.withOpacity(0.1),
        borderRadius: BorderRadius.circular(4),
        border: Border.all(color: color),
      ),
      child: Text(
        label,
        style: TextStyle(
          color: color,
          fontSize: 12,
          fontWeight: FontWeight.w500,
        ),
      ),
    );
  }
}

/// 操作列表：实名认证 / 我的评价 / 我的钱包 / 我的培训。
class _ProfileActions extends ConsumerWidget {
  final dynamic profile; // Profile
  const _ProfileActions({required this.profile});

  Future<void> _showRealNameSheet(
    BuildContext context,
    WidgetRef ref,
  ) async {
    final result = await showModalBottomSheet<_RealNamePayload>(
      context: context,
      isScrollControlled: true,
      builder: (_) => const _RealNameSheet(),
    );
    if (result == null) return;
    try {
      await ref.read(realNameAuthControllerProvider.notifier).submit(
            realName: result.realName,
            idCard: result.idCard,
            idCardPhotoUrl: result.idCardPhotoUrl,
          );
      if (!context.mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('实名认证提交成功')),
      );
    } catch (e) {
      if (!context.mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('提交失败：$e')),
      );
    }
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final p = profile;
    return Card(
      elevation: 1,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
      child: Column(
        children: [
          if (!p.realNameVerified)
            ListTile(
              leading: const Icon(Icons.verified_user_outlined,
                  color: Colors.red),
              title: const Text('实名认证'),
              subtitle: const Text('完成后可正常接单', style: TextStyle(fontSize: 12)),
              trailing: const Icon(Icons.chevron_right),
              onTap: () => _showRealNameSheet(context, ref),
            ),
          if (p.realNameVerified)
            const ListTile(
              leading: Icon(Icons.verified, color: Colors.green),
              title: Text('实名认证'),
              subtitle: Text('已通过', style: TextStyle(fontSize: 12)),
            ),
          const Divider(height: 1),
          ListTile(
            leading: const Icon(Icons.account_balance_wallet_outlined),
            title: const Text('我的钱包'),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => context.push('/home/wallet'),
          ),
          const Divider(height: 1),
          ListTile(
            leading: const Icon(Icons.school_outlined),
            title: const Text('培训课程'),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => context.push('/home/training'),
          ),
          const Divider(height: 1),
          ListTile(
            leading: const Icon(Icons.star_outline),
            title: const Text('我的评价'),
            trailing: const Icon(Icons.chevron_right),
            onTap: () {
              ScaffoldMessenger.of(context).showSnackBar(
                const SnackBar(content: Text('评价列表（占位，后续 plan）')),
              );
            },
          ),
        ],
      ),
    );
  }
}

/// 退出登录按钮。
class _LogoutButton extends ConsumerWidget {
  Future<void> _confirm(BuildContext context, WidgetRef ref) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (_) => AlertDialog(
        title: const Text('退出登录'),
        content: const Text('确认退出当前账号？'),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(false),
            child: const Text('取消'),
          ),
          FilledButton(
            onPressed: () => Navigator.of(context).pop(true),
            child: const Text('退出'),
          ),
        ],
      ),
    );
    if (ok != true) return;
    try {
      await ref.read(logoutControllerProvider.notifier).logout();
      if (!context.mounted) return;
      context.go('/auth/login');
    } catch (e) {
      if (!context.mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('退出失败：$e')),
      );
    }
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return OutlinedButton.icon(
      onPressed: () => _confirm(context, ref),
      icon: const Icon(Icons.logout, color: Colors.red),
      label: const Text('退出登录', style: TextStyle(color: Colors.red)),
      style: OutlinedButton.styleFrom(
        side: const BorderSide(color: Colors.red),
        padding: const EdgeInsets.symmetric(vertical: 12),
      ),
    );
  }
}

/// 实名认证表单 payload。
class _RealNamePayload {
  final String realName;
  final String idCard;
  final String? idCardPhotoUrl;
  const _RealNamePayload(this.realName, this.idCard, this.idCardPhotoUrl);
}

/// 实名认证底部表单。
class _RealNameSheet extends StatefulWidget {
  const _RealNameSheet();
  @override
  State<_RealNameSheet> createState() => _RealNameSheetState();
}

class _RealNameSheetState extends State<_RealNameSheet> {
  final _nameCtrl = TextEditingController();
  final _idCardCtrl = TextEditingController();
  final _photoCtrl = TextEditingController();
  final _formKey = GlobalKey<FormState>();

  @override
  void dispose() {
    _nameCtrl.dispose();
    _idCardCtrl.dispose();
    _photoCtrl.dispose();
    super.dispose();
  }

  String? _validateName(String? v) {
    final s = v?.trim() ?? '';
    if (s.isEmpty) return '请输入真实姓名';
    if (s.length < 2) return '姓名至少 2 个字符';
    return null;
  }

  String? _validateIdCard(String? v) {
    final s = v?.trim() ?? '';
    if (s.isEmpty) return '请输入身份证号';
    if (s.length != 18 && s.length != 15) return '身份证号应为 15 或 18 位';
    return null;
  }

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: EdgeInsets.only(
        bottom: MediaQuery.of(context).viewInsets.bottom,
        left: 16,
        right: 16,
        top: 16,
      ),
      child: Form(
        key: _formKey,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            const Text('实名认证',
                style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
            const SizedBox(height: 16),
            TextFormField(
              controller: _nameCtrl,
              decoration: const InputDecoration(
                labelText: '真实姓名',
                border: OutlineInputBorder(),
              ),
              validator: _validateName,
            ),
            const SizedBox(height: 12),
            TextFormField(
              controller: _idCardCtrl,
              decoration: const InputDecoration(
                labelText: '身份证号',
                border: OutlineInputBorder(),
              ),
              validator: _validateIdCard,
            ),
            const SizedBox(height: 12),
            TextFormField(
              controller: _photoCtrl,
              decoration: const InputDecoration(
                labelText: '证件照 URL（可选）',
                border: OutlineInputBorder(),
                helperText: '开发阶段可留空',
              ),
            ),
            const SizedBox(height: 16),
            FilledButton(
              onPressed: () {
                if (_formKey.currentState?.validate() != true) return;
                Navigator.of(context).pop(_RealNamePayload(
                  _nameCtrl.text.trim(),
                  _idCardCtrl.text.trim(),
                  _photoCtrl.text.trim().isEmpty
                      ? null
                      : _photoCtrl.text.trim(),
                ));
              },
              child: const Text('提交认证'),
            ),
            const SizedBox(height: 16),
          ],
        ),
      ),
    );
  }
}