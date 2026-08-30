<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import {
  NButton,
  NCheckbox,
  NDatePicker,
  NInput,
  NInputNumber,
  NModal,
  NSelect,
  NSpace,
} from 'naive-ui'
import type { DataTableRowKey } from 'naive-ui'
import DataTable from '@/components/DataTable.vue'
import PermissionButton from '@/components/PermissionButton.vue'
import type { TableColumnList } from '@/components/ui-types'
import { PERMISSIONS } from '@/constants/permissions'
import { couponCategoryService, memberService, storeService } from '@/api/services'
import { http } from '@/api/http'
import { API_PATHS } from '@/constants/api-paths'
import { useDataTable } from '@/composables/useDataTable'
import { runAudited } from '@/composables/useAuditedAction'
import { textColumn } from '@/utils/columns'
import { formatDateTime } from '@/utils/format'
import { toastError } from '@/utils/feedback'
import type { CouponCategory, Member } from '@/api/models'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ 'update:show': [show: boolean] }>()

const searchKeyword = ref('')
const initializing = ref(false)
const submitting = ref(false)
const selectedMembers = ref(new Map<string, Member>())
const couponKinds = ref<CouponCategory[]>([])
const couponTemplateOptions = ref<Array<{ label: string; value: string }>>([])
const storeOptions = ref<Array<{ label: string; value: string }>>([])
const form = reactive<{
  reason: string
}>({
  reason: '',
})

interface CouponGrantLine {
  key: number
  templateId: string | null
  storeTarget: string | null
  expiresAt: number | null
  quantity: number | null
}

let nextGrantLineKey = 0
const grantLines = ref<CouponGrantLine[]>([])

const memberTable = useDataTable<Member>({
  fetcher: memberService.list,
  immediate: false,
  defaultPageSize: 10,
})

const columns: TableColumnList<Member> = [
  { type: 'selection', multiple: true },
  textColumn<Member>('昵称', 'nickname', { minWidth: 140 }),
  textColumn<Member>('手机号', 'phone', { width: 140 }),
  textColumn<Member>('会员 ID', 'id', { width: 100 }),
]

const selectedList = computed(() => Array.from(selectedMembers.value.values()))
const checkedRowKeys = computed(() => Array.from(selectedMembers.value.keys()))
const currentPageKeys = computed(() => memberTable.rows.value.map((member) => String(member.id)))
const selectedCurrentPageCount = computed(() =>
  currentPageKeys.value.filter((key) => selectedMembers.value.has(key)).length,
)
const allCurrentPageSelected = computed(
  () => currentPageKeys.value.length > 0 && selectedCurrentPageCount.value === currentPageKeys.value.length,
)
const partiallySelectedCurrentPage = computed(
  () => selectedCurrentPageCount.value > 0 && !allCurrentPageSelected.value,
)
const selectedTemplateIds = computed(() => new Set(
  grantLines.value
    .map((line) => line.templateId)
    .filter((templateId): templateId is string => templateId != null),
))
const couponKindsPerMember = computed(() => selectedTemplateIds.value.size)
const couponsPerMember = computed(() => grantLines.value.reduce((total, line) => {
  if (!line.templateId) return total
  const quantity = Number(line.quantity)
  return total + (Number.isInteger(quantity) && quantity > 0 ? quantity : 0)
}, 0))
const totalCouponCount = computed(() => couponsPerMember.value * selectedList.value.length)
const canAddCouponKind = computed(
  () => grantLines.value.length < Math.min(20, couponTemplateOptions.value.length),
)
const grantStoreOptions = computed(() => [
  { label: '全部门店', value: 'global' },
  ...storeOptions.value.map((store) => ({
    label: `指定门店 · ${store.label}`,
    value: store.value,
  })),
])

watch(
  () => props.show,
  (show) => {
    if (show) void initialize()
  },
)

function reset(): void {
  searchKeyword.value = ''
  selectedMembers.value = new Map()
  form.reason = ''
  grantLines.value = [createGrantLine()]
  memberTable.pagination.page = 1
  delete memberTable.filters.keyword
}

