/**
 * PackagesPage：admin-web 套餐管理页（升级版）。
 *
 * 功能：
 *   - 套餐列表（按 status=on|off 过滤）；
 *   - 创建套餐（弹 Modal）；
 *   - 编辑套餐（弹 Modal + 上下架开关）；
 *   - viewer 不显示任何操作按钮（RBAC）。
 *
 * 数据流：
 *   - list:    GET  /api/v1/admin/packages?status=
 *   - create:  POST /api/v1/admin/packages
 *   - update:  PATCH /api/v1/admin/packages/:id   （含 status 上下架）
 *
 * 对应 spec：2026-09-24-admin-web-design.md §Task 27
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
  Switch,
  Tag,
} from 'antd';
import { EditOutlined, PlusOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { PageHeader } from '@/components/PageHeader';
import { ProTable } from '@/components/ProTable';
import {
  createPackage,
  fetchPackages,
  packageQueryKeys,
  updatePackage,
  type PackageDuration,
  type PackageItem,
  type PackageStatus,
} from '@/api/admin/packages';
import { useAuthStore } from '@/stores/authStore';

interface CreateFormValues {
  name: string;
  price: number;
  duration: PackageDuration;
}

interface EditFormValues {
  name: string;
  price: number;
  duration: PackageDuration;
  status: PackageStatus;
}

const DURATION_LABEL: Record<PackageDuration, string> = {
  half_day: '半日',
  full_day: '全日',
};

const STATUS_LABEL: Record<PackageStatus, string> = {
  on: '上架',
  off: '下架',
};

const STATUS_COLOR: Record<PackageStatus, string> = {
  on: 'green',
  off: 'default',
};

export default function PackagesPage() {
  const qc = useQueryClient();
  const role = useAuthStore((s) => s.role);
  const canManage = role === 'super_admin' || role === 'order_admin';

  const [statusFilter, setStatusFilter] = useState<string | undefined>();

  const [createOpen, setCreateOpen] = useState(false);
  const [createForm] = Form.useForm<CreateFormValues>();

  const [editOpen, setEditOpen] = useState(false);
  const [editTarget, setEditTarget] = useState<PackageItem | null>(null);
  const [editForm] = Form.useForm<EditFormValues>();

  const { data, isLoading, isError, error } = useQuery({
    queryKey: packageQueryKeys.list({ status: statusFilter }),
    queryFn: () => fetchPackages({ status: statusFilter }),
    refetchOnWindowFocus: false,
  });

  const createMut = useMutation({
    mutationFn: (body: CreateFormValues) => createPackage(body),
    onSuccess: (r) => {
      message.success(`已创建套餐 #${r.id}`);
      qc.invalidateQueries({ queryKey: ['packages'] });
    },
    onError: (e: Error) => message.error(`创建失败：${e.message}`),
  });

  const updateMut = useMutation({
    mutationFn: ({ id, body }: { id: number; body: EditFormValues }) =>
      updatePackage(id, body),
    onSuccess: (r) => {
      message.success(`已更新套餐 #${r.id}`);
      qc.invalidateQueries({ queryKey: ['packages'] });
    },
    onError: (e: Error) => message.error(`更新失败：${e.message}`),
  });

  const items: PackageItem[] = data?.data ?? [];

  const columns: ColumnsType<PackageItem> = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 80,
      render: (v: number) => <Tag>#{v}</Tag>,
    },
    { title: '名称', dataIndex: 'name', width: 200 },
    {
      title: '价格',
      dataIndex: 'price',
      width: 110,
      align: 'right',
      render: (v: number) => <Tag color="orange">¥{v}</Tag>,
    },
    {
      title: '时长',
      dataIndex: 'duration',
      width: 90,
      render: (v: PackageDuration) => (
        <Tag color="blue">{DURATION_LABEL[v]}</Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 110,
      render: (v: PackageStatus) => (
        <Tag color={STATUS_COLOR[v]}>{STATUS_LABEL[v]}</Tag>
      ),
    },
    {
      title: '操作',
      width: 220,
      fixed: 'right',
      render: (_: unknown, r: PackageItem) =>
        canManage && (
          <Space size="small">
            <Button
              size="small"
              icon={<EditOutlined />}
              onClick={() => {
                setEditTarget(r);
                editForm.setFieldsValue({
                  name: r.name,
                  price: r.price,
                  duration: r.duration,
                  status: r.status,
                });
                setEditOpen(true);
              }}
              data-testid={`btn-edit-${r.id}`}
            >
              编辑
            </Button>
            <Button
              size="small"
              onClick={() => {
                updateMut.mutate({
                  id: r.id,
                  body: {
                    name: r.name,
                    price: r.price,
                    duration: r.duration,
                    status: r.status === 'on' ? 'off' : 'on',
                  },
                });
              }}
              data-testid={`btn-toggle-${r.id}`}
            >
              {r.status === 'on' ? '下架' : '上架'}
            </Button>
          </Space>
        ),
    },
  ];

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
      // ignore
    }
  };

  const onSubmitEdit = async () => {
    try {
      const values = await editForm.validateFields();
      if (!editTarget) return;
      updateMut.mutate(
        { id: editTarget.id, body: values },
        {
          onSuccess: () => {
            setEditOpen(false);
            setEditTarget(null);
          },
        },
      );
    } catch {
      // ignore
    }
  };

  return (
    <div data-testid="packages-page">
      <PageHeader
        title="套餐管理"
        subtitle="套餐列表 + 创建 / 编辑 / 上下架"
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
              新增套餐
            </Button>
          )
        }
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
            { value: 'on', label: STATUS_LABEL.on },
            { value: 'off', label: STATUS_LABEL.off },
          ]}
        />
        <span style={{ color: '#999' }}>共 {data?.total ?? 0} 条</span>
      </Space>

      <ProTable<PackageItem>
        testId="packages-table"
        rowKey="id"
        columns={columns}
        dataSource={items}
        loading={isLoading}
        density="middle"
        scroll={{ x: 900 }}
      />

      {isError && (
        <div data-testid="packages-error" style={{ color: '#f5222d', marginTop: 8 }}>
          加载失败：{(error as Error)?.message ?? '未知错误'}
        </div>
      )}

      {/* 创建 Modal */}
      <Modal
        title="新增套餐"
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
            label="套餐名称"
            name="name"
            rules={[{ required: true, message: '请输入套餐名称' }]}
          >
            <Input placeholder="例：半日陪诊" data-testid="create-name" maxLength={40} />
          </Form.Item>
          <Form.Item
            label="价格"
            name="price"
            rules={[{ required: true, message: '请输入价格' }]}
          >
            <InputNumber
              placeholder="例：300"
              data-testid="create-price"
              min={0}
              style={{ width: '100%' }}
            />
          </Form.Item>
          <Form.Item
            label="时长"
            name="duration"
            rules={[{ required: true, message: '请选择时长' }]}
          >
            <Select
              data-testid="create-duration"
              options={[
                { value: 'half_day', label: DURATION_LABEL.half_day },
                { value: 'full_day', label: DURATION_LABEL.full_day },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* 编辑 Modal */}
      <Modal
        title={`编辑套餐 #${editTarget?.id ?? '—'}`}
        open={editOpen}
        onCancel={() => {
          setEditOpen(false);
          setEditTarget(null);
        }}
        onOk={onSubmitEdit}
        confirmLoading={updateMut.isPending}
        okButtonProps={{ 'data-testid': 'btn-edit-confirm' }}
        cancelButtonProps={{ 'data-testid': 'btn-edit-cancel' }}
        destroyOnClose
      >
        <Form form={editForm} layout="vertical" preserve={false}>
          <Form.Item
            label="套餐名称"
            name="name"
            rules={[{ required: true, message: '请输入套餐名称' }]}
          >
            <Input data-testid="edit-name" maxLength={40} />
          </Form.Item>
          <Form.Item label="价格" name="price" rules={[{ required: true }]}>
            <InputNumber data-testid="edit-price" min={0} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label="时长" name="duration" rules={[{ required: true }]}>
            <Select
              data-testid="edit-duration"
              options={[
                { value: 'half_day', label: DURATION_LABEL.half_day },
                { value: 'full_day', label: DURATION_LABEL.full_day },
              ]}
            />
          </Form.Item>
          <Form.Item label="上架状态" name="status" valuePropName="checked" getValueFromEvent={(e) => e.target.checked ? 'on' : 'off'}>
            <Switch
              checkedChildren="上架"
              unCheckedChildren="下架"
              data-testid="edit-status-switch"
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}