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
import type { DataTableColumns, DataTableRowKey } from 'naive-ui'
import { DataTable, PermissionButton } from '@/components/common'
import { couponService, memberService } from '@/api/services'
import { useAsyncList } from '@/composables/useAsyncList'
import { confirm } from '@/composables/useConfirm'
import { PERM } from '@/constants/permissions'
import { feedback } from '@/utils/feedback'
import { formatDateTime } from '@/utils/format'
import type { CouponCategory, Member } from '@/types/models'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ 'update:show': [show: boolean] }>()

const searchKeyword = ref('')
const initializing = ref(false)
const submitting = ref(false)
const selectedMembers = ref(new Map<string, Member>())
const couponKinds = ref<CouponCategory[]>([])
const couponTemplateOptions = ref<Array<{ label: string; value: number }>>([])
const form = reactive<{ reason: string }>({
  reason: '',
})

interface CouponGrantLine {
  key: number
  templateId: number | null
  expiresAt: number | null
  quantity: number | null
}

let nextGrantLineKey = 0
const grantLines = ref<CouponGrantLine[]>([])

const memberList = useAsyncList<Member>((params) => memberService.list(params), {
  immediate: false,
  pageSize: 10,
})

const columns: DataTableColumns<Member> = [
  { type: 'selection', multiple: true },
  { title: '昵称', key: 'nickname', minWidth: 140, render: (member) => member.nickname || '—' },
  { title: '手机号', key: 'phone', width: 140, render: (member) => member.phone || '—' },
  { title: '会员 ID', key: 'id', width: 100 },
]

const selectedList = computed(() => Array.from(selectedMembers.value.values()))
const checkedRowKeys = computed(() => Array.from(selectedMembers.value.keys()))
const currentPageKeys = computed(() => memberList.rows.value.map((member) => String(member.id)))
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
    .filter((templateId): templateId is number => templateId != null),
))
const couponKindsPerMember = computed(() => selectedTemplateIds.value.size)
const couponsPerMember = computed(() => grantLines.value.reduce((total, line) => {
  if (line.templateId == null) return total
  const quantity = Number(line.quantity)
  return total + (Number.isInteger(quantity) && quantity > 0 ? quantity : 0)
}, 0))
const totalCouponCount = computed(() => couponsPerMember.value * selectedList.value.length)
const canAddCouponKind = computed(
  () => grantLines.value.length < Math.min(20, couponTemplateOptions.value.length),
)

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
  memberList.page.value = 1
  delete memberList.filters.keyword
}

function createGrantLine(): CouponGrantLine {
  nextGrantLineKey += 1
  return {
    key: nextGrantLineKey,
    templateId: null,
    expiresAt: Date.now() + 30 * 24 * 60 * 60 * 1000,
    quantity: 1,
  }
}

async function initialize(): Promise<void> {
  reset()
  initializing.value = true
  try {
    const [couponResult] = await Promise.all([
      couponService.categories({ page: 1, pageSize: 100 }),
      memberList.load(),
    ])
    couponKinds.value = couponResult.rows.filter((item) => item.status === 'active')
    couponTemplateOptions.value = couponKinds.value.map((item) => ({
      label: `${item.name} · 默认 ${item.defaultValidityDays} 天`,
      value: Number(item.canonicalTemplateId),
    }))
    if (!couponTemplateOptions.value.length) {
      feedback.message.warning('暂无启用中的券种，请联系总后台配置')
    }
  } catch {
    feedback.message.error('读取补发券信息失败')
  } finally {
    initializing.value = false
  }
}

function close(): void {
  if (!submitting.value) emit('update:show', false)
}

function applySearch(): void {
  const keyword = searchKeyword.value.trim()
  if (keyword) memberList.filters.keyword = keyword
  else delete memberList.filters.keyword
  memberList.page.value = 1
  void memberList.load()
}

function resetSearch(): void {
  searchKeyword.value = ''
  delete memberList.filters.keyword
  memberList.page.value = 1
  void memberList.load()
}

