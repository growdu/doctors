/**
 * ReviewsPage 测试：
 *   - 加载列表 + 行渲染；
 *   - 评分 / 陪诊师 ID 筛选触发 fetchReviews；
 *   - 超级管理员显示审核 + 回复 + 隐藏按钮；
 *   - viewer 角色不显示这些按钮（隐藏不渲染 / 隐藏按钮）；
 *   - 审核弹 Modal 选 pass → 调 auditReview；
 *   - 回复弹 Modal → 调 replyReview。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { useAuthStore } from '@/stores/authStore';

const mocks = vi.hoisted(() => ({
  fetchReviews: vi.fn(),
  auditReview: vi.fn(),
  replyReview: vi.fn(),
  message: { success: vi.fn(), error: vi.fn() },
}));

vi.mock('@/api/admin/reviews', () => ({
  fetchReviews: mocks.fetchReviews,
  auditReview: mocks.auditReview,
  replyReview: mocks.replyReview,
  reviewQueryKeys: {
    list: (params?: { escort_id?: number; min_rating?: number }) =>
      ['reviews', params?.escort_id ?? 'all', params?.min_rating ?? 0] as const,
  },
}));

vi.mock('antd', async (orig) => {
  const actual = await orig<typeof import('antd')>();
  return {
    ...actual,
    message: mocks.message,
  };
});

import ReviewsPage from './ReviewsPage';

const FIXTURES = [
  {
    id: 9001,
    order_id: 1003,
    escort_id: 1003,
    escort_name: '赵陪诊',
    rating: 5,
    content: '服务专业',
    created_at: '2026-09-22T13:00:00Z',
    audit_result: null,
    audit_reason: null,
    admin_reply: null,
  },
  {
    id: 9002,
    order_id: 1004,
    escort_id: 1003,
    escort_name: '赵陪诊',
    rating: 4,
    content: '不错',
    created_at: '2026-09-23T14:00:00Z',
    audit_result: null,
    audit_reason: null,
    admin_reply: null,
  },
];

function setupAuth(
  role: 'super_admin' | 'audit_admin' | 'cs' | 'viewer' = 'super_admin',
) {
  useAuthStore.setState({
    token: 'mock.token',
    user: { id: 1, username: 'admin', display_name: '管理员', role, avatar_url: null },
    role,
    isAuthed: true,
  } as never);
}

function renderPage(
  initialRole: 'super_admin' | 'audit_admin' | 'cs' | 'viewer' = 'super_admin',
) {
  setupAuth(initialRole);
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/reviews']}>
        <ReviewsPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe('ReviewsPage', () => {
  beforeEach(() => {
    mocks.fetchReviews.mockReset();
    mocks.auditReview.mockReset();
    mocks.replyReview.mockReset();
    mocks.message.success.mockReset();
    mocks.message.error.mockReset();
  });

  it('加载列表 + 行渲染', async () => {
    mocks.fetchReviews.mockResolvedValue({
      data: FIXTURES,
      total: FIXTURES.length,
    });
    renderPage();
    await waitFor(() =>
      expect(screen.getByTestId('reviews-table')).toBeInTheDocument(),
    );
    expect(screen.getByText('服务专业')).toBeInTheDocument();
    expect(screen.getByTestId('row-rating-9001')).toHaveTextContent('★ 5');
    expect(screen.getByTestId('row-audit-9001')).toHaveTextContent('待审');
  });

  it('rating 筛选触发 fetchReviews 带 min_rating=5', async () => {
    const user = userEvent.setup();
    mocks.fetchReviews.mockResolvedValue({ data: [FIXTURES[0]], total: 1 });
    renderPage();
    await waitFor(() => expect(mocks.fetchReviews).toHaveBeenCalled());
    await user.type(screen.getByTestId('rating-filter'), '5');
    await waitFor(() => {
      expect(mocks.fetchReviews).toHaveBeenCalledWith(
        expect.objectContaining({ min_rating: 5 }),
      );
    });
  });

  it('escort_id 筛选触发 fetchReviews 带 escort_id=1003', async () => {
    const user = userEvent.setup();
    mocks.fetchReviews.mockResolvedValue({
      data: [FIXTURES[0]],
      total: 1,
    });
    renderPage();
    await waitFor(() => expect(mocks.fetchReviews).toHaveBeenCalled());
    const input = screen.getByTestId('escort-id-input') as HTMLInputElement;
    fireEvent.change(input, { target: { value: '1003' } });
    await waitFor(() => {
      expect(mocks.fetchReviews).toHaveBeenCalledWith(
        expect.objectContaining({ escort_id: 1003 }),
      );
    });
  });

  it('super_admin 显示审核 / 回复 / 隐藏按钮', async () => {
    mocks.fetchReviews.mockResolvedValue({
      data: FIXTURES,
      total: FIXTURES.length,
    });
    renderPage('super_admin');
    await waitFor(() =>
      expect(screen.getByTestId('btn-audit-9001')).toBeInTheDocument(),
    );
    expect(screen.getByTestId('btn-reply-9001')).toBeInTheDocument();
    expect(screen.getByTestId('btn-hide-9001')).toBeInTheDocument();
  });

  it('viewer 不显示审核 / 回复 / 隐藏按钮', async () => {
    mocks.fetchReviews.mockResolvedValue({
      data: FIXTURES,
      total: FIXTURES.length,
    });
    renderPage('viewer');
    await waitFor(() =>
      expect(screen.getByTestId('reviews-table')).toBeInTheDocument(),
    );
    expect(screen.queryByTestId('btn-audit-9001')).not.toBeInTheDocument();
    expect(screen.queryByTestId('btn-reply-9001')).not.toBeInTheDocument();
    // 隐藏按钮通过 display:none 隐藏 → 仍存在但不可见
    const hideBtn = screen.getByTestId('btn-hide-9001');
    expect(hideBtn).toBeInTheDocument();
    expect(hideBtn).toBeDisabled();
  });

  it('审核弹 Modal → 选 pass → 调 auditReview', async () => {
    const user = userEvent.setup();
    mocks.fetchReviews.mockResolvedValue({
      data: FIXTURES,
      total: FIXTURES.length,
    });
    mocks.auditReview.mockResolvedValue({
      ...FIXTURES[0],
      audit_result: 'pass',
    });
    renderPage('super_admin');
    await waitFor(() =>
      expect(screen.getByTestId('btn-audit-9001')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-audit-9001'));
    // Modal 出现 + result radio 默认 pass + 直接提交
    await waitFor(() =>
      expect(screen.getByTestId('audit-result-radio')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-audit-confirm'));
    await waitFor(() => {
      expect(mocks.auditReview).toHaveBeenCalledWith(
        9001,
        expect.objectContaining({ result: 'pass' }),
      );
      expect(mocks.message.success).toHaveBeenCalledWith(
        expect.stringContaining('评价 #9001 已通过'),
      );
    });
  });

  it('回复弹 Modal → 输入回复内容 → 调 replyReview', async () => {
    const user = userEvent.setup();
    mocks.fetchReviews.mockResolvedValue({
      data: FIXTURES,
      total: FIXTURES.length,
    });
    mocks.replyReview.mockResolvedValue({
      ...FIXTURES[0],
      admin_reply: '感谢您的评价',
    });
    renderPage('super_admin');
    await waitFor(() =>
      expect(screen.getByTestId('btn-reply-9001')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-reply-9001'));
    await waitFor(() =>
      expect(screen.getByTestId('reply-content-input')).toBeInTheDocument(),
    );
    await user.type(
      screen.getByTestId('reply-content-input'),
      '感谢您的评价',
    );
    await user.click(screen.getByTestId('btn-reply-confirm'));
    await waitFor(() => {
      expect(mocks.replyReview).toHaveBeenCalledWith(9001, '感谢您的评价');
      expect(mocks.message.success).toHaveBeenCalledWith(
        expect.stringContaining('已回复评价 #9001'),
      );
    });
  });
});
