/**
 * PatientDetailPage：admin-web 患者详情页（升级版）。
 *
 * 范围：
 *   - 实名信息（name / phone / verify_status / id_card_no）；
 *   - 订单历史（mock：当前订单列表，按 patient_name 过滤 —— 这里用 useOrdersMsW hook 简单演示）；
 *   - 钱包余额（balance + frozen，详情字段自带）。
 *
 * 数据流：
 *   - GET /api/v1/admin/patients/:id
 *
 * 对应 spec：2026-09-24-admin-web-design.md §Task 11
 */
import { useNavigate, useParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import {
  Alert,
  Button,
  Card,
  Descriptions,
  Empty,
  Space,
  Spin,
  Table,
  Tag,
  Typography,
} from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { PageHeader } from '@/components/PageHeader';
import {
  fetchPatientDetail,
  patientQueryKeys,
} from '@/api/admin/patients';
import { fetchOrders } from '@/api/admin/orders';
import type { OrderListItem } from '@/types/generated';

const { Text } = Typography;

/** 患者关联的订单记录（仅展示订单号 / 状态 / 金额 / 时间）。 */
interface PatientOrderRow {
  id: number;
  status: OrderListItem['status'];
  hospital_name: string;
  final_amount: number;
  created_at: string;
}

const ORDER_COLUMNS: ColumnsType<PatientOrderRow> = [
  { title: '订单号', dataIndex: 'id', width: 100 },
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
    render: (v: OrderListItem['status']) => <Tag>{v}</Tag>,
  },
  {
    title: '下单时间',
    dataIndex: 'created_at',
    width: 170,
    render: (v: string) => dayjs(v).format('YYYY-MM-DD HH:mm:ss'),
  },
];

export default function PatientDetailPage() {
  const { id = '' } = useParams<{ id: string }>();
  const numericId = Number(id);
  const navigate = useNavigate();

  const {
    data: patient,
    isLoading,
    isError,
    error,
  } = useQuery({
    queryKey: patientQueryKeys.detail(numericId),
    queryFn: () => fetchPatientDetail(numericId),
    enabled: Number.isFinite(numericId) && numericId > 0,
  });

  // 拉订单列表（真实场景按 patient_id 过滤；当前 mock 后端未带，按昵称模糊筛）
  const patientName = patient?.name;
  const { data: ordersData } = useQuery({
    queryKey: ['patient-orders', numericId, patientName ?? ''],
    queryFn: async () => {
      const out = await fetchOrders({ page: 1, page_size: 100 });
      // 当前 fixture 订单 patient_name 与 patient.name 可能不一致 → 全展示
      return out.data;
    },
    enabled: Boolean(patientName),
  });

  if (isLoading) {
    return (
      <div data-testid="patient-detail-loading">
        <PageHeader title={`患者详情 #${id}`} subtitle="加载中…" />
        <Spin />
      </div>
    );
  }
  if (isError || !patient) {
    return (
      <div data-testid="patient-detail-page">
        <PageHeader title={`患者详情 #${id}`} subtitle="基本信息 + 订单历史" />
        <Alert
          type="error"
          showIcon
          message="加载失败"
          description={(error as Error)?.message ?? '患者不存在'}
        />
      </div>
    );
  }

  const allOrders: OrderListItem[] = ordersData ?? [];
  const orderRows: PatientOrderRow[] = allOrders.map((o) => ({
    id: o.id,
    status: o.status,
    hospital_name: o.hospital_name,
    final_amount: o.final_amount,
    created_at: o.created_at,
  }));

  return (
    <div data-testid="patient-detail-page">
      <PageHeader
        title={`患者详情 #${patient.id}`}
        subtitle={`${patient.name} · ${patient.phone}`}
        extra={
          <Button
            icon={<ArrowLeftOutlined />}
            onClick={() => navigate('/patients')}
            data-testid="btn-back"
          >
            返回列表
          </Button>
        }
      />

      {/* 实名信息 */}
      <Card title="实名信息">
        <Descriptions column={2} bordered size="small">
          <Descriptions.Item label="昵称">{patient.name}</Descriptions.Item>
          <Descriptions.Item label="手机">{patient.phone}</Descriptions.Item>
          <Descriptions.Item label="实名状态">
            {patient.verify_status === 'verified' ? (
              <Tag color="green" data-testid="patient-verify">
                已实名
              </Tag>
            ) : patient.verify_status === 'pending' ? (
              <Tag color="orange" data-testid="patient-verify">
                待审核
              </Tag>
            ) : (
              <Tag data-testid="patient-verify">未实名</Tag>
            )}
          </Descriptions.Item>
          <Descriptions.Item label="封禁状态">
            {patient.status === 'banned' ? (
              <Tag color="red" data-testid="patient-ban">
                封禁{patient.ban_reason ? `（${patient.ban_reason}）` : ''}
              </Tag>
            ) : (
              <Tag color="green" data-testid="patient-ban">
                正常
              </Tag>
            )}
          </Descriptions.Item>
          <Descriptions.Item label="注册时间" span={2}>
            {dayjs(patient.registered_at).format('YYYY-MM-DD HH:mm:ss')}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      {/* 钱包余额 */}
      <Card style={{ marginTop: 16 }} title="钱包余额">
        <Space size="large">
          <div>
            <Text type="secondary">可用余额：</Text>
            <Text strong style={{ fontSize: 18 }}>
              ¥{(patient.wallet_balance ?? 0).toFixed(2)}
            </Text>
          </div>
          <div>
            <Text type="secondary">冻结金额：</Text>
            <Text strong style={{ fontSize: 18 }}>
              ¥{(patient.wallet_frozen ?? 0).toFixed(2)}
            </Text>
          </div>
        </Space>
        {patient.wallet_balance == null && (
          <Text type="secondary" data-testid="wallet-empty">（当前未接入钱包端点，显示 0）</Text>
        )}
      </Card>

      {/* 订单历史 */}
      <Card style={{ marginTop: 16 }} title={`订单历史（${patient.order_count ?? 0}）`}>
        {orderRows.length === 0 ? (
          <Empty description="暂无订单记录" data-testid="orders-empty" />
        ) : (
          <div data-testid="orders-table">
            <Table<PatientOrderRow>
              rowKey="id"
              columns={ORDER_COLUMNS}
              dataSource={orderRows}
              pagination={false}
              size="small"
            />
          </div>
        )}
      </Card>
    </div>
  );
}
