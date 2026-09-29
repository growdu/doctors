/**
 * FinancePage：admin-web 财务概览页（升级版）。
 *
 * 功能：
 *   - 4 个核心 Statistic 卡片（GMV / 退款额 / 净收入 / 提现汇总）；
 *   - 范围切换（today / week / month）；
 *   - 每日 GMV 曲线 + 渠道拆分表（mock）。
 *
 * 数据流：
 *   - overview: fetchFinanceOverview(range) （当前 mock 由前端拼装）
 *
 * 对应 spec：2026-09-24-admin-web-design.md §Task 23
 */
import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  Card,
  Col,
  DatePicker,
  Row,
  Select,
  Space,
  Statistic,
  Table,
  Tag,
} from 'antd';
import {
  DollarOutlined,
  RollbackOutlined,
  RiseOutlined,
  BankOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { PageHeader } from '@/components/PageHeader';
import {
  fetchFinanceOverview,
  financeQueryKeys,
  type FinanceChannel,
  type FinanceRange,
} from '@/api/admin/finance';

const RANGE_LABEL: Record<FinanceRange, string> = {
  today: '今日',
  week: '近 7 日',
  month: '近 30 日',
};

export default function FinancePage() {
  const [range, setRange] = useState<FinanceRange>('today');

  const { data, isLoading } = useQuery({
    queryKey: financeQueryKeys.overview(range),
    queryFn: () => fetchFinanceOverview(range),
    refetchOnWindowFocus: false,
  });

  const channelColumns: ColumnsType<FinanceChannel> = [
    {
      title: '渠道',
      dataIndex: 'name',
      render: (v: string) => <Tag color="blue">{v}</Tag>,
    },
    {
      title: '笔数',
      dataIndex: 'count',
      align: 'right',
      render: (v: number) => <Tag>{v}</Tag>,
    },
    {
      title: '金额',
      dataIndex: 'amount',
      align: 'right',
      render: (v: number) => (
        <span style={{ color: '#1677ff' }}>¥{v.toLocaleString()}</span>
      ),
    },
  ];

  const dailyColumns: ColumnsType<{ date: string; amount: number }> = [
    {
      title: '日期',
      dataIndex: 'date',
      render: (v: string) => dayjs(v).format('MM-DD'),
    },
    {
      title: 'GMV',
      dataIndex: 'amount',
      align: 'right',
      render: (v: number) => (
        <span style={{ color: '#52c41a' }}>¥{v.toLocaleString()}</span>
      ),
    },
  ];

  return (
    <div data-testid="finance-page">
      <PageHeader
        title="财务概览"
        subtitle="GMV / 退款 / 净收入 / 提现汇总 + 渠道拆分"
        extra={
          <Space>
            <Select
              value={range}
              onChange={setRange}
              data-testid="range-select"
              style={{ width: 140 }}
              options={[
                { value: 'today', label: RANGE_LABEL.today },
                { value: 'week', label: RANGE_LABEL.week },
                { value: 'month', label: RANGE_LABEL.month },
              ]}
            />
            <DatePicker.RangePicker disabled value={[dayjs(), dayjs()]} />
          </Space>
        }
      />

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} md={6}>
          <Card loading={isLoading} data-testid="card-gmv">
            <Statistic
              title="GMV"
              value={data?.gmv ?? 0}
              prefix={<DollarOutlined />}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card loading={isLoading} data-testid="card-refund">
            <Statistic
              title="退款额"
              value={data?.refund_amount ?? 0}
              prefix={<RollbackOutlined />}
              valueStyle={{ color: '#fa8c16' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card loading={isLoading} data-testid="card-net">
            <Statistic
              title="净收入"
              value={data?.net_revenue ?? 0}
              prefix={<RiseOutlined />}
              valueStyle={{ color: '#1677ff' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card loading={isLoading} data-testid="card-withdraw">
            <Statistic
              title="提现汇总"
              value={data?.withdraw_amount ?? 0}
              prefix={<BankOutlined />}
              valueStyle={{ color: '#722ed1' }}
            />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} md={14}>
          <Card title="每日 GMV 走势" loading={isLoading}>
            <Table
              rowKey="date"
              dataSource={data?.daily_gmv ?? []}
              columns={dailyColumns}
              pagination={false}
              size="middle"
              data-testid="daily-table"
            />
          </Card>
        </Col>
        <Col xs={24} md={10}>
          <Card title="渠道拆分" loading={isLoading}>
            <Table
              rowKey="name"
              dataSource={data?.channels ?? []}
              columns={channelColumns}
              pagination={false}
              size="middle"
              data-testid="channels-table"
            />
          </Card>
        </Col>
      </Row>
    </div>
  );
}