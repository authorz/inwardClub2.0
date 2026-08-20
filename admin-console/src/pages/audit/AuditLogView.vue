<script setup lang="ts">
/**
 * 审计日志（只读）。列表展示运营人员可理解的中文摘要，详情保留完整
 * actor / scope / target / before / after / reason / request ID 追溯信息。
 */
import { h, ref } from 'vue'
import { NButton, NDescriptions, NDescriptionsItem, NModal, NSpace, NTag } from 'naive-ui'
import ResourceListView from '@/components/ResourceListView.vue'
import { actionsColumn, dateTimeColumn, renderColumn } from '@/utils/columns'
import { readonlyLists } from '@/api/services'
import { formatDateTime } from '@/utils/format'
import type { AuditLog } from '@/api/models'
import type { FilterField, TableColumnList } from '@/components/ui-types'
import type { OptionItem } from '@/constants/enums'

const ACTOR_TYPE_LABELS: Record<string, string> = {
  super_admin: '总后台管理员',
  store_admin: '门店超级管理员',
  cashier: '门店管理员',
  staff: '员工',
  member: '会员',
  pre_member: '待完善资料会员',
  system: '系统任务',
}

const ROLE_LABELS: Record<string, string> = {
  super_admin: '超级管理员',
  store_admin: '门店超级管理员',
  cashier: '门店管理员',
  staff: '员工',
  member: '会员',
  pre_member: '待完善资料会员',
}

const ACTION_LABELS: Record<string, string> = {
  'member.wallet.adjust': '会员人工调账',
  login: '登录后台',
  logout: '退出后台',
  create: '新增',
  update: '修改',
  disable: '禁用',
  delete: '删除',
  publish: '发布',
  approve: '审核通过',
  reject: '审核驳回',
  refund: '退款',
}

const TARGET_TYPE_LABELS: Record<string, string> = {
  member: '会员',
  store: '门店',
  staff: '员工账号',
  staff_account: '员工账号',
  cashier: '门店管理员账号',
  admin_account: '管理员账号',
  catalog_item: '商品',
  catalog_category: '商品分类',
  activity: '活动',
  coupon_template: '优惠券模板',
  banner: '广告图',
  order: '订单',
  refund: '退款单',
  rule_definition: '业务规则',
}

const DETAIL_FIELD_LABELS: Record<string, string> = {
  id: 'ID',
  memberId: '会员 ID',
  storeId: '门店 ID',
  assetType: '资产类型',
  availableAmount: '可用余额',
  heldAmount: '冻结余额',
  amount: '数量',
  direction: '变动方向',
  status: '状态',
  name: '名称',
  nickname: '昵称',
  phone: '手机号',
  reason: '原因',
  createdAt: '创建时间',
  updatedAt: '更新时间',
}

const VALUE_LABELS: Record<string, string> = {
  points: '积分',
  coins: '金币',
  cash_balance: '余额',
  growth_value: '成长值',
  credit: '增加',
  debit: '扣减',
  active: '启用',
  disabled: '禁用',
  pending: '待处理',
  completed: '已完成',
}

const actorTypeOptions: OptionItem[] = Object.entries(ACTOR_TYPE_LABELS).map(([value, label]) => ({
  label,
  value,
}))
const actionOptions: OptionItem[] = [
  { label: '会员人工调账', value: 'member.wallet.adjust' },
  { label: '登录后台', value: 'login' },
  { label: '退出后台', value: 'logout' },
]
const targetTypeOptions: OptionItem[] = Object.entries(TARGET_TYPE_LABELS).map(
  ([value, label]) => ({ label, value }),
)

const fields: FilterField[] = [
  { key: 'actorType', label: '操作者类型', type: 'select', options: actorTypeOptions },
  { key: 'action', label: '操作类型', type: 'select', options: actionOptions, width: 180 },
  { key: 'targetType', label: '操作对象', type: 'select', options: targetTypeOptions },
  { key: 'created', label: '操作时间', type: 'daterange' },
]

const detailShow = ref(false)
const current = ref<AuditLog | null>(null)

function actorTypeLabel(value?: string): string {
  return ACTOR_TYPE_LABELS[value ?? ''] ?? value ?? '未知操作者'
}

function roleLabel(value?: string): string {
  return ROLE_LABELS[value ?? ''] ?? value ?? '—'
}

function actionLabel(value?: string): string {
  if (!value) return '未知操作'
  if (ACTION_LABELS[value]) return ACTION_LABELS[value]
  return value
    .split('.')
    .map((part) => TARGET_TYPE_LABELS[part] ?? ACTION_LABELS[part] ?? part)
    .join(' · ')
}

