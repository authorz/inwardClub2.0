<script setup lang="ts">
/**
 * 总后台商品管理：商品归属门店，并从同一门店的商品分类中选择分类。
 * 商品图片通过统一资产服务上传，列表直接展示服务端解析后的图片地址。
 */
import { computed, h, onMounted, reactive, ref, watch } from 'vue'
import {
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NSelect,
  NSpace,
  NSwitch,
  NText,
} from 'naive-ui'
import ResourceListView from '@/components/ResourceListView.vue'
import type { FilterField, ResourceListInstance } from '@/components/ui-types'
import FormDrawer from '@/components/FormDrawer.vue'
import PermissionButton from '@/components/PermissionButton.vue'
import AssetImage from '@/components/AssetImage.vue'
import AssetUpload from '@/components/AssetUpload.vue'
import {
  actionsColumn,
  dateTimeColumn,
  moneyColumn,
  renderColumn,
  statusColumn,
  textColumn,
} from '@/utils/columns'
import {
  PAY_CHANNEL_OPTIONS,
  RESOURCE_STATUS,
  RESOURCE_STATUS_OPTIONS,
  type OptionItem,
} from '@/constants/enums'
import { PERMISSIONS } from '@/constants/permissions'
import { catalogItemService, categoryService, couponTemplateService, storeService } from '@/api/services'
import { usePublishableActions } from '@/composables/usePublishableActions'
import type { CatalogCategory, CatalogItem, CouponTemplate } from '@/api/models'
import { toastError, toastSuccess } from '@/utils/feedback'

const listRef = ref<ResourceListInstance | null>(null)
const stores = ref<OptionItem[]>([])
const categories = ref<CatalogCategory[]>([])
const couponTemplates = ref<CouponTemplate[]>([])
const uploadKey = ref(0)

const itemStatusOptions = RESOURCE_STATUS_OPTIONS.filter(({ value }) =>
  [RESOURCE_STATUS.DRAFT, RESOURCE_STATUS.PUBLISHED].includes(
    value as typeof RESOURCE_STATUS.DRAFT | typeof RESOURCE_STATUS.PUBLISHED,
  ),
)
const onlinePayChannelOptions = PAY_CHANNEL_OPTIONS
function couponTemplateName(id: string | number): string {
  return couponTemplates.value.find((template) => String(template.id) === String(id))?.name
    ?? `券 ID ${id}`
}
const categoryFilterOptions = computed<OptionItem[]>(() =>
  categories.value.map((category) => ({
    label: category.storeName ? `${category.storeName} / ${category.name}` : category.name,
    value: String(category.id),
  })),
)

const fields = computed<FilterField[]>(() => [
  { key: 'storeId', label: '所属门店', type: 'select', options: stores.value, width: 200 },
  {
    key: 'keyword',
    label: '商品名称',
    type: 'input',
    placeholder: '支持名称模糊搜索',
    width: 220,
  },
  {
    key: 'categoryId',
    label: '商品分类',
    type: 'select',
    options: categoryFilterOptions.value,
    width: 220,
  },
  { key: 'status', label: '状态', type: 'select', options: itemStatusOptions },
])

const { publish, unpublish } = usePublishableActions(
  catalogItemService,
  { publishedStatus: RESOURCE_STATUS.PUBLISHED, unpublishedStatus: RESOURCE_STATUS.DRAFT },
  () => listRef.value?.reload(),
)

const columns = [
  renderColumn<CatalogItem>(
    '商品图片',
    'imageUrl',
    (row) => h(AssetImage, { src: row.imageUrl, width: 64, height: 44 }),
    92,
  ),
  renderColumn<CatalogItem>(
    '券兑换',
    'couponTemplateIds',
    (row) =>
      row.grantCouponTemplateId
        ? `购买发放：${couponTemplateName(row.grantCouponTemplateId)}`
        : row.couponTemplateIds?.length
        ? row.couponTemplateIds.map(couponTemplateName).join(' / ')
        : '不可兑换',
    180,
  ),
  textColumn<CatalogItem>('商品名称', 'name', { width: 180 }),
  textColumn<CatalogItem>('商品分类', 'categoryName', { width: 150 }),
  renderColumn<CatalogItem>(
    '所属门店',
    'storeName',
    (row) =>
      row.storeName
        ? row.storeName
        : h(NText, { type: 'error', depth: 1 }, () => '未绑定（需修复）'),
    170,
  ),
  moneyColumn<CatalogItem>('价格', 'priceCent'),
  textColumn<CatalogItem>('库存', 'stockQuantity', { width: 90 }),
  renderColumn<CatalogItem>(
    '赠送积分',
    'pointsReward',
    (row) => ((row.pointsReward ?? 0) > 0 ? `${row.pointsReward} 积分/份` : '不赠送'),
    110,
  ),
  statusColumn<CatalogItem>('状态', 'status', RESOURCE_STATUS_OPTIONS),
  dateTimeColumn<CatalogItem>('创建时间', 'createdAt'),
  actionsColumn<CatalogItem>(
    (row) =>
      h(NSpace, {}, () => [
        h(
          PermissionButton,
          { permission: PERMISSIONS.CATALOG_GLOBAL_WRITE, onClick: () => openEdit(row) },
          () => '编辑',
        ),
        row.status === RESOURCE_STATUS.PUBLISHED
          ? h(
              PermissionButton,
              {
                permission: PERMISSIONS.CATALOG_GLOBAL_WRITE,
                type: 'error',
                onClick: () => unpublish(row, row.name),
              },
              () => '下架',
            )
          : h(
              PermissionButton,
              {
                permission: PERMISSIONS.CATALOG_GLOBAL_WRITE,
                type: 'primary',
                onClick: () => publish(row, row.name),
              },
              () => '发布',
            ),
      ]),
    180,
  ),
]