function updateCheckedRows(keys: DataTableRowKey[]): void {
  const nextKeys = new Set(keys.map(String))
  const next = new Map(selectedMembers.value)
  for (const member of memberList.rows.value) {
    const key = String(member.id)
    if (nextKeys.has(key)) next.set(key, member)
    else next.delete(key)
  }
  selectedMembers.value = next
}

function toggleCurrentPage(checked: boolean): void {
  const next = new Map(selectedMembers.value)
  for (const member of memberList.rows.value) {
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

function handleTemplateChange(line: CouponGrantLine, templateId: number | null): void {
  line.templateId = templateId
  const kind = couponKinds.value.find((item) => Number(item.canonicalTemplateId) === templateId)
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

async function grantToMembers(
  members: Member[],
  items: Array<{ templateId: number; expiresAt: string; quantity: number }>,
  reason: string,
): Promise<Member[]> {
  const failures: Member[] = []
  let nextIndex = 0
  const worker = async () => {
    while (nextIndex < members.length) {
      const member = members[nextIndex++]
      try {
        await memberService.grantCouponBatch(member.id, { items, reason })
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
  if (!members.length) {
    feedback.message.error('请至少选择一位会员')
    return
  }
  if (!reason) {
    feedback.message.error('请填写补发原因')
    return
  }

  const templateIds = new Set<number>()
  const items: Array<{ templateId: number; expiresAt: string; quantity: number }> = []
  const itemSummaries: string[] = []
  for (const [index, line] of grantLines.value.entries()) {
    const position = `第 ${index + 1} 种券`
    if (line.templateId == null) {
      feedback.message.error(`${position}尚未选择券种`)
      return
    }
    if (templateIds.has(line.templateId)) {
      feedback.message.error('同一券种请合并数量后补发')
      return
    }
    templateIds.add(line.templateId)
    if (!line.expiresAt || line.expiresAt <= Date.now()) {
      feedback.message.error(`${position}的有效期必须晚于当前时间`)
      return
    }
    const quantity = Number(line.quantity)
    if (!Number.isInteger(quantity) || quantity < 1 || quantity > 99) {
      feedback.message.error(`${position}的数量须为 1 至 99 的整数`)
      return
    }
    const expiresAt = new Date(line.expiresAt).toISOString()
    const templateLabel = couponKinds.value.find(
      (item) => Number(item.canonicalTemplateId) === line.templateId,
    )?.name ?? `券种 #${line.templateId}`
    items.push({ templateId: line.templateId, expiresAt, quantity })
    itemSummaries.push(`${templateLabel} × ${quantity}（至 ${formatDateTime(expiresAt)}）`)
  }

  const perMemberCount = items.reduce((total, item) => total + item.quantity, 0)
  const totalCount = perMemberCount * members.length
  const confirmed = await confirm({
    content: `确认向 ${members.length} 位会员补发 ${items.length} 种本店券，每人 ${perMemberCount} 张、共 ${totalCount} 张？${itemSummaries.join('；')}。操作将写入审计日志。`,
    danger: true,
  })
  if (!confirmed) return

  submitting.value = true
  try {
    const failures = await grantToMembers(members, items, reason)
    if (failures.length) {
      selectedMembers.value = new Map(failures.map((member) => [String(member.id), member]))
      const succeeded = members.length - failures.length
      feedback.message.error(`已成功补发 ${succeeded} 人、${succeeded * perMemberCount} 张，失败 ${failures.length} 人；失败会员已保留，可重试`)
      return
    }
    feedback.message.success(`已向 ${members.length} 位会员补发 ${items.length} 种本店券，共 ${totalCount} 张`)
    emit('update:show', false)
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
    style="width: 1080px; max-width: 96vw"
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
              :loading="memberList.loading.value"
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
              :disabled="!memberList.rows.value.length"
              @update:checked="toggleCurrentPage"
            >
              全选本页
            </NCheckbox>
            <span>本页已选 {{ selectedCurrentPageCount }} / {{ memberList.rows.value.length }}</span>
          </div>
          <DataTable
            :columns="columns"
            :data="memberList.rows.value"
            :loading="initializing || memberList.loading.value"
            :page="memberList.page.value"
            :page-size="memberList.pageSize.value"
            :total="memberList.total.value"
            :checked-row-keys="checkedRowKeys"
            :row-key="(member) => String(member.id)"
            :scroll-x="560"
            empty-text="暂无匹配会员"
            @update:page="memberList.setPage"
            @update:page-size="memberList.setPageSize"
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
            <span>每位已选会员都会收到以下全部券种，且仅限当前门店使用</span>
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
          {{ selectedList.length }} 人，共 {{ totalCouponCount }} 张
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
          :permissions="[PERM.couponWrite]"
          type="primary"
          :loading="submitting"
          :disabled="!selectedList.length"
          @click="submit"
        >
          确认补发（{{ selectedList.length }} 人 / {{ totalCouponCount }} 张）
        </PermissionButton>
      </NSpace>
    </template>
  </NModal>
</template>

<style scoped>
.bulk-grant {
  display: flex;
  flex-direction: column;
  gap: var(--ic-space-5);
}
.member-picker {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 280px;
}
.member-picker__results {
  min-width: 0;
  padding-right: var(--ic-space-5);
}
.member-picker__search {
  display: grid;
  grid-template-columns: minmax(180px, 1fr) auto auto;
  gap: var(--ic-space-2);
  margin-bottom: var(--ic-space-4);
}
.member-picker__page-select {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--ic-space-2);
  margin-bottom: var(--ic-space-2);
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-sm);
}
.selected-members {
  min-width: 0;
  padding-left: var(--ic-space-5);
  border-left: 1px solid var(--ic-color-border);
}
.selected-members__header,
.selected-members__item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--ic-space-2);
}
.selected-members__header {
  min-height: 34px;
  padding-bottom: var(--ic-space-2);
  border-bottom: 1px solid var(--ic-color-border);
}
.selected-members__list {
  max-height: 340px;
  overflow: auto;
}
.selected-members__item {
  padding: var(--ic-space-2) 0;
  border-bottom: 1px solid var(--ic-color-border);
}
.selected-members__item > div {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: var(--ic-space-1);
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
  margin: var(--ic-space-5) 0 0;
  text-align: center;
}
.grant-form {
  display: flex;
  flex-direction: column;
  gap: var(--ic-space-3);
  padding-top: var(--ic-space-5);
  border-top: 1px solid var(--ic-color-border);
}
.grant-form__header,
.grant-form__row {
  display: grid;
  align-items: end;
  gap: var(--ic-space-3);
}
.grant-form__header {
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
}
.grant-form__header > div {
  display: flex;
  flex-direction: column;
  gap: var(--ic-space-1);
}
.grant-form__header span,
.grant-form__summary {
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-sm);
}
.grant-form__row {
  grid-template-columns: minmax(180px, 1.2fr) minmax(210px, 1fr) minmax(110px, 0.55fr) auto;
  padding-bottom: var(--ic-space-3);
  border-bottom: 1px solid var(--ic-color-border);
}
.grant-form label {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: var(--ic-space-2);
  font-size: var(--ic-font-sm);
}
.grant-form__remove {
  align-self: center;
}
.grant-form__summary {
  margin: 0;
}
.grant-form__reason {
  margin-top: var(--ic-space-1);
}
@media (max-width: 840px) {
  .member-picker {
    grid-template-columns: 1fr;
  }
  .member-picker__results {
    padding-right: 0;
  }
  .selected-members {
    margin-top: var(--ic-space-5);
    padding: var(--ic-space-5) 0 0;
    border-top: 1px solid var(--ic-color-border);
    border-left: 0;
  }
  .selected-members__list {
    max-height: 220px;
  }
  .grant-form__row {
    grid-template-columns: 1fr;
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
}
</style>
