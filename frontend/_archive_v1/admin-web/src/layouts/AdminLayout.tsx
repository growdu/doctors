/**
 * AdminLayout：经典 Admin 三段式（侧边栏 + 顶栏 + 内容区）。
 *
 * 设计要点（spec §3.2 菜单 + §3.3 RBAC + §5.1 Auth + §5.2 TraceId）：
 *   - 12 项菜单定义于 MENU_ITEMS；roles 未定义 = 全部角色可见；
 *   - 当前 role 不在 roles 数组 → 该菜单项不渲染（RBAC 过滤）；
 *   - 菜单点击 → useNavigate() 跳 path；
 *   - 当前路由高亮：selectedKeys 按 path 前缀匹配（最长前缀优先）；
 *   - 顶部 Header：左 = 面包屑（看板 / 当前项），右 = TraceId + 用户下拉；
 *   - 用户下拉：昵称 / 角色（只读） / 登出；
 *   - 登出：useAuthStore.logout() + useNavigate('/login')。
 *
 * 对应 spec：2026-09-24-admin-web-design.md §3.2 + §3.3 + §5.1 + §5.2
 */
import { useMemo, type ReactNode } from 'react';
import { Layout, Menu, theme, Breadcrumb, Dropdown, Avatar, Space } from 'antd';
import { Outlet, useLocation, useNavigate } from 'react-router-dom';
import {
  DashboardOutlined,
  OrderedListOutlined,
  UserOutlined,
  AuditOutlined,
  DollarOutlined,
  WalletOutlined,
  ToolOutlined,
  StarOutlined,
  MessageOutlined,
  AlertOutlined,
  BarChartOutlined,
  SettingOutlined,
  LogoutOutlined,
} from '@ant-design/icons';
import { useAuthStore, type AdminRole } from '@/stores/authStore';
import { TraceId } from '@/components/TraceId';

const { Header, Sider, Content } = Layout;

// ── 菜单定义（12 项）────────────────────────────────────────────────
interface MenuItemDef {
  key: string;
  path: string;
  label: string;
  icon: ReactNode;
  /** 允许访问的角色；undefined = 全部角色可见 */
  roles?: AdminRole[];
  /** 面包屑末级文案 */
  breadcrumb: string;
}

const MENU_ITEMS: MenuItemDef[] = [
  {
    key: 'dashboard',
    path: '/dashboard',
    label: '看板',
    icon: <DashboardOutlined />,
    breadcrumb: '看板',
  },
  {
    key: 'orders',
    path: '/orders',
    label: '订单',
    icon: <OrderedListOutlined />,
    roles: ['super_admin', 'order_admin', 'refund_admin', 'cs', 'viewer'],
    breadcrumb: '订单管理',
  },
  {
    key: 'escorts',
    path: '/escorts',
    label: '陪诊师',
    icon: <UserOutlined />,
    roles: ['super_admin', 'audit_admin', 'viewer'],
    breadcrumb: '陪诊师',
  },
  {
    key: 'escorts-audit',
    path: '/escorts/audit',
    label: '陪诊师审核',
    icon: <AuditOutlined />,
    roles: ['super_admin', 'audit_admin'],
    breadcrumb: '陪诊师审核',
  },
  {
    key: 'refunds',
    path: '/refunds',
    label: '退款审批',
    icon: <DollarOutlined />,
    roles: ['super_admin', 'refund_admin', 'cs', 'viewer'],
    breadcrumb: '退款审批',
  },
  {
    key: 'wallets',
    path: '/wallets',
    label: '钱包',
    icon: <WalletOutlined />,
    roles: ['super_admin', 'viewer'],
    breadcrumb: '钱包',
  },
  {
    key: 'work-orders',
    path: '/work-orders',
    label: '工单',
    icon: <ToolOutlined />,
    roles: ['super_admin', 'order_admin', 'refund_admin', 'audit_admin', 'cs'],
    breadcrumb: '工单',
  },
  {
    key: 'reviews',
    path: '/reviews',
    label: '评价',
    icon: <StarOutlined />,
    breadcrumb: '评价',
  },
  {
    key: 'messages',
    path: '/messages',
    label: '站内信',
    icon: <MessageOutlined />,
    roles: ['super_admin', 'cs'],
    breadcrumb: '站内信',
  },
  {
    key: 'sos',
    path: '/sos',
    label: 'SOS',
    icon: <AlertOutlined />,
    roles: ['super_admin', 'cs', 'viewer'],
    breadcrumb: 'SOS',
  },
  {
    key: 'reports',
    path: '/reports',
    label: '报表',
    icon: <BarChartOutlined />,
    roles: ['super_admin', 'order_admin', 'refund_admin', 'audit_admin', 'viewer'],
    breadcrumb: '报表',
  },
  {
    key: 'settings',
    path: '/settings',
    label: '设置',
    icon: <SettingOutlined />,
    roles: ['super_admin'],
    breadcrumb: '设置',
  },
];

