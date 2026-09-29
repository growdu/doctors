/**
 * DashboardPage：admin-web 数据看板（v2 选人模式适配）。
 *
 * v2 增量（基于原 v1 占位 4 卡片）：
 *   - 新增 2 个 Statistic 卡片：
 *       · 待患者选人（pending_selecting_escort，橙色 #fa8c16）
 *       · 待陪诊师确认（pending_escort_acceptance，蓝色 #1677ff）
 *   - 2 张卡片均可点击（Card hoverable），跳转到 /orders 带 status filter：
 *       · 点「待患者选人」 → /orders?status=selecting_escort
 *       · 点「待陪诊师确认」→ /orders?status=escort_pending_acceptance
 *
 * 设计要点：
 *   - 使用 useQuery 拉 OverviewReport，30s 轮询（refetchInterval）；
 *   - loading 状态展示 Spin 占位，避免「数据 undefined 时渲染 0」误导；
 *   - 6 个卡片用 Row gutter + Col span=8 分两行布局（4 + 2）。
 *
 * 对应 spec：2026-09-24-order-matching-redesign.md §5
 *           2026-09-24-admin-web-setup.md §Task 3
 */
import { useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { Card, Col, Row, Statistic, Spin, Typography } from 'antd';
import {
  TeamOutlined,
  ClockCircleOutlined,
  ShoppingOutlined,
  DollarOutlined,
  RollbackOutlined,
  UserSwitchOutlined,
} from '@ant-design/icons';
import { fetchOverview, reportsQueryKeys } from '@/api/admin/reports';
import type { OverviewReport } from '@/types/generated';

const { Title } = Typography;

// v2 配色规范：与 StatusBadge 对齐
const COLOR_SELECTING = '#fa8c16';   // 橙色 = selecting_escort
const COLOR_PENDING_ACCEPT = '#1677ff'; // 蓝色 = escort_pending_acceptance

export default function DashboardPage() {
  const navigate = useNavigate();
  const { data, isLoading } = useQuery<OverviewReport>({
    queryKey: reportsQueryKeys.overview,
    queryFn: fetchOverview,
    refetchInterval: 30_000,
    refetchOnWindowFocus: false,
  });

  if (isLoading || !data) {
    return (
      <div data-testid="dashboard-loading">
        <Spin />
      </div>
    );
  }

  const stats = data;

  return (
    <div data-testid="dashboard">
      <Title level={3}>数据看板</Title>

      <Row gutter={[16, 16]}>
        {/* ── v1 既有 4 个卡片 ─────────────────────────────────────── */}
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title="今日订单"
              value={stats.today_orders}
              prefix={<ShoppingOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title="今日 GMV"
              value={stats.today_gmv}
              prefix={<DollarOutlined />}
              precision={2}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title="待审核陪诊师"
              value={stats.pending_escorts}
              prefix={<TeamOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title="待审核退款"
              value={stats.pending_refunds}
              prefix={<RollbackOutlined />}
            />
          </Card>
        </Col>

        {/* ── v2 新增 2 个待确认卡片（可点击跳列表） ─────────────── */}
        <Col xs={24} sm={12} md={6}>
          <Card
            hoverable
            data-testid="card-selecting-escort"
            onClick={() => navigate('/orders?status=selecting_escort')}
          >
            <Statistic
              title="待患者选人"
              value={stats.pending_selecting_escort ?? 0}
              prefix={<UserSwitchOutlined />}
              valueStyle={{ color: COLOR_SELECTING }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card
            hoverable
            data-testid="card-escort-pending-acceptance"
            onClick={() => navigate('/orders?status=escort_pending_acceptance')}
          >
            <Statistic
              title="待陪诊师确认"
              value={stats.pending_escort_acceptance ?? 0}
              prefix={<ClockCircleOutlined />}
              valueStyle={{ color: COLOR_PENDING_ACCEPT }}
            />
          </Card>
        </Col>
      </Row>

      <Card style={{ marginTop: 16 }} type="inner">
        Dashboard 图表区（v1 后续接入 @ant-design/charts）
      </Card>
    </div>
  );
}