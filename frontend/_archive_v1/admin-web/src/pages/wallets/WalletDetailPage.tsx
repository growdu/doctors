/**
 * WalletDetailPage：admin-web 钱包主体详情页（升级版）。
 *
 * 范围：
 *   - 顶部：主体信息（id / 类型 / 名称 / 余额 / 冻结金额）；
 *   - 中部：余额 + 冻结 2 个 Statistic 卡片；
 *   - 底部：最近 10 条流水列表（来自 subject.recent_transactions）。
 *
 * 数据流：
 *   - detail: GET /api/v1/admin/wallets/:id
 *
 * 对应 spec：2026-09-24-admin-web-design.md §Task 14
 */
import { useNavigate, useParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import {
  Alert,
  Button,
  Card,
  Col,
  Descriptions,
  Row,
  Spin,
  Statistic,
  Table,
  Tag,
  Typography,
} from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { PageHeader } from '@/components/PageHeader';
import {
  fetchWalletDetail,
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

export default function WalletDetailPage() {
  const { id = '' } = useParams<{ id: string }>();
  const numericId = Number(id);
  const navigate = useNavigate();

  const {
    data: subject,
    isLoading,
    isError,
    error,
  } = useQuery({
    queryKey: walletQueryKeys.detail(numericId),
    queryFn: () => fetchWalletDetail(numericId),
    enabled: Number.isFinite(numericId) && numericId > 0,
  });

  if (isLoading) {
    return (
      <div data-testid="wallet-detail-loading">
        <PageHeader
          title={`钱包详情 #${id}`}
          subtitle="主体信息 + 余额 + 流水明细"
        />
        <Spin />
      </div>
    );
  }
  if (isError || !subject) {
    return (
      <div data-testid="wallet-detail-page">
        <PageHeader
          title={`钱包详情 #${id}`}
          subtitle="主体信息 + 余额 + 流水明细"
        />
        <Alert
          type="error"
          showIcon
          message="加载失败"
          description={(error as Error)?.message ?? '主体不存在'}
        />
      </div>
    );
  }

  const txColumns: ColumnsType<WalletTxItem> = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 80,
      render: (v: number) => <Tag>#{v}</Tag>,
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
  ];

  return (
    <div data-testid="wallet-detail-page">
      <PageHeader
        title={`钱包详情 #${subject.id}`}
        subtitle={`${SUBJECT_LABEL[subject.subject_type] ?? subject.subject_type} · ${subject.subject_name}`}
        extra={
          <Button
            icon={<ArrowLeftOutlined />}
            onClick={() => navigate('/wallets')}
            data-testid="btn-back"
          >
            返回流水列表
          </Button>
        }
      />

      {/* 余额 / 冻结 卡片 */}
      <Row gutter={16}>
        <Col xs={24} sm={12}>
          <Card data-testid="card-balance">
            <Statistic
              title="可用余额"
              value={subject.balance}
              precision={2}
              valueStyle={{ color: '#1677ff' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12}>
          <Card data-testid="card-frozen">
            <Statistic
              title="冻结金额"
              value={subject.frozen}
              precision={2}
              valueStyle={{ color: '#fa8c16' }}
            />
          </Card>
        </Col>
      </Row>

      {/* 主体信息 */}
      <Card style={{ marginTop: 16 }} title="主体信息">
        <Descriptions column={2} bordered size="small">
          <Descriptions.Item label="主体 ID">#{subject.id}</Descriptions.Item>
          <Descriptions.Item label="主体类型">
            <Tag color={subject.subject_type === 'patient' ? 'blue' : 'purple'}>
              {SUBJECT_LABEL[subject.subject_type] ?? subject.subject_type}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label="名称" span={2}>
            {subject.subject_name}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      {/* 最近流水 */}
      <Card style={{ marginTop: 16 }} title="最近流水（最多 10 条）">
        <Table<WalletTxItem>
          rowKey="id"
          dataSource={subject.recent_transactions}
          columns={txColumns}
          pagination={false}
          size="middle"
          data-testid="recent-tx-table"
        />
      </Card>
    </div>
  );
}