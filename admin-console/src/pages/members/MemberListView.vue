<script setup lang="ts">
/**
 * 会员列表 + 人工调账（高风险）。
 * 人工调账涉及钱包，必须选择资产类型、填写原因、二次确认，携带幂等键并写入审计。
 */
import { computed, h, reactive, ref } from 'vue'
import {
  NAvatar,
  NButton,
  NDatePicker,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NModal,
  NSelect,
  NSpace,
  NSpin,
  NTabPane,
  NTabs,
  NTag,
} from 'naive-ui'
import type { DataTableSortState, DataTableSortOrder } from 'naive-ui'
import ResourceListView from '@/components/ResourceListView.vue'
import DataTable from '@/components/DataTable.vue'
import type { ResourceListInstance } from '@/components/ui-types'
import FormDrawer from '@/components/FormDrawer.vue'
import PermissionButton from '@/components/PermissionButton.vue'
import { dateTimeColumn, statusColumn, textColumn, actionsColumn, renderColumn } from '@/utils/columns'
import {
  ASSET_TYPE,
  ASSET_TYPE_OPTIONS,
  COUPON_ENTITLEMENT_STATUS_OPTIONS,
  RESOURCE_STATUS_OPTIONS,
} from '@/constants/enums'
import { PERMISSIONS } from '@/constants/permissions'
import { runAudited } from '@/composables/useAuditedAction'
import { useDataTable } from '@/composables/useDataTable'
import { couponCategoryService, memberService, readonlyLists, storeService } from '@/api/services'
import { http } from '@/api/http'
import { API_PATHS } from '@/constants/api-paths'
import { formatDateTime } from '@/utils/format'
import type {
  CouponCategory,
  Member,
  MemberCouponEntitlement,
  MemberDetail,
  WalletLedgerEntry,
} from '@/api/models'
import type { ListQuery } from '@/api/types'
import type { FilterField, TableColumnList } from '@/components/ui-types'
import { toastError } from '@/utils/feedback'

const listRef = ref<ResourceListInstance | null>(null)
const target = ref<Member | null>(null)
type MemberSortField = 'pointsBalance' | 'coinsBalance' | 'vipLevel'
const sortBy = ref<MemberSortField | ''>('')
const sortOrder = ref<'asc' | 'desc'>('desc')
const balanceFormatter = new Intl.NumberFormat('zh-CN')
const adjustAssetTypeOptions = ASSET_TYPE_OPTIONS.filter(
  ({ value }) => value !== ASSET_TYPE.CASH_BALANCE,
).map(({ label, value }) => ({ label, value }))

const fields: FilterField[] = [
  {
    key: 'keyword',
    label: '昵称 / 手机号',
    type: 'input',
    placeholder: '支持昵称、手机号模糊搜索',
    width: 280,
  },
  {
    key: 'created',
    label: '注册时间',
    type: 'daterange',
    width: 280,
  },
]

const columns = computed<TableColumnList<Member>>(() => [
  textColumn<Member>('ID', 'id', { width: 80 }),
  renderColumn<Member>(
    '头像',
    'avatarUrl',
    (row) => {
      const fallback = () => row.nickname?.trim().slice(0, 1) || String(row.id).slice(-1)
      return h(
        NAvatar,
        { size: 32, round: true, src: row.avatarUrl || undefined, objectFit: 'cover' },
        row.avatarUrl ? { fallback } : { default: fallback },
      )
    },
    64,
  ),
  textColumn<Member>('昵称', 'nickname'),
  textColumn<Member>('手机号', 'phone', { width: 140 }),
  textColumn<Member>('当前积分', 'pointsBalance', {
    width: 120,
    sorter: true,
    sortOrder: columnSortOrder('pointsBalance'),
    render: (row) => balanceFormatter.format(row.pointsBalance ?? 0),
  }),
  textColumn<Member>('金币', 'coinsBalance', {
    width: 120,
    sorter: true,
    sortOrder: columnSortOrder('coinsBalance'),
    render: (row) => balanceFormatter.format(row.coinsBalance ?? 0),
  }),
  textColumn<Member>('VIP 等级', 'vipLevel', {
    width: 150,
    sorter: true,
    sortOrder: columnSortOrder('vipLevel'),
    render: (row) =>
      row.vipLevel
        ? h(
            NTag,
            { size: 'small', bordered: false },
            { default: () => `VIP${row.vipLevel} · ${row.vipTierName || '会员'}` },
          )
        : '—',
  }),
  statusColumn<Member>('状态', 'status', RESOURCE_STATUS_OPTIONS, 100),
  dateTimeColumn<Member>('注册时间', 'createdAt'),
  actionsColumn<Member>(
    (row) =>
      h(NSpace, {}, () => [
        h(
          PermissionButton,
          { permission: PERMISSIONS.MEMBER_READ, onClick: () => openDetail(row) },
          () => '详情',
        ),
        h(
          PermissionButton,
          {
            permission: PERMISSIONS.MEMBER_WALLET_ADJUST,
            type: 'primary',
            onClick: () => openAdjust(row),
          },
          () => '人工调账',
        ),
      ]),
    180,
  ),
])

