<script setup lang="ts">
import { computed, h, reactive, ref } from 'vue'
import {
  NForm, NFormItemGi, NGrid, NInput, NInputNumber, NSelect, NSpace, NTabPane, NTabs,
} from 'naive-ui'
import ResourceListView from '@/components/ResourceListView.vue'
import type { FilterField, ResourceListInstance } from '@/components/ui-types'
import FormDrawer from '@/components/FormDrawer.vue'
import PermissionButton from '@/components/PermissionButton.vue'
import { actionsColumn, dateTimeColumn, renderColumn, statusColumn, textColumn } from '@/utils/columns'
import { PERMISSIONS } from '@/constants/permissions'
import { API_PATHS } from '@/constants/api-paths'
import { runAudited } from '@/composables/useAuditedAction'
import { couponCategoryService, couponTemplateService } from '@/api/services'
import type { CouponCategory, CouponTemplate } from '@/api/models'
import { toastError, toastSuccess } from '@/utils/feedback'

const categoryListRef = ref<ResourceListInstance | null>(null)
const templateListRef = ref<ResourceListInstance | null>(null)
const categories = ref<CouponCategory[]>([])

const usageOptions = [
  { label: '赛事使用', value: 'event_ticket' },
  { label: '活动门票兑换', value: 'admission_ticket' },
  { label: '小吃兑换', value: 'snack' },
  { label: '酒水兑换', value: 'alcohol' },
  { label: '饮料兑换', value: 'beverage' },
  { label: '饮品兑换', value: 'drink' },
  { label: '餐食兑换', value: 'meal' },
  { label: '礼品兑换', value: 'gift' },
]
const categoryStatusOptions = [
  { label: '启用', value: 'active', tone: 'success' as const },
  { label: '停用', value: 'disabled', tone: 'warning' as const },
]
const templateStatusOptions = [
  { label: '草稿', value: 'draft', tone: 'default' as const },
  { label: '已发布', value: 'published', tone: 'success' as const },
  { label: '已停用', value: 'disabled', tone: 'warning' as const },
]
const scopeOptions = [
  { label: '总部券', value: 'global', tone: 'success' as const },
  { label: '门店券', value: 'store', tone: 'default' as const },
]
const categoryFields: FilterField[] = [
  { key: 'keyword', label: '类型名称', type: 'input' },
  { key: 'status', label: '状态', type: 'select', options: categoryStatusOptions },
]
const templateFields: FilterField[] = [
  { key: 'keyword', label: '券名称', type: 'input' },
  { key: 'status', label: '状态', type: 'select', options: templateStatusOptions },
]

function giftLimitText(row: CouponCategory): string {
  return row.giftDailyUsageLimit === 0 ? '不限' : `每日 ${row.giftDailyUsageLimit} 张`
}

const categoryColumns = [
  textColumn<CouponCategory>('ID', 'id', { width: 72 }),
  textColumn<CouponCategory>('券类型名称', 'name', { minWidth: 180 }),
  statusColumn<CouponCategory>('使用方式', 'businessType', usageOptions, 150),
  renderColumn<CouponCategory>('赠券使用限制', 'giftDailyUsageLimit', giftLimitText, 140),
  textColumn<CouponCategory>('排序', 'sortOrder', { width: 80 }),
  statusColumn<CouponCategory>('状态', 'status', categoryStatusOptions, 90),
  dateTimeColumn<CouponCategory>('更新时间', 'updatedAt', 170),
  actionsColumn<CouponCategory>((row) => h(NSpace, { size: 6 }, () => [
    h(PermissionButton, {
      permission: PERMISSIONS.COUPON_GLOBAL_WRITE,
      onClick: () => openCategory(row),
    }, () => '编辑'),
    h(PermissionButton, {
      permission: PERMISSIONS.COUPON_GLOBAL_WRITE,
      type: 'error',
      onClick: () => removeCategory(row),
    }, () => '删除'),
  ]), 130),
]

const templateColumns = [
  textColumn<CouponTemplate>('ID', 'id', { width: 72 }),
  textColumn<CouponTemplate>('券名称', 'name', { minWidth: 200 }),
  textColumn<CouponTemplate>('券类型', 'categoryName', { width: 140 }),
  statusColumn<CouponTemplate>('来源', 'scopeType', scopeOptions, 100),
  renderColumn<CouponTemplate>('所属门店', 'storeId', (row) => row.scopeType === 'global' ? '全部门店' : `门店 ${row.storeId ?? '-'}`, 110),
  statusColumn<CouponTemplate>('状态', 'status', templateStatusOptions, 90),
  dateTimeColumn<CouponTemplate>('更新时间', 'updatedAt', 170),
  actionsColumn<CouponTemplate>((row) => h(NSpace, { size: 6, wrap: false }, () => row.scopeType === 'store'
    ? [h('span', { class: 'readonly-text' }, '门店维护')]
    : [
        h(PermissionButton, {
          permission: PERMISSIONS.COUPON_GLOBAL_WRITE,
          onClick: () => openTemplate(row),
        }, () => '编辑'),
        h(PermissionButton, {
          permission: PERMISSIONS.COUPON_GLOBAL_WRITE,
          type: row.status === 'published' ? 'default' : 'primary',
          onClick: () => row.status === 'published' ? disableTemplate(row) : publishTemplate(row),
        }, () => row.status === 'published' ? '停用' : '发布'),
        h(PermissionButton, {
          permission: PERMISSIONS.COUPON_GLOBAL_WRITE,
          type: 'error',
          onClick: () => removeTemplate(row),
        }, () => '删除'),
      ]), 210),
]

