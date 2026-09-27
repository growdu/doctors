/**
 * EscortPendingCountdown：admin-web 陪诊师 30s 确认窗口倒计时组件。
 *
 * 用途（v2 order-matching-redesign）：
 *   - 订单进入 `escort_pending_acceptance` 后，后端给定截止时间
 *     `escort_pending_expire_at`（≤30s）；
 *   - 列表页与详情页实时显示剩余秒数，< 60s 时红色高亮提示紧急；
 *   - 减到 0 文案改为「已超时」，提示运营介入。
 *
 * 实现策略：
 *   - useEffect + setInterval 1s tick（不引第三方库，YAGNI）；
 *   - 卸载 / expireAt 变更时 clearInterval 防内存泄漏；
 *   - < 60s 红色 (#ff4d4f) / > 60s 蓝色 (#1677ff)；
 *   - 过期文案「已超时」并继续心跳（不会清掉 interval，方便运营对照真实时间）。
 *
 * 对应 spec：
 *   - docs/superpowers/plans/2026-09-24-order-matching-redesign.md §4 + §5
 *   - 2026-09-24-admin-web-setup.md §Task 2
 *   - dev.md §38 admin-web v1 完整化收官
 */
import { useEffect, useState } from 'react';
import dayjs from 'dayjs';

export interface EscortPendingCountdownProps {
  /** 截止时间（Date 实例或 ISO 字符串均可，dayjs 自动解析） */
  expireAt: Date | string;
  /** 紧急高亮阈值（秒）；默认 60s。剩余 < threshold 时变红 */
  criticalThresholdSeconds?: number;
  /** 失效文案；默认「已超时」 */
  expiredText?: string;
}

const BLUE = '#1677ff';
const RED = '#ff4d4f';

function toDayjs(input: Date | string) {
  return dayjs(input);
}

export function EscortPendingCountdown({
  expireAt,
  criticalThresholdSeconds = 60,
  expiredText = '已超时',
}: EscortPendingCountdownProps) {
  const compute = () => Math.max(0, toDayjs(expireAt).diff(dayjs(), 'second'));

  const [remaining, setRemaining] = useState<number>(compute);

  useEffect(() => {
    // 同步初始值：父组件 expireAt 在挂载后才传入 / 变化时，再算一次
    setRemaining(compute());
    const tick = setInterval(() => {
      setRemaining(compute());
    }, 1000);
    return () => clearInterval(tick);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [
    typeof expireAt === 'string' ? expireAt : expireAt.toISOString(),
    criticalThresholdSeconds,
  ]);

  const expired = remaining === 0;
  const isCritical = !expired && remaining < criticalThresholdSeconds;
  const color = isCritical ? RED : BLUE;

  return (
    <span
      style={{ color, fontWeight: 600 }}
      data-testid="escort-countdown"
      data-expired={expired ? 'true' : 'false'}
      data-critical={isCritical ? 'true' : 'false'}
    >
      {expired ? expiredText : `剩余 ${remaining}s`}
    </span>
  );
}

export default EscortPendingCountdown;