function columnSortOrder(field: MemberSortField): DataTableSortOrder {
  if (sortBy.value !== field) return false
  return sortOrder.value === 'asc' ? 'ascend' : 'descend'
}

function fetchMembers(query: ListQuery) {
  return memberService.list({
    ...query,
    ...(sortBy.value ? { sortBy: sortBy.value, sortOrder: sortOrder.value } : {}),
  })
}

function handleSorter(sorter: DataTableSortState | DataTableSortState[] | null): void {
  const current = Array.isArray(sorter) ? sorter[0] : sorter
  const field = current?.columnKey
  if (
    !current?.order ||
    (field !== 'pointsBalance' && field !== 'coinsBalance' && field !== 'vipLevel')
  ) {
    sortBy.value = ''
    sortOrder.value = 'desc'
  } else {
    sortBy.value = field
    sortOrder.value = current.order === 'ascend' ? 'asc' : 'desc'
  }
  void listRef.value?.reload()
}

// —— 会员详情（只读） ——
const detailDrawerShow = ref(false)
const detailLoading = ref(false)
const detail = ref<MemberDetail | null>(null)
const detailTab = ref<'coupons' | 'ledger' | 'adjust'>('coupons')
const detailMemberId = ref('')

const ledgerTable = useDataTable<WalletLedgerEntry>({
  fetcher: readonlyLists.walletLedger,
  immediate: false,
})

const couponTable = useDataTable<MemberCouponEntitlement>({
  fetcher: (query) => readonlyLists.memberCouponEntitlements(detailMemberId.value, query),
  immediate: false,
  defaultPageSize: 10,
})

const ledgerReasonLabels: Record<string, string> = {
  point_saving: '存积分',
  point_saving_reward: '存积分审核到账',
  point_saving_coin_reward: '存积分金币奖励',
  point_withdrawal: '取积分',
  food_order_reward: '购买餐品赠送积分',
  food_order_cancel_clawback: '取消订单扣回赠送积分',
  food_order_cancel_rollback: '取消订单失败返还积分',
  sign_in: '签到奖励',
  recharge: '充值到账',
  first_recharge_reward: '用户首充获得积分',
  high_value_recharge_reward: '满额充值获得积分',
  recharge_growth: '充值成长值',
  wechat_payment_growth: '微信支付获得成长值',
  order_payment: '订单支付',
  refund: '订单退款返还',
  admin_adjustment: '管理员调账',
  low_spend_reward: '预约低消达标奖励',
}

const ledgerColumns: TableColumnList<WalletLedgerEntry> = [
  renderColumn<WalletLedgerEntry>(
    '类型',
    'assetType',
    (row) => assetTypeLabel(row.assetType),
    88,
  ),
  renderColumn<WalletLedgerEntry>(
    '变动',
    'amount',
    (row) => `${row.direction === 'debit' ? '-' : '+'}${balanceFormatter.format(row.amount)}`,
    100,
  ),
  renderColumn<WalletLedgerEntry>(
    '变动后余额',
    'balanceAfter',
    (row) => balanceFormatter.format(row.balanceAfter ?? 0),
    120,
  ),
  renderColumn<WalletLedgerEntry>(
    '原因',
    'reason',
    (row) => ledgerReasonLabels[row.reason ?? ''] ?? row.reason ?? '—',
  ),
  dateTimeColumn<WalletLedgerEntry>('时间', 'createdAt', 168),
]

