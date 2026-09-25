/**
 * HospitalsPage：admin-web 医院字典管理页（升级版）。
 *
 * 功能：
 *   - 医院列表（按 city / status 过滤）；
 *   - 创建医院（弹 Modal）；
 *   - 编辑医院（弹 Modal）；
 *   - 切换状态 active/inactive；
 *   - viewer 不显示任何操作按钮（RBAC，仅 super_admin 可访问该路由）。
 *
 * 数据流：
 *   - list:    GET  /api/v1/admin/hospitals?city=&status=
 *   - create:  POST /api/v1/admin/hospitals
 *   - update:  PATCH /api/v1/admin/hospitals/:id
 *
 * 对应 spec：2026-09-24-admin-web-design.md §Task 26
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
import { EditOutlined, PlusOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { PageHeader } from '@/components/PageHeader';
import { ProTable } from '@/components/ProTable';
import {
  createHospital,
  fetchHospitals,
  hospitalQueryKeys,
  updateHospital,
  type HospitalItem,
  type HospitalLevel,
  type HospitalStatus,
} from '@/api/admin/hospitals';
import { useAuthStore } from '@/stores/authStore';

interface CreateFormValues {
  name: string;
  city: string;
  level: HospitalLevel;
}

interface EditFormValues {
  name: string;
  city: string;
  level: HospitalLevel;
  status: HospitalStatus;
}

const STATUS_LABEL: Record<HospitalStatus, string> = {
  active: '已启用',
  inactive: '已停用',
};

const STATUS_COLOR: Record<HospitalStatus, string> = {
  active: 'green',
  inactive: 'red',
};

export default function HospitalsPage() {
  const qc = useQueryClient();
  const role = useAuthStore((s) => s.role);
  const canManage = role === 'super_admin';

  const [cityFilter, setCityFilter] = useState<string | undefined>();
  const [statusFilter, setStatusFilter] = useState<string | undefined>();

  const [createOpen, setCreateOpen] = useState(false);
  const [createForm] = Form.useForm<CreateFormValues>();

  const [editOpen, setEditOpen] = useState(false);
  const [editTarget, setEditTarget] = useState<HospitalItem | null>(null);
  const [editForm] = Form.useForm<EditFormValues>();

  const { data, isLoading, isError, error } = useQuery({
    queryKey: hospitalQueryKeys.list({ city: cityFilter, status: statusFilter }),
    queryFn: () =>
      fetchHospitals({ city: cityFilter, status: statusFilter }),
    refetchOnWindowFocus: false,
  });

  const createMut = useMutation({
    mutationFn: (body: CreateFormValues) => createHospital(body),
    onSuccess: (r) => {
      message.success(`已创建医院 #${r.id}`);
      qc.invalidateQueries({ queryKey: ['hospitals'] });
    },
    onError: (e: Error) => message.error(`创建失败：${e.message}`),
  });

  const updateMut = useMutation({
    mutationFn: ({
      id,
      body,
    }: {
      id: number;
      body: EditFormValues;
    }) => updateHospital(id, body),
    onSuccess: (r) => {
      message.success(`已更新医院 #${r.id}`);
      qc.invalidateQueries({ queryKey: ['hospitals'] });
    },
    onError: (e: Error) => message.error(`更新失败：${e.message}`),
  });

  const items: HospitalItem[] = data?.data ?? [];

  const columns: ColumnsType<HospitalItem> = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 80,
      render: (v: number) => <Tag>#{v}</Tag>,
    },
    { title: '名称', dataIndex: 'name', width: 240 },
    { title: '城市', dataIndex: 'city', width: 120 },
    {
      title: '等级',
      dataIndex: 'level',
      width: 100,
      render: (v: HospitalLevel) => <Tag color="blue">{v}</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (v: HospitalStatus) => (
        <Tag color={STATUS_COLOR[v]}>{STATUS_LABEL[v]}</Tag>
      ),
    },
    {
      title: '操作',
      width: 100,
      fixed: 'right',
      render: (_: unknown, r: HospitalItem) =>
        canManage && (
          <Button
            size="small"
            icon={<EditOutlined />}
            onClick={() => {
              setEditTarget(r);
              editForm.setFieldsValue({
                name: r.name,
                city: r.city,
                level: r.level,
                status: r.status,
              });
              setEditOpen(true);
            }}
            data-testid={`btn-edit-${r.id}`}
          >
            编辑
          </Button>
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
    <div data-testid="hospitals-page">
      <PageHeader
        title="医院字典"
        subtitle="医院列表 + 创建 / 编辑 / 启用停用"
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
              新增医院
            </Button>
          )
        }
      />

      <Space style={{ marginBottom: 16 }}>
        <Input
          placeholder="城市"
          allowClear
          style={{ width: 160 }}
          value={cityFilter}
          onChange={(e) => setCityFilter(e.target.value || undefined)}
          data-testid="filter-city"
        />
        <Select
          placeholder="状态"
          allowClear
          style={{ width: 140 }}
          value={statusFilter}
          onChange={setStatusFilter}
          data-testid="filter-status"
          options={[
            { value: 'active', label: STATUS_LABEL.active },
            { value: 'inactive', label: STATUS_LABEL.inactive },
          ]}
        />
        <span style={{ color: '#999' }}>共 {data?.total ?? 0} 条</span>
      </Space>

      <ProTable<HospitalItem>
        testId="hospitals-table"
        rowKey="id"
        columns={columns}
        dataSource={items}
        loading={isLoading}
        density="middle"
        scroll={{ x: 900 }}
      />

      {isError && (
        <div data-testid="hospitals-error" style={{ color: '#f5222d', marginTop: 8 }}>
          加载失败：{(error as Error)?.message ?? '未知错误'}
        </div>
      )}

      {/* 创建 Modal */}
      <Modal
        title="新增医院"
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
            label="医院名称"
            name="name"
            rules={[{ required: true, message: '请输入医院名称' }]}
          >
            <Input placeholder="例：北京协和医院" data-testid="create-name" maxLength={40} />
          </Form.Item>
          <Form.Item
            label="城市"
            name="city"
            rules={[{ required: true, message: '请输入城市' }]}
          >
            <Input placeholder="例：北京" data-testid="create-city" maxLength={20} />
          </Form.Item>
          <Form.Item
            label="等级"
            name="level"
            rules={[{ required: true, message: '请选择等级' }]}
          >
            <Select
              placeholder="三甲 / 三乙 / 二甲"
              data-testid="create-level"
              options={[
                { value: '三甲', label: '三甲' },
                { value: '三乙', label: '三乙' },
                { value: '二甲', label: '二甲' },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* 编辑 Modal */}
      <Modal
        title={`编辑医院 #${editTarget?.id ?? '—'}`}
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
            label="医院名称"
            name="name"
            rules={[{ required: true, message: '请输入医院名称' }]}
          >
            <Input data-testid="edit-name" maxLength={40} />
          </Form.Item>
          <Form.Item
            label="城市"
            name="city"
            rules={[{ required: true, message: '请输入城市' }]}
          >
            <Input data-testid="edit-city" maxLength={20} />
          </Form.Item>
          <Form.Item
            label="等级"
            name="level"
            rules={[{ required: true, message: '请选择等级' }]}
          >
            <Select
              data-testid="edit-level"
              options={[
                { value: '三甲', label: '三甲' },
                { value: '三乙', label: '三乙' },
                { value: '二甲', label: '二甲' },
              ]}
            />
          </Form.Item>
          <Form.Item
            label="状态"
            name="status"
            rules={[{ required: true, message: '请选择状态' }]}
          >
            <Select
              data-testid="edit-status"
              options={[
                { value: 'active', label: STATUS_LABEL.active },
                { value: 'inactive', label: STATUS_LABEL.inactive },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}