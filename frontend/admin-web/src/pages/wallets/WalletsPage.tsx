/**
 * WalletsPage：admin-web 钱包流水列表页（升级版）。
 *
 * 功能：
 *   - 钱包流水列表（按 subject_type / tx_type 过滤）；
 *   - 行操作：详情（跳 /wallets/:id 看主体 + 最近流水）。
 *
 * 数据流：
 *   - list: GET /api/v1/admin/wallets?type=&tx_type=
 *
 * 对应 spec：2026-09-24-admin-web-design.md §Task 14
 */
import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { Button, Select, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { PageHeader } from '@/components/PageHeader';
import { ProTable } from '@/components/ProTable';
import {
  fetchWalletTransactions,
  walletQueryKeys,
  type WalletTxItem,
  type WalletTxType,
} from '@/api/admin/wallets';

const { Text } = Typography;

const TX_LABEL: Record<WalletTxType, string> = {
  recharge: '充值',
  payment: '扣款',
  refund: '退款',
  withdraw: '提现',
};

const TX_COLOR: Record<WalletTxType, string> = {
  recharge: 'green',
  payment: 'blue',
  refund: 'orange',
  withdraw: 'red',
};

const SUBJECT_LABEL: Record<string, string> = {
  patient: '患者',
  escort: '陪诊师',
};

export default function WalletsPage() {
  const navigate = useNavigate();
  const [typeFilter, setTypeFilter] = useState<string | undefined>();
  const [txTypeFilter, setTxTypeFilter] = useState<string | undefined>();

  const { data, isLoading, isError, error } = useQuery({
    queryKey: walletQueryKeys.list({ type: typeFilter, tx_type: txTypeFilter }),
    queryFn: () =>
      fetchWalletTransactions({ type: typeFilter, tx_type: txTypeFilter }),
    refetchOnWindowFocus: false,
  });

  const items: WalletTxItem[] = data?.data ?? [];

  const columns: ColumnsType<WalletTxItem> = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 80,
      render: (v: number) => <Tag>#{v}</Tag>,
    },
    {
      title: '主体',
      dataIndex: 'subject_type',
      width: 100,
      render: (v: string, r: WalletTxItem) => (
        <Space size={4}>
          <Tag color={v === 'patient' ? 'blue' : 'purple'}>
            {SUBJECT_LABEL[v] ?? v}
          </Tag>
          <Text>#{r.subject_id}</Text>
        </Space>
      ),
    },
    {
      title: '类型',
      dataIndex: 'tx_type',
      width: 90,
      render: (v: WalletTxType) => (
        <Tag color={TX_COLOR[v]}>{TX_LABEL[v]}</Tag>
      ),
    },
    {
      title: '金额',
      dataIndex: 'amount',
      width: 120,
      align: 'right',
      render: (v: number) => (
        <Text type={v >= 0 ? 'success' : 'danger'}>
          {v >= 0 ? '+' : ''}
          {v.toFixed(2)}
        </Text>
      ),
    },
    {
      title: '余额（交易后）',
      dataIndex: 'balance_after',
      width: 130,
      align: 'right',
      render: (v: number) => <Text strong>{v.toFixed(2)}</Text>,
    },
    {
      title: '时间',
      dataIndex: 'created_at',
      width: 170,
      render: (v: string) => dayjs(v).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: '操作',
      width: 100,
      fixed: 'right',
      render: (_: unknown, r: WalletTxItem) => (
        <Button
          size="small"
          onClick={() => navigate(`/wallets/${r.subject_id}`)}
          data-testid={`btn-detail-${r.id}`}
        >
          主体详情
        </Button>
      ),
    },
  ];

  return (
    <div data-testid="wallets-page">
      <PageHeader
        title="钱包流水"
        subtitle="患者 / 陪诊师 钱包流水 + 类型筛选 + 主体详情"
      />

      <Space style={{ marginBottom: 16 }}>
        <Select
          placeholder="主体"
          allowClear
          style={{ width: 140 }}
          value={typeFilter}
          onChange={setTypeFilter}
          data-testid="filter-type"
          options={[
            { value: 'patient', label: '患者' },
            { value: 'escort', label: '陪诊师' },
          ]}
        />
        <Select
          placeholder="流水类型"
          allowClear
          style={{ width: 140 }}
          value={txTypeFilter}
          onChange={setTxTypeFilter}
          data-testid="filter-tx-type"
          options={[
            { value: 'recharge', label: TX_LABEL.recharge },
            { value: 'payment', label: TX_LABEL.payment },
            { value: 'refund', label: TX_LABEL.refund },
            { value: 'withdraw', label: TX_LABEL.withdraw },
          ]}
        />
        <span style={{ color: '#999' }}>共 {data?.total ?? 0} 条</span>
      </Space>

      <ProTable<WalletTxItem>
        testId="wallets-table"
        rowKey="id"
        columns={columns}
        dataSource={items}
        loading={isLoading}
        density="middle"
        scroll={{ x: 1000 }}
      />

      {isError && (
        <div data-testid="wallets-error" style={{ color: '#f5222d', marginTop: 8 }}>
          加载失败：{(error as Error)?.message ?? '未知错误'}
        </div>
      )}
    </div>
  );
}