const couponTypeLabels: Record<string, string> = {
  event_ticket: '赛事券',
  admission_ticket: '门票券',
  snack: '小吃券',
  alcohol: '酒水券',
  beverage: '饮料券',
  drink: '饮品或啤酒券',
  meal: '餐食券',
  gift: '礼品券',
}

const couponColumns: TableColumnList<MemberCouponEntitlement> = [
  textColumn<MemberCouponEntitlement>('券名称', 'templateName', { minWidth: 140 }),
  renderColumn<MemberCouponEntitlement>(
    '类型',
    'couponType',
    (row) => couponTypeLabels[row.couponType] ?? row.couponType,
    100,
  ),
  renderColumn<MemberCouponEntitlement>(
    '适用门店',
    'storeName',
    (row) => row.storeName || '全部门店',
    120,
  ),
  statusColumn<MemberCouponEntitlement>(
    '状态',
    'status',
    COUPON_ENTITLEMENT_STATUS_OPTIONS,
    92,
  ),
  dateTimeColumn<MemberCouponEntitlement>('有效期至', 'expiresAt', 168),
  actionsColumn<MemberCouponEntitlement>(
    (row) => {
      if (row.status !== 'active' && row.status !== 'expired') return '—'
      return h(NSpace, { size: 6, wrap: false }, () => [
        h(
          PermissionButton,
          {
            permission: PERMISSIONS.COUPON_GLOBAL_WRITE,
            onClick: () => openCouponExpiry(row),
          },
          () => '改期',
        ),
        h(
          PermissionButton,
          {
            permission: PERMISSIONS.COUPON_GLOBAL_WRITE,
            type: 'error',
            onClick: () => openCouponVoid(row),
          },
          () => '删除',
        ),
      ])
    },
    150,
  ),
]

function assetTypeLabel(assetType: string): string {
  return ASSET_TYPE_OPTIONS.find((o) => o.value === assetType)?.label ?? assetType
}

function statusLabel(status: string): string {
  return RESOURCE_STATUS_OPTIONS.find((o) => o.value === status)?.label ?? status
}

async function openDetail(row: Member): Promise<void> {
  detail.value = null
  detailTab.value = 'coupons'
  detailMemberId.value = String(row.id)
  target.value = row
  resetAdjustForm()
  detailDrawerShow.value = true
  detailLoading.value = true
  ledgerTable.filters.memberId = row.id
  ledgerTable.pagination.page = 1
  void ledgerTable.load()
  couponTable.pagination.page = 1
  void couponTable.load()
  try {
    detail.value = await http.get<MemberDetail>(API_PATHS.members.detail(row.id))
  } catch (e) {
    toastError((e as { message?: string }).message ?? '加载会员详情失败')
    detailDrawerShow.value = false
  } finally {
    detailLoading.value = false
  }
}

// —— 人工调账表单 ——
const drawerShow = ref(false)
const submitting = ref(false)
const adjust = reactive<{ assetType: string | null; amount: number | null; reason: string }>({
  assetType: null,
  amount: null,
  reason: '',
})

function resetAdjustForm(): void {
  adjust.assetType = null
  adjust.amount = null
  adjust.reason = ''
}

function openAdjust(row: Member): void {
  target.value = row
  resetAdjustForm()
  drawerShow.value = true
}

async function submitAdjust(closeOnSuccess = true): Promise<void> {
  if (!target.value) return
  if (!adjust.assetType) return toastError('请选择资产类型')
  if (adjust.amount == null || adjust.amount === 0) return toastError('请输入调整数量（非零，正增负减）')
  if (!adjust.reason.trim()) return toastError('人工调账必须填写原因')

  const member = target.value
  submitting.value = true
  try {
    const ok = await runAudited({
      title: '确认人工调账',
      content: `将对会员「${member.nickname ?? member.id}」的${assetTypeLabel(adjust.assetType)}调整 ${adjust.amount}，原因：${adjust.reason}。该操作不可逆，携带幂等键并写入审计。`,
      highRisk: true,
      positiveText: '确认调账',
      execute: () =>
        http.post(
          API_PATHS.members.walletAdjustments(member.id),
          {
            assetType: adjust.assetType,
            direction: adjust.amount! >= 0 ? 'credit' : 'debit',
            amount: Math.abs(adjust.amount!),
            reason: adjust.reason,
          },
          { idempotent: true },
        ),
      successText: '调账已提交',
    })
    if (ok) {
      if (closeOnSuccess) drawerShow.value = false
      resetAdjustForm()
      listRef.value?.reload()
      if (detailDrawerShow.value && detail.value && String(detail.value.id) === String(member.id)) {
        detail.value = await http.get<MemberDetail>(API_PATHS.members.detail(member.id))
        void ledgerTable.reload()
      }
    }
  } finally {
    submitting.value = false
  }
}

