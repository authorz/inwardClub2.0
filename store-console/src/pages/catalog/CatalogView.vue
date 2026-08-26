<script setup lang="ts">
/** 本店商品：商品创建后直接归属当前登录门店。 */
import { computed, h, onMounted, reactive, ref, watch } from 'vue'
import {
  NButton,
  NCheckboxGroup,
  NCheckbox,
  NInput,
  NInputNumber,
  NModal,
  NSelect,
  NSpace,
  NSwitch,
  NTooltip,
  type DataTableColumns,
} from 'naive-ui'
import type { DataTableRowKey } from 'naive-ui'
import { catalogService, couponService } from '@/api/services'
import { useAsyncList } from '@/composables/useAsyncList'
import { useAsyncAction } from '@/composables/useAsyncAction'
import {
  PAY_CHANNEL,
  PUBLISH_STATUS,
  toOptions,
  type PayChannel,
} from '@/constants/enums'
import { PERM } from '@/constants/permissions'
import { centToYuan, formatCent, yuanToCent } from '@/utils/format'
import { feedback } from '@/utils/feedback'
import { moneyColumn, statusColumn, textColumn } from '@/utils/columns'
import {
  AppIcon,
  AssetImage,
  AssetUpload,
  DataTable,
  PageHeader,
  PermissionButton,
  StatusFilterBar,
} from '@/components/common'
import type { CatalogCategory, CatalogItem, CouponTemplate } from '@/types/models'

const list = useAsyncList<CatalogItem>((params) => catalogService.items(params), {
  initialFilters: { status: '', keyword: '', categoryId: '' },
})
const selectedItemIds = ref<DataTableRowKey[]>([])
const action = useAsyncAction()
const categories = ref<CatalogCategory[]>([])
const couponTemplates = ref<CouponTemplate[]>([])
const categoryOptions = computed(() => categories.value.map((row) => ({
  label: `${row.name} · ${row.categoryType === 'coupon' ? '券商品' : '普通商品'}`,
  value: String(row.id),
})))
const categoryFilterOptions = computed(() => [
  { label: '全部分类', value: '' },
  ...categoryOptions.value,
])
const itemTypeOptions = [
  { label: '餐品', value: 'food' },
  { label: '积分兑换', value: 'redeemable' }, { label: '实物', value: 'physical' },
]
const publishOptions = toOptions(PUBLISH_STATUS).map(({ label, value }) => ({ label, value }))
const couponTemplateOptions = computed(() => couponTemplates.value
  .filter((template) => ['snack', 'alcohol', 'beverage', 'drink', 'meal', 'gift'].includes(template.couponType))
  .map((template) => ({
    label: `${template.name}（ID ${template.id}）${template.status === 'published' ? '' : '（未发布）'}`,
    value: String(template.id),
    disabled: template.status !== 'published',
  })))
const saleCouponTemplateOptions = computed(() => couponTemplates.value
  .filter((template) => ['event_ticket', 'admission_ticket', 'snack', 'alcohol', 'beverage', 'drink', 'meal', 'gift'].includes(template.couponType))
  .map((template) => ({
    label: `${template.name}（ID ${template.id}）${template.status === 'published' ? '' : '（未发布）'}`,
    value: String(template.id),
    disabled: template.status !== 'published',
  })))

// 商品可选支付方式（点餐/积分商城）：微信、金币。
const ITEM_PAY_CHANNELS: PayChannel[] = ['wechat', 'coin']

const editShow = ref(false)
const editForm = reactive<{
  id: string | number | null
  name: string
  categoryId: string | null
  description: string
  assetId: string | null
  imageUrl: string
  itemType: string
  priceYuan: number
  stockQuantity: number
  payChannels: PayChannel[]
  couponTemplateIds: string[]
  grantCouponTemplateId: string | null
  rewardPointsEnabled: boolean
  pointsReward: number
  sortOrder: number
  status: string
}>({ id: null, name: '', categoryId: null, description: '', assetId: null, imageUrl: '', itemType: 'food', priceYuan: 0, stockQuantity: 0, payChannels: [], couponTemplateIds: [], grantCouponTemplateId: null, rewardPointsEnabled: false, pointsReward: 0, sortOrder: 0, status: 'draft' })

