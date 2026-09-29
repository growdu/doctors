/**
 * WorkOrdersPage：admin-web 客服工单列表页（升级版）。
 *
 * 功能：
 *   - 工单列表（按 status / category / priority 过滤）；
 *   - 行操作：分配客服 / 关闭工单（弹 Modal 收集处理意见）；
 *   - 创建工单（弹 Modal 收集 3 字段）；
 *   - viewer 角色不显示操作按钮（RBAC）。
 *
 * 数据流：
 *   - list:    GET /api/v1/admin/work-orders?status=&category=&priority=
 *   - create:  POST /api/v1/admin/work-orders
 *   - assign:  POST /api/v1/admin/work-orders/:id/assign
 *   - close:   POST /api/v1/admin/work-orders/:id/close
 *
 * 关键技术：
 *   - TanStack Query + invalidate；
 *   - antd Modal + Form；
 *   - antd message 反馈。
 *
 * 对应 spec：2026-09-24-admin-web-design.md §Task 15
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
import { PlusOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { PageHeader } from '@/components/PageHeader';
import { ProTable } from '@/components/ProTable';
import { StatusBadge } from '@/components/StatusBadge';
import {
  assignWorkOrder,
  closeWorkOrder,
  createWorkOrder,
  fetchWorkOrders,
  workOrderQueryKeys,
  type WorkOrderCategory,
  type WorkOrderItem,
  type WorkOrderPriority,
  type WorkOrderStatus,
} from '@/api/admin/work_orders';
import { useAuthStore } from '@/stores/authStore';

interface CreateFormValues {
  category: WorkOrderCategory;
  subject: string;
  priority: WorkOrderPriority;
}

interface AssignFormValues {
  assignee: string;
}

interface CloseFormValues {
  resolution: string;
}

const STATUS_LABEL: Record<WorkOrderStatus, string> = {
  open: '待处理',
  in_progress: '处理中',
  closed: '已关闭',
};

const CATEGORY_LABEL: Record<WorkOrderCategory, string> = {
  complaint: '投诉',
  appeal: '申诉',
  inquiry: '咨询',
};

const PRIORITY_LABEL: Record<WorkOrderPriority, string> = {
  low: '低',
  medium: '中',
  high: '高',
};

const PRIORITY_COLOR: Record<WorkOrderPriority, string> = {
  low: 'default',
  medium: 'blue',
  high: 'red',
};

export default function WorkOrdersPage() {
  const qc = useQueryClient();
  const role = useAuthStore((s) => s.role);
  /** 仅 super_admin / order_admin / refund_admin / cs 可操作 */
  const canManage =
    role === 'super_admin' ||
    role === 'order_admin' ||
    role === 'refund_admin' ||
    role === 'cs';

  const [statusFilter, setStatusFilter] = useState<string | undefined>();
  const [categoryFilter, setCategoryFilter] = useState<string | undefined>();
  const [priorityFilter, setPriorityFilter] = useState<string | undefined>();

  const [createOpen, setCreateOpen] = useState(false);
  const [createForm] = Form.useForm<CreateFormValues>();

  const [assignOpen, setAssignOpen] = useState(false);
  const [assignTarget, setAssignTarget] = useState<WorkOrderItem | null>(null);
  const [assignForm] = Form.useForm<AssignFormValues>();

  const [closeOpen, setCloseOpen] = useState(false);
  const [closeTarget, setCloseTarget] = useState<WorkOrderItem | null>(null);
  const [closeForm] = Form.useForm<CloseFormValues>();

  // ── 查询 ──────────────────────────────────────────────────────────
  const { data, isLoading, isError, error } = useQuery({
    queryKey: workOrderQueryKeys.list({
      status: statusFilter,
      category: categoryFilter,
      priority: priorityFilter,
    }),
    queryFn: () =>
      fetchWorkOrders({
        status: statusFilter,
        category: categoryFilter,
        priority: priorityFilter,
      }),
    refetchOnWindowFocus: false,
  });

  // ── 创建 ──────────────────────────────────────────────────────────
  const createMut = useMutation({
    mutationFn: (body: CreateFormValues) =>
      createWorkOrder({
        category: body.category,
        subject: body.subject,
        priority: body.priority,
      }),
    onSuccess: (r) => {
      message.success(`已创建工单 #${r.id}`);
      qc.invalidateQueries({ queryKey: ['work-orders'] });
    },
    onError: (e: Error) => {
      message.error(`创建失败：${e.message}`);
    },
  });

  // ── 分配 ──────────────────────────────────────────────────────────
  const assignMut = useMutation({
    mutationFn: ({ id, assignee }: { id: number; assignee: string }) =>
      assignWorkOrder(id, assignee),
    onSuccess: (r) => {
      message.success(`已分配工单 #${r.id}`);
      qc.invalidateQueries({ queryKey: ['work-orders'] });
    },
    onError: (e: Error) => {
      message.error(`分配失败：${e.message}`);
    },
  });

  // ── 关闭 ──────────────────────────────────────────────────────────
  const closeMut = useMutation({
    mutationFn: ({ id, resolution }: { id: number; resolution: string }) =>
      closeWorkOrder(id, resolution),
    onSuccess: (r) => {
      message.success(`已关闭工单 #${r.id}`);
      qc.invalidateQueries({ queryKey: ['work-orders'] });
    },
    onError: (e: Error) => {
      message.error(`关闭失败：${e.message}`);
    },
  });

  const items: WorkOrderItem[] = data?.data ?? [];

  // ── 列定义 ────────────────────────────────────────────────────────
  const columns: ColumnsType<WorkOrderItem> = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 80,
      render: (v: number) => <Tag>#{v}</Tag>,
    },
    {
      title: '类别',
      dataIndex: 'category',
      width: 90,
      render: (v: WorkOrderCategory) => <Tag>{CATEGORY_LABEL[v]}</Tag>,
    },
    { title: '主题', dataIndex: 'subject', width: 240 },
    {
      title: '优先级',
      dataIndex: 'priority',
      width: 90,
      render: (v: WorkOrderPriority) => (
        <Tag color={PRIORITY_COLOR[v]}>{PRIORITY_LABEL[v]}</Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 110,
      render: (_: unknown, r: WorkOrderItem) => (
        <StatusBadge status={r.status} testId={`row-status-${r.id}`} />
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      width: 170,
      render: (v: string) => dayjs(v).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: '操作',
      width: 220,
      fixed: 'right',
      render: (_: unknown, r: WorkOrderItem) => (
        <Space size="small">
          {canManage && r.status !== 'closed' && (
            <Button
              size="small"
              onClick={() => {
                setAssignTarget(r);
                assignForm.resetFields();
                setAssignOpen(true);
              }}
              data-testid={`btn-assign-${r.id}`}
            >
              分配
            </Button>
          )}
          {canManage && r.status !== 'closed' && (
            <Button
              size="small"
              danger
              onClick={() => {
                setCloseTarget(r);
                closeForm.resetFields();
                setCloseOpen(true);
              }}
              data-testid={`btn-close-${r.id}`}
            >
              关闭
            </Button>
          )}
        </Space>
      ),
    },
  ];

  // ── 提交各 Modal ──────────────────────────────────────────────────
  const onSubmitCreate = async () => {
    try {
      const values = await createForm.validateFields();
      createMut.mutate(values, {
        onSuccess: () => {
          setCreateOpen(false);
          createForm.resetFields();
        },
      });
    } catch {
      // antd 校验未通过
    }
  };

  const onSubmitAssign = async () => {
    try {
      const values = await assignForm.validateFields();
      if (!assignTarget) return;
      assignMut.mutate(
        { id: assignTarget.id, assignee: values.assignee },
        {
          onSuccess: () => {
            setAssignOpen(false);
            setAssignTarget(null);
          },
        },
      );
    } catch {
      // ignore
    }
  };

  const onSubmitClose = async () => {
    try {
      const values = await closeForm.validateFields();
      if (!closeTarget) return;
      closeMut.mutate(
        { id: closeTarget.id, resolution: values.resolution },
        {
          onSuccess: () => {
            setCloseOpen(false);
            setCloseTarget(null);
          },
        },
      );
    } catch {
      // ignore
    }
  };

  return (
    <div data-testid="work-orders-page">
      <PageHeader
        title="客服工单"
        subtitle="投诉 / 申诉 / 咨询 + 状态 / 优先级 / 类别 多维筛选"
        extra={
          canManage && (
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => {
                createForm.resetFields();
                setCreateOpen(true);
              }}
              data-testid="btn-create"
            >
              新建工单
            </Button>
          )
        }
      />

      <Space style={{ marginBottom: 16 }} wrap>
        <Select
          placeholder="状态"
          allowClear
          style={{ width: 140 }}
          value={statusFilter}
          onChange={setStatusFilter}
          data-testid="filter-status"
          options={[
            { value: 'open', label: STATUS_LABEL.open },
            { value: 'in_progress', label: STATUS_LABEL.in_progress },
            { value: 'closed', label: STATUS_LABEL.closed },
          ]}
        />
        <Select
          placeholder="类别"
          allowClear
          style={{ width: 140 }}
          value={categoryFilter}
          onChange={setCategoryFilter}
          data-testid="filter-category"
          options={[
            { value: 'complaint', label: CATEGORY_LABEL.complaint },
            { value: 'appeal', label: CATEGORY_LABEL.appeal },
            { value: 'inquiry', label: CATEGORY_LABEL.inquiry },
          ]}
        />
        <Select
          placeholder="优先级"
          allowClear
          style={{ width: 140 }}
          value={priorityFilter}
          onChange={setPriorityFilter}
          data-testid="filter-priority"
          options={[
            { value: 'high', label: PRIORITY_LABEL.high },
            { value: 'medium', label: PRIORITY_LABEL.medium },
            { value: 'low', label: PRIORITY_LABEL.low },
          ]}
        />
        <span style={{ color: '#999' }}>共 {data?.total ?? 0} 条</span>
      </Space>

      <ProTable<WorkOrderItem>
        testId="work-orders-table"
        rowKey="id"
        columns={columns}
        dataSource={items}
        loading={isLoading}
        density="middle"
        scroll={{ x: 1100 }}
      />

      {isError && (
        <div data-testid="work-orders-error" style={{ color: '#f5222d', marginTop: 8 }}>
          加载失败：{(error as Error)?.message ?? '未知错误'}
        </div>
      )}

      {/* 创建工单 Modal */}
      <Modal
        title="新建工单"
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        onOk={onSubmitCreate}
        confirmLoading={createMut.isPending}
        okButtonProps={{ 'data-testid': 'btn-create-confirm' }}
        cancelButtonProps={{ 'data-testid': 'btn-create-cancel' }}
        destroyOnClose
      >
        <Form form={createForm} layout="vertical" preserve={false}>
          <Form.Item
            label="类别"
            name="category"
            rules={[{ required: true, message: '请选择类别' }]}
          >
            <Select
              placeholder="投诉 / 申诉 / 咨询"
              data-testid="create-category"
              options={[
                { value: 'complaint', label: CATEGORY_LABEL.complaint },
                { value: 'appeal', label: CATEGORY_LABEL.appeal },
                { value: 'inquiry', label: CATEGORY_LABEL.inquiry },
              ]}
            />
          </Form.Item>
          <Form.Item
            label="主题"
            name="subject"
            rules={[{ required: true, message: '请输入主题' }]}
          >
            <Input
              placeholder="例：陪诊师迟到 30 分钟"
              data-testid="create-subject"
              maxLength={80}
            />
          </Form.Item>
          <Form.Item
            label="优先级"
            name="priority"
            rules={[{ required: true, message: '请选择优先级' }]}
          >
            <Select
              placeholder="高 / 中 / 低"
              data-testid="create-priority"
              options={[
                { value: 'high', label: PRIORITY_LABEL.high },
                { value: 'medium', label: PRIORITY_LABEL.medium },
                { value: 'low', label: PRIORITY_LABEL.low },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* 分配 Modal */}
      <Modal
        title={`分配工单 #${assignTarget?.id ?? '—'}`}
        open={assignOpen}
        onCancel={() => {
          setAssignOpen(false);
          setAssignTarget(null);
        }}
        onOk={onSubmitAssign}
        confirmLoading={assignMut.isPending}
        okButtonProps={{ 'data-testid': 'btn-assign-confirm' }}
        cancelButtonProps={{ 'data-testid': 'btn-assign-cancel' }}
        destroyOnClose
      >
        <Form form={assignForm} layout="vertical" preserve={false}>
          <Form.Item
            label="处理人"
            name="assignee"
            rules={[{ required: true, message: '请输入处理人' }]}
          >
            <Input
              placeholder="例：cs_zhang"
              data-testid="assign-input"
              maxLength={40}
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* 关闭 Modal */}
      <Modal
        title={`关闭工单 #${closeTarget?.id ?? '—'}`}
        open={closeOpen}
        onCancel={() => {
          setCloseOpen(false);
          setCloseTarget(null);
        }}
        onOk={onSubmitClose}
        confirmLoading={closeMut.isPending}
        okButtonProps={{ danger: true, 'data-testid': 'btn-close-confirm' }}
        cancelButtonProps={{ 'data-testid': 'btn-close-cancel' }}
        destroyOnClose
      >
        <Form form={closeForm} layout="vertical" preserve={false}>
          <Form.Item
            label="处理结论"
            name="resolution"
            rules={[{ required: true, message: '请输入处理结论' }]}
          >
            <Input.TextArea
              rows={4}
              placeholder="例：已联系陪诊师补赔差价，患者接受"
              data-testid="close-resolution-input"
              maxLength={200}
              showCount
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}