function createGrantLine(): CouponGrantLine {
  nextGrantLineKey += 1
  return {
    key: nextGrantLineKey,
    templateId: null,
    storeTarget: null,
    expiresAt: Date.now() + 30 * 24 * 60 * 60 * 1000,
    quantity: 1,
  }
}

async function initialize(): Promise<void> {
  reset()
  initializing.value = true
  try {
    const [couponResult, storeResult] = await Promise.all([
      couponCategoryService.list({ status: 'active', page: 1, pageSize: 100 }),
      storeService.list({ page: 1, pageSize: 100 }),
      memberTable.load(),
    ])
    couponKinds.value = couponResult.items
    couponTemplateOptions.value = couponResult.items.map((item) => ({
      label: `${item.name} · 默认 ${item.defaultValidityDays} 天`,
      value: String(item.canonicalTemplateId),
    }))
    storeOptions.value = storeResult.items.map((store) => ({
      label: store.name,
      value: String(store.id),
    }))
    if (!couponTemplateOptions.value.length) {
      toastError('暂无启用中的券种，请先在券种管理中启用')
    }
  } catch (error) {
    toastError((error as { message?: string }).message ?? '读取补发券信息失败')
  } finally {
    initializing.value = false
  }
}

function close(): void {
  if (!submitting.value) emit('update:show', false)
}

function applySearch(): void {
  const keyword = searchKeyword.value.trim()
  if (keyword) memberTable.filters.keyword = keyword
  else delete memberTable.filters.keyword
  memberTable.pagination.page = 1
  void memberTable.load()
}

function resetSearch(): void {
  searchKeyword.value = ''
  delete memberTable.filters.keyword
  memberTable.pagination.page = 1
  void memberTable.load()
}

function updateCheckedRows(keys: DataTableRowKey[]): void {
  const nextKeys = new Set(keys.map(String))
  const next = new Map(selectedMembers.value)
  for (const member of memberTable.rows.value) {
    const key = String(member.id)
    if (nextKeys.has(key)) next.set(key, member)
    else next.delete(key)
  }
  selectedMembers.value = next
}

function toggleCurrentPage(checked: boolean): void {
  const next = new Map(selectedMembers.value)
  for (const member of memberTable.rows.value) {
    const key = String(member.id)
    if (checked) next.set(key, member)
    else next.delete(key)
  }
  selectedMembers.value = next
}

function removeSelected(member: Member): void {
  const next = new Map(selectedMembers.value)
  next.delete(String(member.id))
  selectedMembers.value = next
}

function clearSelected(): void {
  selectedMembers.value = new Map()
}

function templateOptionsFor(line: CouponGrantLine) {
  return couponTemplateOptions.value.map((option) => ({
    ...option,
    disabled: option.value !== line.templateId && selectedTemplateIds.value.has(option.value),
  }))
}

function handleTemplateChange(line: CouponGrantLine, templateId: string | null): void {
  line.templateId = templateId
  line.storeTarget = templateId ? 'global' : null
  const kind = couponKinds.value.find((item) => String(item.canonicalTemplateId) === templateId)
  line.expiresAt = Date.now() + (kind?.defaultValidityDays || 30) * 24 * 60 * 60 * 1000
}

function addGrantLine(): void {
  if (!canAddCouponKind.value) return
  grantLines.value.push(createGrantLine())
}

function removeGrantLine(key: number): void {
  if (grantLines.value.length <= 1) return
  grantLines.value = grantLines.value.filter((line) => line.key !== key)
}

async function grantToMembers(members: Member[], payload: Record<string, unknown>): Promise<Member[]> {
  const failures: Member[] = []
  let nextIndex = 0
  const worker = async () => {
    while (nextIndex < members.length) {
      const member = members[nextIndex++]
      try {
        await http.post(API_PATHS.members.couponEntitlementBatch(String(member.id)), payload, {
          idempotent: true,
        })
      } catch {
        failures.push(member)
      }
    }
  }
  await Promise.all(Array.from({ length: Math.min(5, members.length) }, worker))
  return failures
}

