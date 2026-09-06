/**
 * 通用后台 UI 的 schema 类型（集中定义）。
 *
 * FilterBar、ResourceListView、工具栏按钮都以这些 schema 驱动，
 * 页面只写「配置」，不重复写筛选/表格/按钮布局代码。
 */
import type { DataTableBaseColumn, DataTableSelectionColumn } from 'naive-ui'
import type { OptionItem } from '@/constants/enums'
import type { PermissionCode } from '@/constants/permissions'

/**
 * 统一的表格列列表类型。
 * 业务数据列使用 `DataTableBaseColumn<T>`（由 utils/columns 工厂生成），
 * 需要批量操作的列表可在首列加入 Naive UI 的选择列。
 */
export type TableColumnList<T> = Array<DataTableBaseColumn<T> | DataTableSelectionColumn<T>>

/**
 * ResourceListView 通过 defineExpose 暴露的实例形状。
 * 页面用 `ref<ResourceListInstance | null>(null)` 引用，避免对泛型组件使用
 * `InstanceType<typeof ...>`（泛型函数式组件无法用 InstanceType）。
 */
export interface ResourceListInstance {
  reload: () => Promise<void>
}

export type FilterFieldType = 'input' | 'select' | 'daterange'

export interface FilterField {
  /** 对应 query 参数名 */
  key: string
  label: string
  type: FilterFieldType
  placeholder?: string
  /** select 选项 */
  options?: OptionItem[]
  /** daterange 会写入 `${key}From` / `${key}To` 两个参数 */
  width?: number
  mobileNative?: boolean
}

export interface ToolbarAction {
  key: string
  label: string
  /** 需要的操作权限码；无则所有登录用户可见 */
  permission?: PermissionCode
  /** 是否高风险（视觉强调 + 提示） */
  type?: 'default' | 'primary' | 'error'
  /** 当前状态下是否禁用（例如批量操作尚未选择记录） */
  disabled?: boolean
  /** 点击回调 */
  onClick: () => void
}