interface ItemForm {
  storeId: string | null
  categoryId: string | null
  name: string
  description: string
  assetId: string | null
  imageUrl: string
  itemType: string
  priceYuan: number
  stockQuantity: number
  payChannels: string[]
  couponTemplateIds: string[]
  grantCouponTemplateId: string | null
  rewardPointsEnabled: boolean
  pointsReward: number
  sortOrder: number
  status: string
}

const drawerShow = ref(false)
const submitting = ref(false)
const editingId = ref<string | null>(null)
const form = reactive<ItemForm>({
  storeId: null,
  categoryId: null,
  name: '',
  description: '',
  assetId: null,
  imageUrl: '',
  itemType: 'food',
  priceYuan: 0,
  stockQuantity: 0,
  payChannels: ['wechat'],
  couponTemplateIds: [],
  grantCouponTemplateId: null,
  rewardPointsEnabled: false,
  pointsReward: 0,
  sortOrder: 0,
  status: RESOURCE_STATUS.DRAFT,
})

const formCategoryOptions = computed<OptionItem[]>(() =>
  categories.value
    .filter((category) => String(category.storeId ?? '') === String(form.storeId ?? ''))
    .map((category) => ({
      label: `${category.name} · ${category.categoryType === 'coupon' ? '券商品' : '普通商品'}`,
      value: String(category.id),
    })),
)

const selectedCategory = computed(() => categories.value.find(
  (category) => String(category.id) === String(form.categoryId ?? ''),
))
const categoryIsCoupon = computed(() => selectedCategory.value?.categoryType === 'coupon')

const formCouponTemplateOptions = computed(() =>
  couponTemplates.value
    .filter((template) =>
      ['snack', 'alcohol', 'beverage', 'meal'].includes(template.couponType)
      && (template.scopeType === 'global'
        || String(template.storeId ?? '') === String(form.storeId ?? '')),
    )
    .map((template) => ({
      label: `${template.name}（ID ${template.id}）${template.status === 'published' ? '' : '（未发布）'}`,
      value: String(template.id),
      disabled: template.status !== 'published',
    })),
)
const saleCouponTemplateOptions = computed(() =>
  couponTemplates.value
    .filter((template) =>
      ['event_ticket', 'snack', 'alcohol', 'beverage', 'meal'].includes(template.couponType)
      && (template.scopeType === 'global'
        || String(template.storeId ?? '') === String(form.storeId ?? '')),
    )
    .map((template) => ({
      label: `${template.name}（ID ${template.id}）${template.status === 'published' ? '' : '（未发布）'}`,
      value: String(template.id),
      disabled: template.status !== 'published',
    })),
)

async function loadReferences(): Promise<void> {
  try {
    const [storeResult, categoryResult, couponResult] = await Promise.all([
      storeService.list({ page: 1, pageSize: 100 }),
      categoryService.list({ page: 1, pageSize: 100 }),
      couponTemplateService.list({ page: 1, pageSize: 100 }),
    ])
    stores.value = storeResult.items.map((store) => ({
      label: store.name,
      value: String(store.id),
    }))
    categories.value = categoryResult.items
    couponTemplates.value = couponResult.items
  } catch (e) {
    toastError((e as { message?: string }).message ?? '商品基础数据加载失败')
  }
}

function resetForm(): void {
  form.storeId = null
  form.categoryId = null
  form.name = ''
  form.description = ''
  form.assetId = null
  form.imageUrl = ''
  form.itemType = 'food'
  form.priceYuan = 0
  form.stockQuantity = 0
  form.payChannels = ['wechat']
  form.couponTemplateIds = []
  form.grantCouponTemplateId = null
  form.rewardPointsEnabled = false
  form.pointsReward = 0
  form.sortOrder = 0
  form.status = RESOURCE_STATUS.DRAFT
  uploadKey.value += 1
}