const selectedCategory = computed(() => categories.value.find(
  (category) => String(category.id) === String(editForm.categoryId ?? ''),
))
const categoryIsCoupon = computed(() => selectedCategory.value?.categoryType === 'coupon')

function openEdit(row?: CatalogItem) {
  editForm.id = row?.id ?? null
  editForm.name = row?.name ?? ''
  const categoryId = row?.categoryId == null ? null : String(row.categoryId)
  const categoryIsValid = categoryId != null && categories.value.some(
    (category) => String(category.id) === categoryId,
  )
  editForm.categoryId = categoryIsValid ? categoryId : null
  if (row && !categoryIsValid) {
    feedback.message.warning('该商品原分类已失效，请重新选择有效分类后保存')
  }
  editForm.description = row?.description ?? ''
  editForm.assetId = row?.assetId == null ? null : String(row.assetId)
  editForm.imageUrl = row?.imageUrl ?? ''
  editForm.itemType = row?.itemType ?? 'food'
  editForm.priceYuan = centToYuan(row?.priceCent ?? 0)
  editForm.stockQuantity = row?.stockQuantity ?? 0
  editForm.payChannels = (row?.payChannels ?? []).map((channel) =>
    (channel as string) === 'balance' ? 'coin' : channel,
  ).filter((channel, index, channels) =>
    ITEM_PAY_CHANNELS.includes(channel) && channels.indexOf(channel) === index,
  )
  editForm.couponTemplateIds = (row?.couponTemplateIds ?? []).map(String)
  editForm.grantCouponTemplateId = row?.grantCouponTemplateId == null
    ? null
    : String(row.grantCouponTemplateId)
  editForm.rewardPointsEnabled = (row?.pointsReward ?? 0) > 0
  editForm.pointsReward = row?.pointsReward ?? 0
  editForm.sortOrder = row?.sortOrder ?? 0
  editForm.status = row?.status ?? 'draft'
  editShow.value = true
}

async function saveEdit() {
  const id = editForm.id
  if (!editForm.categoryId) {
    feedback.message.warning('请选择有效的商品分类')
    return
  }
  await action.run(
    async () => {
      if (!editForm.name.trim()) throw new Error('请填写商品名称')
      if (categoryIsCoupon.value && !editForm.grantCouponTemplateId) {
        throw new Error('请选择购买后发放的券')
      }
      const payload = {
        categoryId: editForm.categoryId ? Number(editForm.categoryId) : undefined,
        name: editForm.name.trim(),
        description: editForm.description.trim(),
        assetId: editForm.assetId ? Number(editForm.assetId) : undefined,
        itemType: categoryIsCoupon.value ? 'coupon' : editForm.itemType,
        priceCent: yuanToCent(editForm.priceYuan),
        stockQuantity: editForm.stockQuantity,
        payChannels: editForm.payChannels,
        couponTemplateIds: categoryIsCoupon.value ? [] : editForm.couponTemplateIds.map(Number),
        grantCouponTemplateId: categoryIsCoupon.value ? Number(editForm.grantCouponTemplateId) : null,
        pointsReward: editForm.rewardPointsEnabled ? editForm.pointsReward : 0,
        sortOrder: editForm.sortOrder,
        status: editForm.status,
      }
      if (id == null) await catalogService.create(payload)
      else await catalogService.update(id, payload)
      return true
    },
    {
      successMessage: '商品已保存',
      onSuccess: () => {
        editShow.value = false
        list.refresh()
      },
    },
  )
}

