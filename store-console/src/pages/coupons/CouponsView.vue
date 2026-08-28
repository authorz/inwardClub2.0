<script setup lang="ts">
import { computed, h, onMounted, reactive, ref } from 'vue'
import { NButton, NForm, NFormItemGi, NGrid, NInput, NInputNumber, NModal, NSelect, NSpace, type DataTableColumns } from 'naive-ui'
import { couponService } from '@/api/services'
import { useAsyncList } from '@/composables/useAsyncList'
import { confirm } from '@/composables/useConfirm'
import { feedback } from '@/utils/feedback'
import { PERM } from '@/constants/permissions'
import type { EnumOption } from '@/constants/enums'
import { DataTable, PageHeader, PermissionButton, StatusFilterBar } from '@/components/common'
import { dateColumn, statusColumn, textColumn } from '@/utils/columns'
import type { CouponCategory, CouponTemplate } from '@/types/models'

const categories = ref<CouponCategory[]>([])
const categoryOptions = computed(() => categories.value.map((item) => ({ label: item.name, value: item.id })))
const statusMap: Record<string, EnumOption> = {
  draft: { label: '草稿', value: 'draft', tone: 'default' },
  published: { label: '已发布', value: 'published', tone: 'success' },
  disabled: { label: '已停用', value: 'disabled', tone: 'warning' },
}
const list = useAsyncList<CouponTemplate>((params) => couponService.list(params))
const status = ref<string | null>(null)
const keyword = ref('')
const show = ref(false)
const saving = ref(false)
const form = reactive({
  id: null as string | number | null, name: '', description: '', categoryId: null as string | number | null, admissionCount: 1,
})
const selectedBusinessType = computed(() =>
  categories.value.find((item) => String(item.id) === String(form.categoryId))?.businessType ?? '',
)

async function loadCategories(): Promise<void> {
  try {
    const result = await couponService.categories({ page: 1, pageSize: 100 })
    categories.value = result.rows
  } catch (error) {
    feedback.message.error((error as { message?: string }).message ?? '券类型加载失败')
  }
}

async function open(row?: CouponTemplate): Promise<void> {
  await loadCategories()
  let target = row
  if (row) {
    try { target = await couponService.detail(row.id) }
    catch (error) { return void feedback.message.error((error as { message?: string }).message ?? '详情加载失败') }
  }
  if (target && !categories.value.some((item) => String(item.id) === String(target?.categoryId))) {
    categories.value.push({
      id: target.categoryId,
      name: target.categoryName,
      businessType: target.couponType,
      sortOrder: 0,
      giftDailyUsageLimit: 0,
      status: 'disabled',
    })
  }
  Object.assign(form, target ? {
    id: target.id, name: target.name, description: target.description ?? '',
    categoryId: target.categoryId, admissionCount: target.admissionCount || 1,
  } : { id: null, name: '', description: '', categoryId: categoryOptions.value[0]?.value ?? null, admissionCount: 1 })
  if (!form.categoryId) return void feedback.message.error('请联系总后台先启用券类型')
  show.value = true
}

async function save(): Promise<void> {
  if (!form.name.trim()) return void feedback.message.error('请填写券商品名称')
  if (!form.categoryId) return void feedback.message.error('请选择券类型')
  if (selectedBusinessType.value === 'admission_ticket' && (!Number.isInteger(form.admissionCount) || form.admissionCount < 1 || form.admissionCount > 99)) {
    return void feedback.message.error('请填写正确的可兑人数')
  }
  saving.value = true
  try {
    const body = {
      name: form.name.trim(), description: form.description.trim(), categoryId: form.categoryId,
      admissionCount: selectedBusinessType.value === 'admission_ticket' ? form.admissionCount : 1,
    }
    if (form.id == null) await couponService.create(body)
    else await couponService.update(form.id, body)
    feedback.message.success('券商品已保存')
    show.value = false
    list.refresh()
  } catch (error) { feedback.message.error((error as { message?: string }).message ?? '保存失败') }
  finally { saving.value = false }
}

async function remove(row: CouponTemplate): Promise<void> {
  if (!await confirm({ content: `确认删除券商品“${row.name}”？已发放的数据不会被删除。`, danger: true })) return
  try { await couponService.remove(row.id); feedback.message.success('券商品已删除'); list.refresh() }
  catch (error) { feedback.message.error((error as { message?: string }).message ?? '删除失败') }
}

