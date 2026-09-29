/**
 * SettingsPage：admin-web 系统设置页（升级版）。
 *
 * 功能：
 *   - 4 类分类 Tabs：基础 / 支付 / 短信 / 推送；
 *   - 每个分类独立 Form（站点名 / Logo / 联系电话 / 支付渠道 / 短信签名 / 推送 provider）；
 *   - 保存到 localStorage（mock 持久化）。
 *
 * 数据流：
 *   - 读 / 写：loadSettings() / saveSettings() / saveCategory() （来自 settings API）
 *
 * 对应 spec：2026-09-24-admin-web-design.md §Task 19
 */
import { useEffect, useState } from 'react';
import {
  Button,
  Card,
  Form,
  Input,
  InputNumber,
  Space,
  Switch,
  Tabs,
  Tag,
  message,
} from 'antd';
import { SaveOutlined, ReloadOutlined } from '@ant-design/icons';
import { PageHeader } from '@/components/PageHeader';
import {
  DEFAULT_SETTINGS,
  loadSettings,
  resetSettings,
  saveCategory,
  type AdminSettings,
  type GeneralSettings,
  type PaymentSettings,
  type PushSettings,
  type SmsSettings,
} from '@/api/admin/settings';

export default function SettingsPage() {
  const [settings, setSettings] = useState<AdminSettings>(DEFAULT_SETTINGS);
  const [generalForm] = Form.useForm<GeneralSettings>();
  const [paymentForm] = Form.useForm<PaymentSettings>();
  const [smsForm] = Form.useForm<SmsSettings>();
  const [pushForm] = Form.useForm<PushSettings>();

  // 加载初始值
  useEffect(() => {
    const loaded = loadSettings();
    setSettings(loaded);
    generalForm.setFieldsValue(loaded.general);
    paymentForm.setFieldsValue(loaded.payment);
    smsForm.setFieldsValue(loaded.sms);
    pushForm.setFieldsValue(loaded.push);
  }, [generalForm, paymentForm, smsForm, pushForm]);

  // ── 保存 handler ──────────────────────────────────────────────────
  const onSaveGeneral = async () => {
    try {
      const values = await generalForm.validateFields();
      const next = saveCategory('general', values);
      setSettings(next);
      message.success('已保存基础设置');
    } catch {
      // ignore
    }
  };

  const onSavePayment = async () => {
    try {
      const values = await paymentForm.validateFields();
      const next = saveCategory('payment', values);
      setSettings(next);
      message.success('已保存支付设置');
    } catch {
      // ignore
    }
  };

  const onSaveSms = async () => {
    try {
      const values = await smsForm.validateFields();
      const next = saveCategory('sms', values);
      setSettings(next);
      message.success('已保存短信设置');
    } catch {
      // ignore
    }
  };

  const onSavePush = async () => {
    try {
      const values = await pushForm.validateFields();
      const next = saveCategory('push', values);
      setSettings(next);
      message.success('已保存推送设置');
    } catch {
      // ignore
    }
  };

  const onReset = () => {
    const next = resetSettings();
    setSettings(next);
    generalForm.setFieldsValue(next.general);
    paymentForm.setFieldsValue(next.payment);
    smsForm.setFieldsValue(next.sms);
    pushForm.setFieldsValue(next.push);
    message.success('已重置为默认值');
  };

  return (
    <div data-testid="settings-page">
      <PageHeader
        title="系统设置"
        subtitle="基础 / 支付 / 短信 / 推送 四类持久化配置"
        extra={
          <Button
            icon={<ReloadOutlined />}
            onClick={onReset}
            data-testid="btn-reset"
          >
            重置默认
          </Button>
        }
      />

      <Card>
        <Tabs
          data-testid="settings-tabs"
          items={[
            {
              key: 'general',
              label: '基础',
              children: (
                <Form form={generalForm} layout="vertical">
                  <Form.Item
                    label="站点名称"
                    name="site_name"
                    rules={[{ required: true, message: '请输入站点名称' }]}
                  >
                    <Input data-testid="general-site-name" maxLength={40} />
                  </Form.Item>
                  <Form.Item label="Logo URL" name="logo_url">
                    <Input
                      placeholder="https://..."
                      data-testid="general-logo"
                      maxLength={200}
                    />
                  </Form.Item>
                  <Form.Item
                    label="联系电话"
                    name="contact_phone"
                    rules={[{ required: true, message: '请输入联系电话' }]}
                  >
                    <Input data-testid="general-phone" maxLength={20} />
                  </Form.Item>
                  <Button
                    type="primary"
                    icon={<SaveOutlined />}
                    onClick={onSaveGeneral}
                    data-testid="btn-save-general"
                  >
                    保存基础
                  </Button>
                </Form>
              ),
            },
            {
              key: 'payment',
              label: '支付',
              children: (
                <Form form={paymentForm} layout="vertical">
                  <Form.Item
                    label="启用支付"
                    name="enabled"
                    valuePropName="checked"
                  >
                    <Switch data-testid="payment-enabled" />
                  </Form.Item>
                  <Form.Item
                    label="费率（0~1）"
                    name="fee_rate"
                    rules={[{ required: true, message: '请输入费率' }]}
                  >
                    <InputNumber
                      min={0}
                      max={1}
                      step={0.001}
                      style={{ width: 200 }}
                      data-testid="payment-fee-rate"
                    />
                  </Form.Item>
                  <Form.Item label="当前渠道快照">
                    <Space wrap>
                      {settings.payment.channels.map((c) => (
                        <Tag
                          key={c.name}
                          color={c.enabled ? 'green' : 'default'}
                          data-testid={`payment-channel-${c.name}`}
                        >
                          {c.name} · {c.enabled ? '开启' : '关闭'}
                        </Tag>
                      ))}
                    </Space>
                  </Form.Item>
                  <Button
                    type="primary"
                    icon={<SaveOutlined />}
                    onClick={onSavePayment}
                    data-testid="btn-save-payment"
                  >
                    保存支付
                  </Button>
                </Form>
              ),
            },
            {
              key: 'sms',
              label: '短信',
              children: (
                <Form form={smsForm} layout="vertical">
                  <Form.Item
                    label="短信服务商"
                    name="provider"
                    rules={[{ required: true, message: '请选择服务商' }]}
                  >
                    <Input data-testid="sms-provider" maxLength={20} />
                  </Form.Item>
                  <Form.Item
                    label="签名"
                    name="signature"
                    rules={[{ required: true, message: '请输入签名' }]}
                  >
                    <Input data-testid="sms-signature" maxLength={20} />
                  </Form.Item>
                  <Form.Item
                    label="模板 ID"
                    name="template_id"
                    rules={[{ required: true, message: '请输入模板 ID' }]}
                  >
                    <Input data-testid="sms-template" maxLength={40} />
                  </Form.Item>
                  <Button
                    type="primary"
                    icon={<SaveOutlined />}
                    onClick={onSaveSms}
                    data-testid="btn-save-sms"
                  >
                    保存短信
                  </Button>
                </Form>
              ),
            },
            {
              key: 'push',
              label: '推送',
              children: (
                <Form form={pushForm} layout="vertical">
                  <Form.Item
                    label="启用推送"
                    name="enabled"
                    valuePropName="checked"
                  >
                    <Switch data-testid="push-enabled" />
                  </Form.Item>
                  <Form.Item
                    label="推送服务商"
                    name="provider"
                    rules={[{ required: true, message: '请选择推送服务商' }]}
                  >
                    <Input
                      data-testid="push-provider"
                      maxLength={20}
                      placeholder="jpush / getui / firebase"
                    />
                  </Form.Item>
                  <Form.Item
                    label="免打扰开始时间"
                    name={['quiet_hours', 'start']}
                    rules={[{ required: true, message: '请输入开始时间' }]}
                  >
                    <Input
                      placeholder="22:00"
                      data-testid="push-quiet-start"
                      maxLength={5}
                    />
                  </Form.Item>
                  <Form.Item
                    label="免打扰结束时间"
                    name={['quiet_hours', 'end']}
                    rules={[{ required: true, message: '请输入结束时间' }]}
                  >
                    <Input
                      placeholder="08:00"
                      data-testid="push-quiet-end"
                      maxLength={5}
                    />
                  </Form.Item>
                  <Button
                    type="primary"
                    icon={<SaveOutlined />}
                    onClick={onSavePush}
                    data-testid="btn-save-push"
                  >
                    保存推送
                  </Button>
                </Form>
              ),
            },
          ]}
        />
      </Card>
    </div>
  );
}