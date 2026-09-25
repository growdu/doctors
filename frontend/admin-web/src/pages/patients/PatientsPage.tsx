/**
 * PatientsPage：admin-web 患者列表页（升级版）。
 *
 * 范围：
 *   - 患者列表（按 name / phone 关键词搜索，500ms 防抖）；
 *   - 行操作：详情（跳详情页）+ 封禁 / 解封；
 *   - 封禁弹 Modal 收集原因；
 *   - viewer 不显示封禁 / 解封按钮。
 *
 * 数据流：
 *   - list:      GET /api/v1/admin/patients?keyword=...
 *   - ban/unban: POST /api/v1/admin/patients/:id/{ban|unban}
 *
 * 关键技术：
 *   - useState 输入值 + useDeferredValue（debounce 替代；更轻）；
 *   - TanStack Query 拉取；
 *   - RBAC 控制按钮可见性。
 *
 * 对应 spec：2026-09-24-admin-web-design.md §Task 11
 */
import { useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Button,
  Form,
  Input,
  message,
  Modal,
  Space,
  Tag,
} from 'antd';
import { StopOutlined, CheckCircleOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { PageHeader } from '@/components/PageHeader';
import { ProTable } from '@/components/ProTable';
import {
  banPatient,
  fetchPatients,
  patientQueryKeys,
  unbanPatient,
  type PatientListItem,
} from '@/api/admin/patients';
import { useAuthStore } from '@/stores/authStore';

interface BanFormValues {
  reason: string;
}

export default function PatientsPage() {
  const navigate = useNavigate();
  const qc = useQueryClient();
  const role = useAuthStore((s) => s.role);
  const canBan = role === 'super_admin' || role === 'order_admin' || role === 'cs';

  // 防抖搜索：输入框本地立即更新，应用到 queryKey 时 500ms 延迟。
  const [keywordInput, setKeywordInput] = useState('');
  const [keyword, setKeyword] = useState('');
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => setKeyword(keywordInput), 500);
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [keywordInput]);
  const [banModalOpen, setBanModalOpen] = useState(false);
  const [pendingBan, setPendingBan] = useState<PatientListItem | null>(null);
  const [banForm] = Form.useForm<BanFormValues>();

  const {
    data,
    isLoading,
    isError,
    error,
  } = useQuery({
    queryKey: patientQueryKeys.list({ keyword }),
    queryFn: () => fetchPatients({ keyword: keyword || undefined }),
    refetchOnWindowFocus: false,
  });

  const banMut = useMutation({
    mutationFn: ({ id, reason }: { id: number; reason: string }) =>
      banPatient(id, reason),
    onSuccess: (r) => {
      message.success(`已封禁患者 #${r.id}`);
      qc.invalidateQueries({ queryKey: ['patients'] });
    },
    onError: (e: Error) => {
      message.error(`封禁失败：${e.message}`);
    },
  });

  const unbanMut = useMutation({
    mutationFn: (id: number) => unbanPatient(id),
    onSuccess: (r) => {
      message.success(`已解封患者 #${r.id}`);
      qc.invalidateQueries({ queryKey: ['patients'] });
    },
    onError: (e: Error) => {
      message.error(`解封失败：${e.message}`);
    },
  });

  const items: PatientListItem[] = useMemo(() => data?.data ?? [], [data?.data]);

  const columns: ColumnsType<PatientListItem> = [
    { title: 'ID', dataIndex: 'id', width: 80 },
    { title: '昵称', dataIndex: 'name', width: 140 },
    { title: '手机', dataIndex: 'phone', width: 140 },
    {
      title: '实名状态',
      dataIndex: 'verify_status',
      width: 110,
      render: (v: string | undefined, r: PatientListItem) =>
        v === 'verified' ? (
          <Tag color="green" data-testid={`row-verify-${r.id}`}>
            已实名
          </Tag>
        ) : v === 'pending' ? (
          <Tag color="orange" data-testid={`row-verify-${r.id}`}>
            待审核
          </Tag>
        ) : (
          <Tag data-testid={`row-verify-${r.id}`}>未实名</Tag>
        ),
    },
    {
      title: '封禁状态',
      dataIndex: 'status',
      width: 110,
      render: (_v: string | undefined, r: PatientListItem) =>
        r.status === 'banned' ? (
          <Tag color="red" data-testid={`row-ban-${r.id}`}>
            封禁
          </Tag>
        ) : (
          <Tag color="green" data-testid={`row-ban-${r.id}`}>
            正常
          </Tag>
        ),
    },
    {
      title: '订单数',
      dataIndex: 'order_count',
      width: 90,
      render: (v: number) => <Tag color="blue">{v}</Tag>,
    },
    {
      title: '注册时间',
      dataIndex: 'registered_at',
      width: 170,
      render: (v: string) => dayjs(v).format('YYYY-MM-DD'),
    },
    {
      title: '操作',
      width: 220,
      fixed: 'right',
      render: (_: unknown, r: PatientListItem) => (
        <Space size="small">
          <Button
            size="small"
            onClick={() => navigate(`/patients/${r.id}`)}
            data-testid={`btn-detail-${r.id}`}
          >
            详情
          </Button>
          {canBan && r.status !== 'banned' && (
            <Button
              size="small"
              danger
              icon={<StopOutlined />}
              onClick={() => {
                setPendingBan(r);
                banForm.resetFields();
                setBanModalOpen(true);
              }}
              data-testid={`btn-ban-${r.id}`}
            >
              封禁
            </Button>
          )}
          {canBan && r.status === 'banned' && (
            <Button
              size="small"
              type="primary"
              icon={<CheckCircleOutlined />}
              onClick={() => unbanMut.mutate(r.id)}
              loading={unbanMut.isPending && unbanMut.variables === r.id}
              data-testid={`btn-unban-${r.id}`}
            >
              解封
            </Button>
          )}
        </Space>
      ),
    },
  ];

  const onSubmitBan = async () => {
    try {
      const values = await banForm.validateFields();
      if (!pendingBan) return;
      banMut.mutate(
        { id: pendingBan.id, reason: values.reason },
        {
          onSuccess: () => {
            setBanModalOpen(false);
            setPendingBan(null);
          },
        },
      );
    } catch {
      // antd validate fail silent
    }
  };

  return (
    <div data-testid="patients-page">
      <PageHeader
        title="患者管理"
        subtitle="患者列表 · 关键词搜索 · 封禁 / 解封"
      />

      <div style={{ marginBottom: 16 }}>
        <Input
          placeholder="搜索：昵称 / 手机号"
          value={keywordInput}
          allowClear
          onChange={(e) => setKeywordInput(e.target.value)}
          style={{ width: 280 }}
          data-testid="patient-keyword-input"
        />
        <span style={{ marginLeft: 12, color: '#999' }}>
          共 {data?.total ?? 0} 条
        </span>
      </div>

      <ProTable<PatientListItem>
        testId="patient-table"
        rowKey="id"
        columns={columns}
        dataSource={items}
        loading={isLoading}
        density="middle"
        scroll={{ x: 1100 }}
      />

      {isError && (
        <div data-testid="patients-error" style={{ color: '#f5222d', marginTop: 8 }}>
          加载失败：{(error as Error)?.message ?? '未知错误'}
        </div>
      )}

      <Modal
        title={`封禁患者 #${pendingBan?.id ?? '—'}`}
        open={banModalOpen}
        onCancel={() => {
          setBanModalOpen(false);
          setPendingBan(null);
        }}
        onOk={onSubmitBan}
        confirmLoading={banMut.isPending}
        okButtonProps={{ danger: true, 'data-testid': 'btn-ban-confirm' }}
        cancelButtonProps={{ 'data-testid': 'btn-ban-cancel' }}
        destroyOnClose
      >
        <Form form={banForm} layout="vertical" preserve={false}>
          <Form.Item
            label="封禁原因"
            name="reason"
            rules={[{ required: true, message: '请输入封禁原因' }]}
          >
            <Input.TextArea
              rows={4}
              placeholder="例：恶意下单、辱骂客服"
              data-testid="ban-reason-input"
              maxLength={200}
              showCount
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