function openCreate(): void {
  editingId.value = null
  resetForm()
  drawerShow.value = true
}

function openEdit(row: CatalogItem): void {
  editingId.value = row.id
  resetForm()
  form.storeId = row.storeId == null ? null : String(row.storeId)
  form.categoryId = row.categoryId == null ? null : String(row.categoryId)
  form.name = row.name
  form.description = row.description ?? ''
  form.assetId = row.assetId == null ? null : String(row.assetId)
  form.imageUrl = row.imageUrl ?? ''
  form.itemType = row.itemType || 'food'
  form.priceYuan = (row.priceCent ?? 0) / 100
  form.stockQuantity = row.stockQuantity ?? 0
  form.payChannels = row.payChannels?.length ? [...row.payChannels] : ['wechat']
  form.couponTemplateIds = (row.couponTemplateIds ?? []).map(String)
  form.grantCouponTemplateId = row.grantCouponTemplateId == null
    ? null
    : String(row.grantCouponTemplateId)
  form.rewardPointsEnabled = (row.pointsReward ?? 0) > 0
  form.pointsReward = row.pointsReward ?? 0
  form.sortOrder = row.sortOrder ?? 0
  form.status = row.status ?? RESOURCE_STATUS.DRAFT
  drawerShow.value = true
}

watch(
  () => form.storeId,
  (storeId, previousStoreId) => {
    if (previousStoreId == null || storeId === previousStoreId) return
    if (form.categoryId) {
      const selected = categories.value.find(
        (category) => String(category.id) === String(form.categoryId),
      )
      if (String(selected?.storeId ?? '') !== String(storeId ?? '')) form.categoryId = null
    }
    const availableCouponIDs = new Set(formCouponTemplateOptions.value.map(({ value }) => value))
    form.couponTemplateIds = form.couponTemplateIds.filter((id) => availableCouponIDs.has(id))
    const availableSaleCouponIDs = new Set(saleCouponTemplateOptions.value.map(({ value }) => value))
    if (form.grantCouponTemplateId && !availableSaleCouponIDs.has(form.grantCouponTemplateId)) {
      form.grantCouponTemplateId = null
    }
  },
)

watch(
  () => form.categoryId,
  (_categoryId, previousCategoryId) => {
    const previousWasCoupon = categories.value.find(
      (category) => String(category.id) === String(previousCategoryId ?? ''),
    )?.categoryType === 'coupon'
    if (categoryIsCoupon.value) {
      form.itemType = 'coupon'
      form.couponTemplateIds = []
    } else {
      if (previousWasCoupon) form.itemType = 'food'
      form.grantCouponTemplateId = null
    }
  },
)

async function submit(): Promise<void> {
  if (!form.storeId) return toastError('请选择所属门店')
  if (!form.categoryId) return toastError('请选择商品分类')
  if (!form.name.trim()) return toastError('请填写商品名称')
  if (!form.assetId) return toastError('请上传商品图片')
  if (!form.payChannels.length) return toastError('请选择至少一种支付方式')
  if (categoryIsCoupon.value && !form.grantCouponTemplateId) {
    return toastError('请选择购买后发放的券')
  }
  if (form.rewardPointsEnabled && form.pointsReward <= 0) {
    return toastError('请填写每份商品赠送的积分')
  }

  const payload: Partial<CatalogItem> = {
    storeId: Number(form.storeId),
    categoryId: Number(form.categoryId),
    name: form.name.trim(),
    description: form.description.trim(),
    assetId: Number(form.assetId),
    itemType: categoryIsCoupon.value ? 'coupon' : (form.itemType || 'food'),
    priceCent: Math.round(form.priceYuan * 100),
    stockQuantity: form.stockQuantity,
    payChannels: form.payChannels,
    couponTemplateIds: categoryIsCoupon.value ? [] : form.couponTemplateIds.map(Number),
    grantCouponTemplateId: categoryIsCoupon.value ? Number(form.grantCouponTemplateId) : null,
    pointsReward: form.rewardPointsEnabled ? Math.floor(form.pointsReward) : 0,
    sortOrder: form.sortOrder,
    status: form.status,
  }
  submitting.value = true
  try {
    if (editingId.value) await catalogItemService.update(editingId.value, payload)
    else await catalogItemService.create(payload)
    toastSuccess('已保存')
    drawerShow.value = false
    await Promise.all([listRef.value?.reload(), loadReferences()])
  } catch (e) {
    toastError((e as { message?: string }).message ?? '保存失败')
  } finally {
    submitting.value = false
  }
}