function togglePublish(row: CatalogItem) {
  const publishing = row.status !== 'published'
  void action.run(
    async () => {
      const current = await catalogService.detail(row.id)
      return catalogService.update(row.id, {
        categoryId: current.categoryId,
        name: current.name,
        description: current.description ?? '',
        assetId: current.assetId,
        itemType: current.itemType ?? 'food',
        priceCent: current.priceCent,
        stockQuantity: current.stockQuantity,
        payChannels: current.payChannels,
        couponTemplateIds: current.couponTemplateIds ?? [],
        grantCouponTemplateId: current.grantCouponTemplateId ?? null,
        pointsReward: current.pointsReward ?? 0,
        sortOrder: current.sortOrder ?? 0,
        status: publishing ? 'published' : 'unpublished',
      })
    },
    {
      confirm: { content: `确认${publishing ? '上架' : '下架'}「${row.name}」？` },
      successMessage: publishing ? '已上架' : '已下架',
      onSuccess: () => list.refresh(),
    },
  )
}

async function removeSelected(): Promise<void> {
  const ids = selectedItemIds.value.map(Number)
  if (!ids.length) return
  await action.run(
    () => catalogService.batchRemove(ids),
    {
      confirm: {
        title: '批量删除商品',
        content: `确认永久删除已选择的 ${ids.length} 个商品？商品及其规格数据将被物理删除。`,
        positiveText: '确认批量删除',
        danger: true,
      },
      successMessage: `已删除 ${ids.length} 个商品`,
      onSuccess: () => {
        selectedItemIds.value = []
        list.refresh()
      },
    },
  )
}

const columns = computed<DataTableColumns<CatalogItem>>(() => [
  { type: 'selection', multiple: true },
  textColumn<CatalogItem>('ID', (r) => r.id, { width: 72 }),
  {
    title: '商品图片',
    key: 'image',
    width: 76,
    render: (row: CatalogItem) => h(AssetImage, {
      src: row.imageUrl,
      assetId: row.assetId,
      width: 48,
      height: 48,
    }),
  },
  textColumn<CatalogItem>('商品名称', (r) => r.name, { width: 180, ellipsis: { tooltip: true } }),
  textColumn<CatalogItem>('分类', (r) => r.categoryName, { width: 120, ellipsis: { tooltip: true } }),
  moneyColumn<CatalogItem>('价格', (r) => r.priceCent, { width: 100 }),
  textColumn<CatalogItem>('赠送积分', (r) => r.pointsReward ?? 0, { width: 90 }),
  textColumn<CatalogItem>('库存', (r) => r.stockQuantity, { width: 80, align: 'right' }),
  {
    title: '支付方式',
    key: 'pay',
    width: 150,
    render: (row: CatalogItem) =>
      (row.payChannels ?? []).map((c) => PAY_CHANNEL[c]?.label ?? c).join(' / ') || '-',
  },
  statusColumn<CatalogItem>('状态', PUBLISH_STATUS, (r) => r.status, { width: 96 }),
  {
    title: '操作',
    key: 'actions',
    width: 112,
    fixed: 'right',
    render: (row: CatalogItem) =>
      h(NSpace, { size: 6, wrap: false }, {
        default: () => [
          h(
            NTooltip,
            { trigger: 'hover' },
            {
              trigger: () => h(
                PermissionButton,
                {
                  permissions: [PERM.catalogWrite],
                  ariaLabel: `编辑${row.name}`,
                  onClick: () => openEdit(row),
                },
                { default: () => h(AppIcon, { name: 'edit', size: 17 }) },
              ),
              default: () => '编辑商品',
            },
          ),
          h(
            NTooltip,
            { trigger: 'hover' },
            {
              trigger: () => h(
                PermissionButton,
                {
                  permissions: [PERM.catalogWrite],
                  ariaLabel: `${row.status === 'published' ? '下架' : '上架'}${row.name}`,
                  onClick: () => togglePublish(row),
                },
                {
                  default: () => h(AppIcon, {
                    name: row.status === 'published' ? 'eyeOff' : 'eye',
                    size: 17,
                  }),
                },
              ),
              default: () => (row.status === 'published' ? '下架商品' : '上架商品'),
            },
          ),
        ],
      }),
  },
])

