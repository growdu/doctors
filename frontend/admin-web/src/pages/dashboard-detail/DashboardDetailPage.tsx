/**
 * DashboardDetailPage：admin-web 数据看板明细下钻页（升级版）。
 *
 * 路由参数：
 *   /dashboard-detail/:id
 *   id 取值（mock 阶段）：
 *     - orders_pending      → 待处理订单明细
 *     - refunds_pending     → 待审退款明细
 *     - escorts_pending     → 待审核陪诊师明细
 *     - sos_open            → 未关闭 SOS 明细
 *
 * 数据流：
 *   - fetchDashboardDetail(id)（mock 由前端组装，依赖 seed fixtures）
 *
 * 对应 spec：2026-09-24-admin-web-design.md §Task 28
 */
import { useMemo } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import {
  Alert,
  Button,
  Card,
  Empty,
  Space,
  Spin,
  Table,
  Tag,
} from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { PageHeader } from '@/components/PageHeader';
import {
  dashboardDetailQueryKeys,
  fetchDashboardDetail,
  type DashboardDetailRow,
  type DashboardDetailType,
} from '@/api/admin/dashboard_detail';

const VALID_TYPES: DashboardDetailType[] = [
  'orders_pending',
  'refunds_pending',
  'escorts_pending',
  'sos_open',
];

const TYPE_LABEL: Record<DashboardDetailType, string> = {
  orders_pending: '待处理订单',
  refunds_pending: '待审退款',
  escorts_pending: '待审核陪诊师',
  sos_open: '未关闭 SOS',
};

export default function DashboardDetailPage() {
  const { id = '' } = useParams<{ id: string }>();
  const navigate = useNavigate();

  const isValid = VALID_TYPES.includes(id as DashboardDetailType);

  // mock 数据组装（同步，不走 useQuery）
  const payload = useMemo(() => {
    if (!isValid) return null;
    return fetchDashboardDetail(id as DashboardDetailType);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id, isValid]);

  if (!isValid) {
    return (
      <div data-testid="dashboard-detail-page">
        <PageHeader
          title="看板明细"
          subtitle={`未知类型：${id || '—'}`}
          extra={
            <Button
              icon={<ArrowLeftOutlined />}
              onClick={() => navigate('/dashboard')}
              data-testid="btn-back"
            >
              返回看板
            </Button>
          }
        />
        <Alert
          type="error"
          showIcon
          message="未知下钻类型"
          description={`支持的类型：${VALID_TYPES.join(' / ')}`}
        />
      </div>
    );
  }

  const columns: ColumnsType<DashboardDetailRow> = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 80,
      render: (v: number) => <Tag>#{v}</Tag>,
    },
    { title: '标题', dataIndex: 'title', width: 260 },
    { title: '详情', dataIndex: 'subtitle', width: 280 },
    {
      title: '状态',
      dataIndex: 'status',
      width: 130,
      render: (v: string) => <Tag color="blue">{v}</Tag>,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      width: 170,
      render: (v: string) => dayjs(v).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: '操作',
      width: 100,
      fixed: 'right',
      render: (_: unknown, r: DashboardDetailRow) => (
        <Button
          size="small"
          onClick={() => navigate(r.href)}
          data-testid={`btn-jump-${r.id}`}
        >
          跳转
        </Button>
      ),
    },
  ];

  return (
    <div data-testid="dashboard-detail-page">
      <PageHeader
        title={payload?.label ?? TYPE_LABEL[id as DashboardDetailType]}
        subtitle={`下钻类型：${id}`}
        extra={
          <Button
            icon={<ArrowLeftOutlined />}
            onClick={() => navigate('/dashboard')}
            data-testid="btn-back"
          >
            返回看板
          </Button>
        }
      />

      {!payload ? (
        <Spin />
      ) : payload.rows.length === 0 ? (
        <Card>
          <Empty description="该类型当前没有明细" />
        </Card>
      ) : (
        <Card>
          <Space style={{ marginBottom: 12 }}>
            <span style={{ color: '#999' }}>
              共 {payload.rows.length} 条 · queryKey：
              {JSON.stringify(dashboardDetailQueryKeys.byType(payload.type))}
            </span>
          </Space>
          <Table<DashboardDetailRow>
            rowKey="id"
            dataSource={payload.rows}
            columns={columns}
            pagination={{ pageSize: 20 }}
            size="middle"
            data-testid="dashboard-detail-table"
          />
        </Card>
      )}
    </div>
  );
}