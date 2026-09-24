import { Card, Col, Row, Statistic, Typography } from 'antd';

const { Title } = Typography;

/**
 * Dashboard 占位页（Task 3 雏形）：
 * 后续 Task（4+）会按 spec §4.4 接 useQuery + @ant-design/charts
 * （今日 GMV / 订单 / 退款率 / 待审核陪诊师 + 趋势图）。
 */
export default function DashboardPage() {
  return (
    <div>
      <Title level={3}>数据看板</Title>
      <Row gutter={16}>
        <Col span={6}>
          <Card>
            <Statistic title="今日 GMV" value={0} prefix="¥" />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="今日订单" value={0} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="退款率" value={0} suffix="%" />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="待审核陪诊师" value={0} />
          </Card>
        </Col>
      </Row>
      <Card style={{ marginTop: 16 }} type="inner">
        Dashboard 图表区（v1 后续接入 @ant-design/charts）
      </Card>
    </div>
  );
}