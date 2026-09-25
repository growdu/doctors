/**
 * admin-settings API 客户端（14 P0 页依赖）。
 *
 * 设置数据持久化方案：
 *   - 当前为 mock：localStorage key = `doctors-admin-settings`；
 *   - 真实接入时替换为 fetch GET/PUT /api/v1/admin/settings。
 *
 * 分类：
 *   - general     站点名 / Logo / 联系电话
 *   - payment     支付开关 / 渠道 / 费率
 *   - sms         短信签名 / 模板 / provider
 *   - push        推送开关 / provider / 时段
 *
 * 对应 spec：2026-09-24-admin-web-design.md §Task 19
 */

const STORAGE_KEY = 'doctors-admin-settings';

export type SettingsCategory = 'general' | 'payment' | 'sms' | 'push';

export interface GeneralSettings {
  site_name: string;
  logo_url: string;
  contact_phone: string;
}

export interface PaymentSettings {
  enabled: boolean;
  channels: Array<{ name: 'wechat' | 'alipay' | 'unionpay'; enabled: boolean }>;
  fee_rate: number; // 0~1
}

export interface SmsSettings {
  provider: string;
  signature: string;
  template_id: string;
}

export interface PushSettings {
  enabled: boolean;
  provider: 'jpush' | 'getui' | 'firebase';
  quiet_hours: { start: string; end: string };
}

export interface AdminSettings {
  general: GeneralSettings;
  payment: PaymentSettings;
  sms: SmsSettings;
  push: PushSettings;
}

export const DEFAULT_SETTINGS: AdminSettings = {
  general: {
    site_name: 'Doctors Admin',
    logo_url: '',
    contact_phone: '400-000-0000',
  },
  payment: {
    enabled: true,
    channels: [
      { name: 'wechat', enabled: true },
      { name: 'alipay', enabled: true },
      { name: 'unionpay', enabled: false },
    ],
    fee_rate: 0.006,
  },
  sms: {
    provider: 'aliyun',
    signature: 'Doctors',
    template_id: 'SMS_0001',
  },
  push: {
    enabled: true,
    provider: 'jpush',
    quiet_hours: { start: '22:00', end: '08:00' },
  },
};

/** 读取全部设置（localStorage 缺省时返回 DEFAULT_SETTINGS）。 */
export function loadSettings(): AdminSettings {
  if (typeof window === 'undefined') return DEFAULT_SETTINGS;
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return DEFAULT_SETTINGS;
    const parsed = JSON.parse(raw) as Partial<AdminSettings>;
    return {
      general: { ...DEFAULT_SETTINGS.general, ...parsed.general },
      payment: {
        ...DEFAULT_SETTINGS.payment,
        ...parsed.payment,
        channels: parsed.payment?.channels ?? DEFAULT_SETTINGS.payment.channels,
      },
      sms: { ...DEFAULT_SETTINGS.sms, ...parsed.sms },
      push: {
        ...DEFAULT_SETTINGS.push,
        ...parsed.push,
        quiet_hours:
          parsed.push?.quiet_hours ?? DEFAULT_SETTINGS.push.quiet_hours,
      },
    };
  } catch {
    return DEFAULT_SETTINGS;
  }
}

/** 持久化全部设置。 */
export function saveSettings(s: AdminSettings): void {
  if (typeof window === 'undefined') return;
  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(s));
}

/** 仅保存某个分类的设置并合并回全量。 */
export function saveCategory<K extends SettingsCategory>(
  category: K,
  payload: AdminSettings[K],
): AdminSettings {
  const cur = loadSettings();
  const next: AdminSettings = { ...cur, [category]: payload };
  saveSettings(next);
  return next;
}

/** 重置为默认值。 */
export function resetSettings(): AdminSettings {
  saveSettings(DEFAULT_SETTINGS);
  return DEFAULT_SETTINGS;
}

export const settingsQueryKeys = {
  all: ['admin-settings'] as const,
};