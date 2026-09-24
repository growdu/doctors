/**
 * OrderListPage：admin-web 订单列表页（v2 选人模式适配）。
 *
 * v2 增量（基于原 v1 列表雏形）：
 *   - 状态筛选新增 2 选项：selecting_escort / escort_pending_acceptance；
 *   - 列新增 2 列：已选陪诊师 ID（selected_escort_id）+ 确认截止时间
 *     （escort_pending_expire_at，仅 escort_pending_acceptance 时非空）；
 *   - 订单号变可点击 <a>，点击跳详情页 /orders/:id；
 *   - 「详情」按钮保留作为辅助入口。
 *
 * 实现要点：
 *   - 使用 useSearchParams 读 URL `?status=` 实现「看板卡片跳转 → 列表自动过滤」；
 *   - 使用 TanStack Query 拉数据，5s 轮询（polling）；
 *   - 使用 useNavigate 实现订单号 / 详情按钮跳转。
 *
 * 对应 spec：2026-09-24-order-matching-redesign.md §4 + §5
 *           2026-09-24-admin-web-setup.md §Task 1
 */
import { useMemo } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import {
  Table,
  Tag,
  Button,
  Card,
  Select,
  Space,
  Typography,
  Input,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { StatusBadge } from '@/components/StatusBadge';
import { fetchOrders, orderQueryKeys } from '@/api/admin/orders';
import type { OrderListItem, OrderStatus } from '@/types/generated';

const { Title } = Typography;

// v2 新增 2 状态枚举（与 StatusBadge 对齐）
const ORDER_STATUS_OPTIONS: { value: OrderStatus | ''; label: string }[] = [
  { value: '', label: '全部' },
  { value: 'created', label: '已创建' },
  { value: 'paid', label: '已支付' },
  { value: 'matching', label: '匹配中' },
  // v2 新增 ↓
  { value: 'selecting_escort', label: '待患者选人' },
  { value: 'escort_pending_acceptance', label: '待陪诊师确认' },
  // v2 新增 ↑
  { value: 'accepted', label: '已接单' },
  { value: 'in_service', label: '服务中' },
  { value: 'completed', label: '已完成' },
  { value: 'refunding', label: '退款中' },
  { value: 'refunded', label: '已退款' },
  { value: 'disputed', label: '申诉中' },
  { value: 'closed', label: '已关闭' },
  { value: 'canceled', label: '已取消' },
];

export default function OrderListPage() {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const statusFilter = searchParams.get('status') ?? '';

  const { data, isLoading, isFetching } = useQuery({
    queryKey: orderQueryKeys.list({ status: statusFilter || undefined }),
    queryFn: () =>
      fetchOrders({ status: statusFilter || undefined, page: 1, page_size: 50 }),
    refetchInterval: 5000,
    refetchOnWindowFocus: false,
  });

  const orders: OrderListItem[] = useMemo(() => data?.data ?? [], [data?.data]);

  const columns: ColumnsType<OrderListItem> = [
    {
      title: '订单号',
      dataIndex: 'id',
      width: 90,
      fixed: 'left',
      render: (v: number) => (
        <a
          onClick={() => navigate(`/orders/${v}`)}
          data-testid={`order-link-${v}`}
        >
          {v}
        </a>
      ),
    },
    { title: '医院', dataIndex: 'hospital_name', width: 160 },
    {
      title: '金额',
      dataIndex: 'final_amount',
      width: 100,
      render: (v: number) => `¥${v}`,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 140,
      render: (_: unknown, r: OrderListItem) => (
        <StatusBadge status={r.status} testId={`row-status-${r.id}`} />
      ),
    },
    // v2 新增列：已选陪诊师
    {
      title: '已选陪诊师',
      dataIndex: 'selected_escort_id',
      width: 120,
      render: (v: number | null) =>
        v != null ? <Tag color="blue">#{v}</Tag> : <Tag>未选</Tag>,
    },
    // v2 新增列：escort_pending_acceptance 时显示截止时间
    {
      title: '确认截止',
      dataIndex: 'escort_pending_expire_at',
      width: 170,
      render: (v: string | null) =>
        v ? dayjs(v).format('YYYY-MM-DD HH:mm:ss') : '—',
    },
    {
      title: '下单时间',
      dataIndex: 'created_at',
      width: 160,
      render: (v: string) => dayjs(v).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: '操作',
      width: 120,
      fixed: 'right',
      render: (_: unknown, r: OrderListItem) => (
        <Button type="link" onClick={() => navigate(`/orders/${r.id}`)}>
          详情
        </Button>
      ),
    },
  ];

  const handleStatusChange = (val: OrderStatus | '') => {
    const next = new URLSearchParams(searchParams);
    if (val) next.set('status', val);
    else next.delete('status');
    setSearchParams(next);
  };

  return (
    <div data-testid="order-list-page">
      <Title level={3}>订单管理</Title>

      <Card style={{ marginBottom: 16 }} size="small">
        <Space wrap>
          <span>状态：</span>
          <Select
            value={statusFilter || ''}
            style={{ width: 200 }}
            options={ORDER_STATUS_OPTIONS}
            onChange={handleStatusChange}
            data-testid="order-status-filter"
          />
          <Input.Search
            placeholder="搜索（占位）"
            style={{ width: 200 }}
            disabled
          />
          <span style={{ color: '#999' }}>
            {isFetching ? '刷新中…' : `共 ${data?.total ?? 0} 条`}
          </span>
        </Space>
      </Card>

      <Table<OrderListItem>
        rowKey="id"
        columns={columns}
        dataSource={orders}
        loading={isLoading}
        scroll={{ x: 1000 }}
        pagination={{
          pageSize: 20,
          showSizeChanger: true,
          showTotal: (t) => `共 ${t} 条`,
        }}
      />
    </div>
  );
}