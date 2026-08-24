<script setup lang="ts">
import { computed, h, reactive, ref } from 'vue'
import {
  NButton,
  NDatePicker,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NModal,
  NSelect,
  NSpace,
  type DataTableColumns,
} from 'naive-ui'
import { activityService } from '@/api/services'
import { useAsyncList } from '@/composables/useAsyncList'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { PUBLISH_STATUS, toOptions } from '@/constants/enums'
import { PERM } from '@/constants/permissions'
import { statusColumn, textColumn } from '@/utils/columns'
import { formatDateTime } from '@/utils/format'
import {
  AssetImage,
  AssetUpload,
  DataTable,
  PageHeader,
  PermissionButton,
  RichTextEditor,
} from '@/components/common'
import { feedback } from '@/utils/feedback'
import type { ActivityTicketType, StoreActivity } from '@/types/models'

interface TicketTypeForm {
  key: number
  id: string | number | null
  name: string
  admissionCount: number
  priceYuan: number | null
  stockQuantity: number
  saleRange: [number, number] | null
  payChannels: string[]
  maxTicketsPerOrder: number
  status: string
}

const list = useAsyncList<StoreActivity>((params) => activityService.list(params), {
  initialFilters: { status: '', keyword: '' },
})
const action = useAsyncAction()
const editShow = ref(false)
const originalTicketTypeIds = ref<Array<string | number>>([])
const statusOptions = toOptions(PUBLISH_STATUS).map(({ label, value }) => ({ label, value }))
const ticketNameOptions = ['早鸟票', '预售票', '单人票', '双人票'].map((value) => ({ label: value, value }))
const ticketStatusOptions = [
  { label: '启用', value: 'active' },
  { label: '停用', value: 'inactive' },
]
const ticketPayChannelOptions = [
  { label: '微信', value: 'wechat' },
  { label: '金币', value: 'coin' },
]
const activityPayChannelOptions = [
  ...ticketPayChannelOptions,
  { label: '券兑换', value: 'coupon' },
]
let ticketKeySeed = 0

function newTicketType(name = '单人票'): TicketTypeForm {
  return {
    key: ++ticketKeySeed,
    id: null,
    name,
    admissionCount: 1,
    priceYuan: null,
    stockQuantity: 0,
    saleRange: null,
    payChannels: ['wechat'],
    maxTicketsPerOrder: 0,
    status: 'active',
  }
}

function mapTicketType(ticket: ActivityTicketType): TicketTypeForm {
  const payChannels = (ticket.payChannels ?? []).filter((channel) => channel !== 'coupon')
  return {
    key: ++ticketKeySeed,
    id: ticket.id,
    name: ticket.name,
    admissionCount: ticket.admissionCount || 1,
    priceYuan: ticket.priceCent / 100,
    stockQuantity: ticket.stockQuantity,
    saleRange:
      ticket.saleStartAt && ticket.saleEndAt
        ? [new Date(ticket.saleStartAt).getTime(), new Date(ticket.saleEndAt).getTime()]
        : null,
    payChannels: payChannels.length ? payChannels : ['wechat'],
    maxTicketsPerOrder: ticket.maxTicketsPerOrder ?? 0,
    status: ticket.status || 'active',
  }
}

function isTimedTicket(ticket: TicketTypeForm): boolean {
  return ticket.name === '早鸟票' || ticket.name === '预售票'
}

const form = reactive({
  id: null as string | number | null,
  title: '',
  description: '',
  content: '',
  assetId: null as string | null,
  imageUrl: '',
  activityRange: null as [number, number] | null,
  payChannels: ['wechat'] as string[],
  purchaseLimitPerMember: 0,
  ticketTypes: [newTicketType()] as TicketTypeForm[],
  status: 'published',
})

