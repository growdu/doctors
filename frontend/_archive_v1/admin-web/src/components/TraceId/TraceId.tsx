/**
 * TraceId：展示 X-Trace-Id + 一键复制。
 *
 * 设计要点：
 *   - 默认 monospace <code>，配 copy-to-clipboard 按钮；
 *   - 不可用 traceId（空/null/undefined）→ 渲染 placeholder；
 *   - 复制成功 → message.success 反馈（轻量包装，不引入 App.useApp 耦合）；
 *
 * 对应 spec：2026-09-24-admin-web-design.md §5.2 (TraceId)
 */
import { useState } from 'react';
import { Button, message, Typography } from 'antd';
import { CopyOutlined } from '@ant-design/icons';

const { Text } = Typography;

export interface TraceIdProps {
  /** X-Trace-Id 字符串；空/null → 渲染 placeholder */
  traceId?: string | null;
  /** 顶层 data-testid */
  testId?: string;
  /** 自定义 placeholder */
  placeholder?: string;
}

export function TraceId({
  traceId,
  testId,
  placeholder = '— 无 trace_id —',
}: TraceIdProps) {
  const [copied, setCopied] = useState(false);

  if (!traceId) {
    return (
      <Text type="secondary" data-testid={testId ?? 'trace-id'}>
        {placeholder}
      </Text>
    );
  }

  const handleCopy = async () => {
    try {
      if (navigator?.clipboard?.writeText) {
        await navigator.clipboard.writeText(traceId);
      } else {
        // 老浏览器 fallback：textarea + execCommand
        const ta = document.createElement('textarea');
        ta.value = traceId;
        ta.style.position = 'fixed';
        ta.style.opacity = '0';
        document.body.appendChild(ta);
        ta.select();
        document.execCommand('copy');
        document.body.removeChild(ta);
      }
      setCopied(true);
      message.success('Trace ID 已复制');
      setTimeout(() => setCopied(false), 1200);
    } catch (e) {
      message.error('复制失败，请手动选中');
      // eslint-disable-next-line no-console
      console.error('[TraceId.copy]', e);
    }
  };

  return (
    <span data-testid={testId ?? 'trace-id'}>
      <code
        style={{
          fontFamily:
            'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
          background: 'rgba(0, 0, 0, 0.04)',
          padding: '2px 6px',
          borderRadius: 4,
          marginRight: 8,
        }}
      >
        {traceId}
      </code>
      <Button
        type="link"
        size="small"
        icon={<CopyOutlined />}
        onClick={handleCopy}
        data-testid="trace-id-copy"
      >
        {copied ? '已复制' : '复制'}
      </Button>
    </span>
  );
}

export default TraceId;