const categoryDrawerShow = ref(false)
const categorySaving = ref(false)
const editingCategoryId = ref<string | null>(null)
const categoryForm = reactive({
  name: '', businessType: 'alcohol', sortOrder: 0, status: 'active', giftDailyUsageLimit: 1,
})

function openCategory(row?: CouponCategory): void {
  editingCategoryId.value = row ? String(row.id) : null
  Object.assign(categoryForm, row ? {
    name: row.name,
    businessType: row.businessType,
    sortOrder: row.sortOrder,
    status: row.status,
    giftDailyUsageLimit: row.giftDailyUsageLimit,
  } : { name: '', businessType: 'alcohol', sortOrder: 0, status: 'active', giftDailyUsageLimit: 1 })
  categoryDrawerShow.value = true
}

async function saveCategory(): Promise<void> {
  if (!categoryForm.name.trim()) return toastError('请填写券类型名称')
  categorySaving.value = true
  try {
    const payload = { ...categoryForm, name: categoryForm.name.trim() }
    if (editingCategoryId.value) await couponCategoryService.update(editingCategoryId.value, payload)
    else await couponCategoryService.create(payload)
    toastSuccess('券类型已保存')
    categoryDrawerShow.value = false
    await categoryListRef.value?.reload()
  } catch (error) {
    toastError((error as { message?: string }).message ?? '保存失败')
  } finally {
    categorySaving.value = false
  }
}

async function removeCategory(row: CouponCategory): Promise<void> {
  const ok = await runAudited({
    title: '删除券类型',
    content: `确认删除“${row.name}”？已被使用的券类型不能删除，只能停用。`,
    highRisk: true,
    execute: () => couponCategoryService.remove(String(row.id)),
    successText: '券类型已删除',
  })
  if (ok) await categoryListRef.value?.reload()
}

const templateDrawerShow = ref(false)
const templateSaving = ref(false)
const editingTemplateId = ref<string | null>(null)
const templateForm = reactive({
  name: '', description: '', categoryId: null as string | number | null, admissionCount: 1,
})
const categoryOptions = computed(() => categories.value.map((item) => ({
  label: item.name,
  value: item.id,
  disabled: item.status !== 'active',
})))
const selectedBusinessType = computed(() => categories.value.find(
  (item) => String(item.id) === String(templateForm.categoryId),
)?.businessType ?? '')

async function loadCategories(): Promise<void> {
  const result = await couponCategoryService.list({ page: 1, pageSize: 100 })
  categories.value = result.items
}

async function openTemplate(row?: CouponTemplate): Promise<void> {
  try {
    await loadCategories()
    const target = row ? await couponTemplateService.get(String(row.id)) : null
    editingTemplateId.value = target ? String(target.id) : null
    Object.assign(templateForm, target ? {
      name: target.name,
      description: target.description ?? '',
      categoryId: target.categoryId,
      admissionCount: target.admissionCount || 1,
    } : {
      name: '', description: '',
      categoryId: categoryOptions.value.find((item) => !item.disabled)?.value ?? null,
      admissionCount: 1,
    })
    if (!templateForm.categoryId) return toastError('请先启用至少一个券类型')
    templateDrawerShow.value = true
  } catch (error) {
    toastError((error as { message?: string }).message ?? '券模板加载失败')
  }
}

async function saveTemplate(): Promise<void> {
  if (!templateForm.name.trim()) return toastError('请填写券名称')
  if (!templateForm.categoryId) return toastError('请选择券类型')
  if (selectedBusinessType.value === 'admission_ticket'
    && (!Number.isInteger(templateForm.admissionCount) || templateForm.admissionCount < 1 || templateForm.admissionCount > 99)) {
    return toastError('请填写正确的可兑人数')
  }
  templateSaving.value = true
  try {
    const payload = {
      name: templateForm.name.trim(),
      description: templateForm.description.trim(),
      categoryId: templateForm.categoryId,
      admissionCount: selectedBusinessType.value === 'admission_ticket' ? templateForm.admissionCount : 1,
    }
    if (editingTemplateId.value) await couponTemplateService.update(editingTemplateId.value, payload)
    else await couponTemplateService.create(payload)
    toastSuccess('券模板已保存')
    templateDrawerShow.value = false
    await templateListRef.value?.reload()
  } catch (error) {
    toastError((error as { message?: string }).message ?? '保存失败')
  } finally {
    templateSaving.value = false
  }
}