async function openEdit(row?: StoreActivity): Promise<void> {
  let target = row
  let ticketTypes: ActivityTicketType[] = []
  if (row) {
    try {
      ;[target, ticketTypes] = await Promise.all([
        activityService.detail(row.id),
        activityService.ticketTypes(row.id),
      ])
    } catch (error) {
      feedback.message.error((error as { message?: string }).message ?? '详情加载失败')
      return
    }
  }
  Object.assign(
    form,
    target
      ? {
          id: target.id,
          title: target.title,
          description: target.description ?? '',
          content: target.content ?? '',
          assetId: target.assetId == null ? null : String(target.assetId),
          imageUrl: target.imageUrl ?? '',
          activityRange:
            target.startAt && target.endAt
              ? [new Date(target.startAt).getTime(), new Date(target.endAt).getTime()]
              : null,
          payChannels: target.payChannels?.length ? target.payChannels : ['wechat'],
          purchaseLimitPerMember: target.purchaseLimitPerMember ?? 0,
          ticketTypes: ticketTypes.length ? ticketTypes.map(mapTicketType) : [newTicketType()],
          status: target.status,
        }
      : {
          id: null,
          title: '',
          description: '',
          content: '',
          assetId: null,
          imageUrl: '',
          activityRange: null,
          payChannels: ['wechat'],
          purchaseLimitPerMember: 0,
          ticketTypes: [newTicketType()],
          status: 'published',
        },
  )
  originalTicketTypeIds.value = ticketTypes.map((ticket) => ticket.id)
  editShow.value = true
}

function validateForm(): boolean {
  if (!form.title.trim()) {
    feedback.message.error('请填写活动名称')
    return false
  }
  if (!form.payChannels.length) {
    feedback.message.error('请选择活动支付方式')
    return false
  }
  if (form.activityRange && form.activityRange[0] >= form.activityRange[1]) {
    feedback.message.error('活动结束时间必须晚于开始时间')
    return false
  }
  if (!form.ticketTypes.length) {
    feedback.message.error('请至少添加一个票档')
    return false
  }
  for (const ticket of form.ticketTypes) {
    if (!ticket.name.trim()) {
      feedback.message.error('请填写票档名称')
      return false
    }
    if (!Number.isInteger(ticket.admissionCount) || ticket.admissionCount < 1 || ticket.admissionCount > 99) {
      feedback.message.error(`请填写“${ticket.name}”的正确入场人数`)
      return false
    }
    if (ticket.priceYuan == null || ticket.priceYuan <= 0) {
      feedback.message.error(`请填写“${ticket.name}”的正确价格`)
      return false
    }
    if (ticket.stockQuantity < 0) {
      feedback.message.error(`“${ticket.name}”的库存不能小于 0`)
      return false
    }
    if (!ticket.payChannels.length) {
      feedback.message.error(`请选择“${ticket.name}”的支付方式`)
      return false
    }
    if (isTimedTicket(ticket) && !ticket.saleRange) {
      feedback.message.error(`请设置“${ticket.name}”的售卖时间`)
      return false
    }
    if (ticket.saleRange && ticket.saleRange[0] >= ticket.saleRange[1]) {
      feedback.message.error(`“${ticket.name}”的售卖结束时间必须晚于开始时间`)
      return false
    }
  }
  return true
}

async function saveEdit(): Promise<void> {
  if (!validateForm()) return
  await action.run(
    async () => {
      const payload = {
        title: form.title.trim(),
        description: form.description.trim(),
        content: form.content.trim(),
        assetId: form.assetId ? Number(form.assetId) : undefined,
        startAt: form.activityRange ? new Date(form.activityRange[0]).toISOString() : undefined,
        endAt: form.activityRange ? new Date(form.activityRange[1]).toISOString() : undefined,
        payChannels: form.payChannels,
        purchaseLimitPerMember: form.purchaseLimitPerMember,
        status: form.status,
      }
      const activity =
        form.id == null
          ? await activityService.create(payload)
          : await activityService.update(form.id, payload)
      const retainedIds = new Set<string>()
      for (const ticket of form.ticketTypes) {
        const ticketPayload = {
          name: ticket.name.trim(),
          admissionCount: ticket.admissionCount,
          priceCent: Math.round((ticket.priceYuan ?? 0) * 100),
          stockQuantity: ticket.stockQuantity,
          saleStartAt: ticket.saleRange
            ? new Date(ticket.saleRange[0]).toISOString()
            : undefined,
          saleEndAt: ticket.saleRange ? new Date(ticket.saleRange[1]).toISOString() : undefined,
          payChannels: ticket.payChannels,
          maxTicketsPerOrder: ticket.maxTicketsPerOrder,
          status: ticket.status,
        }
        if (ticket.id != null) {
          await activityService.updateTicketType(activity.id, ticket.id, ticketPayload)
          retainedIds.add(String(ticket.id))
        } else {
          const created = await activityService.createTicketType(activity.id, ticketPayload)
          retainedIds.add(String(created.id))
        }
      }
      for (const id of originalTicketTypeIds.value) {
        if (!retainedIds.has(String(id))) {
          await activityService.removeTicketType(activity.id, id)
        }
      }
      return true
    },
    {
      successMessage: '活动及票档已保存',
      onSuccess: () => {
        editShow.value = false
        list.refresh()
      },
    },
  )
}

