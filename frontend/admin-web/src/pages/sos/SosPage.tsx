/**
 * SosPage：SOS 紧急报警页骨架（v1 Task 18 占位）。
 *
 * 范围：
 *   - 实时报警列表（最近 24h）；
 *   - 报警详情（位置 / 患者 / 陪诊师 / 联系方式）；
 *   - 行操作：联系 / 升级 / 关闭。
 *
 * TODO: 实现页面（来自 plan v1 Task 18）
 */
import { Card } from 'antd';
import { PageHeader } from '@/components/PageHeader';

export default function SosPage() {
  return (
    <div data-testid="sos-page">
      <PageHeader
        title="SOS 紧急报警"
        subtitle="实时报警列表（24h 内）+ 联系方式 + 升级"
      />
      <Card>
        <p style={{ color: '#999' }}>
          TODO: 接入 MSW GET /api/v1/admin/sos + 轮询 30s + 联系/升级按钮
        </p>
      </Card>
    </div>
  );
}