function targetTypeLabel(value?: string): string {
  return TARGET_TYPE_LABELS[value ?? ''] ?? value ?? '未知对象'
}

function actorName(row: AuditLog): string {
  const snapshot = row.actorSnapshot
  return snapshot?.displayName || snapshot?.name || snapshot?.username || actorTypeLabel(row.actorType)
}

function actorAccount(row: AuditLog): string {
  const snapshot = row.actorSnapshot
  const account = snapshot?.username ? `账号：${snapshot.username}` : actorTypeLabel(row.actorType)
  return `${account} · ID ${snapshot?.id ?? row.actorId ?? '—'}`
}

function targetLabel(row: AuditLog): string {
  const type = targetTypeLabel(row.targetType)
  const snapshot = row.targetSnapshot
  if (snapshot?.nickname) return `${snapshot.nickname}（${type} ID ${snapshot.id}）`
  return row.targetId ? `${type} ID ${row.targetId}` : type
}

function targetPhone(row: AuditLog): string {
  return row.targetSnapshot?.phone || '未保存'
}

function snapshotSourceLabel(row: AuditLog): string {
  const source = row.targetSnapshot?.source || row.actorSnapshot?.source || row.scopeSnapshot?.source
  return source === 'backfill_current_state' ? '历史补录（迁移时的当前资料）' : '操作时入库快照'
}

function scopeLabel(row: AuditLog): string {
  if (row.storeId == null) return '总部 / 全局'
  const snapshot = row.scopeSnapshot
  return snapshot?.storeName ? `${snapshot.storeName}（ID ${snapshot.storeId}）` : `门店 ID ${row.storeId}`
}

function asRecord(value: unknown): Record<string, unknown> | null {
  return value && typeof value === 'object' && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : null
}

function operationSummary(row: AuditLog): string {
  const before = asRecord(row.before)
  const after = asRecord(row.after)
  if (row.action === 'member.wallet.adjust' && before && after) {
    const asset = VALUE_LABELS[String(after.assetType ?? before.assetType ?? '')] ?? '资产'
    const from = before.availableAmount ?? '—'
    const to = after.availableAmount ?? '—'
    return `${asset}余额 ${String(from)} → ${String(to)}${row.reason ? `；${row.reason}` : ''}`
  }
  return row.reason || `${targetLabel(row)}：${actionLabel(row.action)}`
}

function isSensitiveField(key: string): boolean {
  return /(password|token|secret|privateKey|credential)/i.test(key)
}

function localizeDetail(value: unknown, parentKey = ''): unknown {
  if (Array.isArray(value)) return value.map((item) => localizeDetail(item, parentKey))
  const record = asRecord(value)
  if (record) {
    return Object.fromEntries(
      Object.entries(record).map(([key, item]) => [
        DETAIL_FIELD_LABELS[key] ?? key,
        isSensitiveField(key) ? '已隐藏敏感内容' : localizeDetail(item, key),
      ]),
    )
  }
  if (typeof value === 'string') return VALUE_LABELS[value] ?? value
  return value
}

function formatDetail(value: unknown): string {
  if (value == null) return '无记录'
  return JSON.stringify(localizeDetail(value), null, 2)
}

function openDetail(row: AuditLog): void {
  current.value = row
  detailShow.value = true
}

const columns: TableColumnList<AuditLog> = [
  dateTimeColumn<AuditLog>('操作时间', 'createdAt', 170),
  renderColumn<AuditLog>(
    '操作者',
    'actorType',
    (row) =>
      h('div', { class: 'audit-log__identity' }, [
        h('strong', actorName(row)),
        h('span', actorAccount(row)),
      ]),
    190,
  ),
  renderColumn<AuditLog>(
    '操作',
    'action',
    (row) => h(NTag, { size: 'small', bordered: false }, { default: () => actionLabel(row.action) }),
    160,
  ),
  renderColumn<AuditLog>(
    '操作对象',
    'targetType',
    (row) =>
      h('div', { class: 'audit-log__identity' }, [
        h('strong', targetLabel(row)),
        h('span', `手机号：${targetPhone(row)}`),
      ]),
    250,
  ),
  renderColumn<AuditLog>('操作范围', 'storeId', (row) => scopeLabel(row), 180),
  renderColumn<AuditLog>('变更摘要', 'summary', (row) => operationSummary(row)),
  actionsColumn<AuditLog>(
    (row) =>
      h(NSpace, {}, () => [
        h(NButton, { size: 'small', onClick: () => openDetail(row) }, { default: () => '详情' }),
      ]),
    90,
  ),
]
</script>

