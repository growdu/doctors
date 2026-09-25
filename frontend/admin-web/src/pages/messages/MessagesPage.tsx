/**
 * MessagesPage：admin-web 站内信消息中心页（升级版）。
 *
 * 功能：
 *   - 消息列表（按 category 过滤：系统 / 公告 / 工单）；
 *   - 行操作：发送站内信（指定 user_id）/ 全体广播；
 *   - viewer 角色不显示发送 / 广播按钮（RBAC）。
 *
 * 数据流：
 *   - list:      GET /api/v1/admin/messages?category=
 *   - send:      POST /api/v1/admin/messages
 *   - broadcast: POST /api/v1/admin/messages/broadcast
 *
 * 对应 spec：2026-09-24-admin-web-design.md §Task 17
 */
import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Button,
  Form,
  Input,
  InputNumber,
  message,
  Modal,
  Select,
  Space,
  Tag,
} from 'antd';
import { NotificationOutlined, SendOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { PageHeader } from '@/components/PageHeader';
import { ProTable } from '@/components/ProTable';
import {
  broadcastMessage,
  fetchMessages,
  messageQueryKeys,
  sendMessage,
  type MessageCategory,
  type MessageItem,
} from '@/api/admin/messages';
import { useAuthStore } from '@/stores/authStore';

interface SendFormValues {
  target_user_id: number;
  title: string;
  category: MessageCategory;
}

interface BroadcastFormValues {
  title: string;
  category: MessageCategory;
}

const CATEGORY_LABEL: Record<MessageCategory, string> = {
  system: '系统通知',
  announcement: '公告',
  work_order: '工单消息',
};

const CATEGORY_COLOR: Record<MessageCategory, string> = {
  system: 'blue',
  announcement: 'orange',
  work_order: 'purple',
};

export default function MessagesPage() {
  const qc = useQueryClient();
  const role = useAuthStore((s) => s.role);
  /** 仅 super_admin / cs 可发送 / 广播 */
  const canSend = role === 'super_admin' || role === 'cs';

  const [categoryFilter, setCategoryFilter] = useState<string | undefined>();

  const [sendOpen, setSendOpen] = useState(false);
  const [sendForm] = Form.useForm<SendFormValues>();

  const [broadcastOpen, setBroadcastOpen] = useState(false);
  const [broadcastForm] = Form.useForm<BroadcastFormValues>();

  // ── 查询 ──────────────────────────────────────────────────────────
  const { data, isLoading, isError, error } = useQuery({
    queryKey: messageQueryKeys.list({ category: categoryFilter }),
    queryFn: () => fetchMessages({ category: categoryFilter }),
    refetchOnWindowFocus: false,
  });

  // ── 发送 ──────────────────────────────────────────────────────────
  const sendMut = useMutation({
    mutationFn: (body: SendFormValues) =>
      sendMessage({
        target_user_id: body.target_user_id,
        title: body.title,
        category: body.category,
      }),
    onSuccess: (r) => {
      message.success(`已发送站内信 #${r.id}`);
      qc.invalidateQueries({ queryKey: ['messages'] });
    },
    onError: (e: Error) => {
      message.error(`发送失败：${e.message}`);
    },
  });

  // ── 广播 ──────────────────────────────────────────────────────────
  const broadcastMut = useMutation({
    mutationFn: (body: BroadcastFormValues) =>
      broadcastMessage({ title: body.title, category: body.category }),
    onSuccess: (r) => {
      message.success(`已广播消息 #${r.id}`);
      qc.invalidateQueries({ queryKey: ['messages'] });
    },
    onError: (e: Error) => {
      message.error(`广播失败：${e.message}`);
    },
  });

  const items: MessageItem[] = data?.data ?? [];

  const columns: ColumnsType<MessageItem> = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 80,
      render: (v: number) => <Tag>#{v}</Tag>,
    },
    {
      title: '类型',
      dataIndex: 'category',
      width: 110,
      render: (v: MessageCategory) => (
        <Tag color={CATEGORY_COLOR[v]}>{CATEGORY_LABEL[v]}</Tag>
      ),
    },
    { title: '标题', dataIndex: 'title', width: 360 },
    {
      title: '受众',
      dataIndex: 'audience',
      width: 100,
      render: (v: string | undefined, r: MessageItem) => {
        if (v === 'all') return <Tag color="red">全员</Tag>;
        if (r.target_user_id != null) return <Tag>#{r.target_user_id}</Tag>;
        return <Tag color="default">—</Tag>;
      },
    },
    {
      title: '状态',
      dataIndex: 'read',
      width: 90,
      render: (v: boolean) =>
        v ? <Tag color="green">已读</Tag> : <Tag color="orange">未读</Tag>,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      width: 170,
      render: (v: string) => dayjs(v).format('YYYY-MM-DD HH:mm:ss'),
    },
  ];

  // ── 提交 Modal ────────────────────────────────────────────────────
  const onSubmitSend = async () => {
    try {
      const values = await sendForm.validateFields();
      sendMut.mutate(values, {
        onSuccess: () => {
          setSendOpen(false);
          sendForm.resetFields();
        },
      });
    } catch {
      // ignore
    }
  };

  const onSubmitBroadcast = async () => {
    try {
      const values = await broadcastForm.validateFields();
      broadcastMut.mutate(values, {
        onSuccess: () => {
          setBroadcastOpen(false);
          broadcastForm.resetFields();
        },
      });
    } catch {
      // ignore
    }
  };

  return (
    <div data-testid="messages-page">
      <PageHeader
        title="消息中心"
        subtitle="系统通知 / 公告 / 工单消息 + 发送 / 广播"
        extra={
          canSend && (
            <Space>
              <Button
                icon={<SendOutlined />}
                onClick={() => {
                  sendForm.resetFields();
                  setSendOpen(true);
                }}
                data-testid="btn-send"
              >
                发送站内信
              </Button>
              <Button
                type="primary"
                icon={<NotificationOutlined />}
                onClick={() => {
                  broadcastForm.resetFields();
                  setBroadcastOpen(true);
                }}
                data-testid="btn-broadcast"
              >
                全体广播
              </Button>
            </Space>
          )
        }
      />

      <Space style={{ marginBottom: 16 }}>
        <Select
          placeholder="类型"
          allowClear
          style={{ width: 160 }}
          value={categoryFilter}
          onChange={setCategoryFilter}
          data-testid="filter-category"
          options={[
            { value: 'system', label: CATEGORY_LABEL.system },
            { value: 'announcement', label: CATEGORY_LABEL.announcement },
            { value: 'work_order', label: CATEGORY_LABEL.work_order },
          ]}
        />
        <span style={{ color: '#999' }}>共 {data?.total ?? 0} 条</span>
      </Space>

      <ProTable<MessageItem>
        testId="messages-table"
        rowKey="id"
        columns={columns}
        dataSource={items}
        loading={isLoading}
        density="middle"
        scroll={{ x: 1100 }}
      />

      {isError && (
        <div data-testid="messages-error" style={{ color: '#f5222d', marginTop: 8 }}>
          加载失败：{(error as Error)?.message ?? '未知错误'}
        </div>
      )}

      {/* 发送站内信 */}
      <Modal
        title="发送站内信"
        open={sendOpen}
        onCancel={() => setSendOpen(false)}
        onOk={onSubmitSend}
        confirmLoading={sendMut.isPending}
        okButtonProps={{ 'data-testid': 'btn-send-confirm' }}
        cancelButtonProps={{ 'data-testid': 'btn-send-cancel' }}
        destroyOnClose
      >
        <Form form={sendForm} layout="vertical" preserve={false}>
          <Form.Item
            label="目标用户 ID"
            name="target_user_id"
            rules={[{ required: true, message: '请输入用户 ID' }]}
          >
            <InputNumber
              placeholder="例：7001"
              data-testid="send-target-input"
              style={{ width: '100%' }}
              min={1}
            />
          </Form.Item>
          <Form.Item
            label="消息标题"
            name="title"
            rules={[{ required: true, message: '请输入标题' }]}
          >
            <Input
              placeholder="例：您的订单已退款"
              data-testid="send-title-input"
              maxLength={80}
            />
          </Form.Item>
          <Form.Item
            label="类型"
            name="category"
            rules={[{ required: true, message: '请选择类型' }]}
          >
            <Select
              placeholder="系统 / 公告 / 工单"
              data-testid="send-category-select"
              options={[
                { value: 'system', label: CATEGORY_LABEL.system },
                { value: 'announcement', label: CATEGORY_LABEL.announcement },
                { value: 'work_order', label: CATEGORY_LABEL.work_order },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* 全体广播 */}
      <Modal
        title="全体广播"
        open={broadcastOpen}
        onCancel={() => setBroadcastOpen(false)}
        onOk={onSubmitBroadcast}
        confirmLoading={broadcastMut.isPending}
        okButtonProps={{ danger: true, 'data-testid': 'btn-broadcast-confirm' }}
        cancelButtonProps={{ 'data-testid': 'btn-broadcast-cancel' }}
        destroyOnClose
      >
        <Form form={broadcastForm} layout="vertical" preserve={false}>
          <Form.Item
            label="广播标题"
            name="title"
            rules={[{ required: true, message: '请输入标题' }]}
          >
            <Input
              placeholder="例：9/25 系统维护通知"
              data-testid="broadcast-title-input"
              maxLength={80}
            />
          </Form.Item>
          <Form.Item
            label="类型"
            name="category"
            rules={[{ required: true, message: '请选择类型' }]}
          >
            <Select
              placeholder="系统 / 公告 / 工单"
              data-testid="broadcast-category-select"
              options={[
                { value: 'system', label: CATEGORY_LABEL.system },
                { value: 'announcement', label: CATEGORY_LABEL.announcement },
                { value: 'work_order', label: CATEGORY_LABEL.work_order },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}