function removeActivity(row: StoreActivity): void {
  void action.run(
    () => activityService.remove(row.id),
    {
      confirm: { title: '删除活动', content: `确认删除活动「${row.title}」？删除后无法恢复。` },
      successMessage: '活动已删除',
      onSuccess: () => list.refresh(),
    },
  )
}

function activityPeriod(row: StoreActivity) {
  if (!row.startAt && !row.endAt) return '-'
  return h('div', { class: 'activity-period' }, [
    h('span', {}, row.startAt ? formatDateTime(row.startAt) : '未设置开始时间'),
    h(
      'span',
      { class: 'activity-period__end' },
      `至 ${row.endAt ? formatDateTime(row.endAt) : '未设置结束时间'}`,
    ),
  ])
}

function activityAuditTime(row: StoreActivity): string {
  const createdTime = row.createdAt ? Date.parse(row.createdAt) : Number.NaN
  const updatedTime = row.updatedAt ? Date.parse(row.updatedAt) : Number.NaN
  const isUpdated = Number.isFinite(updatedTime) && (!Number.isFinite(createdTime) || updatedTime > createdTime)
  const value = isUpdated ? row.updatedAt : row.createdAt
  return value ? `${isUpdated ? '更新' : '创建'} ${formatDateTime(value)}` : '-'
}

const columns = computed<DataTableColumns<StoreActivity>>(() => [
  textColumn<StoreActivity>('ID', (row) => row.id, { width: 72 }),
  {
    title: '活动封面',
    key: 'imageUrl',
    width: 112,
    render: (row: StoreActivity) =>
      h(AssetImage, {
        src: row.imageUrl ?? null,
        assetId: row.assetId ?? null,
        width: 88,
        height: 50,
      }),
  },
  textColumn<StoreActivity>('活动标题', (row) => row.title, { width: 220 }),
  {
    title: '活动时间',
    key: 'activityPeriod',
    width: 190,
    render: activityPeriod,
  },
  textColumn<StoreActivity>('创建/更新时间', activityAuditTime, { width: 190 }),
  statusColumn<StoreActivity>('状态', PUBLISH_STATUS, (row) => row.status, { width: 96 }),
  {
    title: '操作',
    key: 'actions',
    width: 180,
    fixed: 'right',
    render: (row: StoreActivity) =>
      h(NSpace, { size: 8, wrap: false }, {
        default: () => [
          h(PermissionButton, { permissions: [PERM.activityWrite], onClick: () => openEdit(row) }, { default: () => '编辑详情' }),
          h(PermissionButton, { permissions: [PERM.activityWrite], type: 'error', onClick: () => removeActivity(row) }, { default: () => '删除' }),
        ],
      }),
  },
])
</script>

