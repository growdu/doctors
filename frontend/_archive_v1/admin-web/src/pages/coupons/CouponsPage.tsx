/**
 * CouponsPage：admin-web 优惠券管理页（升级版）。
 *
 * 功能：
 *   - 优惠券列表（按 type=amount_off|discount 过滤）；
 *   - 创建优惠券（弹 Modal）；
 *   - 停用优惠券（弹 confirm modal 确认）；
 *   - viewer 不显示任何操作按钮（RBAC）。
 *
 * 数据流：
 *   - list:    GET  /api/v1/admin/coupons?type=
 *   - create:  POST /api/v1/admin/coupons
 *   - disable: POST /api/v1/admin/coupons/:id/disable
 *
 * 对应 spec：2026-09-24-admin-web-design.md §Task 25
 */
import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Button,
  DatePicker,
  Form,
  Input,
  InputNumber,
  message,
  Modal,
  Select,
  Space,
  Tag,
} from 'antd';
import { PlusOutlined, StopOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { PageHeader } from '@/components/PageHeader';
import { ProTable } from '@/components/ProTable';
import {
  createCoupon,
  disableCoupon,
  fetchCoupons,
  couponQueryKeys,
  type CouponItem,
  type CouponType,
} from '@/api/admin/coupons';
import { useAuthStore } from '@/stores/authStore';

interface CreateFormValues {
  name: string;
  type: CouponType;
  value: number;
  valid_until: dayjs.Dayjs;
}

const TYPE_LABEL: Record<CouponType, string> = {
  amount_off: '满减',
  discount: '折扣',
};

const TYPE_COLOR: Record<CouponType, string> = {
  amount_off: 'orange',
  discount: 'purple',
};

export default function CouponsPage() {
  const qc = useQueryClient();
  const role = useAuthStore((s) => s.role);
  const canManage = role === 'super_admin' || role === 'order_admin';

  const [typeFilter, setTypeFilter] = useState<string | undefined>();

  const [createOpen, setCreateOpen] = useState(false);
  const [createForm] = Form.useForm<CreateFormValues>();

  const [disableTarget, setDisableTarget] = useState<CouponItem | null>(null);

  const { data, isLoading, isError, error } = useQuery({
    queryKey: couponQueryKeys.list({ type: typeFilter }),
    queryFn: () => fetchCoupons({ type: typeFilter }),
    refetchOnWindowFocus: false,
  });

  const createMut = useMutation({
    mutationFn: (body: {
      name: string;
      type: CouponType;
      value: number;
      valid_until: string;
    }) => createCoupon(body),
    onSuccess: (r) => {
      message.success(`已创建优惠券 #${r.id}`);
      qc.invalidateQueries({ queryKey: ['coupons'] });
    },
    onError: (e: Error) => message.error(`创建失败：${e.message}`),
  });

  const disableMut = useMutation({
    mutationFn: (id: number) => disableCoupon(id),
    onSuccess: (r) => {
      message.success(`已停用优惠券 #${r.id}`);
      qc.invalidateQueries({ queryKey: ['coupons'] });
    },
    onError: (e: Error) => message.error(`停用失败：${e.message}`),
  });

  const items: CouponItem[] = data?.data ?? [];

  const columns: ColumnsType<CouponItem> = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 80,
      render: (v: number) => <Tag>#{v}</Tag>,
    },
    { title: '名称', dataIndex: 'name', width: 240 },
    {
      title: '类型',
      dataIndex: 'type',
      width: 100,
      render: (v: CouponType) => (
        <Tag color={TYPE_COLOR[v]}>{TYPE_LABEL[v]}</Tag>
      ),
    },
    {
      title: '面值 / 折扣',
      dataIndex: 'value',
      width: 130,
      align: 'right',
      render: (v: number, r: CouponItem) =>
        r.type === 'amount_off' ? <Tag color="orange">¥{v}</Tag> : (
          <Tag color="purple">{(v * 100).toFixed(0)} 折</Tag>
        ),
    },
    {
      title: '有效期',
      dataIndex: 'valid_until',
      width: 170,
      render: (v: string) => dayjs(v).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: '状态',
      dataIndex: 'disabled',
      width: 100,
      render: (v: boolean | undefined) =>
        v ? <Tag color="red">已停用</Tag> : <Tag color="green">启用中</Tag>,
    },
    {
      title: '操作',
      width: 110,
      fixed: 'right',
      render: (_: unknown, r: CouponItem) =>
        canManage && !r.disabled && (
          <Button
            size="small"
            danger
            icon={<StopOutlined />}
            onClick={() => setDisableTarget(r)}
            data-testid={`btn-disable-${r.id}`}
          >
            停用
          </Button>
        ),
    },
  ];

  const onSubmitCreate = async () => {
    try {
      const values = await createForm.validateFields();
      createMut.mutate(
        {
          name: values.name,
          type: values.type,
          value: values.value,
          valid_until: values.valid_until.toISOString(),
        },
        {
          onSuccess: () => {
            setCreateOpen(false);
            createForm.resetFields();
          },
        },
      );
    } catch {
      // ignore
    }
  };

  const onConfirmDisable = () => {
    if (!disableTarget) return;
    disableMut.mutate(disableTarget.id, {
      onSuccess: () => setDisableTarget(null),
    });
  };

  return (
    <div data-testid="coupons-page">
      <PageHeader
        title="优惠券管理"
        subtitle="卡券模板 + 创建 / 停用 + 类型筛选"
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
              新增卡券
            </Button>
          )
        }
      />

      <Space style={{ marginBottom: 16 }}>
        <Select
          placeholder="类型"
          allowClear
          style={{ width: 140 }}
          value={typeFilter}
          onChange={setTypeFilter}
          data-testid="filter-type"
          options={[
            { value: 'amount_off', label: TYPE_LABEL.amount_off },
            { value: 'discount', label: TYPE_LABEL.discount },
          ]}
        />
        <span style={{ color: '#999' }}>共 {data?.total ?? 0} 条</span>
      </Space>

      <ProTable<CouponItem>
        testId="coupons-table"
        rowKey="id"
        columns={columns}
        dataSource={items}
        loading={isLoading}
        density="middle"
        scroll={{ x: 1000 }}
      />

      {isError && (
        <div data-testid="coupons-error" style={{ color: '#f5222d', marginTop: 8 }}>
          加载失败：{(error as Error)?.message ?? '未知错误'}
        </div>
      )}

      {/* 创建 Modal */}
      <Modal
        title="新增卡券"
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
            label="卡券名称"
            name="name"
            rules={[{ required: true, message: '请输入卡券名称' }]}
          >
            <Input
              placeholder="例：新人 50 元券"
              data-testid="create-name"
              maxLength={40}
            />
          </Form.Item>
          <Form.Item
            label="类型"
            name="type"
            rules={[{ required: true, message: '请选择类型' }]}
          >
            <Select
              data-testid="create-type"
              options={[
                { value: 'amount_off', label: TYPE_LABEL.amount_off },
                { value: 'discount', label: TYPE_LABEL.discount },
              ]}
            />
          </Form.Item>
          <Form.Item
            noStyle
            shouldUpdate={(prev, cur) => prev.type !== cur.type}
          >
            {({ getFieldValue }) => {
              const t = getFieldValue('type');
              return (
                <Form.Item
                  label={t === 'discount' ? '折扣（0~1，例如 0.8 表示 8 折）' : '面值（元）'}
                  name="value"
                  rules={[{ required: true, message: '请输入面值 / 折扣' }]}
                >
                  <InputNumber
                    min={t === 'discount' ? 0.01 : 1}
                    max={t === 'discount' ? 0.99 : undefined}
                    step={t === 'discount' ? 0.05 : 1}
                    style={{ width: '100%' }}
                    data-testid="create-value"
                  />
                </Form.Item>
              );
            }}
          </Form.Item>
          <Form.Item
            label="有效期至"
            name="valid_until"
            rules={[{ required: true, message: '请选择有效期' }]}
          >
            <DatePicker
              style={{ width: '100%' }}
              data-testid="create-valid-until"
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* 停用确认 Modal */}
      <Modal
        title={`停用卡券 #${disableTarget?.id ?? '—'}`}
        open={!!disableTarget}
        onCancel={() => setDisableTarget(null)}
        onOk={onConfirmDisable}
        confirmLoading={disableMut.isPending}
        okButtonProps={{ danger: true, 'data-testid': 'btn-disable-confirm' }}
        cancelButtonProps={{ 'data-testid': 'btn-disable-cancel' }}
        destroyOnClose
      >
        <p>确认停用卡券「{disableTarget?.name}」？停用后用户已领取的卡券仍然有效。</p>
      </Modal>
    </div>
  );
}