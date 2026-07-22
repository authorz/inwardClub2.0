<script setup lang="ts">
/**
 * 本店商品：本店自建商品 + 采用全局商品的本店覆盖（价格、库存、支付方式、上下架）。
 * 门店不可修改全局模板；库存调整/发布为高风险写操作，服务端带 Idempotency-Key。
 */
import { computed, h, reactive, ref } from 'vue'
import {
  NButton,
  NCheckboxGroup,
  NCheckbox,
  NInputNumber,
  NModal,
  NSpace,
  type DataTableColumns,
} from 'naive-ui'
import { catalogService } from '@/api/services'
import { useAsyncList } from '@/composables/useAsyncList'
import { useAsyncAction } from '@/composables/useAsyncAction'
import {
  PAY_CHANNEL,
  PUBLISH_STATUS,
  SCOPE_TYPE,
  toOptions,
  type PayChannel,
} from '@/constants/enums'
import { PERM } from '@/constants/permissions'
import { centToYuan, formatCent, yuanToCent } from '@/utils/format'
import { moneyColumn, statusColumn, textColumn } from '@/utils/columns'
import { DataTable, PageHeader, PermissionButton, StatusFilterBar } from '@/components/common'
import type { CatalogItem } from '@/types/models'

const list = useAsyncList<CatalogItem>((params) => catalogService.items(params), {
  initialFilters: { status: '', keyword: '' },
})
const action = useAsyncAction()

// 商品可选支付方式（点餐/积分商城）：微信、金币、余额。
const ITEM_PAY_CHANNELS: PayChannel[] = ['wechat', 'coin', 'balance']

const editShow = ref(false)
const editForm = reactive<{
  id: string | number | null
  name: string
  priceYuan: number
  stockQuantity: number
  payChannels: PayChannel[]
}>({ id: null, name: '', priceYuan: 0, stockQuantity: 0, payChannels: [] })

function openEdit(row: CatalogItem) {
  editForm.id = row.id
  editForm.name = row.name
  editForm.priceYuan = centToYuan(row.priceCent)
  editForm.stockQuantity = row.stockQuantity
  editForm.payChannels = [...row.payChannels]
  editShow.value = true
}

async function saveEdit() {
  const id = editForm.id
  if (id == null) return
  // 价格、库存、支付方式分别调用对应覆盖接口。
  await action.run(
    async () => {
      await catalogService.updatePrice(id, yuanToCent(editForm.priceYuan))
      await catalogService.updateStock(id, editForm.stockQuantity)
      await catalogService.updatePaymentRules(id, editForm.payChannels)
      return true
    },
    {
      successMessage: '已保存本店覆盖',
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
    () => (publishing ? catalogService.publish(row.id) : catalogService.unpublish(row.id)),
    {
      confirm: { content: `确认${publishing ? '上架' : '下架'}「${row.name}」？` },
      successMessage: publishing ? '已上架' : '已下架',
      onSuccess: () => list.refresh(),
    },
  )
}

const columns = computed<DataTableColumns<CatalogItem>>(() => [
  textColumn<CatalogItem>('商品', (r) => r.name),
  statusColumn<CatalogItem>('来源', SCOPE_TYPE, (r) => r.scopeType, { width: 96 }),
  textColumn<CatalogItem>('分类', (r) => r.categoryName, { width: 110 }),
  moneyColumn<CatalogItem>('本店价', (r) => r.priceCent, { width: 100 }),
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
    width: 170,
    fixed: 'right',
    render: (row: CatalogItem) =>
      h(NSpace, { size: 4 }, {
        default: () => [
          h(
            PermissionButton,
            { permissions: [PERM.catalogWrite], type: 'primary', text: true, onClick: () => openEdit(row) },
            { default: () => '价格/库存' },
          ),
          h(
            PermissionButton,
            { permissions: [PERM.catalogWrite], text: true, onClick: () => togglePublish(row) },
            { default: () => (row.status === 'published' ? '下架' : '上架') },
          ),
        ],
      }),
  },
])
</script>

<template>
  <div>
    <PageHeader
      title="本店商品 / 库存 / 价格覆盖"
      description="维护本店自建商品与全局商品的本店覆盖"
    />

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
    />

    <DataTable
      :columns="columns"
      :data="list.rows.value"
      :loading="list.loading.value"
      :page="list.page.value"
      :page-size="list.pageSize.value"
      :total="list.total.value"
      empty-text="暂无商品"
      @update:page="list.setPage"
      @update:page-size="list.setPageSize"
    />

    <NModal
      v-model:show="editShow"
      preset="card"
      title="本店覆盖"
      style="width: 420px"
    >
      <div class="edit-form">
        <div class="edit-form__name">
          {{ editForm.name }}
        </div>
        <label class="edit-form__field">
          <span class="ic-muted">本店售价（元）</span>
          <NInputNumber
            v-model:value="editForm.priceYuan"
            :min="0"
            :precision="2"
          />
        </label>
        <label class="edit-form__field">
          <span class="ic-muted">本店库存</span>
          <NInputNumber
            v-model:value="editForm.stockQuantity"
            :min="0"
            :precision="0"
          />
        </label>
        <div class="edit-form__field">
          <span class="ic-muted">支付方式</span>
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
  gap: var(--ic-space-4);
}
.edit-form__name {
  font-weight: 600;
}
.edit-form__field {
  display: flex;
  flex-direction: column;
  gap: var(--ic-space-2);
  font-size: var(--ic-font-sm);
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
</style>