async function submit(): Promise<void> {
  const members = selectedList.value
  const reason = form.reason.trim()
  if (!members.length) return toastError('请至少选择一位会员')
  if (!grantLines.value.length) return toastError('请至少添加一种优惠券')
  if (!reason) return toastError('请填写补发原因')

  const templateIds = new Set<string>()
  const items: Array<{
    templateId: number
    scopeType: 'global' | 'store'
    storeId: number | null
    expiresAt: string
    quantity: number
  }> = []
  const itemSummaries: string[] = []
  for (const [index, line] of grantLines.value.entries()) {
    const position = `第 ${index + 1} 种券`
    if (!line.templateId) return toastError(`${position}尚未选择券种`)
    if (templateIds.has(line.templateId)) return toastError('同一券种请合并数量后补发')
    templateIds.add(line.templateId)
    if (!line.storeTarget) return toastError(`${position}尚未选择适用门店`)
    if (!line.expiresAt || line.expiresAt <= Date.now()) return toastError(`${position}的有效期必须晚于当前时间`)
    const quantity = Number(line.quantity)
    if (!Number.isInteger(quantity) || quantity < 1 || quantity > 99) {
      return toastError(`${position}的数量须为 1 至 99 的整数`)
    }
    const expiresAt = new Date(line.expiresAt).toISOString()
    const templateLabel = couponKinds.value.find(
      (item) => String(item.canonicalTemplateId) === line.templateId,
    )?.name ?? `券种 #${line.templateId}`
    const storeLabel = line.storeTarget === 'global'
      ? '全部门店'
      : (storeOptions.value.find((item) => item.value === line.storeTarget)?.label
        ?? `门店 #${line.storeTarget}`)
    items.push({
      templateId: Number(line.templateId),
      scopeType: line.storeTarget === 'global' ? 'global' : 'store',
      storeId: line.storeTarget === 'global' ? null : Number(line.storeTarget),
      expiresAt,
      quantity,
    })
    itemSummaries.push(`${templateLabel} × ${quantity}（${storeLabel}，至 ${formatDateTime(expiresAt)}）`)
  }

  const perMemberCount = items.reduce((total, item) => total + item.quantity, 0)
  const totalCount = perMemberCount * members.length
  submitting.value = true
  try {
    const ok = await runAudited({
      title: '确认批量补发优惠券',
      content: `将向 ${members.length} 位会员补发 ${items.length} 种券，每人 ${perMemberCount} 张，共 ${totalCount} 张。${itemSummaries.join('；')}。原因：${reason}`,
      highRisk: true,
      positiveText: `确认补发（共 ${totalCount} 张）`,
      execute: async () => {
        const failures = await grantToMembers(members, {
          items,
          reason,
        })
        if (failures.length) {
          selectedMembers.value = new Map(failures.map((member) => [String(member.id), member]))
          const succeeded = members.length - failures.length
          throw new Error(`已成功补发 ${succeeded} 人、${succeeded * perMemberCount} 张，失败 ${failures.length} 人；失败会员已保留，可重试`)
        }
      },
      successText: `已向 ${members.length} 位会员补发 ${items.length} 种优惠券，共 ${totalCount} 张`,
    })
    if (ok) emit('update:show', false)
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    title="补发券"
    :mask-closable="false"
    :close-on-esc="!submitting"
    style="width: 1120px; max-width: 96vw"
    content-style="max-height: calc(100vh - 190px); overflow: auto"
    @update:show="(value) => !value && close()"
  >
    <div class="bulk-grant">
      <div class="member-picker">
        <section class="member-picker__results">
          <div class="member-picker__search">
            <NInput
              v-model:value="searchKeyword"
              clearable
              placeholder="输入手机号、昵称模糊搜索"
              @keyup.enter="applySearch"
            />
            <NButton
              type="primary"
              :loading="memberTable.loading.value"
              @click="applySearch"
            >
              搜索
            </NButton>
            <NButton @click="resetSearch">
              重置
            </NButton>
          </div>
          <div class="member-picker__page-select">
            <NCheckbox
              :checked="allCurrentPageSelected"
              :indeterminate="partiallySelectedCurrentPage"
              :disabled="!memberTable.rows.value.length"
              @update:checked="toggleCurrentPage"
            >
              全选本页
            </NCheckbox>
            <span>本页已选 {{ selectedCurrentPageCount }} / {{ memberTable.rows.value.length }}</span>
          </div>
          <DataTable
            :columns="columns"
            :data="memberTable.rows.value"
            :loading="initializing || memberTable.loading.value"
            :page="memberTable.pagination.page"
            :page-size="memberTable.pagination.pageSize"
            :item-count="memberTable.pagination.itemCount"
            :checked-row-keys="checkedRowKeys"
            :row-key="(member) => String(member.id)"
            empty-text="暂无匹配会员"
            @update:page="memberTable.handlePageChange"
            @update:page-size="memberTable.handlePageSizeChange"
            @update:checked-row-keys="updateCheckedRows"
          />
        </section>

        <aside class="selected-members">
          <div class="selected-members__header">
            <strong>已选会员（{{ selectedList.length }}）</strong>
            <NButton
              v-if="selectedList.length"
              text
              size="tiny"
              @click="clearSelected"
            >
              清空
            </NButton>
          </div>
          <div
            v-if="selectedList.length"
            class="selected-members__list"
          >
            <div
              v-for="member in selectedList"
              :key="String(member.id)"
              class="selected-members__item"
            >
              <div>
                <strong>{{ member.nickname || `会员 ${member.id}` }}</strong>
                <span>{{ member.phone || '未绑定手机号' }}</span>
              </div>
              <NButton
                text
                size="tiny"
                @click="removeSelected(member)"
              >
                移除
              </NButton>
            </div>
          </div>
          <p
            v-else
            class="selected-members__empty"
          >
            尚未选择会员
          </p>
        </aside>
      </div>

      <section class="grant-form">
        <div class="grant-form__header">
          <div>
            <strong>补发券明细</strong>
            <span>每位已选会员都会收到以下全部券种</span>
          </div>
          <NButton
            size="small"
            :disabled="!canAddCouponKind"
            @click="addGrantLine"
          >
            添加券种
          </NButton>
        </div>
        <div
          v-for="(line, index) in grantLines"
          :key="line.key"
          class="grant-form__row"
        >
          <label>
            <span>券种 {{ index + 1 }}</span>
            <NSelect
              v-model:value="line.templateId"
              :options="templateOptionsFor(line)"
              :loading="initializing"
              filterable
              placeholder="选择启用中的券种"
              @update:value="(value) => handleTemplateChange(line, value)"
            />
          </label>
          <label>
            <span>适用门店</span>
            <NSelect
              v-model:value="line.storeTarget"
              :options="grantStoreOptions"
              :disabled="!line.templateId"
              placeholder="请先选择券种"
            />
          </label>
          <label>
            <span>有效期至</span>
            <NDatePicker
              v-model:value="line.expiresAt"
              type="datetime"
              clearable
              style="width: 100%"
            />
          </label>
          <label>
            <span>每人数量</span>
            <NInputNumber
              v-model:value="line.quantity"
              :min="1"
              :max="99"
              :precision="0"
              placeholder="数量"
              style="width: 100%"
            />
          </label>
          <NButton
            v-if="grantLines.length > 1"
            class="grant-form__remove"
            text
            type="error"
            @click="removeGrantLine(line.key)"
          >
            移除
          </NButton>
        </div>
        <p
          class="grant-form__summary"
          aria-live="polite"
        >
          每人 {{ couponKindsPerMember }} 种、{{ couponsPerMember }} 张；当前已选
          {{ selectedList.length }} 人，共补发 {{ totalCouponCount }} 张
        </p>
        <label class="grant-form__reason">
          <span>补发原因</span>
          <NInput
            v-model:value="form.reason"
            type="textarea"
            :autosize="{ minRows: 2, maxRows: 4 }"
            maxlength="200"
            show-count
            placeholder="必填，将写入审计日志"
          />
        </label>
      </section>
    </div>

    <template #footer>
      <NSpace justify="end">
        <NButton
          :disabled="submitting"
          @click="close"
        >
          取消
        </NButton>
        <PermissionButton
          :permission="PERMISSIONS.COUPON_GLOBAL_WRITE"
          type="primary"
          :disabled="submitting || !selectedList.length || !totalCouponCount"
          @click="submit"
        >
          {{ submitting ? '补发中…' : `确认补发（${totalCouponCount} 张）` }}
        </PermissionButton>
      </NSpace>
    </template>
  </NModal>
</template>

<style scoped>
.bulk-grant {
  display: flex;
  flex-direction: column;
  gap: var(--ic-space-lg);
}
.member-picker {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 280px;
}
.member-picker__results {
  min-width: 0;
  padding-right: var(--ic-space-lg);
}
.member-picker__search {
  display: grid;
  grid-template-columns: minmax(180px, 1fr) auto auto;
  gap: var(--ic-space-sm);
  margin-bottom: var(--ic-space-md);
}
.member-picker__page-select {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--ic-space-sm);
  margin-bottom: var(--ic-space-sm);
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-sm);
}
.member-picker__results :deep(.data-table) {
  padding: 0;
  border: 0;
  border-radius: 0;
}
.selected-members {
  min-width: 0;
  padding-left: var(--ic-space-lg);
  border-left: 1px solid var(--ic-color-divider);
}
.selected-members__header,
.selected-members__item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--ic-space-sm);
}
.selected-members__header {
  min-height: 34px;
  padding-bottom: var(--ic-space-sm);
  border-bottom: 1px solid var(--ic-color-divider);
}
.selected-members__list {
  max-height: 340px;
  overflow: auto;
}
.selected-members__item {
  padding: var(--ic-space-sm) 0;
  border-bottom: 1px solid var(--ic-color-divider);
}
.selected-members__item > div {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: var(--ic-space-xs);
}
.selected-members__item strong,
.selected-members__item span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.selected-members__item span,
.selected-members__empty {
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-sm);
}
.selected-members__empty {
  margin: var(--ic-space-lg) 0 0;
  text-align: center;
}
.grant-form {
  display: flex;
  flex-direction: column;
  padding-top: var(--ic-space-lg);
  border-top: 1px solid var(--ic-color-divider);
}
.grant-form__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--ic-space-md);
  padding-bottom: var(--ic-space-md);
}
.grant-form__header > div {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: var(--ic-space-xs);
}
.grant-form__header span,
.grant-form__summary {
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-sm);
}
.grant-form__row {
  display: grid;
  grid-template-columns: minmax(180px, 1.35fr) minmax(160px, 1fr) minmax(190px, 1.2fr) 104px auto;
  align-items: end;
  gap: var(--ic-space-md);
  padding: var(--ic-space-md) 0;
  border-top: 1px solid var(--ic-color-divider);
}
.grant-form label {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: var(--ic-space-sm);
  font-size: var(--ic-font-sm);
}
.grant-form__remove {
  min-height: 34px;
}
.grant-form__summary {
  margin: 0;
  padding: var(--ic-space-sm) 0 var(--ic-space-md);
  text-align: right;
}
.grant-form__reason {
  padding-top: var(--ic-space-md);
  border-top: 1px solid var(--ic-color-divider);
}
@media (max-width: 840px) {
  .member-picker {
    grid-template-columns: 1fr;
  }
  .member-picker__results {
    padding-right: 0;
  }
  .selected-members {
    margin-top: var(--ic-space-lg);
    padding: var(--ic-space-lg) 0 0;
    border-top: 1px solid var(--ic-color-divider);
    border-left: 0;
  }
  .selected-members__list {
    max-height: 220px;
  }
  .grant-form__row {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .grant-form__remove {
    justify-self: start;
  }
}
@media (max-width: 560px) {
  .member-picker__search {
    grid-template-columns: 1fr 1fr;
  }
  .member-picker__search :deep(.n-input) {
    grid-column: 1 / -1;
  }
  .member-picker__page-select {
    align-items: flex-start;
    flex-direction: column;
  }
  .grant-form__header {
    align-items: flex-start;
  }
  .grant-form__row {
    grid-template-columns: 1fr;
  }
  .grant-form__summary {
    text-align: left;
  }
}
</style>
