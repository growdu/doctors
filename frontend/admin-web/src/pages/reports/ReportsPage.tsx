/**
 * ReportsPage：admin-web 业务报表页（升级版）。
 *
 * 功能：
 *   - 6 个汇总 Statistic（订单数 / GMV / 退款数 / 退款额 / 净收入 / 平均评分）；
 *   - 时间区间选择（from / to）+ 维度筛选（hospital/package/escort）；
 *   - 明细表（rows）渲染。
 *
 * 数据流：
 *   - business: fetchBusinessReport({ from, to, dimension }) （当前 mock 由前端拼装）
 *
 * 对应 spec：2026-09-24-admin-web-design.md §Task 24
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
  ShoppingOutlined,
  DollarOutlined,
  RollbackOutlined,
  RiseOutlined,
  StarOutlined,
  OrderedListOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import dayjs, { type Dayjs } from 'dayjs';
import { PageHeader } from '@/components/PageHeader';
import {
  fetchBusinessReport,
  reportsQueryKeys,
  type BusinessReportRow,
} from '@/api/admin/reports';

const DIMENSION_LABEL: Record<BusinessReportRow['dimension'], string> = {
  hospital: '医院',
  package: '套餐',
  escort: '陪诊师',
};

export default function ReportsPage() {
  const [from, setFrom] = useState<Dayjs>(dayjs().subtract(7, 'day'));
  const [to, setTo] = useState<Dayjs>(dayjs());
  const [dimension, setDimension] = useState<BusinessReportRow['dimension']>('hospital');

  const { data, isLoading } = useQuery({
    queryKey: reportsQueryKeys.business({
      from: from.toISOString(),
      to: to.toISOString(),
    }),
    queryFn: () =>
      fetchBusinessReport({
        from: from.toISOString(),
        to: to.toISOString(),
        dimension,
      }),
    refetchOnWindowFocus: false,
  });

  const columns: ColumnsType<BusinessReportRow> = [
    {
      title: '维度',
      dataIndex: 'dimension',
      width: 100,
      render: (v: BusinessReportRow['dimension']) => (
        <Tag color="blue">{DIMENSION_LABEL[v]}</Tag>
      ),
    },
    { title: '名称', dataIndex: 'label', width: 200 },
    {
      title: '订单数',
      dataIndex: 'order_count',
      align: 'right',
      render: (v: number) => <Tag color="blue">{v}</Tag>,
    },
    {
      title: 'GMV',
      dataIndex: 'gmv',
      align: 'right',
      render: (v: number) => (
        <span style={{ color: '#52c41a' }}>¥{v.toLocaleString()}</span>
      ),
    },
    {
      title: '退款数',
      dataIndex: 'refund_count',
      align: 'right',
      render: (v: number) => <Tag color="orange">{v}</Tag>,
    },
    {
      title: '退款额',
      dataIndex: 'refund_amount',
      align: 'right',
      render: (v: number) => (
        <span style={{ color: '#f5222d' }}>¥{v.toLocaleString()}</span>
      ),
    },
    {
      title: '净收入',
      align: 'right',
      render: (_: unknown, r: BusinessReportRow) => (
        <span style={{ color: '#1677ff', fontWeight: 600 }}>
          ¥{(r.gmv - r.refund_amount).toLocaleString()}
        </span>
      ),
    },
  ];

  return (
    <div data-testid="reports-page">
      <PageHeader
        title="业务报表"
        subtitle="自定义时间区间 + 多维度交叉 + 明细表"
        extra={
          <Space>
            <DatePicker.RangePicker
              value={[from, to]}
              onChange={(vals) => {
                if (vals && vals[0] && vals[1]) {
                  setFrom(vals[0]);
                  setTo(vals[1]);
                }
              }}
              data-testid="range-picker"
            />
            <Select
              value={dimension}
              onChange={setDimension}
              data-testid="dimension-select"
              style={{ width: 140 }}
              options={[
                { value: 'hospital', label: '按医院' },
                { value: 'package', label: '按套餐' },
                { value: 'escort', label: '按陪诊师' },
              ]}
            />
          </Space>
        }
      />

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} md={4}>
          <Card loading={isLoading} data-testid="stat-orders">
            <Statistic
              title="订单数"
              value={data?.totals.order_count ?? 0}
              prefix={<OrderedListOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={4}>
          <Card loading={isLoading} data-testid="stat-gmv">
            <Statistic
              title="GMV"
              value={data?.totals.gmv ?? 0}
              prefix={<DollarOutlined />}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={4}>
          <Card loading={isLoading} data-testid="stat-refund-count">
            <Statistic
              title="退款数"
              value={data?.totals.refund_count ?? 0}
              prefix={<ShoppingOutlined />}
              valueStyle={{ color: '#fa8c16' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={4}>
          <Card loading={isLoading} data-testid="stat-refund-amount">
            <Statistic
              title="退款额"
              value={data?.totals.refund_amount ?? 0}
              prefix={<RollbackOutlined />}
              valueStyle={{ color: '#f5222d' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={4}>
          <Card loading={isLoading} data-testid="stat-net">
            <Statistic
              title="净收入"
              value={data?.totals.net_revenue ?? 0}
              prefix={<RiseOutlined />}
              valueStyle={{ color: '#1677ff' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={4}>
          <Card loading={isLoading} data-testid="stat-rating">
            <Statistic
              title="平均评分"
              value={
                data && data.rows.length > 0
                  ? data.totals.avg_rating / data.rows.length
                  : 0
              }
              precision={2}
              prefix={<StarOutlined />}
            />
          </Card>
        </Col>
      </Row>

      <Card style={{ marginTop: 16 }} title="明细" loading={isLoading}>
        <Table<BusinessReportRow>
          rowKey={(r) => `${r.dimension}-${r.key}`}
          dataSource={data?.rows ?? []}
          columns={columns}
          pagination={false}
          size="middle"
          data-testid="rows-table"
        />
      </Card>
    </div>
  );
}