onMounted(async () => {
  try {
    const [categoryResult, couponResult] = await Promise.all([
      catalogService.categories({ page: 1, pageSize: 100 }),
      couponService.list({ page: 1, pageSize: 100, includeGlobal: true }),
    ])
    categories.value = categoryResult.rows
    couponTemplates.value = couponResult.rows
  } catch {
    categories.value = []
    couponTemplates.value = []
  }
})

watch(
  () => editForm.categoryId,
  (_categoryId, previousCategoryId) => {
    const previousWasCoupon = categories.value.find(
      (category) => String(category.id) === String(previousCategoryId ?? ''),
    )?.categoryType === 'coupon'
    if (categoryIsCoupon.value) {
      editForm.itemType = 'coupon'
      editForm.couponTemplateIds = []
    } else {
      if (previousWasCoupon) editForm.itemType = 'food'
      editForm.grantCouponTemplateId = null
    }
  },
)
</script>

<template>
  <div>
    <PageHeader
      title="本店商品"
      description="维护当前门店的商品、库存、支付方式和购买赠送积分"
    >
      <template #actions>
        <NSpace>
          <PermissionButton
            :permissions="[PERM.catalogWrite]"
            type="error"
            :disabled="selectedItemIds.length === 0"
            @click="removeSelected"
          >
            批量删除{{ selectedItemIds.length ? `（${selectedItemIds.length}）` : '' }}
          </PermissionButton>
          <PermissionButton
            :permissions="[PERM.catalogWrite]"
            type="primary"
            @click="openEdit()"
          >
            新增商品
          </PermissionButton>
        </NSpace>
      </template>
    </PageHeader>

    <StatusFilterBar
      :status-options="toOptions(PUBLISH_STATUS)"
      :status="(list.filters.status as string) ?? null"
      :keyword="(list.filters.keyword as string) ?? ''"
      :loading="list.loading.value"
      search-placeholder="搜索商品名"
      @update:status="list.filters.status = $event ?? ''"
      @update:keyword="list.filters.keyword = $event"
      @apply="list.applyFilters({})"
      @reset="list.reset()"
    >
      <template #prefix-filters>
        <NSelect
          class="catalog-category-filter"
          :value="(list.filters.categoryId as string) ?? ''"
          :options="categoryFilterOptions"
          placeholder="全部分类"
          @update:value="list.filters.categoryId = $event ?? ''"
        />
      </template>
    </StatusFilterBar>

    <DataTable
      v-model:checked-row-keys="selectedItemIds"
      :columns="columns"
      :data="list.rows.value"
      :loading="list.loading.value"
      :page="list.page.value"
      :page-size="list.pageSize.value"
      :total="list.total.value"
      :scroll-x="1110"
      empty-text="暂无商品"
      @update:page="list.setPage"
      @update:page-size="list.setPageSize"
    />

    <NModal
      v-model:show="editShow"
      preset="card"
      :title="editForm.id == null ? '新增商品' : '编辑商品'"
      style="width: min(680px, calc(100vw - 24px))"
    >
      <div class="edit-form">
        <div class="edit-form__grid">
          <label class="edit-form__field edit-form__span-2"><span class="ic-muted">商品名称</span><NInput v-model:value="editForm.name" /></label>
          <label class="edit-form__field"><span class="ic-muted">商品分类</span><NSelect
            v-model:value="editForm.categoryId"
            :options="categoryOptions"
            clearable
          /></label>
          <label v-if="!categoryIsCoupon" class="edit-form__field"><span class="ic-muted">商品类型</span><NSelect
            v-model:value="editForm.itemType"
            :options="itemTypeOptions"
          /></label>
          <div v-else class="edit-form__field">
            <span class="ic-muted">商品类型</span>
            <span>券商品</span>
          </div>
          <label class="edit-form__field"><span class="ic-muted">商品说明</span><NInput
            v-model:value="editForm.description"
            type="textarea"
            :autosize="{ minRows: 1, maxRows: 2 }"
          /></label>
          <div class="edit-form__field">
            <span class="ic-muted">商品图片</span><AssetUpload
              v-model:asset-id="editForm.assetId"
              v-model:preview-url="editForm.imageUrl"
              purpose="product"
              :width="88"
              :height="54"
              compact
            />
          </div>
          <label class="edit-form__field">
            <span class="ic-muted">售价（元）</span>
            <NInputNumber
              v-model:value="editForm.priceYuan"
              :min="0"
              :precision="2"
            />
          </label>
          <div class="edit-form__field">
            <span class="ic-muted">购买后赠送积分</span>
            <NSwitch v-model:value="editForm.rewardPointsEnabled" />
          </div>
          <label v-if="editForm.rewardPointsEnabled" class="edit-form__field"><span class="ic-muted">每份赠送积分</span><NInputNumber
            v-model:value="editForm.pointsReward"
            :min="1"
            :precision="0"
          /></label>
          <label class="edit-form__field">
            <span class="ic-muted">库存</span>
            <NInputNumber
              v-model:value="editForm.stockQuantity"
              :min="0"
              :precision="0"
            />
          </label>
          <label class="edit-form__field"><span class="ic-muted">排序</span><NInputNumber
            v-model:value="editForm.sortOrder"
            :min="0"
            :precision="0"
          /></label>
          <label class="edit-form__field"><span class="ic-muted">状态</span><NSelect
            v-model:value="editForm.status"
            :options="publishOptions"
          /></label>
          <div class="edit-form__field">
            <span class="ic-muted">支付方式</span>
            <div class="edit-form__checks">
              <NCheckboxGroup v-model:value="editForm.payChannels">
                <NSpace>
                  <NCheckbox
                    v-for="c in ITEM_PAY_CHANNELS"
                    :key="c"
                    :value="c"
                    :label="PAY_CHANNEL[c].label"
                  />
                </NSpace>
              </NCheckboxGroup>
            </div>
          </div>
          <label v-if="categoryIsCoupon" class="edit-form__field edit-form__span-2">
            <span class="ic-muted">购买后发放的券</span>
            <NSelect
              v-model:value="editForm.grantCouponTemplateId"
              clearable
              filterable
              :options="saleCouponTemplateOptions"
              placeholder="一份商品对应发放一张券"
            />
            <span class="ic-muted">支付成功后按购买数量自动到账，每张券有效期 30 天。</span>
          </label>
          <label v-else class="edit-form__field edit-form__span-2">
            <span class="ic-muted">允许兑换该商品的券</span>
            <NSelect
              v-model:value="editForm.couponTemplateIds"
              multiple
              clearable
              filterable
              :options="couponTemplateOptions"
              placeholder="请从当前门店可用的券列表中选择"
            />
          </label>
        </div>
        <p class="ic-muted edit-form__hint">
          当前价格：{{ formatCent(yuanToCent(editForm.priceYuan)) }}
        </p>
      </div>
      <template #footer>
        <div class="edit-form__footer">
          <NButton @click="editShow = false">
            取消
          </NButton>
          <NButton
            type="primary"
            :loading="action.running.value"
            @click="saveEdit"
          >
            保存
          </NButton>
        </div>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.edit-form {
  display: flex;
  flex-direction: column;
  gap: var(--ic-space-2);
}
.catalog-category-filter {
  width: 180px;
}
.edit-form__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--ic-space-3) var(--ic-space-4);
}
.edit-form__field {
  display: flex;
  flex-direction: column;
  gap: var(--ic-space-2);
  font-size: var(--ic-font-sm);
}
.edit-form__span-2 {
  grid-column: 1 / -1;
}
.edit-form__checks {
  display: flex;
  align-items: center;
  min-height: 34px;
}
.edit-form__hint {
  font-size: var(--ic-font-xs);
  margin: 0;
}
.edit-form__footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--ic-space-2);
}
@media (max-width: 560px) {
  .edit-form__grid {
    gap: var(--ic-space-2) var(--ic-space-3);
  }
}
</style>
