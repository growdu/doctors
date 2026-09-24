/**
 * ProTable 测试契约（待 vitest 启用后跑）：
 *   - 顶层挂载 data-testid；
 *   - loading=true 时 antd Table 内含 .ant-spin；
 *   - pagination=false 时不渲染分页；
 *   - density='compact' → Table size=small；'middle'/'large' → middle/large；
 *   - rowSelection 透传 selectedRowKeys。
 *
 * 注：当前骨架未装 vitest / @testing-library/react，本文件为契约样。
 */
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { ProTable } from './ProTable';
import type { ColumnsType } from 'antd/es/table';

interface Row {
  id: number;
  name: string;
}

const columns: ColumnsType<Row> = [
  { title: 'ID', dataIndex: 'id' },
  { title: '名称', dataIndex: 'name' },
];
const data: Row[] = [
  { id: 1, name: 'A' },
  { id: 2, name: 'B' },
];

describe('ProTable', () => {
  it('顶层挂载 data-testid', () => {
    render(<ProTable testId="pt" columns={columns} dataSource={data} />);
    expect(screen.getByTestId('pt')).toBeInTheDocument();
  });

  it('loading=true 时含 .ant-spin', () => {
    const { container } = render(
      <ProTable testId="pt" columns={columns} dataSource={data} loading />,
    );
    expect(container.querySelector('.ant-spin')).not.toBeNull();
  });

  it('pagination=false 时不渲染分页', () => {
    const { container } = render(
      <ProTable
        testId="pt"
        columns={columns}
        dataSource={data}
        pagination={false}
      />,
    );
    expect(container.querySelector('.ant-pagination')).toBeNull();
  });

  it('density=compact → size=small（antd .ant-table-small）', () => {
    const { container } = render(
      <ProTable
        testId="pt"
        columns={columns}
        dataSource={data}
        density="compact"
      />,
    );
    expect(container.querySelector('.ant-table-sm')).not.toBeNull();
  });

  it('rowSelection.selectedRowKeys 透传', () => {
    render(
      <ProTable<Row>
        testId="pt"
        columns={columns}
        dataSource={data}
        rowSelection={{ selectedRowKeys: [1] }}
      />,
    );
    // 选中行 1 应有 .ant-checkbox-checked
    const checkboxes = document.querySelectorAll('.ant-checkbox-checked');
    expect(checkboxes.length).toBeGreaterThanOrEqual(1);
  });
});