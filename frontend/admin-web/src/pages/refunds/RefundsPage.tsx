/**
 * RefundsPage：admin-web 退款审批列表页（升级版）。
 *
 * 功能：
 *   - 退款列表（按 status=pending|approved|rejected 过滤）；
 *   - 行操作：审批通过 / 驳回（弹 Modal 收集原因）+ 详情；
 *   - viewer 不显示审批按钮（RBAC）。
 *
 * 数据流：
 *   - list:    GET /api/v1/admin/refunds?status=
 *   - approve: POST /api/v1/admin/refunds/:id/approve
 *   - reject:  POST /api/v1/admin/refunds/:id/reject
 *
 * 对应 spec：2026-09-24-admin-web-design.md §Task 13
 */
import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
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
import { CheckOutlined, CloseOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { PageHeader } from '@/components/PageHeader';
import { ProTable } from '@/components/ProTable';
import {
  approveRefund,
  fetchRefunds,
  rejectRefund,
  refundQueryKeys,
  type RefundItem,
  type RefundStatus,
} from '@/api/admin/refunds';
import { useAuthStore } from '@/stores/authStore';

interface ApproveFormValues {
  note: string;
}

interface RejectFormValues {
  reason: string;
}

const STATUS_LABEL: Record<RefundStatus, string> = {
  pending: '待审',
  approved: '已通过',
  rejected: '已驳回',
};

const STATUS_COLOR: Record<RefundStatus, string> = {
  pending: 'orange',
  approved: 'green',
  rejected: 'red',
};

export default function RefundsPage() {
  const navigate = useNavigate();
  const qc = useQueryClient();
  const role = useAuthStore((s) => s.role);
  /** 仅 super_admin / refund_admin 可审批 */
  const canApprove = role === 'super_admin' || role === 'refund_admin';

  const [statusFilter, setStatusFilter] = useState<string | undefined>();

  const [approveOpen, setApproveOpen] = useState(false);
  const [approveTarget, setApproveTarget] = useState<RefundItem | null>(null);
  const [approveForm] = Form.useForm<ApproveFormValues>();

  const [rejectOpen, setRejectOpen] = useState(false);
  const [rejectTarget, setRejectTarget] = useState<RefundItem | null>(null);
  const [rejectForm] = Form.useForm<RejectFormValues>();

  // ── 查询 ──────────────────────────────────────────────────────────
  const { data, isLoading, isError, error } = useQuery({
    queryKey: refundQueryKeys.list({ status: statusFilter }),
    queryFn: () => fetchRefunds({ status: statusFilter }),
    refetchOnWindowFocus: false,
  });

  // ── 通过 ──────────────────────────────────────────────────────────
  const approveMut = useMutation({
    mutationFn: ({ id, note }: { id: number; note: string }) =>
      approveRefund(id, note),
    onSuccess: (r) => {
      message.success(`已通过退款 #${r.id}`);
      qc.invalidateQueries({ queryKey: ['refunds'] });
    },
    onError: (e: Error) => {
      message.error(`通过失败：${e.message}`);
    },
  });

  // ── 驳回 ──────────────────────────────────────────────────────────
  const rejectMut = useMutation({
    mutationFn: ({ id, reason }: { id: number; reason: string }) =>
      rejectRefund(id, reason),
    onSuccess: (r) => {
      message.success(`已驳回退款 #${r.id}`);
      qc.invalidateQueries({ queryKey: ['refunds'] });
    },
    onError: (e: Error) => {
      message.error(`驳回失败：${e.message}`);
    },
  });

  const items: RefundItem[] = data?.data ?? [];

  const columns: ColumnsType<RefundItem> = [
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
    { title: '申请人', dataIndex: 'patient_name', width: 100 },
    {
      title: '金额',
      dataIndex: 'amount',
      width: 110,
      align: 'right',
      render: (v: number) => <Tag color="orange">¥{v}</Tag>,
    },
    { title: '退款原因', dataIndex: 'reason', width: 240 },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (v: RefundStatus) => (
        <Tag color={STATUS_COLOR[v]}>{STATUS_LABEL[v]}</Tag>
      ),
    },
    {
      title: '审批备注',
      dataIndex: 'refund_note',
      width: 200,
      render: (v: string | null) => v ?? <Tag color="default">—</Tag>,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      width: 170,
      render: (v: string) => dayjs(v).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: '操作',
      width: 260,
      fixed: 'right',
      render: (_: unknown, r: RefundItem) => (
        <Space size="small">
          <Button
            size="small"
            onClick={() => navigate(`/refunds/${r.id}`)}
            data-testid={`btn-detail-${r.id}`}
          >
            详情
          </Button>
          {canApprove && r.status === 'pending' && (
            <>
              <Button
                size="small"
                type="primary"
                icon={<CheckOutlined />}
                onClick={() => {
                  setApproveTarget(r);
                  approveForm.resetFields();
                  setApproveOpen(true);
                }}
                data-testid={`btn-approve-${r.id}`}
              >
                通过
              </Button>
              <Button
                size="small"
                danger
                icon={<CloseOutlined />}
                onClick={() => {
                  setRejectTarget(r);
                  rejectForm.resetFields();
                  setRejectOpen(true);
                }}
                data-testid={`btn-reject-${r.id}`}
              >
                驳回
              </Button>
            </>
          )}
        </Space>
      ),
    },
  ];

  // ── 提交 Modal ────────────────────────────────────────────────────
  const onSubmitApprove = async () => {
    try {
      const values = await approveForm.validateFields();
      if (!approveTarget) return;
      approveMut.mutate(
        { id: approveTarget.id, note: values.note },
        {
          onSuccess: () => {
            setApproveOpen(false);
            setApproveTarget(null);
          },
        },
      );
    } catch {
      // ignore
    }
  };

  const onSubmitReject = async () => {
    try {
      const values = await rejectForm.validateFields();
      if (!rejectTarget) return;
      rejectMut.mutate(
        { id: rejectTarget.id, reason: values.reason },
        {
          onSuccess: () => {
            setRejectOpen(false);
            setRejectTarget(null);
          },
        },
      );
    } catch {
      // ignore
    }
  };

  return (
    <div data-testid="refunds-page">
      <PageHeader
        title="退款审批"
        subtitle="待审 / 已批准 / 已驳回 退款工单"
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
            { value: 'pending', label: STATUS_LABEL.pending },
            { value: 'approved', label: STATUS_LABEL.approved },
            { value: 'rejected', label: STATUS_LABEL.rejected },
          ]}
        />
        <span style={{ color: '#999' }}>共 {data?.total ?? 0} 条</span>
      </Space>

      <ProTable<RefundItem>
        testId="refunds-table"
        rowKey="id"
        columns={columns}
        dataSource={items}
        loading={isLoading}
        density="middle"
        scroll={{ x: 1200 }}
      />

      {isError && (
        <div data-testid="refunds-error" style={{ color: '#f5222d', marginTop: 8 }}>
          加载失败：{(error as Error)?.message ?? '未知错误'}
        </div>
      )}

      {/* 通过 Modal */}
      <Modal
        title={`通过退款 #${approveTarget?.id ?? '—'}`}
        open={approveOpen}
        onCancel={() => {
          setApproveOpen(false);
          setApproveTarget(null);
        }}
        onOk={onSubmitApprove}
        confirmLoading={approveMut.isPending}
        okButtonProps={{ 'data-testid': 'btn-approve-confirm' }}
        cancelButtonProps={{ 'data-testid': 'btn-approve-cancel' }}
        destroyOnClose
      >
        <Form form={approveForm} layout="vertical" preserve={false}>
          <Form.Item
            label="审批备注（可选）"
            name="note"
          >
            <Input.TextArea
              rows={3}
              placeholder="例：已联系财务，全额退款"
              data-testid="approve-note-input"
              maxLength={120}
              showCount
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* 驳回 Modal */}
      <Modal
        title={`驳回退款 #${rejectTarget?.id ?? '—'}`}
        open={rejectOpen}
        onCancel={() => {
          setRejectOpen(false);
          setRejectTarget(null);
        }}
        onOk={onSubmitReject}
        confirmLoading={rejectMut.isPending}
        okButtonProps={{ danger: true, 'data-testid': 'btn-reject-confirm' }}
        cancelButtonProps={{ 'data-testid': 'btn-reject-cancel' }}
        destroyOnClose
      >
        <Form form={rejectForm} layout="vertical" preserve={false}>
          <Form.Item
            label="驳回原因"
            name="reason"
            rules={[{ required: true, message: '请输入驳回原因' }]}
          >
            <Input.TextArea
              rows={4}
              placeholder="例：患者已在其他渠道退款"
              data-testid="reject-reason-input"
              maxLength={200}
              showCount
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}