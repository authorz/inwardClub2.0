/**
 * 表格列构造助手：集中生成状态标签列、金额列、时间列、手机号掩码列等，
 * 避免每个列表页重复写 render 函数。
 */

import { h } from 'vue'
import type { DataTableBaseColumn } from 'naive-ui'
import StatusTag from '@/components/common/StatusTag.vue'
import type { EnumOption } from '@/constants/enums'
import { formatCent, formatDateTime, maskPhone } from './format'

/** 普通文本列，空值回退为 '-'。 */
export function textColumn<T>(
  title: string,
  get: (row: T) => string | number | null | undefined,
  extra: Partial<DataTableBaseColumn<T>> = {},
): DataTableBaseColumn<T> {
  return {
    title,
    key: title,
    render: (row: T) => {
      const v = get(row)
      return v == null || v === '' ? '-' : String(v)
    },
    ...extra,
  }
}

/** 金额列（整数分 -> ¥）。 */
export function moneyColumn<T>(
  title: string,
  get: (row: T) => number | null | undefined,
  extra: Partial<DataTableBaseColumn<T>> = {},
): DataTableBaseColumn<T> {
  return {
    title,
    key: `${title}_money`,
    align: 'right',
    render: (row: T) => formatCent(get(row)),
    ...extra,
  }
}

/** 时间列。 */
export function dateColumn<T>(
  title: string,
  get: (row: T) => string | null | undefined,
  extra: Partial<DataTableBaseColumn<T>> = {},
): DataTableBaseColumn<T> {
  return {
    title,
    key: `${title}_date`,
    render: (row: T) => formatDateTime(get(row)),
    ...extra,
  }
}

/** 手机号掩码列。 */
export function phoneColumn<T>(
  title: string,
  get: (row: T) => string | null | undefined,
  extra: Partial<DataTableBaseColumn<T>> = {},
): DataTableBaseColumn<T> {
  return {
    title,
    key: `${title}_phone`,
    render: (row: T) => maskPhone(get(row)),
    ...extra,
  }
}

/** 状态标签列，使用集中枚举字典。 */
export function statusColumn<T>(
  title: string,
  dict: Record<string, EnumOption>,
  get: (row: T) => string | null | undefined,
  extra: Partial<DataTableBaseColumn<T>> = {},
): DataTableBaseColumn<T> {
  return {
    title,
    key: `${title}_status`,
    render: (row: T) => h(StatusTag, { dict, value: get(row) }),
    ...extra,
  }
}

/** 操作列：render 返回操作按钮组。 */
export function actionColumn<T>(
  render: (row: T) => ReturnType<typeof h>,
  title = '操作',
  width = 200,
): DataTableBaseColumn<T> {
  return {
    title,
    key: 'actions',
    width,
    fixed: 'right',
    render,
  }
}
