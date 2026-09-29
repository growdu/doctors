/**
 * ReviewsPage：admin-web 评价列表页（升级版）。
 *
 * 范围：
 *   - 评价列表（min_rating 评分筛选 + escort_id 陪诊师筛选）；
 *   - 行操作：详情 + 审核（通过 / 隐藏）+ 客服回复；
 *   - 审核弹 Modal 选 pass/hide + 备注；
 *   - 回复弹 Modal 输入回复内容；
 *   - viewer 不显示审核 / 回复按钮。
 *
 * 数据流：
 *   - list:  GET /api/v1/admin/reviews?escort_id=&min_rating=
 *   - audit: POST /api/v1/admin/reviews/:id/audit { result, reason }
 *   - reply: POST /api/v1/admin/reviews/:id/reply { reply }
 *
 * 对应 spec：2026-09-24-admin-web-design.md §Task 16
 */
import { useState } from 'react';
import {
  Button,
  Form,
  Input,
  InputNumber,
  message,
  Modal,
  Radio,
  Space,
  Tag,
} from 'antd';
import {
  CheckCircleOutlined,
  EyeInvisibleOutlined,
  MessageOutlined,
} from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { PageHeader } from '@/components/PageHeader';
import { ProTable } from '@/components/ProTable';
import {
  auditReview,
  fetchReviews,
  replyReview,
  reviewQueryKeys,
  type ReviewAuditResult,
  type ReviewListItem,
} from '@/api/admin/reviews';
import { useAuthStore } from '@/stores/authStore';

interface AuditFormValues {
  result: ReviewAuditResult;
  reason?: string;
}

interface ReplyFormValues {
  reply: string;
}

/** 评分 → 5 段颜色（绿-黄-红）。 */
function ratingColor(rating: number): string {
  if (rating >= 5) return 'green';
  if (rating >= 4) return 'lime';
  if (rating >= 3) return 'gold';
  if (rating >= 2) return 'orange';
  return 'red';
}