<template>
  <div>
    <PageHeader
      title="活动管理"
      description="维护活动基本信息、票档、封面、图文详情、支付方式和发布状态"
    >
      <template #actions>
        <PermissionButton
          :permissions="[PERM.activityWrite]"
          type="primary"
          @click="openEdit()"
        >
          新增活动
        </PermissionButton>
      </template>
    </PageHeader>

    <div class="activity-filters">
      <div class="activity-filter-field">
        <label>活动标题</label>
        <NInput
          :value="(list.filters.keyword as string) ?? ''"
          placeholder="支持标题模糊搜索"
          clearable
          @update:value="list.filters.keyword = $event"
          @keyup.enter="list.applyFilters({})"
        />
      </div>
      <div class="activity-filter-field activity-filter-field--status">
        <label>状态</label>
        <NSelect
          :value="(list.filters.status as string) ?? ''"
          :options="[{ label: '全部', value: '' }, ...statusOptions]"
          @update:value="list.filters.status = $event"
        />
      </div>
      <div class="activity-filter-actions">
        <NButton
          type="primary"
          :loading="list.loading.value"
          @click="list.applyFilters({})"
        >
          查询
        </NButton>
        <NButton @click="list.reset()">
          重置
        </NButton>
      </div>
    </div>

    <DataTable
      :columns="columns"
      :data="list.rows.value"
      :loading="list.loading.value"
      :page="list.page.value"
      :page-size="list.pageSize.value"
      :total="list.total.value"
      empty-text="暂无活动"
      @update:page="list.setPage"
      @update:page-size="list.setPageSize"
    />

    <NModal
      v-model:show="editShow"
      preset="card"
      :title="form.id == null ? '新增活动' : '编辑活动'"
      class="activity-modal"
      style="width: min(880px, calc(100vw - 32px))"
    >
      <NForm label-placement="top">
        <NFormItem
          label="发布状态"
          required
          class="activity-form__status"
        >
          <NSelect
            v-model:value="form.status"
            :options="statusOptions"
          />
        </NFormItem>

        <NFormItem
          label="活动标题"
          required
        >
          <NInput
            v-model:value="form.title"
            placeholder="请输入活动标题"
            maxlength="128"
            show-count
          />
        </NFormItem>

        <div class="activity-form__grid">
          <NFormItem label="活动时间">
            <NDatePicker
              v-model:value="form.activityRange"
              type="datetimerange"
              clearable
              style="width: 100%"
            />
          </NFormItem>
          <NFormItem label="每人限购">
            <NInputNumber
              v-model:value="form.purchaseLimitPerMember"
              :min="0"
              :precision="0"
              placeholder="0 表示不限制"
              style="width: 100%"
            />
          </NFormItem>
        </div>

        <NFormItem label="活动摘要">
          <NInput
            v-model:value="form.description"
            type="textarea"
            :autosize="{ minRows: 2, maxRows: 4 }"
            placeholder="用于活动列表和分享摘要"
            maxlength="500"
            show-count
          />
        </NFormItem>

        <NFormItem
          label="活动支付方式"
          required
        >
          <NSelect
            v-model:value="form.payChannels"
            :options="activityPayChannelOptions"
            multiple
            placeholder="请选择支付方式"
          />
        </NFormItem>

        <section class="ticket-types">
          <div class="ticket-types__header">
            <div>
              <h3>票档设置</h3>
              <p>活动勾选“券兑换”后，所有票档均可使用赛事门票券；票档单独设置微信或金币支付。库存为 0 表示不限量。</p>
            </div>
            <NButton
              secondary
              @click="form.ticketTypes.push(newTicketType())"
            >
              新增票档
            </NButton>
          </div>
          <div
            v-for="(ticket, index) in form.ticketTypes"
            :key="ticket.key"
            class="ticket-type"
          >
            <div class="ticket-type__heading">
              <strong>票档 {{ index + 1 }}</strong>
              <NButton
                v-if="form.ticketTypes.length > 1"
                text
                type="error"
                @click="form.ticketTypes.splice(index, 1)"
              >
                删除
              </NButton>
            </div>
            <div class="ticket-type-grid ticket-type-grid--primary">
              <NFormItem
                label="票档名称"
                required
              >
                <NSelect
                  v-model:value="ticket.name"
                  :options="ticketNameOptions"
                  tag
                  filterable
                />
              </NFormItem>
              <NFormItem
                label="入场人数"
                required
              >
                <NInputNumber
                  v-model:value="ticket.admissionCount"
                  :min="1"
                  :max="99"
                  :precision="0"
                  style="width: 100%"
                />
              </NFormItem>
              <NFormItem
                label="价格"
                required
              >
                <NInputNumber
                  v-model:value="ticket.priceYuan"
                  :min="0.01"
                  :precision="2"
                  style="width: 100%"
                >
                  <template #prefix>
                    ¥
                  </template>
                </NInputNumber>
              </NFormItem>
              <NFormItem label="库存">
                <NInputNumber
                  v-model:value="ticket.stockQuantity"
                  :min="0"
                  :precision="0"
                  placeholder="0 表示不限量"
                  style="width: 100%"
                />
              </NFormItem>
              <NFormItem label="状态">
                <NSelect
                  v-model:value="ticket.status"
                  :options="ticketStatusOptions"
                />
              </NFormItem>
            </div>
            <div class="ticket-type-grid ticket-type-grid--rules">
              <NFormItem
                :label="isTimedTicket(ticket) ? '售卖时间（必填）' : '售卖时间（选填）'"
                :required="isTimedTicket(ticket)"
              >
                <NDatePicker
                  v-model:value="ticket.saleRange"
                  type="datetimerange"
                  clearable
                  style="width: 100%"
                />
              </NFormItem>
              <NFormItem label="单次限购">
                <NInputNumber
                  v-model:value="ticket.maxTicketsPerOrder"
                  :min="0"
                  :precision="0"
                  placeholder="0 表示不限制"
                  style="width: 100%"
                />
              </NFormItem>
              <NFormItem
                label="票档支付方式"
                required
              >
                <NSelect
                  v-model:value="ticket.payChannels"
                  :options="ticketPayChannelOptions"
                  multiple
                />
              </NFormItem>
            </div>
          </div>
        </section>

        <NFormItem label="活动封面">
          <div class="activity-form__asset">
            <AssetUpload
              v-model:asset-id="form.assetId"
              v-model:preview-url="form.imageUrl"
              purpose="activity"
              :width="240"
              :height="320"
            />
            <p>建议上传 3:4 竖版海报（如 750×1000px），支持 JPG、PNG、WebP。</p>
          </div>
        </NFormItem>

        <NFormItem label="图文详情">
          <RichTextEditor
            v-model="form.content"
            placeholder="编辑活动详情，可插入标题、列表、链接和图片"
          />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
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
        </NSpace>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.activity-filters {
  display: flex;
  align-items: flex-end;
  gap: var(--ic-space-4);
  margin-bottom: var(--ic-space-4);
  padding: var(--ic-space-4);
  border: 1px solid var(--ic-color-border);
  border-radius: var(--ic-radius-lg);
  background: var(--ic-color-surface);
}

