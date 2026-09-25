/**
 * SosPage：admin-web SOS 紧急报警列表页（升级版）。
 *
 * 功能：
 *   - 报警列表（按 status=open|closed 过滤）；
 *   - 行操作：处置（resolve）+ 升级（escalate，弹 Modal 选级别 + 原因）；
 *   - viewer 角色不显示操作按钮（RBAC）。
 *
 * 数据流：
 *   - list:      GET /api/v1/admin/sos?status=
 *   - resolve:   POST /api/v1/admin/sos/:id/resolve
 *   - escalate:  POST /api/v1/admin/sos/:id/escalate
 *
 * 对应 spec：2026-09-24-admin-web-design.md §Task 18
 */
import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Button,
  Form,
  Input,
  message,
  Modal,
  Select,
  Space,
  Tag,
} from 'antd';
import { CheckOutlined, ArrowUpOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { PageHeader } from '@/components/PageHeader';
import { ProTable } from '@/components/ProTable';
import {
  escalateSos,
  fetchSosAlerts,
  resolveSos,
  sosQueryKeys,
  type SosItem,
  type SosStatus,
} from '@/api/admin/sos';
import { useAuthStore } from '@/stores/authStore';

interface ResolveFormValues {
  note: string;
}

interface EscalateFormValues {
  level: string;
  reason: string;
}

const STATUS_LABEL: Record<SosStatus, string> = {
  open: '待处置',
  closed: '已关闭',
};

export default function SosPage() {
  const qc = useQueryClient();
  const role = useAuthStore((s) => s.role);
  /** 仅 super_admin / cs 可处置 / 升级 */
  const canResolve = role === 'super_admin' || role === 'cs';

  const [statusFilter, setStatusFilter] = useState<string | undefined>();

  const [resolveOpen, setResolveOpen] = useState(false);
  const [resolveTarget, setResolveTarget] = useState<SosItem | null>(null);
  const [resolveForm] = Form.useForm<ResolveFormValues>();

  const [escalateOpen, setEscalateOpen] = useState(false);
  const [escalateTarget, setEscalateTarget] = useState<SosItem | null>(null);
  const [escalateForm] = Form.useForm<EscalateFormValues>();

  // ── 查询 ──────────────────────────────────────────────────────────
  const { data, isLoading, isError, error } = useQuery({
    queryKey: sosQueryKeys.list({ status: statusFilter }),
    queryFn: () => fetchSosAlerts({ status: statusFilter }),
    refetchInterval: 30_000,
    refetchOnWindowFocus: false,
  });

  // ── 处置 ──────────────────────────────────────────────────────────
  const resolveMut = useMutation({
    mutationFn: ({ id, note }: { id: number; note: string }) =>
      resolveSos(id, note),
    onSuccess: (r) => {
      message.success(`已处置报警 #${r.id}`);
      qc.invalidateQueries({ queryKey: ['sos'] });
    },
    onError: (e: Error) => {
      message.error(`处置失败：${e.message}`);
    },
  });

  // ── 升级 ──────────────────────────────────────────────────────────
  const escalateMut = useMutation({
    mutationFn: ({
      id,
      level,
      reason,
    }: {
      id: number;
      level: string;
      reason: string;
    }) => escalateSos(id, { level, reason }),
    onSuccess: (r) => {
      message.success(`已升级报警 #${r.id} 至 ${r.escalate_level}`);
      qc.invalidateQueries({ queryKey: ['sos'] });
    },
    onError: (e: Error) => {
      message.error(`升级失败：${e.message}`);
    },
  });

  const items: SosItem[] = data?.data ?? [];

  const columns: ColumnsType<SosItem> = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 80,
      render: (v: number) => <Tag>#{v}</Tag>,
    },
    {
      title: '订单',
      dataIndex: 'order_id',
      width: 80,
      render: (v: number) => <Tag color="blue">#{v}</Tag>,
    },
    { title: '患者', dataIndex: 'patient_name', width: 100 },
    { title: '陪诊师', dataIndex: 'escort_name', width: 100 },
    { title: '位置', dataIndex: 'location', width: 180 },
    { title: '联系电话', dataIndex: 'contact', width: 130 },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (v: SosStatus) => (
        <Tag color={v === 'open' ? 'red' : 'green'}>
          {STATUS_LABEL[v]}
        </Tag>
      ),
    },
    {
      title: '升级级别',
      dataIndex: 'escalate_level',
      width: 100,
      render: (v: string | undefined) =>
        v ? <Tag color="orange">{v}</Tag> : <Tag color="default">—</Tag>,
    },
    {
      title: '报警时间',
      dataIndex: 'created_at',
      width: 170,
      render: (v: string) => dayjs(v).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: '操作',
      width: 200,
      fixed: 'right',
      render: (_: unknown, r: SosItem) => (
        <Space size="small">
          {canResolve && r.status === 'open' && (
            <Button
              size="small"
              type="primary"
              icon={<CheckOutlined />}
              onClick={() => {
                setResolveTarget(r);
                resolveForm.resetFields();
                setResolveOpen(true);
              }}
              data-testid={`btn-resolve-${r.id}`}
            >
              处置
            </Button>
          )}
          {canResolve && r.status === 'open' && (
            <Button
              size="small"
              danger
              icon={<ArrowUpOutlined />}
              onClick={() => {
                setEscalateTarget(r);
                escalateForm.resetFields();
                setEscalateOpen(true);
              }}
              data-testid={`btn-escalate-${r.id}`}
            >
              升级
            </Button>
          )}
        </Space>
      ),
    },
  ];

  // ── 提交 Modal ────────────────────────────────────────────────────
  const onSubmitResolve = async () => {
    try {
      const values = await resolveForm.validateFields();
      if (!resolveTarget) return;
      resolveMut.mutate(
        { id: resolveTarget.id, note: values.note },
        {
          onSuccess: () => {
            setResolveOpen(false);
            setResolveTarget(null);
          },
        },
      );
    } catch {
      // ignore
    }
  };

  const onSubmitEscalate = async () => {
    try {
      const values = await escalateForm.validateFields();
      if (!escalateTarget) return;
      escalateMut.mutate(
        {
          id: escalateTarget.id,
          level: values.level,
          reason: values.reason,
        },
        {
          onSuccess: () => {
            setEscalateOpen(false);
            setEscalateTarget(null);
          },
        },
      );
    } catch {
      // ignore
    }
  };

  return (
    <div data-testid="sos-page">
      <PageHeader
        title="SOS 紧急报警"
        subtitle="24h 报警实时列表（30s 自动刷新）"
      />

      <Space style={{ marginBottom: 16 }}>
        <Select
          placeholder="状态"
          allowClear
          style={{ width: 140 }}
          value={statusFilter}
          onChange={setStatusFilter}
          data-testid="filter-status"
          options={[
            { value: 'open', label: STATUS_LABEL.open },
            { value: 'closed', label: STATUS_LABEL.closed },
          ]}
        />
        <span style={{ color: '#999' }}>共 {data?.total ?? 0} 条</span>
      </Space>

      <ProTable<SosItem>
        testId="sos-table"
        rowKey="id"
        columns={columns}
        dataSource={items}
        loading={isLoading}
        density="middle"
        scroll={{ x: 1400 }}
      />

      {isError && (
        <div data-testid="sos-error" style={{ color: '#f5222d', marginTop: 8 }}>
          加载失败：{(error as Error)?.message ?? '未知错误'}
        </div>
      )}

      {/* 处置 Modal */}
      <Modal
        title={`处置报警 #${resolveTarget?.id ?? '—'}`}
        open={resolveOpen}
        onCancel={() => {
          setResolveOpen(false);
          setResolveTarget(null);
        }}
        onOk={onSubmitResolve}
        confirmLoading={resolveMut.isPending}
        okButtonProps={{ 'data-testid': 'btn-resolve-confirm' }}
        cancelButtonProps={{ 'data-testid': 'btn-resolve-cancel' }}
        destroyOnClose
      >
        <Form form={resolveForm} layout="vertical" preserve={false}>
          <Form.Item
            label="处置说明"
            name="note"
            rules={[{ required: true, message: '请输入处置说明' }]}
          >
            <Input.TextArea
              rows={4}
              placeholder="例：已联系陪诊师陪同患者转急诊"
              data-testid="resolve-note-input"
              maxLength={200}
              showCount
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* 升级 Modal */}
      <Modal
        title={`升级报警 #${escalateTarget?.id ?? '—'}`}
        open={escalateOpen}
        onCancel={() => {
          setEscalateOpen(false);
          setEscalateTarget(null);
        }}
        onOk={onSubmitEscalate}
        confirmLoading={escalateMut.isPending}
        okButtonProps={{ danger: true, 'data-testid': 'btn-escalate-confirm' }}
        cancelButtonProps={{ 'data-testid': 'btn-escalate-cancel' }}
        destroyOnClose
      >
        <Form form={escalateForm} layout="vertical" preserve={false}>
          <Form.Item
            label="升级级别"
            name="level"
            rules={[{ required: true, message: '请选择升级级别' }]}
          >
            <Select
              placeholder="L1 / L2 / L3"
              data-testid="escalate-level-select"
              options={[
                { value: 'L1', label: 'L1 - 主管介入' },
                { value: 'L2', label: 'L2 - 经理介入' },
                { value: 'L3', label: 'L3 - 总监介入' },
              ]}
            />
          </Form.Item>
          <Form.Item
            label="升级原因"
            name="reason"
            rules={[{ required: true, message: '请输入升级原因' }]}
          >
            <Input.TextArea
              rows={4}
              placeholder="例：患者生命体征异常，需立即转急诊"
              data-testid="escalate-reason-input"
              maxLength={200}
              showCount
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}