/** 从 pathname 计算面包屑末级（最长前缀匹配，避免 /escorts 截胡 /escorts/audit） */
function lastBreadcrumb(pathname: string): string {
  const sorted = [...MENU_ITEMS].sort((a, b) => b.path.length - a.path.length);
  for (const item of sorted) {
    if (pathname === item.path || pathname.startsWith(item.path + '/')) {
      return item.breadcrumb;
    }
  }
  return '首页';
}

/**
 * AdminLayout 主体组件。
 *
 * RBAC 过滤：从 useAuthStore() 读 role；roles 未定义 = 全部可见；
 * roles 含当前 role → 渲染；否则过滤。
 */
export default function AdminLayout() {
  const navigate = useNavigate();
  const location = useLocation();
  const user = useAuthStore((s) => s.user);
  const role = useAuthStore((s) => s.role);
  const logout = useAuthStore((s) => s.logout);
  const {
    token: { colorBgContainer, borderRadiusLG },
  } = theme.useToken();

  // 当前 role 可见的菜单项（按定义顺序）
  const visibleItems = useMemo<MenuItemDef[]>(() => {
    if (!role) return [];
    return MENU_ITEMS.filter((it) => !it.roles || it.roles.includes(role));
  }, [role]);

  // antd Menu items 配置（label 包一层 span 注入 data-testid）
  const menuItems = useMemo(
    () =>
      visibleItems.map((it) => ({
        key: it.key,
        icon: it.icon,
        label: <span data-testid={`menu-item-${it.key}`}>{it.label}</span>,
      })),
    [visibleItems],
  );

  // 当前高亮 key：按 path 前缀匹配，最长匹配优先
  const selectedKey = useMemo<string | undefined>(() => {
    if (!visibleItems.length) return undefined;
    const sorted = [...visibleItems].sort((a, b) => b.path.length - a.path.length);
    for (const item of sorted) {
      if (
        location.pathname === item.path ||
        location.pathname.startsWith(item.path + '/')
      ) {
        return item.key;
      }
    }
    return undefined;
  }, [location.pathname, visibleItems]);

  // 面包屑：左 = 固定「看板」 + 当前项
  const breadcrumbItems = [
    { title: '看板', href: '/dashboard' },
    { title: lastBreadcrumb(location.pathname) },
  ];

  // 登出：清状态 + 跳 /login
  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  // 用户下拉菜单（昵称 / 角色只读 + 登出）
  const userMenuItems = [
    {
      key: 'nickname',
      label: (
        <span data-testid="dropdown-nickname">
          {user?.display_name ?? user?.username ?? '未登录'}
        </span>
      ),
      disabled: true,
    },
    {
      key: 'role',
      label: (
        <span data-testid="dropdown-role">角色：{role ?? '无'}</span>
      ),
      disabled: true,
    },
    { type: 'divider' as const },
    {
      key: 'logout',
      label: '登出',
      icon: <LogoutOutlined />,
      onClick: handleLogout,
      'data-testid': 'menu-logout',
    },
  ];

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider width={220} theme="dark" data-testid="admin-sider">
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
          data-testid="brand"
        >
          Doctors Admin
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={selectedKey ? [selectedKey] : []}
          items={menuItems}
          onClick={({ key }) => {
            const item = MENU_ITEMS.find((it) => it.key === key);
            if (item) navigate(item.path);
          }}
          data-testid="admin-menu"
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
          data-testid="admin-header"
        >
          <Breadcrumb items={breadcrumbItems} data-testid="admin-breadcrumb" />
          <Space size="middle">
            <TraceId traceId={null} />
            <Dropdown
              menu={{ items: userMenuItems }}
              placement="bottomRight"
              trigger={['click']}
            >
              <Space
                style={{ cursor: 'pointer' }}
                data-testid="user-dropdown-trigger"
              >
                <Avatar
                  size="small"
                  icon={<UserOutlined />}
                  data-testid="user-avatar"
                />
                <span data-testid="user-nickname">
                  {user?.display_name ?? user?.username ?? '未登录'}
                </span>
              </Space>
            </Dropdown>
          </Space>
        </Header>
        <Content
          style={{
            margin: 16,
            padding: 24,
            background: colorBgContainer,
            borderRadius: borderRadiusLG,
            minHeight: 280,
          }}
          data-testid="admin-content"
        >
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
}