// —— 用户持券管理 ——
type CouponActionMode = 'grant' | 'expiry' | 'void'
const couponDrawerShow = ref(false)
const couponSubmitting = ref(false)
const couponActionMode = ref<CouponActionMode>('grant')
const couponActionTarget = ref<MemberCouponEntitlement | null>(null)
const couponKinds = ref<CouponCategory[]>([])
const couponTemplateOptions = ref<{ label: string; value: string }[]>([])
const couponStoreOptions = ref<{ label: string; value: string }[]>([])
const couponForm = reactive<{
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

const selectedCouponTemplate = computed(() =>
  couponKinds.value.find((item) => String(item.canonicalTemplateId) === couponForm.templateId),
)

const couponGrantStoreOptions = computed(() => {
  const template = selectedCouponTemplate.value
  if (!template) return []
  return [
    { label: '全部门店', value: 'global' },
    ...couponStoreOptions.value.map((item) => ({
      label: `指定门店 · ${item.label}`,
      value: item.value,
    })),
  ]
})

const couponDrawerTitle = computed(() => {
  if (couponActionMode.value === 'grant') return '补发优惠券'
  if (couponActionMode.value === 'expiry') return '修改优惠券有效期'
  return '删除用户优惠券'
})

const couponSubmitText = computed(() => {
  if (couponActionMode.value === 'grant') return '确认补发'
  if (couponActionMode.value === 'expiry') return '确认改期'
  return '确认删除'
})

function defaultCouponExpiry(): number {
  return Date.now() + 30 * 24 * 60 * 60 * 1000
}

function resetCouponForm(): void {
  couponForm.templateId = null
  couponForm.storeTarget = null
  couponForm.expiresAt = defaultCouponExpiry()
  couponForm.reason = ''
  couponActionTarget.value = null
}

async function loadCouponTemplateOptions(): Promise<void> {
  const [templateResult, storeResult] = await Promise.all([
    couponCategoryService.list({ status: 'active', page: 1, pageSize: 100 }),
    storeService.list({ page: 1, pageSize: 100 }),
  ])
  couponKinds.value = templateResult.items
  couponStoreOptions.value = storeResult.items.map((store) => ({
    label: store.name,
    value: String(store.id),
  }))
  couponTemplateOptions.value = couponKinds.value.map((item: CouponCategory) => ({
    label: `${item.name} · ${couponTypeLabels[item.businessType] ?? item.businessType} · 默认 ${item.defaultValidityDays} 天`,
    value: String(item.canonicalTemplateId),
  }))
}

function handleCouponTemplateChange(templateId: string | null): void {
  couponForm.templateId = templateId
  couponForm.storeTarget = templateId ? 'global' : null
  const kind = couponKinds.value.find((item) => String(item.canonicalTemplateId) === templateId)
  couponForm.expiresAt = Date.now() + (kind?.defaultValidityDays || 30) * 24 * 60 * 60 * 1000
}

async function openCouponGrant(): Promise<void> {
  resetCouponForm()
  couponActionMode.value = 'grant'
  try {
    await loadCouponTemplateOptions()
  } catch (error) {
    toastError((error as { message?: string }).message ?? '读取券种或门店失败')
    return
  }
  if (!couponTemplateOptions.value.length) {
    toastError('暂无启用中的券种，请先在券种管理中启用')
    return
  }
  couponDrawerShow.value = true
}

function openCouponExpiry(row: MemberCouponEntitlement): void {
  resetCouponForm()
  couponActionMode.value = 'expiry'
  couponActionTarget.value = row
  couponForm.expiresAt = row.expiresAt ? new Date(row.expiresAt).getTime() : defaultCouponExpiry()
  couponDrawerShow.value = true
}

function openCouponVoid(row: MemberCouponEntitlement): void {
  resetCouponForm()
  couponActionMode.value = 'void'
  couponActionTarget.value = row
  couponForm.expiresAt = null
  couponDrawerShow.value = true
}

async function submitCouponAction(): Promise<void> {
  const member = target.value
  if (!member) return
  const reason = couponForm.reason.trim()
  if (!reason) return toastError('请填写操作原因')
  if (couponActionMode.value === 'grant' && !couponForm.templateId) {
    return toastError('请选择需要补发的券种')
  }
  if (couponActionMode.value === 'grant' && !couponForm.storeTarget) {
    return toastError('请选择优惠券适用门店')
  }
  if (couponActionMode.value !== 'void') {
    if (!couponForm.expiresAt || couponForm.expiresAt <= Date.now()) {
      return toastError('有效期必须晚于当前时间')
    }
  }
  if (couponActionMode.value !== 'grant' && !couponActionTarget.value) return

  const memberId = String(member.id)
  const entitlement = couponActionTarget.value
  const mode = couponActionMode.value
  const expiresAt = couponForm.expiresAt
    ? new Date(couponForm.expiresAt).toISOString()
    : undefined
  const grantStoreLabel = couponForm.storeTarget === 'global'
    ? '全部门店'
    : (couponStoreOptions.value.find((item) => item.value === couponForm.storeTarget)?.label
      ?? `门店 #${couponForm.storeTarget}`)
  const title = mode === 'grant' ? '确认补发优惠券' : mode === 'expiry' ? '确认修改有效期' : '确认删除优惠券'
  const content = mode === 'grant'
    ? `将向会员「${member.nickname ?? member.id}」补发所选优惠券，适用于${grantStoreLabel}，有效期至 ${formatDateTime(expiresAt)}。原因：${reason}`
    : mode === 'expiry'
      ? `将“${entitlement?.templateName}”的有效期修改至 ${formatDateTime(expiresAt)}。原因：${reason}`
      : `将从会员账户删除“${entitlement?.templateName}”。已使用券不能删除，操作记录会保留在审计日志中。原因：${reason}`

  couponSubmitting.value = true
  try {
    const ok = await runAudited({
      title,
      content,
      highRisk: true,
      positiveText: couponSubmitText.value,
      execute: () => {
        if (mode === 'grant') {
          return http.post(
            API_PATHS.members.couponEntitlements(memberId),
            {
              templateId: Number(couponForm.templateId),
              scopeType: couponForm.storeTarget === 'global' ? 'global' : 'store',
              storeId: couponForm.storeTarget === 'global' ? null : Number(couponForm.storeTarget),
              expiresAt,
              reason,
            },
            { idempotent: true },
          )
        }
        const entitlementId = String(entitlement!.entitlementId)
        if (mode === 'expiry') {
          return http.patch(
            API_PATHS.members.couponEntitlement(memberId, entitlementId),
            { expiresAt, reason },
            { idempotent: true },
          )
        }
        return http.post(
          API_PATHS.members.voidCouponEntitlement(memberId, entitlementId),
          { reason },
          { idempotent: true },
        )
      },
      successText: mode === 'grant' ? '优惠券已补发' : mode === 'expiry' ? '有效期已更新' : '优惠券已删除',
    })
    if (ok) {
      couponDrawerShow.value = false
      resetCouponForm()
      void couponTable.reload()
    }
  } finally {
    couponSubmitting.value = false
  }
}
</script>

<template>
  <div>
    <ResourceListView
      ref="listRef"
      title="会员列表"
      description="全局会员查询；人工调账为高风险操作"
      :breadcrumb="['用户 / 会员', '会员列表']"
      :fields="fields"
      :columns="columns"
      :fetcher="fetchMembers"
      @update:sorter="handleSorter"
    />
    <FormDrawer
      v-model:show="drawerShow"
      title="人工调账"
      :submitting="submitting"
      high-risk
      submit-text="确认调账"
      @submit="submitAdjust()"
    >
      <NForm label-placement="top">
        <NFormItem
          label="资产类型"
          required
        >
          <NSelect
            v-model:value="adjust.assetType"
            :options="adjustAssetTypeOptions"
            placeholder="选择资产类型"
          />
        </NFormItem>
        <NFormItem
          label="调整数量（正增 / 负减）"
          required
        >
          <NInputNumber
            v-model:value="adjust.amount"
            style="width: 100%"
          />
        </NFormItem>
        <NFormItem
          label="调账原因"
          required
        >
          <NInput
            v-model:value="adjust.reason"
            type="textarea"
            placeholder="请填写调账原因（将写入审计）"
          />
        </NFormItem>
      </NForm>
    </FormDrawer>
    <FormDrawer
      v-model:show="couponDrawerShow"
      :title="couponDrawerTitle"
      :submitting="couponSubmitting"
      high-risk
      :submit-text="couponSubmitText"
      @submit="submitCouponAction"
    >
      <NForm label-placement="top">
        <NFormItem
          v-if="couponActionMode === 'grant'"
          label="券种"
          required
        >
          <NSelect
            v-model:value="couponForm.templateId"
            :options="couponTemplateOptions"
            filterable
            placeholder="选择启用中的券种"
            @update:value="handleCouponTemplateChange"
          />
        </NFormItem>
        <NFormItem
          v-if="couponActionMode === 'grant'"
          label="适用门店"
          required
        >
          <NSelect
            v-model:value="couponForm.storeTarget"
            :options="couponGrantStoreOptions"
            :disabled="!couponForm.templateId"
            placeholder="请先选择券种"
          />
        </NFormItem>
        <NFormItem
          v-if="couponActionMode !== 'void'"
          label="有效期至"
          required
        >
          <NDatePicker
            v-model:value="couponForm.expiresAt"
            type="datetime"
            clearable
            style="width: 100%"
          />
        </NFormItem>
        <NFormItem
          label="操作原因"
          required
        >
          <NInput
            v-model:value="couponForm.reason"
            type="textarea"
            :autosize="{ minRows: 3, maxRows: 5 }"
            :placeholder="couponActionMode === 'void' ? '说明删除原因，将写入审计日志' : '说明补发或改期原因，将写入审计日志'"
            maxlength="200"
            show-count
          />
        </NFormItem>
      </NForm>
    </FormDrawer>
    <NModal
      v-model:show="detailDrawerShow"
      preset="card"
      title="会员详情"
      :mask-closable="false"
      style="width: 980px; max-width: 94vw"
      content-style="max-height: 72vh; overflow: auto"
    >
      <NSpin :show="detailLoading">
        <div
          v-if="detail"
          class="member-detail__profile"
        >
          <NAvatar
            v-if="detail.avatarUrl"
            :size="72"
            round
            :src="detail.avatarUrl"
            object-fit="cover"
          >
            <template #fallback>
              {{ detail.nickname?.trim().slice(0, 1) || String(detail.id).slice(-1) }}
            </template>
          </NAvatar>
          <NAvatar
            v-else
            :size="72"
            round
          >
            {{ detail.nickname?.trim().slice(0, 1) || String(detail.id).slice(-1) }}
          </NAvatar>
          <div class="member-detail__identity">
            <strong>{{ detail.nickname || '—' }}</strong>
            <span>{{ detail.phone || '—' }}</span>
          </div>
        </div>
        <template v-if="detail">
          <div class="member-detail__summary">
            <div>
              <span class="member-detail__label">用户 ID：</span>{{ detail.id }}
            </div>
            <div>
              <span class="member-detail__label">注册时间：</span>{{ formatDateTime(detail.createdAt) }}
            </div>
            <div>
              <span class="member-detail__label">当前积分：</span>{{ detail.pointsBalance }}
            </div>
            <div>
              <span class="member-detail__label">金币：</span>{{ detail.coinsBalance }}
            </div>
            <div>
              <span class="member-detail__label">VIP 等级：</span>
              {{ detail.vipLevel ? `VIP${detail.vipLevel} · ${detail.vipTierName || '会员'}` : '—' }}
            </div>
            <div>
              <span class="member-detail__label">状态：</span>{{ statusLabel(detail.status) }}
            </div>
            <div
              v-for="account in detail.wallet"
              :key="account.assetType"
            >
              <span class="member-detail__label">{{ assetTypeLabel(account.assetType) }}：</span>
              {{ account.availableAmount }}
            </div>
          </div>

          <NTabs
            v-model:value="detailTab"
            type="line"
          >
            <NTabPane
              name="coupons"
              tab="用户优惠券"
            >
              <div class="member-detail__coupon-toolbar">
                <span>删除操作会撤销券资产并保留审计记录，已使用券不可删除。</span>
                <PermissionButton
                  :permission="PERMISSIONS.COUPON_GLOBAL_WRITE"
                  type="primary"
                  @click="openCouponGrant"
                >
                  补发券
                </PermissionButton>
              </div>
              <DataTable
                class="member-detail__coupons"
                :columns="couponColumns"
                :data="couponTable.rows.value"
                :loading="couponTable.loading.value"
                :page="couponTable.pagination.page"
                :page-size="couponTable.pagination.pageSize"
                :item-count="couponTable.pagination.itemCount"
                :row-key="(row) => String(row.entitlementId)"
                empty-text="该用户暂无优惠券"
                @update:page="couponTable.handlePageChange"
                @update:page-size="couponTable.handlePageSizeChange"
              />
            </NTabPane>

            <NTabPane
              name="ledger"
              tab="钱包流水"
            >
              <DataTable
                class="member-detail__ledger"
                :columns="ledgerColumns"
                :data="ledgerTable.rows.value"
                :loading="ledgerTable.loading.value"
                :page="ledgerTable.pagination.page"
                :page-size="ledgerTable.pagination.pageSize"
                :item-count="ledgerTable.pagination.itemCount"
                :row-key="(row) => row.recordKey"
                empty-text="暂无流水"
                @update:page="ledgerTable.handlePageChange"
                @update:page-size="ledgerTable.handlePageSizeChange"
              />
            </NTabPane>

            <NTabPane
              name="adjust"
              tab="人工调账"
            >
              <NForm
                class="member-detail__adjust"
                label-placement="top"
              >
                <NFormItem
                  label="资产类型"
                  required
                >
                  <NSelect
                    v-model:value="adjust.assetType"
                    :options="adjustAssetTypeOptions"
                    placeholder="选择资产类型"
                  />
                </NFormItem>
                <NFormItem
                  label="调整数量（正增 / 负减）"
                  required
                >
                  <NInputNumber
                    v-model:value="adjust.amount"
                    style="width: 100%"
                  />
                </NFormItem>
                <NFormItem
                  label="调账原因"
                  required
                >
                  <NInput
                    v-model:value="adjust.reason"
                    type="textarea"
                    placeholder="请填写调账原因（将写入审计）"
                  />
                </NFormItem>
                <PermissionButton
                  :permission="PERMISSIONS.MEMBER_WALLET_ADJUST"
                  type="primary"
                  :loading="submitting"
                  :disabled="adjust.amount == null || adjust.amount === 0 || !adjust.reason.trim()"
                  @click="submitAdjust(false)"
                >
                  提交调账
                </PermissionButton>
              </NForm>
            </NTabPane>
          </NTabs>
        </template>
      </NSpin>

      <template #footer>
        <div class="member-detail__footer">
          <NButton @click="detailDrawerShow = false">
            关闭
          </NButton>
        </div>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.member-detail__profile {
  display: flex;
  align-items: center;
  gap: var(--ic-space-md);
  padding-bottom: var(--ic-space-md);
  margin-bottom: var(--ic-space-md);
  border-bottom: 1px solid var(--ic-color-border);
}
.member-detail__identity {
  display: flex;
  flex-direction: column;
  gap: var(--ic-space-xs);
}
.member-detail__identity strong {
  font-size: var(--ic-font-lg);
}
.member-detail__identity span {
  color: var(--ic-color-text-secondary);
}
.member-detail__summary {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--ic-space-sm) var(--ic-space-lg);
  margin-bottom: var(--ic-space-md);
  font-size: var(--ic-font-sm);
}
.member-detail__label {
  color: var(--ic-color-text-secondary);
}
.member-detail__adjust {
  max-width: 360px;
  padding-top: var(--ic-space-sm);
}
.member-detail__ledger {
  padding: 0;
  border: 0;
  border-radius: 0;
}
.member-detail__coupon-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--ic-space-md);
  margin-bottom: var(--ic-space-sm);
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-sm);
}
.member-detail__coupons {
  padding: 0;
  border: 0;
  border-radius: 0;
}
.member-detail__footer {
  display: flex;
  justify-content: flex-end;
}
@media (max-width: 640px) {
  .member-detail__summary {
    grid-template-columns: 1fr;
  }
  .member-detail__coupon-toolbar {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