export default function ReviewsPage() {
  const qc = useQueryClient();
  const role = useAuthStore((s) => s.role);
  const canAudit =
    role === 'super_admin' ||
    role === 'audit_admin' ||
    role === 'cs';

  const [minRating, setMinRating] = useState<number>(0);
  const [escortId, setEscortId] = useState<number | undefined>(undefined);
  const [auditModal, setAuditModal] = useState<ReviewListItem | null>(null);
  const [replyModal, setReplyModal] = useState<ReviewListItem | null>(null);
  const [auditForm] = Form.useForm<AuditFormValues>();
  const [replyForm] = Form.useForm<ReplyFormValues>();

  const { data, isLoading, isError, error } = useQuery({
    queryKey: reviewQueryKeys.list({ escort_id: escortId, min_rating: minRating }),
    queryFn: () =>
      fetchReviews({
        escort_id: escortId,
        min_rating: minRating > 0 ? minRating : undefined,
      }),
    refetchOnWindowFocus: false,
  });

  const auditMut = useMutation({
    mutationFn: ({
      id,
      result,
      reason,
    }: {
      id: number;
      result: ReviewAuditResult;
      reason?: string;
    }) => auditReview(id, { result, reason }),
    onSuccess: (r) => {
      message.success(
        r.audit_result === 'pass'
          ? `评价 #${r.id} 已通过`
          : `评价 #${r.id} 已隐藏`,
      );
      qc.invalidateQueries({ queryKey: ['reviews'] });
    },
    onError: (e: Error) => {
      message.error(`审核失败：${e.message}`);
    },
  });

  const replyMut = useMutation({
    mutationFn: ({ id, reply }: { id: number; reply: string }) =>
      replyReview(id, reply),
    onSuccess: (r) => {
      message.success(`已回复评价 #${r.id}`);
      qc.invalidateQueries({ queryKey: ['reviews'] });
    },
    onError: (e: Error) => {
      message.error(`回复失败：${e.message}`);
    },
  });

  const items: ReviewListItem[] = data?.data ?? [];

  const columns: ColumnsType<ReviewListItem> = [
    { title: 'ID', dataIndex: 'id', width: 80 },
    {
      title: '陪诊师',
      dataIndex: 'escort_id',
      width: 120,
      render: (v: number, r: ReviewListItem) =>
        r.escort_name ? `${r.escort_name} (#${v})` : `#${v}`,
    },
    {
      title: '订单号',
      dataIndex: 'order_id',
      width: 100,
    },
    {
      title: '评分',
      dataIndex: 'rating',
      width: 100,
      render: (v: number, r: ReviewListItem) => (
        <Tag color={ratingColor(v)} data-testid={`row-rating-${r.id}`}>
          ★ {v}
        </Tag>
      ),
    },
    {
      title: '评论',
      dataIndex: 'content',
      width: 260,
      ellipsis: true,
      render: (v: string, r: ReviewListItem) => (
        <span data-testid={`row-content-${r.id}`}>{v}</span>
      ),
    },
    {
      title: '审核结果',
      dataIndex: 'audit_result',
      width: 100,
      render: (v: ReviewAuditResult | null | undefined, r: ReviewListItem) =>
        v === 'pass' ? (
          <Tag color="green">通过</Tag>
        ) : v === 'hide' ? (
          <Tag color="red">已隐藏</Tag>
        ) : (
          <Tag data-testid={`row-audit-${r.id}`}>待审</Tag>
        ),
    },
    {
      title: '提交时间',
      dataIndex: 'created_at',
      width: 170,
      render: (v: string) => dayjs(v).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: '操作',
      width: 240,
      fixed: 'right',
      render: (_: unknown, r: ReviewListItem) => (
        <Space size="small">
          {canAudit && (
            <Button
              size="small"
              type="primary"
              ghost
              icon={<CheckCircleOutlined />}
              onClick={() => {
                setAuditModal(r);
                auditForm.resetFields();
                auditForm.setFieldsValue({ result: 'pass' });
              }}
              data-testid={`btn-audit-${r.id}`}
            >
              审核
            </Button>
          )}
          {canAudit && (
            <Button
              size="small"
              icon={<MessageOutlined />}
              onClick={() => {
                setReplyModal(r);
                replyForm.resetFields();
                if (r.admin_reply) {
                  replyForm.setFieldsValue({ reply: r.admin_reply });
                }
              }}
              data-testid={`btn-reply-${r.id}`}
            >
              {r.admin_reply ? '改回复' : '回复'}
            </Button>
          )}
          <Button
            size="small"
            danger
            icon={<EyeInvisibleOutlined />}
            onClick={() => {
              setAuditModal(r);
              auditForm.resetFields();
              auditForm.setFieldsValue({ result: 'hide' });
            }}
            disabled={!canAudit}
            style={canAudit ? undefined : { display: 'none' }}
            data-testid={`btn-hide-${r.id}`}
          >
            隐藏
          </Button>
        </Space>
      ),
    },
  ];

  const onSubmitAudit = async () => {
    try {
      const values = await auditForm.validateFields();
      if (!auditModal) return;
      auditMut.mutate(
        { id: auditModal.id, result: values.result, reason: values.reason },
        {
          onSuccess: () => setAuditModal(null),
        },
      );
    } catch {
      // silent
    }
  };

  const onSubmitReply = async () => {
    try {
      const values = await replyForm.validateFields();
      if (!replyModal) return;
      replyMut.mutate(
        { id: replyModal.id, reply: values.reply },
        {
          onSuccess: () => setReplyModal(null),
        },
      );
    } catch {
      // silent
    }
  };

  return (
    <div data-testid="reviews-page">
      <PageHeader
        title="评价管理"
        subtitle="评价列表 · 评分筛选 · 审核 / 回复"
      />

      <div style={{ marginBottom: 16, display: 'flex', gap: 12, flexWrap: 'wrap' }}>
        <div data-testid="rating-filter-wrap">
          <InputNumber
            min={0}
            max={5}
            value={minRating || undefined}
            onChange={(v) => setMinRating(typeof v === 'number' ? v : 0)}
            placeholder="最低评分"
            style={{ width: 140 }}
            data-testid="rating-filter"
          />
        </div>
        <div data-testid="escort-id-wrap">
          <Input
            type="number"
            placeholder="陪诊师 ID"
            value={escortId ?? ''}
            onChange={(e) => {
              const v = e.target.value.trim();
              setEscortId(v ? Number(v) : undefined);
            }}
            style={{ width: 140 }}
            data-testid="escort-id-input"
          />
        </div>
        <span style={{ color: '#999', alignSelf: 'center' }}>
          共 {data?.total ?? 0} 条
        </span>
      </div>

      <ProTable<ReviewListItem>
        testId="reviews-table"
        rowKey="id"
        columns={columns}
        dataSource={items}
        loading={isLoading}
        density="middle"
        scroll={{ x: 1300 }}
      />

      {isError && (
        <div data-testid="reviews-error" style={{ color: '#f5222d', marginTop: 8 }}>
          加载失败：{(error as Error)?.message ?? '未知错误'}
        </div>
      )}

      {/* 审核 Modal */}
      <Modal
        title={`审核评价 #${auditModal?.id ?? '—'}`}
        open={auditModal != null}
        onCancel={() => setAuditModal(null)}
        onOk={onSubmitAudit}
        confirmLoading={auditMut.isPending}
        okButtonProps={{ 'data-testid': 'btn-audit-confirm' }}
        cancelButtonProps={{ 'data-testid': 'btn-audit-cancel' }}
        destroyOnClose
      >
        <Form form={auditForm} layout="vertical" preserve={false}>
          <Form.Item
            label="审核结果"
            name="result"
            rules={[{ required: true, message: '请选择审核结果' }]}
          >
            <Radio.Group data-testid="audit-result-radio">
              <Radio value="pass">通过</Radio>
              <Radio value="hide">隐藏</Radio>
            </Radio.Group>
          </Form.Item>
          <Form.Item label="理由（可选）" name="reason">
            <Input.TextArea
              rows={3}
              placeholder="例：评价内容违反社区规范"
              data-testid="audit-reason-input"
              maxLength={200}
              showCount
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* 回复 Modal */}
      <Modal
        title={`回复评价 #${replyModal?.id ?? '—'}`}
        open={replyModal != null}
        onCancel={() => setReplyModal(null)}
        onOk={onSubmitReply}
        confirmLoading={replyMut.isPending}
        okButtonProps={{ 'data-testid': 'btn-reply-confirm' }}
        cancelButtonProps={{ 'data-testid': 'btn-reply-cancel' }}
        destroyOnClose
      >
        <Form form={replyForm} layout="vertical" preserve={false}>
          <Form.Item
            label="回复内容"
            name="reply"
            rules={[{ required: true, message: '请输入回复内容' }]}
          >
            <Input.TextArea
              rows={4}
              placeholder="感谢您的评价，我们会持续改进"
              data-testid="reply-content-input"
              maxLength={300}
              showCount
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
