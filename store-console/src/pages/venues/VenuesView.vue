<script setup lang="ts">
import { computed, h, reactive, ref } from 'vue'
import {
  NButton, NForm, NFormItem, NInput, NInputNumber, NModal, NSelect,
  NSpace, NTabPane, NTabs, type DataTableColumns,
} from 'naive-ui'
import { venueService } from '@/api/services'
import { useAsyncList } from '@/composables/useAsyncList'
import { confirm } from '@/composables/useConfirm'
import { feedback } from '@/utils/feedback'
import { AVAILABILITY_STATUS, toOptions } from '@/constants/enums'
import { PERM } from '@/constants/permissions'
import {
  AssetImage, AssetUpload, DataTable, PageHeader, PermissionButton, StatusFilterBar,
} from '@/components/common'
import { dateColumn, statusColumn, textColumn } from '@/utils/columns'
import type { StoreSeat, StoreTable } from '@/types/models'

const tables = useAsyncList<StoreTable>((params) => venueService.tables(params), {
  initialFilters: { keyword: '', status: '' },
  pageSize: 50,
})
const seats = useAsyncList<StoreSeat>((params) => venueService.seats(params), {
  initialFilters: { keyword: '', status: '', tableId: '' },
})
const statuses = toOptions(AVAILABILITY_STATUS)
const statusSelectOptions = statuses.map(({ label, value }) => ({ label, value }))
const seatTableFilter = computed<string | null>({
  get: () => String(seats.filters.tableId ?? '') || null,
  set: (value) => { seats.filters.tableId = value ?? '' },
})
const tableOptions = computed(() => tables.rows.value.map((row) => ({
  label: `${row.name}（${row.code}）`, value: String(row.id),
})))

const tableShow = ref(false)
const tableSaving = ref(false)
const tableForm = reactive({
  id: null as string | number | null,
  name: '', code: '', capacity: 1, basePoints: 0,
  layoutAssetId: null as string | null, layoutUrl: '', status: 'available',
})

function openTable(row?: StoreTable): void {
  Object.assign(tableForm, row ? {
    id: row.id, name: row.name, code: row.code, capacity: row.capacity,
    basePoints: row.basePoints, layoutAssetId: row.layoutAssetId == null ? null : String(row.layoutAssetId),
    layoutUrl: row.layoutUrl ?? '', status: row.status,
  } : {
    id: null, name: '', code: '', capacity: 1, basePoints: 0,
    layoutAssetId: null, layoutUrl: '', status: 'available',
  })
  tableShow.value = true
}

function clearTableLayout(): void {
  tableForm.layoutAssetId = null
  tableForm.layoutUrl = ''
}

async function saveTable(): Promise<void> {
  if (!tableForm.name.trim() || !tableForm.code.trim()) {
    feedback.message.error('请填写桌子名称和编号')
    return
  }
  tableSaving.value = true
  try {
    const body = {
      name: tableForm.name.trim(), code: tableForm.code.trim(),
      capacity: tableForm.capacity, basePoints: tableForm.basePoints,
      layoutAssetId: tableForm.layoutAssetId ? Number(tableForm.layoutAssetId) : null,
      status: tableForm.status,
    }
    if (tableForm.id == null) await venueService.createTable(body)
    else await venueService.updateTable(tableForm.id, body)
    feedback.message.success('桌子已保存')
    tableShow.value = false
    tables.refresh()
  } catch (error) {
    feedback.message.error((error as { message?: string }).message ?? '保存失败')
  } finally {
    tableSaving.value = false
  }
}

async function removeTable(row: StoreTable): Promise<void> {
  const ok = await confirm({ content: `确认删除桌子“${row.name}”？已有座位或预约时不能删除。`, danger: true })
  if (!ok) return
  try {
    await venueService.deleteTable(row.id)
    feedback.message.success('桌子已删除')
    tables.refresh()
  } catch (error) {
    feedback.message.error((error as { message?: string }).message ?? '删除失败')
  }
}

const seatShow = ref(false)
const seatSaving = ref(false)
const seatForm = reactive({
  id: null as string | number | null, tableId: null as string | null,
  name: '', status: 'available',
})

function openSeat(row?: StoreSeat): void {
  Object.assign(seatForm, row ? {
    id: row.id, tableId: String(row.tableId), name: row.name, status: row.status,
  } : { id: null, tableId: null, name: '', status: 'available' })
  seatShow.value = true
}