async function publish(row: CouponTemplate): Promise<void> {
  if (!await confirm({ content: `确认发布券商品“${row.name}”？发布后可用于本店购买或发放。` })) return
  try { await couponService.publish(row.id); feedback.message.success('券商品已发布'); list.refresh() }
  catch (error) { feedback.message.error((error as { message?: string }).message ?? '发布失败') }
}

async function disable(row: CouponTemplate): Promise<void> {
  if (!await confirm({ content: `确认停用券商品“${row.name}”？已发出的券不受影响。`, danger: true })) return
  try { await couponService.disable(row.id); feedback.message.success('券商品已停用'); list.refresh() }
  catch (error) { feedback.message.error((error as { message?: string }).message ?? '停用失败') }
}

function applyFilters(): void {
  list.applyFilters({ status: status.value, keyword: keyword.value.trim() })
}

function resetFilters(): void {
  status.value = null
  keyword.value = ''
  list.reset()
}

const columns = computed<DataTableColumns<CouponTemplate>>(() => [
  textColumn<CouponTemplate>('券商品名称', (row) => row.name),
  textColumn<CouponTemplate>('券类型', (row) => row.categoryName, { width: 130 }),
  statusColumn<CouponTemplate>('状态', statusMap, (row) => row.status, { width: 100 }),
  dateColumn<CouponTemplate>('更新时间', (row) => row.updatedAt, { width: 150 }),
  { title: '操作', key: 'actions', width: 220, render: (row) => h(NSpace, { size: 4, wrap: false }, () => [
    h(PermissionButton, { permissions: [PERM.couponWrite], text: true, onClick: () => open(row) }, () => '编辑'),
    row.status !== 'published'
      ? h(PermissionButton, { permissions: [PERM.couponWrite], type: 'primary', onClick: () => publish(row) }, () => '发布')
      : h(PermissionButton, { permissions: [PERM.couponWrite], type: 'warning', onClick: () => disable(row) }, () => '停用'),
    h(PermissionButton, { permissions: [PERM.couponWrite], text: true, type: 'error', onClick: () => remove(row) }, () => '删除'),
  ]) },
])

onMounted(loadCategories)
</script>

<template>
  <div>
    <PageHeader
      title="本店券商品"
      description="基于总后台维护的券类型创建本店券商品；发布后可用于本店购买或发放"
    />
    <StatusFilterBar
      v-model:status="status"
      v-model:keyword="keyword"
      :status-options="Object.values(statusMap)"
      search-placeholder="搜索券名称"
      :loading="list.loading.value"
      @apply="applyFilters"
      @reset="resetFilters"
    >
      <template #actions>
        <PermissionButton
          :permissions="[PERM.couponWrite]"
          type="primary"
          @click="open()"
        >
          新增券商品
        </PermissionButton>
      </template>
    </StatusFilterBar>
    <DataTable
      :columns="columns"
      :data="list.rows.value"
      :loading="list.loading.value"
      :page="list.page.value"
      :page-size="list.pageSize.value"
      :total="list.total.value"
      empty-text="暂无本店券商品"
      @update:page="list.setPage"
      @update:page-size="list.setPageSize"
    />
    <NModal
      v-model:show="show"
      preset="card"
      :title="form.id == null ? '新增券商品' : '编辑券商品'"
      style="width: min(720px, 92vw)"
    >
      <NForm label-placement="top">
        <NGrid
          :cols="2"
          :x-gap="16"
        >
          <NFormItemGi
            label="券商品名称"
            required
          >
            <NInput v-model:value="form.name" />
          </NFormItemGi>
          <NFormItemGi
            label="券类型"
            required
          >
            <NSelect
              v-model:value="form.categoryId"
              :options="categoryOptions"
            />
          </NFormItemGi>
          <NFormItemGi
            v-if="selectedBusinessType === 'admission_ticket'"
            label="可兑人数"
            required
          >
            <NInputNumber
              v-model:value="form.admissionCount"
              :min="1"
              :max="99"
              :precision="0"
              style="width: 100%"
            />
          </NFormItemGi>
          <NFormItemGi
            label="使用说明"
            :span="2"
          >
            <NInput
              v-model:value="form.description"
              type="textarea"
              :rows="3"
            />
          </NFormItemGi>
        </NGrid>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="show = false">
            取消
          </NButton><NButton
            type="primary"
            :loading="saving"
            @click="save"
          >
            保存
          </NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>
