<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import {
  NButton,
  NCheckbox,
  NDatePicker,
  NInput,
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
  templateId: string | null
  storeTarget: string | null
  expiresAt: number | null
  reason: string
}>({
  templateId: null,
  storeTarget: null,
  expiresAt: null,
  reason: '',
})

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
const selectedCouponKind = computed(() =>
  couponKinds.value.find((item) => String(item.canonicalTemplateId) === form.templateId),
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
  form.templateId = null
  form.storeTarget = null
  form.expiresAt = Date.now() + 30 * 24 * 60 * 60 * 1000
  form.reason = ''
  memberTable.pagination.page = 1
  delete memberTable.filters.keyword
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

function handleTemplateChange(templateId: string | null): void {
  form.templateId = templateId
  form.storeTarget = templateId ? 'global' : null
  const kind = couponKinds.value.find((item) => String(item.canonicalTemplateId) === templateId)
  form.expiresAt = Date.now() + (kind?.defaultValidityDays || 30) * 24 * 60 * 60 * 1000
}

async function grantToMembers(members: Member[], payload: Record<string, unknown>): Promise<Member[]> {
  const failures: Member[] = []
  let nextIndex = 0
  const worker = async () => {
    while (nextIndex < members.length) {
      const member = members[nextIndex++]
      try {
        await http.post(API_PATHS.members.couponEntitlements(String(member.id)), payload, {
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
  if (!form.templateId) return toastError('请选择需要补发的券种')
  if (!form.storeTarget) return toastError('请选择优惠券适用门店')
  if (!form.expiresAt || form.expiresAt <= Date.now()) return toastError('有效期必须晚于当前时间')
  if (!reason) return toastError('请填写补发原因')

  const expiresAt = new Date(form.expiresAt).toISOString()
  const storeLabel = form.storeTarget === 'global'
    ? '全部门店'
    : (storeOptions.value.find((item) => item.value === form.storeTarget)?.label
      ?? `门店 #${form.storeTarget}`)
  const templateLabel = selectedCouponKind.value?.name ?? `券种 #${form.templateId}`
  submitting.value = true
  try {
    const ok = await runAudited({
      title: '确认批量补发优惠券',
      content: `将向 ${members.length} 位会员补发“${templateLabel}”，适用于${storeLabel}，有效期至 ${formatDateTime(expiresAt)}。原因：${reason}`,
      highRisk: true,
      positiveText: `确认补发（${members.length} 人）`,
      execute: async () => {
        const failures = await grantToMembers(members, {
          templateId: Number(form.templateId),
          scopeType: form.storeTarget === 'global' ? 'global' : 'store',
          storeId: form.storeTarget === 'global' ? null : Number(form.storeTarget),
          expiresAt,
          reason,
        })
        if (failures.length) {
          selectedMembers.value = new Map(failures.map((member) => [String(member.id), member]))
          const succeeded = members.length - failures.length
          throw new Error(`已成功补发 ${succeeded} 人，失败 ${failures.length} 人；失败会员已保留，可重试`)
        }
      },
      successText: `已向 ${members.length} 位会员补发优惠券`,
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
        <label>
          <span>券种</span>
          <NSelect
            v-model:value="form.templateId"
            :options="couponTemplateOptions"
            :loading="initializing"
            filterable
            placeholder="选择启用中的券种"
            @update:value="handleTemplateChange"
          />
        </label>
        <label>
          <span>适用门店</span>
          <NSelect
            v-model:value="form.storeTarget"
            :options="grantStoreOptions"
            :disabled="!form.templateId"
            placeholder="请先选择券种"
          />
        </label>
        <label>
          <span>有效期至</span>
          <NDatePicker
            v-model:value="form.expiresAt"
            type="datetime"
            clearable
            style="width: 100%"
          />
        </label>
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
          :disabled="submitting || !selectedList.length"
          @click="submit"
        >
          {{ submitting ? '补发中…' : `确认补发（${selectedList.length} 人）` }}
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
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: var(--ic-space-md);
  padding-top: var(--ic-space-lg);
  border-top: 1px solid var(--ic-color-divider);
}
.grant-form label {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: var(--ic-space-sm);
  font-size: var(--ic-font-sm);
}
.grant-form__reason {
  grid-column: 1 / -1;
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
  .grant-form {
    grid-template-columns: 1fr;
  }
  .grant-form__reason {
    grid-column: auto;
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
}
</style>