<template>
  <div>
    <ResourceListView
      title="审计日志"
      description="用中文查看后台关键操作，并可追溯操作前后数据"
      :breadcrumb="['审计与日志', '审计日志']"
      :fields="fields"
      :columns="columns"
      :fetcher="readonlyLists.auditLogs"
    />

    <NModal
      v-model:show="detailShow"
      preset="card"
      title="操作详情"
      :mask-closable="false"
      style="width: 820px; max-width: 92vw"
      content-style="max-height: 72vh; overflow: auto"
    >
      <template v-if="current">
        <div class="audit-detail__summary">
          <strong>{{ actionLabel(current.action) }}</strong>
          <span>{{ operationSummary(current) }}</span>
        </div>

        <NDescriptions
          :column="2"
          label-placement="left"
          bordered
        >
          <NDescriptionsItem label="日志编号">
            {{ current.id }}
          </NDescriptionsItem>
          <NDescriptionsItem label="操作时间">
            {{ formatDateTime(current.createdAt) }}
          </NDescriptionsItem>
          <NDescriptionsItem label="操作者">
            {{ actorName(current) }}（{{ actorAccount(current) }}）
          </NDescriptionsItem>
          <NDescriptionsItem label="操作者角色">
            {{ roleLabel(current.actorRole) }}
          </NDescriptionsItem>
          <NDescriptionsItem label="操作范围">
            {{ scopeLabel(current) }}
          </NDescriptionsItem>
          <NDescriptionsItem label="操作对象">
            {{ targetLabel(current) }}
          </NDescriptionsItem>
          <NDescriptionsItem label="会员手机号">
            {{ targetPhone(current) }}
          </NDescriptionsItem>
          <NDescriptionsItem label="会员头像地址">
            {{ current.targetSnapshot?.avatarUrl || '未保存' }}
          </NDescriptionsItem>
          <NDescriptionsItem label="身份资料来源">
            {{ snapshotSourceLabel(current) }}
          </NDescriptionsItem>
          <NDescriptionsItem label="操作代码">
            {{ current.action }}
          </NDescriptionsItem>
          <NDescriptionsItem label="请求编号">
            {{ current.requestId || '—' }}
          </NDescriptionsItem>
          <NDescriptionsItem
            label="操作原因"
            :span="2"
          >
            {{ current.reason || '未填写' }}
          </NDescriptionsItem>
        </NDescriptions>

        <div class="audit-detail__changes">
          <section>
            <h3>操作前</h3>
            <pre>{{ formatDetail(current.before) }}</pre>
          </section>
          <section>
            <h3>操作后</h3>
            <pre>{{ formatDetail(current.after) }}</pre>
          </section>
        </div>
      </template>

      <template #footer>
        <div class="audit-detail__footer">
          <NButton @click="detailShow = false">
            关闭
          </NButton>
        </div>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.audit-log__identity {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.audit-log__identity strong {
  font-size: var(--ic-font-sm);
  font-weight: 600;
}
.audit-log__identity span {
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-xs);
}
.audit-detail__summary {
  display: flex;
  flex-direction: column;
  gap: var(--ic-space-xs);
  padding-bottom: var(--ic-space-md);
  margin-bottom: var(--ic-space-md);
  border-bottom: 1px solid var(--ic-color-border);
}
.audit-detail__summary strong {
  font-size: var(--ic-font-xl);
}
.audit-detail__summary span {
  color: var(--ic-color-text-secondary);
}
.audit-detail__changes {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--ic-space-md);
  margin-top: var(--ic-space-md);
}
.audit-detail__changes section {
  min-width: 0;
}
.audit-detail__changes h3 {
  margin: 0 0 var(--ic-space-sm);
  font-size: var(--ic-font-lg);
}
.audit-detail__changes pre {
  min-height: 120px;
  max-height: 320px;
  padding: var(--ic-space-md);
  margin: 0;
  overflow: auto;
  color: var(--ic-color-text);
  background: var(--ic-color-surface-muted);
  border-radius: var(--ic-radius-md);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: var(--ic-font-xs);
  line-height: 1.65;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
.audit-detail__footer {
  display: flex;
  justify-content: flex-end;
}
@media (max-width: 720px) {
  .audit-detail__changes {
    grid-template-columns: 1fr;
  }
}
</style>
