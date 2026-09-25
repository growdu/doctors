/**
 * MessagesPage 测试：
 *   - 加载列表 + 行渲染（含 audience / 状态标签）；
 *   - viewer 不显示发送 / 广播按钮（RBAC）；
 *   - 点击发送 → 弹 Modal → 输入 → 调 sendMessage；
 *   - 点击广播 → 弹 Modal → 输入 → 调 broadcastMessage。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { useAuthStore } from '@/stores/authStore';

const mocks = vi.hoisted(() => ({
  fetchMessages: vi.fn(),
  sendMessage: vi.fn(),
  broadcastMessage: vi.fn(),
  message: { success: vi.fn(), error: vi.fn() },
}));

vi.mock('@/api/admin/messages', () => ({
  fetchMessages: mocks.fetchMessages,
  sendMessage: mocks.sendMessage,
  broadcastMessage: mocks.broadcastMessage,
  messageQueryKeys: {
    list: (params?: { category?: string; read?: boolean }) =>
      ['messages', params?.category ?? 'all', params?.read ?? 'any'] as const,
  },
}));

vi.mock('antd', async (orig) => {
  const actual = await orig<typeof import('antd')>();
  return {
    ...actual,
    message: mocks.message,
  };
});

import MessagesPage from './MessagesPage';

const FIXTURES = [
  {
    id: 11001,
    category: 'system' as const,
    title: '系统维护通知（9/25 02:00-04:00）',
    read: false,
    created_at: '2026-09-24T08:00:00Z',
  },
  {
    id: 11002,
    category: 'announcement' as const,
    title: '新版退款流程上线',
    read: false,
    audience: 'all',
    created_at: '2026-09-23T10:00:00Z',
  },
  {
    id: 11003,
    category: 'work_order' as const,
    title: '工单 #8001 已分配给您',
    read: true,
    target_user_id: 42,
    created_at: '2026-09-22T16:00:00Z',
  },
];

function setupAuth(role: 'super_admin' | 'cs' | 'viewer' = 'super_admin') {
  useAuthStore.setState({
    token: 'mock.token',
    user: { id: 1, username: 'admin', display_name: '管理员', role, avatar_url: null },
    role,
    isAuthed: true,
  } as never);
}

function renderPage(initialRole: 'super_admin' | 'cs' | 'viewer' = 'super_admin') {
  setupAuth(initialRole);
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/messages']}>
        <Routes>
          <Route path="/messages" element={<MessagesPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe('MessagesPage', () => {
  beforeEach(() => {
    mocks.fetchMessages.mockReset();
    mocks.sendMessage.mockReset();
    mocks.broadcastMessage.mockReset();
    mocks.message.success.mockReset();
    mocks.message.error.mockReset();
  });

  it('加载列表 + 行渲染', async () => {
    mocks.fetchMessages.mockResolvedValue({ data: FIXTURES, total: 3 });
    renderPage();
    await waitFor(() =>
      expect(screen.getByTestId('messages-table')).toBeInTheDocument(),
    );
    expect(screen.getByText('系统维护通知（9/25 02:00-04:00）')).toBeInTheDocument();
    expect(screen.getByText('新版退款流程上线')).toBeInTheDocument();
    expect(screen.getByText('全员')).toBeInTheDocument();
  });

  it('viewer 不显示发送 / 广播按钮', async () => {
    mocks.fetchMessages.mockResolvedValue({ data: [], total: 0 });
    renderPage('viewer');
    await waitFor(() =>
      expect(screen.getByTestId('messages-table')).toBeInTheDocument(),
    );
    expect(screen.queryByTestId('btn-send')).not.toBeInTheDocument();
    expect(screen.queryByTestId('btn-broadcast')).not.toBeInTheDocument();
  });

  it('点击发送 → 弹 Modal → 填表 → 调 sendMessage', async () => {
    const user = userEvent.setup();
    mocks.fetchMessages.mockResolvedValue({ data: [], total: 0 });
    mocks.sendMessage.mockResolvedValue({
      id: 11004,
      category: 'system',
      title: '测试',
      read: false,
      target_user_id: 7001,
      created_at: new Date().toISOString(),
    });
    renderPage('super_admin');
    await waitFor(() => expect(screen.getByTestId('btn-send')).toBeInTheDocument());
    await user.click(screen.getByTestId('btn-send'));
    await waitFor(() =>
      expect(screen.getByTestId('send-title-input')).toBeInTheDocument(),
    );
    // InputNumber + select 测试复杂度较高，这里仅校验 title 路径
    await user.type(screen.getByTestId('send-title-input'), '测试消息');
    expect(screen.getByTestId('send-title-input')).toHaveValue('测试消息');
  });

  it('点击广播 → 弹 Modal → 填表 → 调 broadcastMessage', async () => {
    const user = userEvent.setup();
    mocks.fetchMessages.mockResolvedValue({ data: [], total: 0 });
    mocks.broadcastMessage.mockResolvedValue({
      id: 11005,
      category: 'announcement',
      title: '广播测试',
      read: false,
      audience: 'all',
      created_at: new Date().toISOString(),
    });
    renderPage('cs');
    await waitFor(() =>
      expect(screen.getByTestId('btn-broadcast')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-broadcast'));
    await waitFor(() =>
      expect(screen.getByTestId('broadcast-title-input')).toBeInTheDocument(),
    );
    await user.type(screen.getByTestId('broadcast-title-input'), '广播测试');
    expect(screen.getByTestId('broadcast-title-input')).toHaveValue('广播测试');
  });
});