const toolbarActions = [
  {
    key: 'create',
    label: '新增商品',
    type: 'primary' as const,
    permission: PERMISSIONS.CATALOG_GLOBAL_WRITE,
    onClick: openCreate,
  },
]

onMounted(loadReferences)
</script>

<template>
  <div>
    <ResourceListView
      ref="listRef"
      title="商品管理"
      description="按门店维护商品、分类、图片、价格与库存"
      :breadcrumb="['商品管理', '商品管理']"
      :fields="fields"
      :columns="columns"
      :fetcher="catalogItemService.list"
      :toolbar-actions="toolbarActions"
      empty-text="暂无商品，请先创建门店商品分类"
    />

    <FormDrawer
      v-model:show="drawerShow"
      :title="editingId ? '编辑商品' : '新增商品'"
      :submitting="submitting"
      @submit="submit"
    >
      <NForm label-placement="top">
        <NFormItem
          label="所属门店"
          required
        >
          <NSelect
            v-model:value="form.storeId"
            :options="stores"
            placeholder="请选择门店"
            filterable
          />
        </NFormItem>
        <NFormItem
          label="商品分类"
          required
        >
          <NSelect
            v-model:value="form.categoryId"
            :options="formCategoryOptions.map(({ label, value }) => ({ label, value }))"
            :disabled="!form.storeId"
            :placeholder="form.storeId ? '请选择该门店下的分类' : '请先选择门店'"
            filterable
          />
        </NFormItem>
        <NFormItem
          label="商品名称"
          required
        >
          <NInput
            v-model:value="form.name"
            placeholder="请输入商品名称"
          />
        </NFormItem>
        <NFormItem label="商品描述">
          <NInput
            v-model:value="form.description"
            type="textarea"
            :autosize="{ minRows: 2, maxRows: 5 }"
            placeholder="请输入商品描述"
          />
        </NFormItem>
        <NFormItem
          label="商品图片"
          required
        >
          <AssetUpload
            :key="uploadKey"
            v-model:asset-id="form.assetId"
            purpose="product"
            :preview-url="form.imageUrl || null"
            :width="176"
            :height="112"
          />
        </NFormItem>
        <NFormItem
          label="价格（元）"
          required
        >
          <NInputNumber
            v-model:value="form.priceYuan"
            :min="0"
            :precision="2"
            style="width: 100%"
          />
        </NFormItem>
        <NFormItem
          label="库存"
          required
        >
          <NInputNumber
            v-model:value="form.stockQuantity"
            :min="0"
            :precision="0"
            style="width: 100%"
          />
        </NFormItem>
        <NFormItem
          label="支付方式"
          required
        >
          <NSelect
            v-model:value="form.payChannels"
            multiple
            :options="onlinePayChannelOptions.map(({ label, value }) => ({ label, value }))"
            placeholder="请选择支付方式"
          />
        </NFormItem>
        <NFormItem
          v-if="categoryIsCoupon"
          label="购买后发放的券"
          required
        >
          <NSelect
            v-model:value="form.grantCouponTemplateId"
            clearable
            filterable
            :options="saleCouponTemplateOptions"
            placeholder="一份商品对应发放一张券"
          />
          <NText depth="3">
            用户支付成功后按购买数量自动到账，每张券有效期 30 天。
          </NText>
        </NFormItem>
        <NFormItem
          v-else
          label="允许兑换该商品的券"
        >
          <NSelect
            v-model:value="form.couponTemplateIds"
            multiple
            clearable
            filterable
            :options="formCouponTemplateOptions"
            placeholder="请从当前门店可用的券列表中选择"
          />
        </NFormItem>
        <NFormItem label="购买后赠送积分">
          <NSwitch v-model:value="form.rewardPointsEnabled">
            <template #checked>
              赠送
            </template>
            <template #unchecked>
              不赠送
            </template>
          </NSwitch>
        </NFormItem>
        <NFormItem
          v-if="form.rewardPointsEnabled"
          label="每份赠送积分"
          required
        >
          <NInputNumber
            v-model:value="form.pointsReward"
            :min="1"
            :precision="0"
            style="width: 100%"
            placeholder="请输入购买每份商品赠送的积分"
          />
        </NFormItem>
        <NFormItem label="排序">
          <NInputNumber
            v-model:value="form.sortOrder"
            :min="0"
            :precision="0"
            style="width: 100%"
          />
        </NFormItem>
        <NFormItem
          label="状态"
          required
        >
          <NSelect
            v-model:value="form.status"
            :options="itemStatusOptions.map(({ label, value }) => ({ label, value }))"
            placeholder="请选择状态"
          />
        </NFormItem>
      </NForm>
    </FormDrawer>
  </div>
</template>
