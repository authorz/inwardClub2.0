<script setup lang="ts">
import { computed, h, onMounted, reactive, ref } from 'vue'
import {
  NForm,
  NFormItemGi,
  NGrid,
  NInput,
  NInputNumber,
  NSelect,
  NSpace,
  NTabPane,
  NTabs,
} from 'naive-ui'
import ResourceListView from '@/components/ResourceListView.vue'
import type { ResourceListInstance } from '@/components/ui-types'
import FormDrawer from '@/components/FormDrawer.vue'
import PermissionButton from '@/components/PermissionButton.vue'
import { actionsColumn, dateTimeColumn, statusColumn, textColumn } from '@/utils/columns'
import { SCOPE_TYPE_OPTIONS } from '@/constants/enums'
import { PERMISSIONS } from '@/constants/permissions'
import { runAudited } from '@/composables/useAuditedAction'
import { couponCategoryService, couponTemplateService } from '@/api/services'
import { API_PATHS } from '@/constants/api-paths'
import type { CouponCategory, CouponTemplate } from '@/api/models'
import type { FilterField } from '@/components/ui-types'
import { toastError, toastSuccess } from '@/utils/feedback'

const templateListRef = ref<ResourceListInstance | null>(null)
const categoryListRef = ref<ResourceListInstance | null>(null)
const activeTab = ref('templates')

const businessTypes = [
  { label: '赛事门票核销', value: 'event_ticket' },
  { label: '小吃兑换', value: 'snack' },
  { label: '酒水兑换', value: 'alcohol' },
  { label: '饮料兑换', value: 'beverage' },
  { label: '饮品或啤酒兑换', value: 'drink' },
  { label: '餐食兑换', value: 'meal' },
  { label: '礼品兑换', value: 'gift' },
]
const categoryStatuses = [
  { label: '启用', value: 'active', tone: 'success' as const },
  { label: '停用', value: 'disabled', tone: 'warning' as const },
]
const couponStatuses = [
  { label: '草稿', value: 'draft', tone: 'default' as const },
  { label: '已发布', value: 'published', tone: 'success' as const },
  { label: '已停用', value: 'disabled', tone: 'warning' as const },
]
const templateFields: FilterField[] = [
  { key: 'keyword', label: '券名称', type: 'input' },
  { key: 'status', label: '状态', type: 'select', options: couponStatuses },
]
const categoryFields: FilterField[] = [
  { key: 'keyword', label: '类型名称', type: 'input' },
  { key: 'status', label: '状态', type: 'select', options: categoryStatuses },
]

const categories = ref<CouponCategory[]>([])
const categoryOptions = computed(() => categories.value.map((item) => ({
  label: item.name,
  value: item.id,
})))

async function loadCategories(): Promise<void> {
  try {
    const result = await couponCategoryService.list({ status: 'active', page: 1, pageSize: 100 })
    categories.value = result.items
  } catch (error) {
    toastError((error as { message?: string }).message ?? '读取券类型失败')
  }
}

const templateColumns = [
  textColumn<CouponTemplate>('ID', 'id', { width: 72 }),
  textColumn<CouponTemplate>('券名称', 'name', { minWidth: 160 }),
  textColumn<CouponTemplate>('券类型', 'categoryName', { minWidth: 130 }),
  statusColumn<CouponTemplate>('范围', 'scopeType', SCOPE_TYPE_OPTIONS, 90),
  statusColumn<CouponTemplate>('状态', 'status', couponStatuses, 100),
  dateTimeColumn<CouponTemplate>('更新时间', 'updatedAt', 170),
  actionsColumn<CouponTemplate>((row) => h(NSpace, { size: 6, wrap: false }, () => [
    h(PermissionButton, { permission: PERMISSIONS.COUPON_GLOBAL_WRITE, onClick: () => openTemplateEdit(row) }, () => '编辑'),
    row.status !== 'published'
      ? h(PermissionButton, { permission: PERMISSIONS.COUPON_GLOBAL_WRITE, type: 'primary', onClick: () => publish(row) }, () => '发布')
      : h(PermissionButton, { permission: PERMISSIONS.COUPON_GLOBAL_WRITE, type: 'warning', onClick: () => disable(row) }, () => '停用'),
    h(PermissionButton, { permission: PERMISSIONS.COUPON_GLOBAL_WRITE, type: 'error', onClick: () => removeTemplate(row) }, () => '删除'),
  ]), 220),
]

const categoryColumns = [
  textColumn<CouponCategory>('ID', 'id', { width: 72 }),
  textColumn<CouponCategory>('类型名称', 'name', { minWidth: 180 }),
  statusColumn<CouponCategory>('兑换场景', 'businessType', businessTypes, 150),
  textColumn<CouponCategory>('排序', 'sortOrder', { width: 90 }),
  statusColumn<CouponCategory>('状态', 'status', categoryStatuses, 90),
  dateTimeColumn<CouponCategory>('更新时间', 'updatedAt', 170),
  actionsColumn<CouponCategory>((row) => h(NSpace, { size: 6 }, () => [
    h(PermissionButton, { permission: PERMISSIONS.COUPON_GLOBAL_WRITE, onClick: () => openCategoryEdit(row) }, () => '编辑'),
    h(PermissionButton, { permission: PERMISSIONS.COUPON_GLOBAL_WRITE, type: 'error', onClick: () => removeCategory(row) }, () => '删除'),
  ]), 130),
]

