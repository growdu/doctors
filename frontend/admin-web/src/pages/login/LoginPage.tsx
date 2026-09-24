/**
 * LoginPage：登录页骨架（v1 Task 21 占位）。
 *
 * 范围：
 *   - 用户名 / 密码表单；
 *   - mock 登录：username=super → super_admin / order → order_admin / ... / 其他 → viewer；
 *   - 成功后跳回 from 或 /dashboard。
 *
 * TODO: 实现页面（来自 plan v1 Task 21）
 */
import { useState } from 'react';
import { Button, Card, Form, Input, message } from 'antd';
import { useLocation, useNavigate } from 'react-router-dom';
import { PageHeader } from '@/components/PageHeader';
import { useAuthStore } from '@/stores/authStore';

interface LoginValues {
  username: string;
  password: string;
}

export default function LoginPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const login = useAuthStore((s) => s.login);
  const [submitting, setSubmitting] = useState(false);

  const onFinish = async (values: LoginValues) => {
    setSubmitting(true);
    try {
      await login(values.username, values.password);
      message.success('登录成功');
      const from = (location.state as { from?: string } | null)?.from ?? '/dashboard';
      navigate(from, { replace: true });
    } catch (e) {
      message.error(`登录失败：${(e as Error).message}`);
    } finally {
      setSubmitting(false);
    }
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
      <Card style={{ width: 360 }}>
        <PageHeader title="Doctors Admin" subtitle="后台管理系统登录" />
        <Form<LoginValues>
          layout="vertical"
          onFinish={onFinish}
          initialValues={{ username: 'super', password: 'demo' }}
        >
          <Form.Item
            name="username"
            label="用户名"
            rules={[{ required: true, message: '请输入用户名' }]}
          >
            <Input placeholder="super / order / refund / audit / cs / viewer" />
          </Form.Item>
          <Form.Item
            name="password"
            label="密码"
            rules={[{ required: true, message: '请输入密码' }]}
          >
            <Input.Password placeholder="demo" />
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
          <p style={{ color: '#999', fontSize: 12, margin: 0 }}>
            Mock 登录：用户名决定角色（super / order / refund / audit / cs / viewer），密码任意。
          </p>
        </Form>
      </Card>
    </div>
  );
}