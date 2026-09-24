import { Layout, Menu, theme } from 'antd';
import { Outlet, useLocation, useNavigate } from 'react-router-dom';
import {
  DashboardOutlined,
  TeamOutlined,
  FileTextOutlined,
} from '@ant-design/icons';

const { Header, Sider, Content } = Layout;

/**
 * AdminLayout 雏形：经典 Admin 三段式（侧边栏 + 顶栏 + 内容区）。
 * 后续 Task（4+）会在此基础上接入：
 *   - AuthGuard / RequirePermission（router/guards.tsx，spec §3.3）
 *   - 真实菜单 + RBAC 过滤（spec §3.2）
 *   - 用户信息下拉 / 退出登录
 *   - 面包屑 / TraceId
 */
export default function AdminLayout() {
  const navigate = useNavigate();
  const location = useLocation();
  const {
    token: { colorBgContainer, borderRadiusLG },
  } = theme.useToken();

  // 当前选中菜单项：取 pathname 第一段作为 key（如 /orders/123 → orders）
  const selectedKey = location.pathname.split('/')[1] || 'dashboard';

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider width={220} theme="dark">
        <div
          style={{
            height: 64,
            margin: 16,
            color: '#fff',
            fontSize: 18,
            fontWeight: 600,
            textAlign: 'center',
            lineHeight: '32px',
          }}
        >
          Doctors Admin
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[selectedKey]}
          items={[
            { key: 'dashboard', icon: <DashboardOutlined />, label: '数据看板' },
            { key: 'escorts', icon: <TeamOutlined />, label: '陪诊师' },
            { key: 'orders', icon: <FileTextOutlined />, label: '订单' },
          ]}
          onClick={({ key }) => navigate(`/${key}`)}
        />
      </Sider>
      <Layout>
        <Header
          style={{
            padding: '0 24px',
            background: colorBgContainer,
            borderBottom: '1px solid #f0f0f0',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
          }}
        >
          <div style={{ fontSize: 16, fontWeight: 500 }}>后台管理系统</div>
          <div style={{ color: '#999' }}>v0.1.0</div>
        </Header>
        <Content
          style={{
            margin: 16,
            padding: 24,
            background: colorBgContainer,
            borderRadius: borderRadiusLG,
            minHeight: 280,
          }}
        >
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
}