/**
 * LoginPage：admin-web 登录页（升级版）。
 *
 * 功能：
 *   - 用户名 + 密码表单；
 *   - mock 登录（直接调 useAuthStore.login → 写 token + user + role）；
 *   - 登录成功跳 from 或 /dashboard；
 *   - 6 个角色 demo 账号一键填入（super / order / refund / audit / cs / viewer）。
 *
 * 数据流：
 *   - login: POST /api/v1/admin/login  → 当前 mock 由 useAuthStore.login 直接 mock
 *
 * 对应 spec：2026-09-24-admin-web-design.md §Task 21
 */
import { useState } from 'react';
import { Button, Card, Form, Input, Space, Tag } from 'antd';
import { useLocation, useNavigate } from 'react-router-dom';
import { PageHeader } from '@/components/PageHeader';
import { useAuthStore } from '@/stores/authStore';

interface LoginValues {
  username: string;
  password: string;
}

interface DemoAccount {
  username: string;
  label: string;
  role: string;
  color: string;
}

const DEMO_ACCOUNTS: DemoAccount[] = [
  { username: 'super', label: '超级管理员', role: 'super_admin', color: 'red' },
  { username: 'order', label: '订单管理员', role: 'order_admin', color: 'orange' },
  { username: 'refund', label: '退款管理员', role: 'refund_admin', color: 'gold' },
  { username: 'audit', label: '审核管理员', role: 'audit_admin', color: 'green' },
  { username: 'cs', label: '客服', role: 'cs', color: 'blue' },
  { username: 'viewer', label: '只读观察', role: 'viewer', color: 'default' },
];

export default function LoginPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const login = useAuthStore((s) => s.login);
  const [submitting, setSubmitting] = useState(false);
  const [form] = Form.useForm<LoginValues>();

  const onFinish = async (values: LoginValues) => {
    setSubmitting(true);
    try {
      await login(values.username, values.password);
      const from = (location.state as { from?: string } | null)?.from ?? '/dashboard';
      navigate(from, { replace: true });
    } catch (e) {
      // eslint-disable-next-line no-console
      console.error('login error', e);
    } finally {
      setSubmitting(false);
    }
  };

  const fillDemo = (username: string) => {
    form.setFieldsValue({ username, password: 'demo' });
  };

  return (
    <div
      data-testid="login-page"
      style={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: '#f0f2f5',
      }}
    >
      <Card style={{ width: 380 }}>
        <PageHeader title="Doctors Admin" subtitle="后台管理系统登录" />
        <Form<LoginValues>
          form={form}
          layout="vertical"
          onFinish={onFinish}
          initialValues={{ username: 'super', password: 'demo' }}
        >
          <Form.Item
            name="username"
            label="用户名"
            rules={[{ required: true, message: '请输入用户名' }]}
          >
            <Input
              placeholder="super / order / refund / audit / cs / viewer"
              data-testid="login-username"
            />
          </Form.Item>
          <Form.Item
            name="password"
            label="密码"
            rules={[{ required: true, message: '请输入密码' }]}
          >
            <Input.Password
              placeholder="demo"
              data-testid="login-password"
            />
          </Form.Item>
          <Form.Item>
            <Button
              type="primary"
              htmlType="submit"
              loading={submitting}
              block
              data-testid="login-submit"
            >
              登录
            </Button>
          </Form.Item>
          <div style={{ marginBottom: 12 }}>
            <p style={{ color: '#999', fontSize: 12, margin: '0 0 6px' }}>
              Demo 账号一键填入：
            </p>
            <Space wrap>
              {DEMO_ACCOUNTS.map((acc) => (
                <Tag
                  key={acc.username}
                  color={acc.color}
                  style={{ cursor: 'pointer' }}
                  onClick={() => fillDemo(acc.username)}
                  data-testid={`demo-${acc.username}`}
                >
                  {acc.username} · {acc.label}
                </Tag>
              ))}
            </Space>
          </div>
          <p style={{ color: '#999', fontSize: 12, margin: 0 }}>
            Mock 登录：用户名决定角色（super / order / refund / audit / cs / viewer），密码任意。
          </p>
        </Form>
      </Card>
    </div>
  );
}