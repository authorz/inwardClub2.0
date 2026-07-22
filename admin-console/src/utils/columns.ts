/**
 * 表格列工厂（公共工具）。
 *
 * 后台各列表页反复出现「状态标签列、金额列、时间列、范围列、审计信息列、操作列」。
 * 这里用工厂函数集中生成这些列，页面只需组合，避免每页手写 render。
 *
 * 说明：统一返回 Naive UI 的 `DataTableBaseColumn<T>`（而非联合类型 DataTableColumn），
 * 避免联合类型带来的 render/title 推断歧义；行取值按 key 索引，内部做一次安全断言，
 * 因此实体可以是普通 interface（无需强制带索引签名）。
 */
import { h, type VNodeChild } from 'vue'
import type { DataTableBaseColumn } from 'naive-ui'
import StatusTag from '@/components/StatusTag.vue'
import type { OptionItem } from '@/constants/enums'
import { formatCent, formatDateTime } from '@/utils/format'

/** 按 key 安全取值（实体多为无索引签名的 interface） */
function pick(row: unknown, key: string): unknown {
  return (row as Record<string, unknown>)[key]
}

export function textColumn<T>(
  title: string,
  key: string,
  options: Partial<DataTableBaseColumn<T>> = {},
): DataTableBaseColumn<T> {
  return { title, key, ellipsis: { tooltip: true }, ...options }
}

export function statusColumn<T>(
  title: string,
  key: string,
  statusOptions: OptionItem[],
  width = 120,
): DataTableBaseColumn<T> {
  return {
    title,
    key,
    width,
    render: (row) => h(StatusTag, { value: pick(row, key) as string, options: statusOptions }),
  }
}

export function moneyColumn<T>(title: string, key: string, width = 120): DataTableBaseColumn<T> {
  return {
    title,
    key,
    width,
    render: (row) => formatCent(pick(row, key) as number),
  }
}

export function dateTimeColumn<T>(title: string, key: string, width = 170): DataTableBaseColumn<T> {
  return {
    title,
    key,
    width,
    render: (row) => formatDateTime(pick(row, key) as string),
  }
}

/** 自定义 render 列：用于图片、掩码手机号等非标准展示，避免页面手写原始列对象 */
export function renderColumn<T>(
  title: string,
  key: string,
  render: (row: T) => VNodeChild,
  width?: number,
): DataTableBaseColumn<T> {
  return { title, key, ...(width ? { width } : {}), render }
}

/** 操作列：由页面传入 render 函数，宽度统一 */
export function actionsColumn<T>(
  render: (row: T) => VNodeChild,
  width = 200,
): DataTableBaseColumn<T> {
  return {
    title: '操作',
    key: '__actions',
    width,
    fixed: 'right',
    render,
  }
}
