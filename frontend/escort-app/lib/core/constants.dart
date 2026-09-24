// lib/core/constants.dart
//
// 全局常量 —— API base URL / trace prefix / 30s 邀请超时等。
//
// 命名约定：k + PascalCase（如 kApiBaseUrlDev）。

/// API base URL —— 通过 `--dart-define=API_BASE=...` 覆盖；空值时走 dev 默认。
const String kApiBaseDefault = 'https://api.dev.doctors.example.com/api/v1';
const String kApiBaseProd = 'https://api.doctors.example.com/api/v1';

/// Trace ID 前缀 —— 后端日志侧便于按端拆分（escort-{ms}-{rand}）。
const String kTracePrefix = 'escort';

/// 邀请超时秒数（spec §1.2 + §4.1：30s 倒计时）。
const int kInvitationTimeoutSeconds = 30;

/// GPS 签到容差（米）—— spec §3 GPS 校验阈值。
const double kGpsCheckinToleranceMeters = 200.0;

/// SharedPreferences / flutter_secure_storage key 命名。
const String kStorageKeyToken = 'escort_app.jwt';
const String kStorageKeyRefreshToken = 'escort_app.refresh_token';
const String kStorageKeyOnboardingStage = 'escort_app.onboarding_stage';

/// 从 `--dart-define` 读 API base；dev / prod 默认值见上。
String resolveApiBase() {
  const fromDefine = String.fromEnvironment('API_BASE', defaultValue: '');
  return fromDefine.isEmpty ? kApiBaseDefault : fromDefine;
}