const showTemplate = ref(false)
const savingTemplate = ref(false)
const editingTemplateId = ref<string | null>(null)
const templateForm = reactive({
  name: '',
  description: '',
  categoryId: null as string | number | null,
  admissionCount: 1,
})
const selectedBusinessType = computed(() =>
  categories.value.find((item) => String(item.id) === String(templateForm.categoryId))?.businessType ?? '',
)

function resetTemplateForm(): void {
  Object.assign(templateForm, {
    name: '', description: '', categoryId: categoryOptions.value[0]?.value ?? null, admissionCount: 1,
  })
}
async function openTemplateCreate(): Promise<void> {
  editingTemplateId.value = null
  await loadCategories()
  resetTemplateForm()
  if (!templateForm.categoryId) return toastError('请先在“券类型”中启用至少一个类型')
  showTemplate.value = true
}
async function openTemplateEdit(row: CouponTemplate): Promise<void> {
  try {
    const [detail] = await Promise.all([
      couponTemplateService.get(String(row.id)),
      loadCategories(),
    ])
    if (!categories.value.some((item) => String(item.id) === String(detail.categoryId))) {
      categories.value.push({
        id: String(detail.categoryId),
        name: detail.categoryName,
        businessType: detail.couponType,
        sortOrder: 0,
        status: 'disabled',
        createdAt: detail.createdAt,
        updatedAt: detail.updatedAt,
      })
    }
    editingTemplateId.value = String(row.id)
    Object.assign(templateForm, {
      name: detail.name,
      description: detail.description ?? '',
      categoryId: detail.categoryId,
      admissionCount: detail.admissionCount || 1,
    })
    showTemplate.value = true
  } catch (error) {
    toastError((error as { message?: string }).message ?? '读取优惠券详情失败')
  }
}
async function saveTemplate(): Promise<void> {
  if (!templateForm.name.trim()) return toastError('请填写券名称')
  if (!templateForm.categoryId) return toastError('请选择券类型')
  if (selectedBusinessType.value === 'event_ticket' && (!Number.isInteger(templateForm.admissionCount) || templateForm.admissionCount < 1 || templateForm.admissionCount > 99)) {
    return toastError('请填写正确的可兑人数')
  }
  savingTemplate.value = true
  try {
    const payload = {
      name: templateForm.name.trim(),
      description: templateForm.description.trim(),
      categoryId: templateForm.categoryId,
      admissionCount: selectedBusinessType.value === 'event_ticket' ? templateForm.admissionCount : 1,
    }
    if (editingTemplateId.value) await couponTemplateService.update(editingTemplateId.value, payload)
    else await couponTemplateService.create(payload)
    toastSuccess('券模板已保存')
    showTemplate.value = false
    templateListRef.value?.reload()
  } catch (error) {
    toastError((error as { message?: string }).message ?? '保存失败')
  } finally {
    savingTemplate.value = false
  }
}
async function publish(row: CouponTemplate): Promise<void> {
  const ok = await runAudited({
    title: '发布券模板', content: `发布“${row.name}”后可用于 VIP 福利和商品关联。`,
    highRisk: true,
    execute: () => couponTemplateService.action(API_PATHS.coupons.publishTemplate(String(row.id))),
    successText: '券模板已发布',
  })
  if (ok) templateListRef.value?.reload()
}
async function disable(row: CouponTemplate): Promise<void> {
  const ok = await runAudited({
    title: '停用券模板', content: `确认停用“${row.name}”？已发出的券不受影响。`,
    highRisk: true,
    execute: () => couponTemplateService.action(API_PATHS.coupons.disableTemplate(String(row.id))),
    successText: '券模板已停用',
  })
  if (ok) templateListRef.value?.reload()
}
async function removeTemplate(row: CouponTemplate): Promise<void> {
  const ok = await runAudited({
    title: '删除券模板', content: `确认删除“${row.name}”？`, highRisk: true,
    execute: () => couponTemplateService.remove(String(row.id)), successText: '券模板已删除',
  })
  if (ok) templateListRef.value?.reload()
}

const showCategory = ref(false)
const savingCategory = ref(false)
const editingCategoryId = ref<string | null>(null)
const categoryForm = reactive({ name: '', businessType: 'alcohol', sortOrder: 0, status: 'active' })