.activity-filter-field {
  width: 260px;
}

.activity-filter-field--status {
  width: 180px;
}

.activity-filter-field label {
  display: block;
  margin-bottom: var(--ic-space-2);
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-sm);
}

.activity-filter-actions {
  display: flex;
  gap: var(--ic-space-2);
  margin-left: auto;
}

.activity-period {
  display: flex;
  flex-direction: column;
  gap: 2px;
  line-height: 1.45;
  white-space: nowrap;
}

.activity-period__end {
  color: var(--ic-color-text-secondary);
}

.activity-modal :deep(.n-card__content) {
  max-height: calc(100vh - 190px);
  overflow-y: auto;
}

.activity-form__status {
  max-width: 280px;
}

.activity-form__grid {
  display: grid;
  grid-template-columns: minmax(0, 2fr) minmax(220px, 1fr);
  gap: 0 16px;
}

.activity-form__asset p {
  margin: 8px 0 0;
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-xs);
}

.ticket-types__header,
.ticket-type__heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.ticket-types__header h3,
.ticket-types__header p {
  margin: 0;
}

.ticket-types__header p {
  margin-top: 4px;
  color: #737373;
  font-size: 13px;
}

.ticket-type {
  padding: 16px 0;
  border-top: 1px solid #e5e5e5;
}

.ticket-type__heading {
  margin-bottom: 8px;
}

.ticket-type-grid {
  display: grid;
  gap: 0 16px;
}

.ticket-type-grid--primary {
  grid-template-columns: minmax(150px, 1.3fr) minmax(100px, 0.7fr) repeat(3, minmax(110px, 1fr));
}

.ticket-type-grid--rules {
  grid-template-columns: minmax(280px, 2fr) minmax(130px, 1fr) minmax(180px, 1.3fr);
}

@media (max-width: 760px) {
  .activity-filters {
    align-items: stretch;
    flex-direction: column;
  }

  .activity-filter-field,
  .activity-filter-field--status {
    width: 100%;
  }

  .activity-filter-actions {
    margin-left: 0;
  }

  .activity-form__status {
    max-width: none;
  }

  .activity-form__grid,
  .ticket-type-grid--primary,
  .ticket-type-grid--rules {
    grid-template-columns: 1fr;
  }
}
</style>
