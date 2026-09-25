/**
 * EscortDetailPage：admin-web 陪诊师详情页（升级版）。
 *
 * 范围：
 *   - 顶部：基本信息（昵称 / 手机 / 城市 / 评分 / 注册时间 / 健康证 / 服务范围 / 培训记录）；
 *   - 中部：审计时间线（AuditAction 组件）；
 *   - 底部：审批操作按钮（approve / reject），拒绝弹 Modal 收原因；
 *   - viewer 不显示审批按钮。
 *
 * 数据流：
 *   - detail:     GET /api/v1/admin/escorts/:id
 *   - history:    fetchEscortAuditHistory（mock 拼装，后续接真端点）
 *   - approve / reject: POST /api/v1/admin/escorts/:id/{approve|reject}
 *
 * 对应 spec：2026-09-24-admin-web-design.md §Task 12
 */
import { useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Alert,
  Button,
  Card,
  Descriptions,
  Divider,
  Form,
  Input,
  message,
  Modal,
  Space,
  Spin,
  Tag,
  Typography,
} from 'antd';
import {
  ArrowLeftOutlined,
  CheckOutlined,
  CloseOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import { PageHeader } from '@/components/PageHeader';
import { StatusBadge } from '@/components/StatusBadge';
import { AuditAction } from '@/components/AuditAction';
import {
  approveEscort,
  escortQueryKeys,
  fetchEscortAuditHistory,
  fetchEscortDetail,
  rejectEscort,
} from '@/api/admin/escorts';
import { useAuthStore } from '@/stores/authStore';

const { Text } = Typography;

interface RejectFormValues {
  reason: string;
}

export default function EscortDetailPage() {
  const { id = '' } = useParams<{ id: string }>();
  const numericId = Number(id);
  const navigate = useNavigate();
  const qc = useQueryClient();
  const role = useAuthStore((s) => s.role);
  const canAudit = role === 'super_admin' || role === 'audit_admin';

  const [rejectModalOpen, setRejectModalOpen] = useState(false);
  const [rejectForm] = Form.useForm<RejectFormValues>();

  const {
    data: escort,
    isLoading,
    isError,
    error,
  } = useQuery({
    queryKey: escortQueryKeys.detail(numericId),
    queryFn: () => fetchEscortDetail(numericId),
    enabled: Number.isFinite(numericId) && numericId > 0,
  });

  const { data: auditHistory = [] } = useQuery({
    queryKey: escortQueryKeys.auditHistory(numericId),
    queryFn: () => fetchEscortAuditHistory(numericId),
    enabled: Number.isFinite(numericId) && numericId > 0,
  });

  const approveMut = useMutation({
    mutationFn: () => approveEscort(numericId, { note: 'audit approved' }),
    onSuccess: () => {
      message.success(`已通过陪诊师 #${numericId} 审核`);
      qc.invalidateQueries({ queryKey: escortQueryKeys.detail(numericId) });
      qc.invalidateQueries({ queryKey: escortQueryKeys.auditHistory(numericId) });
      qc.invalidateQueries({ queryKey: escortQueryKeys.pendingAudit });
    },
    onError: (e: Error) => {
      message.error(`通过失败：${e.message}`);
    },
  });

  const rejectMut = useMutation({
    mutationFn: (reason: string) =>
      rejectEscort(numericId, { reason }),
    onSuccess: () => {
      message.success(`已拒绝陪诊师 #${numericId} 审核`);
      qc.invalidateQueries({ queryKey: escortQueryKeys.detail(numericId) });
      qc.invalidateQueries({ queryKey: escortQueryKeys.auditHistory(numericId) });
      qc.invalidateQueries({ queryKey: escortQueryKeys.pendingAudit });
    },
    onError: (e: Error) => {
      message.error(`拒绝失败：${e.message}`);
    },
  });

  if (isLoading) {
    return (
      <div data-testid="escort-detail-loading">
        <PageHeader
          title={`陪诊师详情 #${id}`}
          subtitle="基本信息 + 资质 + 审核历史"
        />
        <Spin />
      </div>
    );
  }
  if (isError || !escort) {
    return (
      <div data-testid="escort-detail-page">
        <PageHeader
          title={`陪诊师详情 #${id}`}
          subtitle="基本信息 + 资质 + 审核历史"
        />
        <Alert
          type="error"
          showIcon
          message="加载失败"
          description={(error as Error)?.message ?? '陪诊师不存在'}
        />
      </div>
    );
  }

  const onSubmitReject = async () => {
    try {
      const values = await rejectForm.validateFields();
      rejectMut.mutate(values.reason, {
        onSuccess: () => {
          setRejectModalOpen(false);
          rejectForm.resetFields();
        },
      });
    } catch {
      // antd validation failed silently
    }
  };

  return (
    <div data-testid="escort-detail-page">
      <PageHeader
        title={`陪诊师详情 #${escort.id}`}
        subtitle={`${escort.name} · ${escort.city}`}
        extra={
          <Space>
            <Button
              icon={<ArrowLeftOutlined />}
              onClick={() => navigate('/escorts/audit')}
              data-testid="btn-back"
            >
              返回审核列表
            </Button>
            {canAudit && (
              <>
                <Button
                  type="primary"
                  icon={<CheckOutlined />}
                  onClick={() => approveMut.mutate()}
                  loading={approveMut.isPending}
                  data-testid="btn-approve"
                >
                  通过
                </Button>
                <Button
                  danger
                  icon={<CloseOutlined />}
                  onClick={() => setRejectModalOpen(true)}
                  data-testid="btn-reject"
                >
                  拒绝
                </Button>
              </>
            )}
          </Space>
        }
      />

      {/* 基本信息 */}
      <Card title="基本信息">
        <Descriptions column={2} bordered size="small">
          <Descriptions.Item label="昵称">{escort.name}</Descriptions.Item>
          <Descriptions.Item label="真实姓名">
            {escort.real_name ?? '—'}
          </Descriptions.Item>
          <Descriptions.Item label="手机">{escort.phone}</Descriptions.Item>
          <Descriptions.Item label="身份证号">
            {escort.id_card_no ?? '—'}
          </Descriptions.Item>
          <Descriptions.Item label="城市">{escort.city}</Descriptions.Item>
          <Descriptions.Item label="评分">
            <Tag color="gold">★ {escort.rating.toFixed(1)}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="年龄">{escort.age ?? '—'}</Descriptions.Item>
          <Descriptions.Item label="性别">
            {escort.gender === 'male'
              ? '男'
              : escort.gender === 'female'
              ? '女'
              : '—'}
          </Descriptions.Item>
          <Descriptions.Item label="审核状态">
            <StatusBadge status={escort.audit_status} testId="escort-status" />
          </Descriptions.Item>
          <Descriptions.Item label="健康证">
            {escort.health_cert_status === 'valid' ? (
              <Tag color="green">有效</Tag>
            ) : escort.health_cert_status === 'expired' ? (
              <Tag color="orange">已过期</Tag>
            ) : (
              <Tag color="default">未提交</Tag>
            )}
          </Descriptions.Item>
          <Descriptions.Item label="注册时间" span={2}>
            {dayjs(escort.created_at).format('YYYY-MM-DD HH:mm:ss')}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      {/* 服务范围 */}
      <Card style={{ marginTop: 16 }} title="服务范围">
        {escort.service_cities && escort.service_cities.length > 0 ? (
          <Space wrap>
            {escort.service_cities.map((c) => (
              <Tag key={c} color="blue">
                {c}
              </Tag>
            ))}
          </Space>
        ) : (
          <Text type="secondary">暂无服务城市</Text>
        )}
      </Card>

      {/* 培训记录 */}
      <Card style={{ marginTop: 16 }} title="培训记录">
        {escort.training_records && escort.training_records.length > 0 ? (
          <ul data-testid="training-records">
            {escort.training_records.map((r, idx) => (
              <li key={idx}>
                {r.title} —{' '}
                {r.passed_at ? (
                  <Tag color="green">
                    通过 · {dayjs(r.passed_at).format('YYYY-MM-DD')}
                  </Tag>
                ) : (
                  <Tag color="default">未通过</Tag>
                )}
              </li>
            ))}
          </ul>
        ) : (
          <Text type="secondary">暂无培训记录</Text>
        )}
      </Card>

      <Divider orientation="left">审计历史</Divider>
      <AuditAction items={auditHistory} testId="escort-audit-history" />

      {/* 拒绝原因 Modal */}
      <Modal
        title={`拒绝陪诊师 #${escort.id} 审核`}
        open={rejectModalOpen}
        onCancel={() => {
          setRejectModalOpen(false);
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
            label="拒绝原因"
            name="reason"
            rules={[{ required: true, message: '请输入拒绝原因' }]}
          >
            <Input.TextArea
              rows={4}
              placeholder="例：健康证已过期，请重新上传"
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
