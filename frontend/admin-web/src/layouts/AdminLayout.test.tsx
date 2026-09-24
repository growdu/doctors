/**
 * AdminLayout 测试契约（待 vitest 启用后跑）：
 *   1. super_admin 角色 → 12 菜单项全渲染；
 *   2. role=viewer 隐藏 escorts/audit 与 settings；
 *   3. role=refund_admin 隐藏 escorts + escorts/audit；
 *   4. role=order_admin 隐藏 wallets 与 settings；
 *   5. 点击菜单 → useNavigate('/path') 被调；
 *   6. 当前路由高亮：selectedKeys 在 /escorts/audit 时陪诊师审核 li 加 ant-menu-item-selected。
 *
 * 策略：
 *   - vi.mock('@/stores/authStore') 注入 role/user/logout；
 *   - vi.mock('react-router-dom') 注入 useNavigate/useLocation（保留 MemoryRouter 等其他导出）；
 *   - antd Menu 在 jsdom 下原样可用，通过 data-testid 找菜单项，
 *     通过 .ant-menu-item-selected 类判断高亮。
 *
 * 注：当前骨架未装 vitest / @testing-library/react，本文件为契约样。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';

// ── Mock authStore ───────────────────────────────────────────────────
interface MockUser {
  id: number;
  username: string;
  display_name: string;
  role: string;
}

const mockAuth = vi.hoisted(() => ({
  user: null as MockUser | null,
  role: null as string | null,
  isAuthed: false,
  logout: vi.fn(),
}));

vi.mock('@/stores/authStore', () => ({
  useAuthStore: (selector: (s: typeof mockAuth) => unknown) => selector(mockAuth),
  ADMIN_ROLES: [
    'super_admin',
    'order_admin',
    'refund_admin',
    'audit_admin',
    'cs',
    'viewer',
  ],
}));

// ── Mock react-router-dom（注入 useNavigate / useLocation） ─────────
const navSpy = vi.hoisted(() => vi.fn());
const locationMock = vi.hoisted(() => ({ pathname: '/dashboard' }));

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual<typeof import('react-router-dom')>(
    'react-router-dom',
  );
  return {
    ...actual,
    useNavigate: () => navSpy,
    useLocation: () => ({
      pathname: locationMock.pathname,
      search: '',
      hash: '',
      state: null,
    }),
  };
});

// ── SUT ─────────────────────────────────────────────────────────────
import AdminLayout from './AdminLayout';

function setRole(role: string | null, displayName = '测试用户') {
  mockAuth.role = role;
  mockAuth.user = role
    ? { id: 1, username: 'test', display_name: displayName, role }
    : null;
  mockAuth.isAuthed = !!role;
}

describe('AdminLayout 菜单 + RBAC 过滤', () => {
  beforeEach(() => {
    navSpy.mockReset();
    mockAuth.logout.mockReset();
    locationMock.pathname = '/dashboard';
    setRole(null);
  });

  it('super_admin 渲染 12 菜单项', () => {
    setRole('super_admin');
    render(
      <MemoryRouter>
        <AdminLayout />
      </MemoryRouter>,
    );
    const items = screen.getAllByTestId(/^menu-item-/);
    expect(items.length).toBe(12);
    // 关键项可见
    expect(screen.getByTestId('menu-item-dashboard')).toBeInTheDocument();
    expect(screen.getByTestId('menu-item-orders')).toBeInTheDocument();
    expect(screen.getByTestId('menu-item-escorts')).toBeInTheDocument();
    expect(screen.getByTestId('menu-item-escorts-audit')).toBeInTheDocument();
    expect(screen.getByTestId('menu-item-settings')).toBeInTheDocument();
  });

  it('viewer 隐藏 escorts/audit 与 settings（保留陪诊师）', () => {
    setRole('viewer');
    render(
      <MemoryRouter>
        <AdminLayout />
      </MemoryRouter>,
    );
    // escorts/audit（仅 super_admin / audit_admin）：hidden
    expect(screen.queryByTestId('menu-item-escorts-audit')).toBeNull();
    // settings（仅 super_admin）：hidden
    expect(screen.queryByTestId('menu-item-settings')).toBeNull();
    // 陪诊师（super_admin / audit_admin / viewer）：可见
    expect(screen.getByTestId('menu-item-escorts')).toBeInTheDocument();
    // 钱包（super_admin / viewer）：可见
    expect(screen.getByTestId('menu-item-wallets')).toBeInTheDocument();
  });

  it('refund_admin 隐藏 escorts + escorts/audit（保留退款审批）', () => {
    setRole('refund_admin');
    render(
      <MemoryRouter>
        <AdminLayout />
      </MemoryRouter>,
    );
    // escorts（仅 super_admin / audit_admin / viewer）：hidden
    expect(screen.queryByTestId('menu-item-escorts')).toBeNull();
    // escorts/audit（仅 super_admin / audit_admin）：hidden
    expect(screen.queryByTestId('menu-item-escorts-audit')).toBeNull();
    // refunds（super_admin / refund_admin / cs / viewer）：可见
    expect(screen.getByTestId('menu-item-refunds')).toBeInTheDocument();
  });

  it('order_admin 隐藏 wallets 与 settings（保留工单）', () => {
    setRole('order_admin');
    render(
      <MemoryRouter>
        <AdminLayout />
      </MemoryRouter>,
    );
    // wallets（仅 super_admin / viewer）：hidden
    expect(screen.queryByTestId('menu-item-wallets')).toBeNull();
    // settings（仅 super_admin）：hidden
    expect(screen.queryByTestId('menu-item-settings')).toBeNull();
    // work-orders（super_admin / order_admin / refund_admin / audit_admin / cs）：可见
    expect(screen.getByTestId('menu-item-work-orders')).toBeInTheDocument();
  });

  it('点击菜单 → useNavigate 跳对应 path', () => {
    setRole('super_admin');
    render(
      <MemoryRouter>
        <AdminLayout />
      </MemoryRouter>,
    );
    // 点击「订单」
    fireEvent.click(screen.getByTestId('menu-item-orders'));
    expect(navSpy).toHaveBeenCalledWith('/orders');

    // 点击「陪诊师审核」
    fireEvent.click(screen.getByTestId('menu-item-escorts-audit'));
    expect(navSpy).toHaveBeenCalledWith('/escorts/audit');
  });

  it('当前路由高亮：/escorts/audit → 陪诊师审核 li 有 ant-menu-item-selected', () => {
    locationMock.pathname = '/escorts/audit';
    setRole('super_admin');
    const { container } = render(
      <MemoryRouter>
        <AdminLayout />
      </MemoryRouter>,
    );
    // antd 5 Menu 高亮项：li.ant-menu-item-selected
    const selectedItems = container.querySelectorAll('li.ant-menu-item-selected');
    expect(selectedItems.length).toBe(1);
    // 高亮项文案含「陪诊师审核」
    expect(selectedItems[0].textContent).toContain('陪诊师审核');
    // 同时 /escorts 应不高亮（最长前缀优先）
    const escortsLi = screen
      .getByTestId('menu-item-escorts')
      .closest('li.ant-menu-item');
    expect(escortsLi).not.toBeNull();
    expect(escortsLi!.classList.contains('ant-menu-item-selected')).toBe(false);
  });
});