<script setup lang="ts">
/**
 * 积分审核：审核会员存积分申请（通过/驳回）。
 * 审核为高风险写操作，服务端带 Idempotency-Key 与审计。
 */
import { computed, h, ref } from 'vue'
import { NAlert, NAvatar, NSpin, type DataTableColumns } from 'naive-ui'
import { pointSavingService } from '@/api/services'
import { ApiError } from '@/api/error'
import { useAsyncList } from '@/composables/useAsyncList'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { REVIEW_STATUS, toOptions } from '@/constants/enums'
import { PERM } from '@/constants/permissions'
import { dateColumn, phoneColumn, statusColumn, textColumn } from '@/utils/columns'
import { feedback } from '@/utils/feedback'
import {
  DataTable,
  PageHeader,
  PermissionButton,
  ReviewDialog,
  StatusFilterBar,
} from '@/components/common'
import type { PointSavingRequest } from '@/types/models'

const list = useAsyncList<PointSavingRequest>((params) => pointSavingService.list(params), {
  initialFilters: { status: 'pending', keyword: '' },
})
const action = useAsyncAction()

const dialogShow = ref(false)
const activeRow = ref<PointSavingRequest | null>(null)
const detailLoading = ref(false)
const detailError = ref('')
let detailRequestVersion = 0

async function openReview(row: PointSavingRequest) {
  activeRow.value = row
  dialogShow.value = true
  detailError.value = ''
  detailLoading.value = true
  const version = ++detailRequestVersion
  try {
    const detail = await pointSavingService.detail(row.id)
    if (version === detailRequestVersion) activeRow.value = detail
  } catch (error) {
    if (version === detailRequestVersion) {
      detailError.value = error instanceof ApiError ? error.message : '积分计算规则加载失败'
    }
  } finally {
    if (version === detailRequestVersion) detailLoading.value = false
  }
}

function submit(decision: 'approved' | 'rejected', reason: string) {
  const row = activeRow.value
  if (!row) return
  if (decision === 'approved' && detailError.value) {
    feedback.message.warning('积分计算规则尚未加载，暂不能通过审核')
    return
  }
  void action.run(() => pointSavingService.review(row.id, decision, reason), {
    successMessage: decision === 'approved' ? '已通过' : '已驳回',
    onSuccess: () => {
      dialogShow.value = false
      list.refresh()
    },
  })
}

function initial(value?: string): string {
  return value?.trim().slice(0, 1).toUpperCase() || '会'
}

function memberCell(row: PointSavingRequest) {
  const name = row.memberName || `用户 ${row.memberId}`
  const fallback = () => initial(name)
  return h('div', { class: 'point-review__identity' }, [
    h(
      NAvatar,
      { round: true, size: 40, src: row.memberAvatarUrl || undefined, objectFit: 'cover' },
      row.memberAvatarUrl ? { fallback } : { default: fallback },
    ),
    h('div', { class: 'point-review__identity-copy' }, [
      h('strong', name),
      h('span', row.memberName ? `用户 ID：${row.memberId}` : `用户 ID：${row.memberId} · 资料已不存在`),
    ]),
  ])
}

function reviewerCell(row: PointSavingRequest) {
  const reviewer = row.reviewer
  if (!reviewer) {
    const text = row.status === 'pending'
      ? '待审核'
      : `审核者 ${row.reviewedBy ? `ID ${row.reviewedBy}` : '资料未保存'}`
    return h('span', { class: 'ic-muted' }, text)
  }

  if (reviewer.type === 'staff') {
    const name = reviewer.nickname || reviewer.staffName || '工作人员'
    return h('div', { class: 'point-review__identity' }, [
      h(
        NAvatar,
        { round: true, size: 40, src: reviewer.avatarUrl || undefined },
        { default: () => initial(name) },
      ),
      h('div', { class: 'point-review__identity-copy' }, [
        h('strong', name),
        h('span', `工作人员 · ${reviewer.phone || '未登记手机号'}`),
      ]),
    ])
  }

  const name = reviewer.displayName || reviewer.username || '后台管理员'
  const role = reviewer.type === 'cashier' ? '收银员' : '门店管理员'
  return h('div', { class: 'point-review__identity' }, [
    h(NAvatar, { round: true, size: 40 }, { default: () => initial(name) }),
    h('div', { class: 'point-review__identity-copy' }, [
      h('strong', name),
      h('span', `${role}账号：${reviewer.username || '未保存'}`),
    ]),
  ])
}

