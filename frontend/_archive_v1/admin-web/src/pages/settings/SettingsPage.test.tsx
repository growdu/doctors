/**
 * SettingsPage 测试：
 *   - 加载时表单默认值 = DEFAULT_SETTINGS；
 *   - 点击保存 → 调 saveCategory（mock localStorage）；
 *   - 点击重置 → 回到默认值。
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Routes, Route } from 'react-router-dom';

const mocks = vi.hoisted(() => ({
  message: { success: vi.fn(), error: vi.fn() },
}));

vi.mock('antd', async (orig) => {
  const actual = await orig<typeof import('antd')>();
  // 仅替换 message
  return {
    ...actual,
    message: mocks.message,
  };
});

import SettingsPage from './SettingsPage';

function renderPage() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/settings']}>
        <Routes>
          <Route path="/settings" element={<SettingsPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe('SettingsPage', () => {
  beforeEach(() => {
    window.localStorage.clear();
    mocks.message.success.mockReset();
    mocks.message.error.mockReset();
  });
  afterEach(() => {
    window.localStorage.clear();
  });

  it('加载时基础设置表单默认值', async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getByTestId('settings-page')).toBeInTheDocument(),
    );
    expect(screen.getByTestId('general-site-name')).toHaveValue('Doctors Admin');
    expect(screen.getByTestId('general-phone')).toHaveValue('400-000-0000');
  });

  it('基础 Tab → 修改 site_name → 保存 → message.success', async () => {
    const user = userEvent.setup();
    renderPage();
    await waitFor(() =>
      expect(screen.getByTestId('general-site-name')).toBeInTheDocument(),
    );
    const input = screen.getByTestId('general-site-name');
    await user.clear(input);
    await user.type(input, '新站点名');
    await user.click(screen.getByTestId('btn-save-general'));
    await waitFor(() => {
      expect(mocks.message.success).toHaveBeenCalledWith('已保存基础设置');
    });
    // 校验 localStorage 已写入
    const stored = JSON.parse(
      window.localStorage.getItem('doctors-admin-settings') ?? '{}',
    );
    expect(stored.general?.site_name).toBe('新站点名');
  });

  it('短信 Tab → 修改签名 → 保存', async () => {
    const user = userEvent.setup();
    renderPage();
    await waitFor(() =>
      expect(screen.getByTestId('settings-tabs')).toBeInTheDocument(),
    );
    // 切到短信 Tab
    const smsTab = screen.getByRole('tab', { name: '短信' });
    await user.click(smsTab);
    await waitFor(() =>
      expect(screen.getByTestId('sms-signature')).toBeInTheDocument(),
    );
    await user.clear(screen.getByTestId('sms-signature'));
    await user.type(screen.getByTestId('sms-signature'), 'NewSig');
    await user.click(screen.getByTestId('btn-save-sms'));
    await waitFor(() => {
      expect(mocks.message.success).toHaveBeenCalledWith('已保存短信设置');
    });
  });

  it('点击重置 → 回到默认 + message.success', async () => {
    const user = userEvent.setup();
    renderPage();
    await waitFor(() =>
      expect(screen.getByTestId('general-site-name')).toBeInTheDocument(),
    );
    // 先改一下
    const input = screen.getByTestId('general-site-name');
    await user.clear(input);
    await user.type(input, '脏数据');
    // 点重置
    await user.click(screen.getByTestId('btn-reset'));
    await waitFor(() => {
      expect(screen.getByTestId('general-site-name')).toHaveValue('Doctors Admin');
      expect(mocks.message.success).toHaveBeenCalledWith('已重置为默认值');
    });
  });

  it('支付 / 推送 Tab 加载 + 切换开关', async () => {
    const user = userEvent.setup();
    renderPage();
    await waitFor(() =>
      expect(screen.getByTestId('settings-tabs')).toBeInTheDocument(),
    );
    await user.click(screen.getByRole('tab', { name: '支付' }));
    await waitFor(() =>
      expect(screen.getByTestId('payment-fee-rate')).toBeInTheDocument(),
    );
    await user.click(screen.getByRole('tab', { name: '推送' }));
    await waitFor(() =>
      expect(screen.getByTestId('push-provider')).toBeInTheDocument(),
    );
    expect(screen.getByTestId('push-quiet-start')).toBeInTheDocument();
    expect(screen.getByTestId('push-quiet-end')).toBeInTheDocument();
  });
});