/**
 * ProfilePage：admin-web 个人资料页（升级版）。
 *
 * 功能：
 *   - 展示当前用户基本信息（id / username / display_name / role / 登录态）；
 *   - 修改密码表单（mock：填入新密码 + 确认密码 → message.success）；
 *   - 退出登录按钮（清状态 + 跳 /login）。
 *
 * 数据流：
 *   - 全部走 useAuthStore；
 *   - 修改密码为 mock（暂不调真实接口）。
 *
 * 对应 spec：2026-09-24-admin-web-design.md §Task 20
 */
import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Avatar,
  Button,
  Card,
  Col,
  Descriptions,
  Form,
  Input,
  Row,
  Space,
  Tag,
  Typography,
} from 'antd';
import { LogoutOutlined, LockOutlined, UserOutlined } from '@ant-design/icons';
import { PageHeader } from '@/components/PageHeader';
import { useAuthStore } from '@/stores/authStore';

const { Title } = Typography;

interface ChangePasswordValues {
  old_password: string;
  new_password: string;
  confirm_password: string;
}

const ROLE_LABEL: Record<string, string> = {
  super_admin: '超级管理员',
  order_admin: '订单管理员',
  refund_admin: '退款管理员',
  audit_admin: '审核管理员',
  cs: '客服',
  viewer: '只读观察',
};

export default function ProfilePage() {
  const navigate = useNavigate();
  const user = useAuthStore((s) => s.user);
  const role = useAuthStore((s) => s.role);
  const logout = useAuthStore((s) => s.logout);
  const [form] = Form.useForm<ChangePasswordValues>();

  const onChangePassword = async () => {
    try {
      const values = await form.validateFields();
      if (values.new_password !== values.confirm_password) {
        form.setFields([
          {
            name: 'confirm_password',
            errors: ['两次输入的密码不一致'],
          },
        ]);
        return;
      }
      // mock：直接 message.success，不调真实接口
      // eslint-disable-next-line no-console
      console.info('[ProfilePage] change password mock ok');
      form.resetFields();
    } catch {
      // ignore
    }
  };

  const onLogout = () => {
    logout();
    navigate('/login');
  };

  if (!user) {
    return (
      <div data-testid="profile-page">
        <PageHeader title="个人资料" subtitle="请先登录" />
        <Card>
          <p>当前未登录管理员账号</p>
        </Card>
      </div>
    );
  }

  return (
    <div data-testid="profile-page">
      <PageHeader title="个人资料" subtitle="当前管理员信息 + 修改密码 + 退出登录" />

      <Row gutter={16}>
        {/* 左：头像 + 角色 */}
        <Col xs={24} md={8}>
          <Card data-testid="profile-card-left">
            <Space direction="vertical" align="center" style={{ width: '100%' }}>
              <Avatar
                size={88}
                icon={<UserOutlined />}
                src={user.avatar_url ?? undefined}
                data-testid="profile-avatar"
              />
              <Title level={4} style={{ margin: 0 }} data-testid="profile-name">
                {user.display_name}
              </Title>
              <Tag color="blue" data-testid="profile-role">
                {ROLE_LABEL[role ?? ''] ?? role ?? '未知角色'}
              </Tag>
              <Button
                danger
                icon={<LogoutOutlined />}
                onClick={onLogout}
                data-testid="btn-logout"
              >
                退出登录
              </Button>
            </Space>
          </Card>
        </Col>

        {/* 右：基本信息 + 修改密码 */}
        <Col xs={24} md={16}>
          <Card title="基本信息">
            <Descriptions column={1} bordered size="small">
              <Descriptions.Item label="用户 ID">#{user.id}</Descriptions.Item>
              <Descriptions.Item label="用户名">{user.username}</Descriptions.Item>
              <Descriptions.Item label="显示名">
                {user.display_name}
              </Descriptions.Item>
              <Descriptions.Item label="角色">
                <Tag color="blue">{ROLE_LABEL[role ?? ''] ?? role ?? '—'}</Tag>
              </Descriptions.Item>
            </Descriptions>
          </Card>

          <Card style={{ marginTop: 16 }} title="修改密码">
            <Form
              form={form}
              layout="vertical"
              data-testid="change-password-form"
            >
              <Form.Item
                label="当前密码"
                name="old_password"
                rules={[{ required: true, message: '请输入当前密码' }]}
              >
                <Input.Password
                  prefix={<LockOutlined />}
                  placeholder="当前密码"
                  data-testid="pw-old"
                />
              </Form.Item>
              <Form.Item
                label="新密码"
                name="new_password"
                rules={[
                  { required: true, message: '请输入新密码' },
                  { min: 6, message: '密码至少 6 位' },
                ]}
              >
                <Input.Password
                  prefix={<LockOutlined />}
                  placeholder="新密码（≥6 位）"
                  data-testid="pw-new"
                />
              </Form.Item>
              <Form.Item
                label="确认新密码"
                name="confirm_password"
                rules={[{ required: true, message: '请再次输入新密码' }]}
              >
                <Input.Password
                  prefix={<LockOutlined />}
                  placeholder="再次输入新密码"
                  data-testid="pw-confirm"
                />
              </Form.Item>
              <Button
                type="primary"
                onClick={onChangePassword}
                data-testid="btn-change-password"
              >
                提交修改
              </Button>
            </Form>
          </Card>
        </Col>
      </Row>
    </div>
  );
}