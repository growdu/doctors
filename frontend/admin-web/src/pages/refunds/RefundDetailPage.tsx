/**
 * RefundDetailPage：admin-web 退款详情页（升级版）。
 *
 * 范围：
 *   - 退款工单基本信息（id / 订单 / 患者 / 金额 / 原因 / 状态 / 备注）；
 *   - 审批操作（通过 / 驳回，弹 Modal 收集备注 + 原因）；
 *   - viewer 不显示审批按钮（RBAC）。
 *
 * 数据流：
 *   - detail: GET /api/v1/admin/refunds/:id
 *   - approve: POST /api/v1/admin/refunds/:id/approve
 *   - reject:  POST /api/v1/admin/refunds/:id/reject
 *
 * 对应 spec：2026-09-24-admin-web-design.md §Task 13
 */
import { useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Alert,
  Button,
  Card,
  Descriptions,
  Form,
  Input,
  message,
  Modal,
  Space,
  Spin,
  Tag,
} from 'antd';
import {
  ArrowLeftOutlined,
  CheckOutlined,
  CloseOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import { PageHeader } from '@/components/PageHeader';
import {
  approveRefund,
  fetchRefundDetail,
  rejectRefund,
  refundQueryKeys,
  type RefundStatus,
} from '@/api/admin/refunds';
import { useAuthStore } from '@/stores/authStore';

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

interface ApproveFormValues {
  note: string;
}

interface RejectFormValues {
  reason: string;
}

export default function RefundDetailPage() {
  const { id = '' } = useParams<{ id: string }>();
  const numericId = Number(id);
  const navigate = useNavigate();
  const qc = useQueryClient();
  const role = useAuthStore((s) => s.role);
  const canApprove = role === 'super_admin' || role === 'refund_admin';

  const [approveOpen, setApproveOpen] = useState(false);
  const [approveForm] = Form.useForm<ApproveFormValues>();
  const [rejectOpen, setRejectOpen] = useState(false);
  const [rejectForm] = Form.useForm<RejectFormValues>();

  const {
    data: refund,
    isLoading,
    isError,
    error,
  } = useQuery({
    queryKey: refundQueryKeys.detail(numericId),
    queryFn: () => fetchRefundDetail(numericId),
    enabled: Number.isFinite(numericId) && numericId > 0,
  });

  const approveMut = useMutation({
    mutationFn: ({ id, note }: { id: number; note: string }) =>
      approveRefund(id, note),
    onSuccess: () => {
      message.success('已通过审批');
      qc.invalidateQueries({ queryKey: refundQueryKeys.detail(numericId) });
      qc.invalidateQueries({ queryKey: ['refunds'] });
    },
    onError: (e: Error) => message.error(`通过失败：${e.message}`),
  });

  const rejectMut = useMutation({
    mutationFn: ({ id, reason }: { id: number; reason: string }) =>
      rejectRefund(id, reason),
    onSuccess: () => {
      message.success('已驳回审批');
      qc.invalidateQueries({ queryKey: refundQueryKeys.detail(numericId) });
      qc.invalidateQueries({ queryKey: ['refunds'] });
    },
    onError: (e: Error) => message.error(`驳回失败：${e.message}`),
  });

  if (isLoading) {
    return (
      <div data-testid="refund-detail-loading">
        <PageHeader
          title={`退款详情 #${id}`}
          subtitle="工单信息 + 关联订单 + 审批操作"
        />
        <Spin />
      </div>
    );
  }
  if (isError || !refund) {
    return (
      <div data-testid="refund-detail-page">
        <PageHeader
          title={`退款详情 #${id}`}
          subtitle="工单信息 + 关联订单 + 审批操作"
        />
        <Alert
          type="error"
          showIcon
          message="加载失败"
          description={(error as Error)?.message ?? '退款工单不存在'}
        />
      </div>
    );
  }

  const onSubmitApprove = async () => {
    try {
      const values = await approveForm.validateFields();
      approveMut.mutate(
        { id: refund.id, note: values.note },
        {
          onSuccess: () => {
            setApproveOpen(false);
            approveForm.resetFields();
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
      rejectMut.mutate(
        { id: refund.id, reason: values.reason },
        {
          onSuccess: () => {
            setRejectOpen(false);
            rejectForm.resetFields();
          },
        },
      );
    } catch {
      // ignore
    }
  };

  return (
    <div data-testid="refund-detail-page">
      <PageHeader
        title={`退款详情 #${refund.id}`}
        subtitle={`订单 #${refund.order_id} · 患者 ${refund.patient_name}`}
        extra={
          <Space>
            <Button
              icon={<ArrowLeftOutlined />}
              onClick={() => navigate('/refunds')}
              data-testid="btn-back"
            >
              返回列表
            </Button>
            {canApprove && refund.status === 'pending' && (
              <>
                <Button
                  type="primary"
                  icon={<CheckOutlined />}
                  onClick={() => setApproveOpen(true)}
                  data-testid="btn-approve"
                >
                  通过
                </Button>
                <Button
                  danger
                  icon={<CloseOutlined />}
                  onClick={() => setRejectOpen(true)}
                  data-testid="btn-reject"
                >
                  驳回
                </Button>
              </>
            )}
          </Space>
        }
      />

      <Card title="工单信息">
        <Descriptions column={2} bordered size="small">
          <Descriptions.Item label="工单 ID">#{refund.id}</Descriptions.Item>
          <Descriptions.Item label="订单 ID">
            <Tag color="blue">#{refund.order_id}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="患者">{refund.patient_name}</Descriptions.Item>
          <Descriptions.Item label="金额">
            <Tag color="orange">¥{refund.amount}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="退款原因" span={2}>
            {refund.reason}
          </Descriptions.Item>
          <Descriptions.Item label="状态">
            <Tag color={STATUS_COLOR[refund.status]}>
              {STATUS_LABEL[refund.status]}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label="审批备注">
            {refund.refund_note ?? <Tag color="default">—</Tag>}
          </Descriptions.Item>
          <Descriptions.Item label="创建时间" span={2}>
            {dayjs(refund.created_at).format('YYYY-MM-DD HH:mm:ss')}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      {/* 通过 Modal */}
      <Modal
        title={`通过退款 #${refund.id}`}
        open={approveOpen}
        onCancel={() => {
          setApproveOpen(false);
          approveForm.resetFields();
        }}
        onOk={onSubmitApprove}
        confirmLoading={approveMut.isPending}
        okButtonProps={{ 'data-testid': 'btn-approve-confirm' }}
        cancelButtonProps={{ 'data-testid': 'btn-approve-cancel' }}
        destroyOnClose
      >
        <Form form={approveForm} layout="vertical" preserve={false}>
          <Form.Item label="审批备注" name="note">
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
        title={`驳回退款 #${refund.id}`}
        open={rejectOpen}
        onCancel={() => {
          setRejectOpen(false);
          rejectForm.resetFields();
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