function openCategoryCreate(): void {
  editingCategoryId.value = null
  Object.assign(categoryForm, { name: '', businessType: 'alcohol', sortOrder: 0, status: 'active' })
  showCategory.value = true
}
function openCategoryEdit(row: CouponCategory): void {
  editingCategoryId.value = String(row.id)
  Object.assign(categoryForm, {
    name: row.name,
    businessType: row.businessType,
    sortOrder: row.sortOrder,
    status: row.status,
  })
  showCategory.value = true
}
async function saveCategory(): Promise<void> {
  if (!categoryForm.name.trim()) return toastError('请填写券类型名称')
  savingCategory.value = true
  try {
    const payload = { ...categoryForm, name: categoryForm.name.trim() }
    if (editingCategoryId.value) await couponCategoryService.update(editingCategoryId.value, payload)
    else await couponCategoryService.create(payload)
    toastSuccess('券类型已保存')
    showCategory.value = false
    await Promise.all([categoryListRef.value?.reload(), loadCategories()])
    templateListRef.value?.reload()
  } catch (error) {
    toastError((error as { message?: string }).message ?? '保存失败')
  } finally {
    savingCategory.value = false
  }
}
async function removeCategory(row: CouponCategory): Promise<void> {
  const ok = await runAudited({
    title: '删除券类型', content: `确认删除“${row.name}”？已被券模板使用的类型不能删除，只能停用。`,
    highRisk: true,
    execute: () => couponCategoryService.remove(String(row.id)), successText: '券类型已删除',
  })
  if (ok) {
    await Promise.all([categoryListRef.value?.reload(), loadCategories()])
    templateListRef.value?.reload()
  }
}

const templateToolbarActions = [{
  key: 'create', label: '新增券模板', type: 'primary' as const,
  permission: PERMISSIONS.COUPON_GLOBAL_WRITE, onClick: openTemplateCreate,
}]
const categoryToolbarActions = [{
  key: 'create', label: '新增券类型', type: 'primary' as const,
  permission: PERMISSIONS.COUPON_GLOBAL_WRITE, onClick: openCategoryCreate,
}]

onMounted(loadCategories)
</script>

<template>
  <div>
    <NTabs v-model:value="activeTab" type="line" animated>
      <NTabPane name="templates" tab="券模板">
        <ResourceListView
          ref="templateListRef"
          title="券模板"
          description="创建发放规则时只选择已配置的券类型，兑换场景由券类型统一决定"
          :breadcrumb="['权益规则', '券管理']"
          :fields="templateFields"
          :columns="templateColumns"
          :fetcher="couponTemplateService.list"
          :toolbar-actions="templateToolbarActions"
          empty-text="暂无券模板，点击右上角新增"
        />
      </NTabPane>
      <NTabPane name="categories" tab="券类型">
        <ResourceListView
          ref="categoryListRef"
          title="券类型"
          description="统一管理券类型名称、兑换场景、排序和启停；停用后不可用于新建券模板"
          :breadcrumb="['权益规则', '券管理']"
          :fields="categoryFields"
          :columns="categoryColumns"
          :fetcher="couponCategoryService.list"
          :toolbar-actions="categoryToolbarActions"
          empty-text="暂无券类型，点击右上角新增"
        />
      </NTabPane>
    </NTabs>

    <FormDrawer
      v-model:show="showTemplate"
      :title="editingTemplateId ? '编辑券模板' : '新增券模板'"
      :submitting="savingTemplate"
      :width="720"
      @submit="saveTemplate"
    >
      <NForm label-placement="top">
        <NGrid :cols="2" :x-gap="16">
          <NFormItemGi label="券名称" required>
            <NInput v-model:value="templateForm.name" placeholder="例如：周赛门票" />
          </NFormItemGi>
          <NFormItemGi label="券类型" required>
            <NSelect
              v-model:value="templateForm.categoryId"
              :options="categoryOptions"
              placeholder="选择总后台已配置的券类型"
            />
          </NFormItemGi>
          <NFormItemGi
            v-if="selectedBusinessType === 'event_ticket'"
            label="可兑人数"
            required
          >
            <NInputNumber
              v-model:value="templateForm.admissionCount"
              :min="1"
              :max="99"
              :precision="0"
              style="width: 100%"
            />
          </NFormItemGi>
          <NFormItemGi label="使用说明" :span="2">
            <NInput
              v-model:value="templateForm.description"
              type="textarea"
              :rows="3"
              placeholder="说明适用范围和使用条件"
            />
          </NFormItemGi>
        </NGrid>
      </NForm>
    </FormDrawer>

    <FormDrawer
      v-model:show="showCategory"
      :title="editingCategoryId ? '编辑券类型' : '新增券类型'"
      :submitting="savingCategory"
      :width="640"
      @submit="saveCategory"
    >
      <NForm label-placement="top">
        <NGrid :cols="2" :x-gap="16">
          <NFormItemGi label="类型名称" required>
            <NInput v-model:value="categoryForm.name" placeholder="例如：周赛门票券" />
          </NFormItemGi>
          <NFormItemGi label="兑换场景" required>
            <NSelect
              v-model:value="categoryForm.businessType"
              :options="businessTypes"
              :disabled="Boolean(editingCategoryId)"
            />
          </NFormItemGi>
          <NFormItemGi label="排序">
            <NInputNumber
              v-model:value="categoryForm.sortOrder"
              :min="0"
              :precision="0"
              style="width: 100%"
            />
          </NFormItemGi>
          <NFormItemGi label="状态" required>
            <NSelect v-model:value="categoryForm.status" :options="categoryStatuses" />
          </NFormItemGi>
        </NGrid>
      </NForm>
    </FormDrawer>
  </div>
</template>
