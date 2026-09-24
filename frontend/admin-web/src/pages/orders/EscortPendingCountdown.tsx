/**
 * EscortPendingCountdown：陪诊师 30s 确认窗口倒计时。
 *
 * v2 新增：
 *   - 陪诊师被患者选中后，订单进入 escort_pending_acceptance 状态；
 *   - 后端给定 30s 截止时间 `escort_pending_expire_at`；
 *   - admin 端需在详情页实时显示剩余秒数，< 60s 红色高亮提示紧急。
 *
 * 实现策略：
 *   - 用 useEffect + setInterval 1s tick（不引第三方库，YAGNI）；
 *   - 卸载时 clearInterval 防内存泄漏；
 *   - 剩余 < 60s 时 span 颜色变红 (#ff4d4f)，> 60s 时蓝 (#1677ff)；
 *   - 减到 0 时 clearInterval，文本定格在 "0s"。
 *
 * 对应 spec：2026-09-24-order-matching-redesign.md §4 + §5
 *           2026-09-24-admin-web-setup.md §Task 2
 */
import { useEffect, useState } from 'react';
import dayjs from 'dayjs';

export interface EscortPendingCountdownProps {
  /** ISO datetime 字符串，UTC 或本地时区均可（dayjs 自动解析） */
  expireAt: string;
  /** 高亮阈值（秒）；默认 60s。剩余 < threshold 时变红 */
  criticalThresholdSeconds?: number;
}

const BLUE = '#1677ff';
const RED = '#ff4d4f';

export function EscortPendingCountdown({
  expireAt,
  criticalThresholdSeconds = 60,
}: EscortPendingCountdownProps) {
  // 初值用 dayjs.diff 计算剩余秒（不依赖 useEffect 第一次跑）
  const compute = () => Math.max(0, dayjs(expireAt).diff(dayjs(), 'second'));

  const [remaining, setRemaining] = useState<number>(compute);

  useEffect(() => {
    // 同步初始值（如果父组件 expireAt 在挂载后才传入，需要再算一次）
    setRemaining(compute());
    const tick = setInterval(() => {
      const r = compute();
      setRemaining(r);
      if (r === 0) clearInterval(tick);
    }, 1000);
    return () => clearInterval(tick);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [expireAt, criticalThresholdSeconds]);

  const isCritical = remaining < criticalThresholdSeconds;
  const color = isCritical ? RED : BLUE;

  return (
    <span
      style={{ color, fontWeight: 600 }}
      data-testid="escort-countdown"
      data-critical={isCritical ? 'true' : 'false'}
    >
      {remaining}s
    </span>
  );
}

export default EscortPendingCountdown;