/**
 * EscortAuditPage：admin-web 陪诊师审核队列页（升级版）。
 *
 * 功能：
 *   - 待审核陪诊师列表（status=pending）；
 *   - 行操作：通过 / 拒绝 + 拒绝原因备注；
 *   - 拒绝操作弹出 Modal 收集原因；
 *   - viewer 角色不显示审核按钮（RBAC）。
 *
 * 数据流：
 *   - list: GET /api/v1/admin/escorts?status=pending
 *   - approve: POST /api/v1/admin/escorts/:id/approve
 *   - reject: POST /api/v1/admin/escorts/:id/reject
 *
 * 关键技术：
 *   - TanStack Query 拉取 + 操作后 invalidate；
 *   - antd Modal 收集拒绝原因；
 *   - message.success / message.error 反馈。
 *
 * 对应 spec：2026-09-24-admin-web-design.md §3.2 + §Task 12
 */
import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Button,
  Modal,
  Form,
  Input,
  Space,
  Tag,
  message,
} from 'antd';
import { CheckOutlined, CloseOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { PageHeader } from '@/components/PageHeader';
import { ProTable } from '@/components/ProTable';
import { StatusBadge } from '@/components/StatusBadge';
import {
  approveEscort,
  escortQueryKeys,
  fetchPendingAudit,
  rejectEscort,
  type EscortListItem,
} from '@/api/admin/escorts';
import { useAuthStore } from '@/stores/authStore';

interface RejectFormValues {
  reason: string;
}

export default function EscortAuditPage() {
  const navigate = useNavigate();
  const qc = useQueryClient();
  const role = useAuthStore((s) => s.role);
  /** viewer 不允许审核（按权限控制可见性，UI 隐藏按钮） */
  const canAudit = role === 'super_admin' || role === 'audit_admin';

  const [rejectModalOpen, setRejectModalOpen] = useState(false);
  const [pendingReject, setPendingReject] = useState<EscortListItem | null>(null);
  const [rejectForm] = Form.useForm<RejectFormValues>();

  // ── 查询 ──────────────────────────────────────────────────────────
  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: escortQueryKeys.pendingAudit,
    queryFn: () => fetchPendingAudit({ page: 1, page_size: 50 }),
    refetchOnWindowFocus: false,
  });

  // ── 通过审核（无参数） ─────────────────────────────────────────────
  const approveMut = useMutation({
    mutationFn: (id: number) => approveEscort(id, { note: 'audit approved' }),
    onSuccess: (r) => {
      message.success(`已通过陪诊师 #${r.id} 审核`);
      qc.invalidateQueries({ queryKey: escortQueryKeys.pendingAudit });
    },
    onError: (e: Error) => {
      message.error(`通过失败：${e.message}`);
    },
  });

  // ── 拒绝审核（必填 reason） ───────────────────────────────────────
  const rejectMut = useMutation({
    mutationFn: ({ id, reason }: { id: number; reason: string }) =>
      rejectEscort(id, { reason }),
    onSuccess: (r) => {
      message.success(`已拒绝陪诊师 #${r.id} 审核`);
      qc.invalidateQueries({ queryKey: escortQueryKeys.pendingAudit });
    },
    onError: (e: Error) => {
      message.error(`拒绝失败：${e.message}`);
    },
  });

  const items: EscortListItem[] = data?.data ?? [];

  // ── 列定义 ────────────────────────────────────────────────────────
  const columns: ColumnsType<EscortListItem> = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 80,
      render: (v: number) => <Tag>#{v}</Tag>,
    },
    { title: '昵称', dataIndex: 'name', width: 140 },
    { title: '手机', dataIndex: 'phone', width: 140 },
    { title: '城市', dataIndex: 'city', width: 100 },
    {
      title: '提交时间',
      dataIndex: 'created_at',
      width: 170,
      render: (v: string) => dayjs(v).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: '状态',
      dataIndex: 'audit_status',
      width: 110,
      render: (_: unknown, r: EscortListItem) => (
        <StatusBadge status={r.audit_status} testId={`row-status-${r.id}`} />
      ),
    },
    {
      title: '操作',
      width: 280,
      fixed: 'right',
      render: (_: unknown, r: EscortListItem) => (
        <Space size="small">
          <Button
            size="small"
            onClick={() => navigate(`/escorts/${r.id}`)}
            data-testid={`btn-detail-${r.id}`}
          >
            查看详情
          </Button>
          {canAudit && (
            <>
              <Button
                size="small"
                type="primary"
                icon={<CheckOutlined />}
                onClick={() => approveMut.mutate(r.id)}
                loading={approveMut.isPending && approveMut.variables === r.id}
                data-testid={`btn-approve-${r.id}`}
              >
                通过
              </Button>
              <Button
                size="small"
                danger
                icon={<CloseOutlined />}
                onClick={() => {
                  setPendingReject(r);
                  rejectForm.resetFields();
                  setRejectModalOpen(true);
                }}
                data-testid={`btn-reject-${r.id}`}
              >
                拒绝
              </Button>
            </>
          )}
        </Space>
      ),
    },
  ];

  // ── 拒绝提交 ─────────────────────────────────────────────────────
  const onSubmitReject = async () => {
    try {
      const values = await rejectForm.validateFields();
      if (!pendingReject) return;
      rejectMut.mutate(
        { id: pendingReject.id, reason: values.reason },
        {
          onSuccess: () => {
            setRejectModalOpen(false);
            setPendingReject(null);
          },
        },
      );
    } catch {
      // antd 校验未通过静默
    }
  };

  return (
    <div data-testid="escort-audit-page">
      <PageHeader
        title="陪诊师审核"
        subtitle="待审核陪诊师队列（status=pending）"
        extra={
          <Button onClick={() => refetch()} data-testid="btn-refresh">
            刷新
          </Button>
        }
      />

      <ProTable<EscortListItem>
        testId="audit-table"
        rowKey="id"
        columns={columns}
        dataSource={items}
        loading={isLoading}
        density="middle"
        scroll={{ x: 1100 }}
      />

      {isError && (
        <div data-testid="audit-error" style={{ color: '#f5222d', marginTop: 8 }}>
          加载失败：{(error as Error)?.message ?? '未知错误'}
        </div>
      )}

      {/* 拒绝原因 Modal */}
      <Modal
        title={`拒绝陪诊师 #${pendingReject?.id ?? '—'} 审核`}
        open={rejectModalOpen}
        onCancel={() => {
          setRejectModalOpen(false);
          setPendingReject(null);
        }}
        onOk={onSubmitReject}
        confirmLoading={rejectMut.isPending}
        okButtonProps={{ danger: true, 'data-testid': 'btn-reject-confirm' }}
        cancelButtonProps={{ 'data-testid': 'btn-reject-cancel' }}
        destroyOnClose
      >
        <Form form={rejectForm} layout="vertical" preserve={false}>
          <Form.Item
            label="拒绝原因"
            name="reason"
            rules={[{ required: true, message: '请输入拒绝原因' }]}
          >
            <Input.TextArea
              rows={4}
              placeholder="例：健康证已过期 / 资质不完整"
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
