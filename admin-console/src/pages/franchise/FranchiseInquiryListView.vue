<script setup lang="ts">
import { h, ref } from 'vue'
import { NAvatar } from 'naive-ui'
import ResourceListView from '@/components/ResourceListView.vue'
import PermissionButton from '@/components/PermissionButton.vue'
import { actionsColumn, dateTimeColumn, renderColumn, statusColumn, textColumn } from '@/utils/columns'
import { franchiseInquiryService } from '@/api/services'
import type { FranchiseInquiry } from '@/api/models'
import type { FilterField, ResourceListInstance } from '@/components/ui-types'
import { FRANCHISE_INQUIRY_STATUS_OPTIONS } from '@/constants/enums'
import { PERMISSIONS } from '@/constants/permissions'
import { toastError, toastSuccess } from '@/utils/feedback'

const listRef = ref<ResourceListInstance | null>(null)
const updatingId = ref<number | null>(null)

const fields: FilterField[] = [
  {
    key: 'keyword',
    label: '咨询人信息',
    type: 'input',
    placeholder: '支持称呼、电话、区域模糊搜索',
    width: 300,
  },
  { key: 'source', label: '信息渠道', type: 'input', placeholder: '请输入渠道名称' },
  {
    key: 'status',
    label: '处理状态',
    type: 'select',
    options: FRANCHISE_INQUIRY_STATUS_OPTIONS,
  },
]

const columns = [
  textColumn<FranchiseInquiry>('ID', 'id', { width: 80 }),
  renderColumn<FranchiseInquiry>(
    '用户信息',
    'member',
    (row) => {
      const nickname = row.memberNickname?.trim() || '未关联会员'
      const phone = row.memberPhone?.trim() || '暂无会员手机号'
      const fallback = () => nickname === '未关联会员' ? '未' : nickname.slice(0, 1)
      return h('div', { class: 'franchise-member' }, [
        h(
          NAvatar,
          {
            class: 'franchise-member__avatar',
            size: 36,
            round: true,
            src: row.memberAvatarUrl || undefined,
            objectFit: 'cover',
          },
          row.memberAvatarUrl ? { fallback } : { default: fallback },
        ),
        h('div', { class: 'franchise-member__details' }, [
          h('span', { class: 'franchise-member__nickname', title: nickname }, nickname),
          h('span', { class: 'franchise-member__phone' }, phone),
        ]),
      ])
    },
    210,
  ),
  textColumn<FranchiseInquiry>('称呼', 'contactName', { width: 150 }),
  textColumn<FranchiseInquiry>('联系电话', 'phone', { width: 170 }),
  textColumn<FranchiseInquiry>('预期开设区域', 'expectedRegion'),
  textColumn<FranchiseInquiry>('信息渠道', 'source', { width: 140 }),
  statusColumn<FranchiseInquiry>('处理状态', 'status', FRANCHISE_INQUIRY_STATUS_OPTIONS, 110),
  dateTimeColumn<FranchiseInquiry>('提交时间', 'createdAt'),
  actionsColumn<FranchiseInquiry>((row) =>
    h(
      PermissionButton,
      {
        permission: PERMISSIONS.STORE_WRITE,
        type: row.status === 'processed' ? 'default' : 'primary',
        disabled: updatingId.value === row.id,
        onClick: () => toggleStatus(row),
      },
      () => row.status === 'processed' ? '恢复未处理' : '标记已处理',
    ), 120),
]

async function toggleStatus(row: FranchiseInquiry): Promise<void> {
  const nextStatus = row.status === 'processed' ? 'unprocessed' : 'processed'
  updatingId.value = row.id
  try {
    await franchiseInquiryService.updateStatus(String(row.id), nextStatus)
    toastSuccess(nextStatus === 'processed' ? '已标记为已处理' : '已恢复为未处理')
    await listRef.value?.reload()
  } catch (error) {
    toastError((error as { message?: string }).message ?? '更新处理状态失败')
  } finally {
    updatingId.value = null
  }
}
</script>

<template>
  <ResourceListView
    ref="listRef"
    title="加盟咨询"
    description="查看小程序提交的全球招商加盟咨询"
    :breadcrumb="['加盟咨询']"
    :fields="fields"
    :columns="columns"
    :fetcher="franchiseInquiryService.list"
    empty-text="暂无加盟咨询"
  />
</template>

<style>
.franchise-member {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  padding: 2px 0;
}

.franchise-member__avatar {
  flex: none;
}

.franchise-member__details {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

.franchise-member__nickname {
  overflow: hidden;
  color: var(--ic-color-text);
  font-weight: 600;
  line-height: 20px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.franchise-member__phone {
  color: var(--ic-color-text-secondary);
  font-size: 12px;
  line-height: 18px;
  font-variant-numeric: tabular-nums;
}
</style>