async function publishTemplate(row: CouponTemplate): Promise<void> {
  const ok = await runAudited({
    title: '发布券模板',
    content: `确认发布“${row.name}”？发布后门店后台将可看到该总部券。`,
    highRisk: true,
    execute: () => couponTemplateService.action(API_PATHS.coupons.publishTemplate(String(row.id))),
    successText: '券模板已发布',
  })
  if (ok) await templateListRef.value?.reload()
}

async function disableTemplate(row: CouponTemplate): Promise<void> {
  const ok = await runAudited({
    title: '停用券模板',
    content: `确认停用“${row.name}”？已发出的券不受影响。`,
    highRisk: true,
    execute: () => couponTemplateService.action(API_PATHS.coupons.disableTemplate(String(row.id))),
    successText: '券模板已停用',
  })
  if (ok) await templateListRef.value?.reload()
}

async function removeTemplate(row: CouponTemplate): Promise<void> {
  const ok = await runAudited({
    title: '删除券模板',
    content: `确认删除“${row.name}”？已有发放记录的模板只能停用。`,
    highRisk: true,
    execute: () => couponTemplateService.remove(String(row.id)),
    successText: '券模板已删除',
  })
  if (ok) await templateListRef.value?.reload()
}

const categoryToolbarActions = [{
  key: 'create-category', label: '新增券类型', type: 'primary' as const,
  permission: PERMISSIONS.COUPON_GLOBAL_WRITE, onClick: () => openCategory(),
}]
const templateToolbarActions = [{
  key: 'create-template', label: '新增总部券', type: 'primary' as const,
  permission: PERMISSIONS.COUPON_GLOBAL_WRITE, onClick: () => openTemplate(),
}]
</script>

<template>
  <div>
    <NTabs
      type="line"
      animated
    >
      <NTabPane
        name="templates"
        tab="券模板"
      >
        <ResourceListView
          ref="templateListRef"
          title="券模板"
          description="统一查看总部券与门店券；总部券由这里维护，门店券由所属门店维护"
          :breadcrumb="['权益规则', '券管理']"
          :fields="templateFields"
          :columns="templateColumns"
          :fetcher="couponTemplateService.list"
          :toolbar-actions="templateToolbarActions"
          empty-text="暂无券模板"
        />
      </NTabPane>
      <NTabPane
        name="categories"
        tab="券类型与使用限制"
      >
        <ResourceListView
          ref="categoryListRef"
          title="券类型与赠券使用限制"
          description="管理券类型、兑换方式以及赠送券每位会员每天可使用的张数"
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
      v-model:show="categoryDrawerShow"
      :title="editingCategoryId ? '编辑券类型' : '新增券类型'"
      :submitting="categorySaving"
      :width="720"
      @submit="saveCategory"
    >
      <NForm label-placement="top">
        <NGrid
          :cols="2"
          :x-gap="16"
        >
          <NFormItemGi
            label="券类型名称"
            required
          >
            <NInput
              v-model:value="categoryForm.name"
              placeholder="例如：周赛赛事券"
            />
          </NFormItemGi>
          <NFormItemGi
            label="使用方式"
            required
          >
            <NSelect
              v-model:value="categoryForm.businessType"
              :options="usageOptions"
              :disabled="Boolean(editingCategoryId)"
            />
          </NFormItemGi>
          <NFormItemGi
            label="赠券每日使用上限"
            required
          >
            <NInputNumber
              v-model:value="categoryForm.giftDailyUsageLimit"
              :min="0"
              :max="999"
              :precision="0"
              placeholder="0 表示不限"
              style="width: 100%"
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
          <NFormItemGi
            label="状态"
            required
          >
            <NSelect
              v-model:value="categoryForm.status"
              :options="categoryStatusOptions"
            />
          </NFormItemGi>
        </NGrid>
        <p class="form-hint">
          0 表示不限；大于 0 时，所有非购买来源的赠券按该券类型合计限制。
        </p>
      </NForm>
    </FormDrawer>

    <FormDrawer
      v-model:show="templateDrawerShow"
      :title="editingTemplateId ? '编辑总部券' : '新增总部券'"
      :submitting="templateSaving"
      :width="720"
      @submit="saveTemplate"
    >
      <NForm label-placement="top">
        <NGrid
          :cols="2"
          :x-gap="16"
        >
          <NFormItemGi
            label="券名称"
            required
          >
            <NInput v-model:value="templateForm.name" />
          </NFormItemGi>
          <NFormItemGi
            label="券类型"
            required
          >
            <NSelect
              v-model:value="templateForm.categoryId"
              :options="categoryOptions"
            />
          </NFormItemGi>
          <NFormItemGi
            v-if="selectedBusinessType === 'admission_ticket'"
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
          <NFormItemGi
            label="使用说明"
            :span="2"
          >
            <NInput
              v-model:value="templateForm.description"
              type="textarea"
              :rows="3"
            />
          </NFormItemGi>
        </NGrid>
      </NForm>
    </FormDrawer>
  </div>
</template>

<style scoped>
.readonly-text,
.form-hint {
  color: var(--text-color-3);
  font-size: 13px;
}

.form-hint {
  margin: 0;
}
</style>