async function saveSeat(): Promise<void> {
  if (!seatForm.tableId) return void feedback.message.error('请选择所属桌子')
  if (!seatForm.name.trim()) return void feedback.message.error('请填写座位名称')
  seatSaving.value = true
  try {
    const body = { tableId: Number(seatForm.tableId), name: seatForm.name.trim(), status: seatForm.status }
    if (seatForm.id == null) await venueService.createSeat(body)
    else await venueService.updateSeat(seatForm.id, body)
    feedback.message.success('座位已保存')
    seatShow.value = false
    seats.refresh()
    tables.refresh()
  } catch (error) {
    feedback.message.error((error as { message?: string }).message ?? '保存失败')
  } finally {
    seatSaving.value = false
  }
}

async function removeSeat(row: StoreSeat): Promise<void> {
  const ok = await confirm({ content: `确认删除座位“${row.name}”？已有预约时不能删除。`, danger: true })
  if (!ok) return
  try {
    await venueService.deleteSeat(row.id)
    feedback.message.success('座位已删除')
    seats.refresh()
    tables.refresh()
  } catch (error) {
    feedback.message.error((error as { message?: string }).message ?? '删除失败')
  }
}

const tableColumns = computed<DataTableColumns<StoreTable>>(() => [
  { title: '背景', key: 'layoutUrl', width: 130, render: (row) => h(AssetImage, { src: row.layoutUrl, width: 96, height: 54 }) },
  textColumn<StoreTable>('桌子名称', (row) => row.name),
  textColumn<StoreTable>('编号', (row) => row.code, { width: 110 }),
  textColumn<StoreTable>('座位', (row) => `${row.seatCount} / ${row.capacity}`, { width: 90 }),
  textColumn<StoreTable>('基础积分', (row) => row.basePoints, { width: 90 }),
  statusColumn<StoreTable>('状态', AVAILABILITY_STATUS, (row) => row.status, { width: 110 }),
  dateColumn<StoreTable>('更新时间', (row) => row.updatedAt, { width: 150 }),
  {
    title: '操作', key: 'actions', width: 130,
    render: (row) => h(NSpace, { size: 4 }, () => [
      h(PermissionButton, { permissions: [PERM.reservationWrite], text: true, onClick: () => openTable(row) }, () => '编辑'),
      h(PermissionButton, { permissions: [PERM.reservationWrite], text: true, type: 'error', onClick: () => removeTable(row) }, () => '删除'),
    ]),
  },
])
const seatColumns = computed<DataTableColumns<StoreSeat>>(() => [
  textColumn<StoreSeat>('座位名称', (row) => row.name),
  textColumn<StoreSeat>('所属桌子', (row) => row.tableName),
  statusColumn<StoreSeat>('状态', AVAILABILITY_STATUS, (row) => row.status, { width: 110 }),
  dateColumn<StoreSeat>('更新时间', (row) => row.updatedAt, { width: 150 }),
  {
    title: '操作', key: 'actions', width: 130,
    render: (row) => h(NSpace, { size: 4 }, () => [
      h(PermissionButton, { permissions: [PERM.reservationWrite], text: true, onClick: () => openSeat(row) }, () => '编辑'),
      h(PermissionButton, { permissions: [PERM.reservationWrite], text: true, type: 'error', onClick: () => removeSeat(row) }, () => '删除'),
    ]),
  },
])
</script>