const columns = computed<DataTableColumns<PointSavingRequest>>(() => [
  {
    title: '会员',
    key: 'member',
    width: 230,
    render: memberCell,
  },
  phoneColumn<PointSavingRequest>('手机号', (r) => r.phone),
  textColumn<PointSavingRequest>('存入积分', (r) => r.points, { align: 'right' }),
  statusColumn<PointSavingRequest>('状态', REVIEW_STATUS, (r) => r.status, { width: 100 }),
  {
    title: '审核人',
    key: 'reviewer',
    width: 260,
    render: reviewerCell,
  },
  dateColumn<PointSavingRequest>('提交时间', (r) => r.submittedAt ?? r.createdAt, { width: 150 }),
  dateColumn<PointSavingRequest>('审核时间', (r) => r.reviewedAt, { width: 150 }),
  {
    title: '操作',
    key: 'actions',
    width: 100,
    fixed: 'right',
    render: (row: PointSavingRequest) =>
      h(
        PermissionButton,
        {
          permissions: [PERM.pointReview],
          type: 'primary',
          text: true,
          disabled: row.status !== 'pending',
          onClick: () => void openReview(row),
        },
        { default: () => '审核' },
      ),
  },
])
</script>

<template>
  <div>
    <PageHeader
      title="积分审核"
      description="审核会员存积分申请"
    />

    <StatusFilterBar
      :status-options="toOptions(REVIEW_STATUS)"
      :status="(list.filters.status as string) ?? null"
      :keyword="(list.filters.keyword as string) ?? ''"
      :loading="list.loading.value"
      search-placeholder="搜索会员"
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
      empty-text="暂无待审核申请"
      @update:page="list.setPage"
      @update:page-size="list.setPageSize"
    />

    <ReviewDialog
      v-model:show="dialogShow"
      title="积分存入审核"
      :loading="action.running.value"
      :width="580"
      :approve-disabled="detailLoading || Boolean(detailError)"
      @approve="submit('approved', $event)"
      @reject="submit('rejected', $event)"
    >
      <NSpin :show="detailLoading">
        <div
          v-if="activeRow"
          class="review-info"
        >
          <div class="review-member">
            <NAvatar
              v-if="activeRow.memberAvatarUrl"
              round
              :size="56"
              :src="activeRow.memberAvatarUrl"
              object-fit="cover"
            >
              <template #fallback>
                {{ initial(activeRow.memberName) }}
              </template>
            </NAvatar>
            <NAvatar
              v-else
              round
              :size="56"
            >
              {{ initial(activeRow.memberName) }}
            </NAvatar>
            <div class="review-member__copy">
              <strong>{{ activeRow.memberName || `用户 ${activeRow.memberId}` }}</strong>
              <span>手机号：{{ activeRow.phone || '未登记手机号' }}</span>
            </div>
          </div>

          <NAlert
            v-if="detailError"
            type="error"
            :show-icon="false"
          >
            {{ detailError }}，请关闭弹窗后重试。
          </NAlert>

          <dl class="review-points">
            <div>
              <dt>存入积分</dt>
              <dd>{{ activeRow.points }}</dd>
            </div>
            <div>
              <dt>实际获得积分</dt>
              <dd>{{ activeRow.awardedPoints ?? '—' }}</dd>
            </div>
          </dl>

          <section class="review-calculation">
            <h3>积分计算规则</h3>
            <p>{{ activeRow.calculationDescription || '正在读取当前积分计算规则…' }}</p>
            <span v-if="activeRow.pointsDivisor">
              规则版本 v{{ activeRow.ruleVersion }} · 标准换算为每 {{ activeRow.pointsDivisor }} 存入积分获得 1 积分，除法结果向下取整。审核通过时服务端会按同一规则重新校验。
            </span>
          </section>
        </div>
      </NSpin>
    </ReviewDialog>
  </div>
</template>

<style scoped>
.review-info {
  display: flex;
  flex-direction: column;
  gap: var(--ic-space-4);
  font-size: var(--ic-font-sm);
}
.review-member {
  display: flex;
  gap: var(--ic-space-3);
  align-items: center;
}
.review-member__copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.review-member__copy strong {
  overflow: hidden;
  font-size: var(--ic-font-md);
  text-overflow: ellipsis;
  white-space: nowrap;
}
.review-member__copy span,
.review-calculation span {
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-xs);
}
.review-points {
  margin: 0;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  border-block: 1px solid var(--ic-color-border);
}
.review-points > div {
  padding: var(--ic-space-3) 0;
}
.review-points > div + div {
  padding-left: var(--ic-space-4);
  border-left: 1px solid var(--ic-color-border);
}
.review-points dt {
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-xs);
}
.review-points dd {
  margin: 4px 0 0;
  font-size: var(--ic-font-xl);
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}
.review-calculation h3 {
  margin: 0;
  font-size: var(--ic-font-sm);
  font-weight: 650;
}
.review-calculation p {
  margin: 8px 0 6px;
  line-height: 1.65;
}
.review-calculation span {
  display: block;
  line-height: 1.6;
}
:global(.point-review__identity) {
  display: flex;
  align-items: center;
  gap: var(--ic-space-3);
  min-width: 0;
}
:global(.point-review__identity-copy) {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  line-height: 1.4;
}
:global(.point-review__identity-copy strong) {
  display: block;
  overflow: hidden;
  color: var(--ic-color-text);
  font-size: var(--ic-font-base);
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}
:global(.point-review__identity-copy span) {
  display: block;
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-xs);
  white-space: nowrap;
}
</style>
