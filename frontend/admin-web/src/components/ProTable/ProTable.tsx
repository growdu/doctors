/**
 * ProTable：antd Table 二次封装。
 *
 * 封装 5 个常用 props：
 *   1. loading（boolean）
 *   2. pagination（false | TablePaginationConfig）
 *   3. rowSelection（TableRowSelection<T>）
 *   4. density（'compact' | 'middle' | 'large'，默认 'middle'）
 *   5. data-testid（顶层节点挂载）
 *
 * 设计要点：
 *   - 完全透传 antd TableProps，避免锁死业务侧用法；
 *   - density 通过 Table 上 size 属性实现；
 *   - 默认 rowKey='id'，业务可显式覆盖；
 *   - 默认 scroll x=1000，小屏可滚动；
 *   - 默认 size 与 antd 推荐一致（middle）。
 *
 * 对应 spec：2026-09-24-admin-web-design.md §5.2 (公共组件)
 */
import type { CSSProperties } from 'react';
import { Table } from 'antd';
import type {
  TableProps,
  TablePaginationConfig,
  TableRowSelection,
} from 'antd/es/table';

export type ProTableDensity = 'compact' | 'middle' | 'large';

export interface ProTableProps<T = unknown> extends Omit<TableProps<T>, 'size'> {
  /** 加载态，默认 false */
  loading?: boolean;
  /** 分页配置，false 关闭；不传 → 默认 { pageSize: 20, showSizeChanger, showTotal } */
  pagination?: false | TablePaginationConfig;
  /** 行选择 */
  rowSelection?: TableRowSelection<T>;
  /** 密度，默认 middle */
  density?: ProTableDensity;
  /** 业务数据（已抽出 dataSource 键名） */
  dataSource?: readonly T[];
  /** 顶部 data-testid，便于 e2e / RTL 定位 */
  testId?: string;
}

const DEFAULT_PAGINATION: TablePaginationConfig = {
  pageSize: 20,
  showSizeChanger: true,
  showTotal: (t) => `共 ${t} 条`,
};

const DEFAULT_SCROLL: TableProps['scroll'] = { x: 1000 };

/**
 * ProTable 渲染。
 *
 * @example
 *   <ProTable
 *     testId="order-list"
 *     columns={columns}
 *     dataSource={orders}
 *     loading={isLoading}
 *     rowSelection={{ selectedRowKeys, onChange }}
 *     density="compact"
 *   />
 */
export function ProTable<T extends object = object>(props: ProTableProps<T>) {
  const {
    loading = false,
    pagination = DEFAULT_PAGINATION,
    rowSelection,
    density = 'middle',
    testId,
    scroll = DEFAULT_SCROLL,
    ...rest
  } = props;

  const wrapperStyle: CSSProperties = {
    width: '100%',
  };

  return (
    <div data-testid={testId} style={wrapperStyle}>
      <Table<T>
        {...rest}
        loading={loading}
        pagination={pagination}
        rowSelection={rowSelection}
        size={density}
        scroll={scroll}
        rowKey={rest.rowKey ?? 'id'}
      />
    </div>
  );
}

export default ProTable;