<template>
  <div>
    <PageHeader
      title="桌子与座位"
      description="桌子归属当前门店；座位必须绑定本店已有桌子并自动继承门店"
    />
    <NTabs
      type="line"
      animated
    >
      <NTabPane
        name="tables"
        tab="桌子管理"
      >
        <StatusFilterBar
          :status-options="statuses"
          :status="tables.filters.status as string"
          :keyword="tables.filters.keyword as string"
          :loading="tables.loading.value"
          search-placeholder="搜索桌子名称 / 编号"
          @update:status="tables.filters.status = $event ?? ''"
          @update:keyword="tables.filters.keyword = $event"
          @apply="tables.applyFilters({})"
          @reset="tables.reset()"
        >
          <template #actions>
            <PermissionButton
              :permissions="[PERM.reservationWrite]"
              type="primary"
              @click="openTable()"
            >
              新增桌子
            </PermissionButton>
          </template>
        </StatusFilterBar>
        <DataTable
          :columns="tableColumns"
          :data="tables.rows.value"
          :loading="tables.loading.value"
          :page="tables.page.value"
          :page-size="tables.pageSize.value"
          :total="tables.total.value"
          empty-text="暂无桌子"
          @update:page="tables.setPage"
          @update:page-size="tables.setPageSize"
        />
      </NTabPane>
      <NTabPane
        name="seats"
        tab="座位管理"
      >
        <StatusFilterBar
          :status-options="statuses"
          :status="seats.filters.status as string"
          :keyword="seats.filters.keyword as string"
          :loading="seats.loading.value"
          search-placeholder="搜索座位名称"
          @update:status="seats.filters.status = $event ?? ''"
          @update:keyword="seats.filters.keyword = $event"
          @apply="seats.applyFilters({})"
          @reset="seats.reset()"
        >
          <template #filters>
            <NSelect
              v-model:value="seatTableFilter"
              :options="tableOptions"
              clearable
              placeholder="所属桌子"
              style="width: 180px"
            />
          </template>
          <template #actions>
            <PermissionButton
              :permissions="[PERM.reservationWrite]"
              type="primary"
              :disabled="tables.total.value === 0"
              @click="openSeat()"
            >
              新增座位
            </PermissionButton>
          </template>
        </StatusFilterBar>
        <DataTable
          :columns="seatColumns"
          :data="seats.rows.value"
          :loading="seats.loading.value"
          :page="seats.page.value"
          :page-size="seats.pageSize.value"
          :total="seats.total.value"
          empty-text="暂无座位，请先新增桌子"
          @update:page="seats.setPage"
          @update:page-size="seats.setPageSize"
        />
      </NTabPane>
    </NTabs>

    <NModal
      v-model:show="tableShow"
      preset="card"
      :title="tableForm.id == null ? '新增桌子' : '编辑桌子'"
      style="width: 560px"
    >
      <NForm label-placement="top">
        <NFormItem
          label="桌子名称"
          required
        >
          <NInput v-model:value="tableForm.name" />
        </NFormItem>
        <NFormItem
          label="桌子编号"
          required
        >
          <NInput v-model:value="tableForm.code" />
        </NFormItem>
        <NFormItem
          label="座位容量"
          required
        >
          <NInputNumber
            v-model:value="tableForm.capacity"
            :min="1"
          />
        </NFormItem>
        <NFormItem
          label="基础积分"
          required
        >
          <NInputNumber
            v-model:value="tableForm.basePoints"
            :min="0"
          />
        </NFormItem>
        <NFormItem label="桌子背景（选填）">
          <div class="layout-editor">
            <AssetUpload
              v-model:asset-id="tableForm.layoutAssetId"
              v-model:preview-url="tableForm.layoutUrl"
              purpose="table_layout"
              :width="240"
              :height="135"
            />
            <NButton
              v-if="tableForm.layoutAssetId || tableForm.layoutUrl"
              secondary
              type="error"
              size="small"
              @click="clearTableLayout"
            >
              移除背景
            </NButton>
            <small v-if="!tableForm.layoutAssetId && !tableForm.layoutUrl">未设置自定义背景，将使用默认桌面</small>
          </div>
        </NFormItem>
        <NFormItem label="状态">
          <NSelect
            v-model:value="tableForm.status"
            :options="statusSelectOptions"
          />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="tableShow = false">
            取消
          </NButton><NButton
            type="primary"
            :loading="tableSaving"
            @click="saveTable"
          >
            保存
          </NButton>
        </NSpace>
      </template>
    </NModal>
    <NModal
      v-model:show="seatShow"
      preset="card"
      :title="seatForm.id == null ? '新增座位' : '编辑座位'"
      style="width: 480px"
    >
      <NForm label-placement="top">
        <NFormItem
          label="所属桌子"
          required
        >
          <NSelect
            v-model:value="seatForm.tableId"
            :options="tableOptions"
          />
        </NFormItem>
        <NFormItem
          label="座位名称"
          required
        >
          <NInput v-model:value="seatForm.name" />
        </NFormItem>
        <NFormItem label="状态">
          <NSelect
            v-model:value="seatForm.status"
            :options="statusSelectOptions"
          />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="seatShow = false">
            取消
          </NButton><NButton
            type="primary"
            :loading="seatSaving"
            @click="saveSeat"
          >
            保存
          </NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.layout-editor { display: flex; align-items: flex-start; flex-direction: column; gap: var(--ic-space-3); }
.layout-editor small { color: var(--ic-color-text-secondary); font-size: var(--ic-font-xs); }
</style>
