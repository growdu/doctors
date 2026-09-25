// lib/models/profile.dart
//
// 陪诊师用户资料（spec §A5）。
//
// 字段语义：
//   - realNameVerified：是否完成实名（个人中心显示「已实名」chip）
//   - approved：是否通过平台审核（控制能否接单）
//   - role：escort / patient / admin（陪诊师端恒为 escort）
import '../utils/format.dart';

/// 用户资料。
class Profile {
  final int id;
  final String phone;
  final String role;
  final String nickname;

  /// 头像 URL（可空；为 null 时显示默认头像）。
  final String? avatarUrl;

  /// 是否已完成实名认证。
  final bool realNameVerified;

  /// 是否通过平台审核（true 后才能在邀请页 / 订单页正常工作）。
  final bool approved;

  const Profile({
    required this.id,
    required this.phone,
    required this.role,
    required this.nickname,
    this.avatarUrl,
    required this.realNameVerified,
    required this.approved,
  });

  factory Profile.fromJson(Map<String, dynamic> j) => Profile(
        id: j['id'] as int,
        phone: j['phone'] as String,
        role: j['role'] as String,
        nickname: (j['nickname'] as String?) ?? '',
        avatarUrl: j['avatar_url'] as String?,
        realNameVerified: (j['real_name_verified'] as bool?) ?? false,
        approved: (j['approved'] as bool?) ?? false,
      );

  Map<String, dynamic> toJson() => {
        'id': id,
        'phone': phone,
        'role': role,
        'nickname': nickname,
        if (avatarUrl != null) 'avatar_url': avatarUrl,
        'real_name_verified': realNameVerified,
        'approved': approved,
      };

  /// 手机号脱敏展示（默认用 format.maskPhone）。
  String get maskedPhone => maskPhone(phone);

  Profile copyWith({
    String? nickname,
    String? avatarUrl,
    bool? realNameVerified,
    bool? approved,
  }) =>
      Profile(
        id: id,
        phone: phone,
        role: role,
        nickname: nickname ?? this.nickname,
        avatarUrl: avatarUrl ?? this.avatarUrl,
        realNameVerified: realNameVerified ?? this.realNameVerified,
        approved: approved ?? this.approved,
      );
}