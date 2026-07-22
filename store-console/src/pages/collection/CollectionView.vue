<script setup lang="ts">
/**
 * 线下聚合收款：创建固定金额动态收款码（支持微信 / 支付宝渠道，属门店收单能力）。
 *
 * 前端不传 storeId（门店范围来自 token scope）；memberPhone 仅用于本次匹配，
 * 服务端返回掩码会员信息。创建为高风险写操作，服务端带 Idempotency-Key。
 */
import { reactive, ref } from 'vue'
import { NButton, NForm, NFormItem, NInput, NInputNumber, NSelect } from 'naive-ui'
import { collectionService, type CreateCollectionPayload } from '@/api/services/collection'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { COLLECTION_PAY_CHANNELS, PAY_CHANNEL } from '@/constants/enums'
import { yuanToCent } from '@/utils/format'
import { feedback } from '@/utils/feedback'
import { CollectionCodeDialog, PageHeader } from '@/components/common'
import type { CollectionOrder } from '@/types/models'

const action = useAsyncAction()

const form = reactive({
  amountYuan: 0,
  subject: '',
  businessType: 'general',
  expiresInMinutes: 5,
  memberPhone: '',
})

const businessTypeOptions = [
  { label: '通用收款', value: 'general' },
  { label: '餐饮', value: 'food' },
  { label: '活动', value: 'activity' },
  { label: '充值', value: 'recharge' },
]

const dialogShow = ref(false)
const currentOrder = ref<CollectionOrder | null>(null)

function submit() {
  if (form.amountYuan <= 0) {
    feedback.message.warning('请输入收款金额')
    return
  }
  if (!form.subject.trim()) {
    feedback.message.warning('请输入收款事由')
    return
  }
  const payload: CreateCollectionPayload = {
    amountCent: yuanToCent(form.amountYuan),
    subject: form.subject.trim(),
    businessType: form.businessType,
    expiresInSeconds: form.expiresInMinutes * 60,
  }
  if (form.memberPhone.trim()) payload.memberPhone = form.memberPhone.trim()

  void action.run(() => collectionService.create(payload), {
    successMessage: '收款码已创建',
    onSuccess: (order) => {
      currentOrder.value = order
      dialogShow.value = true
      // 清空手机号，前端不留存原始号码。
      form.memberPhone = ''
    },
  })
}

function onCancelOrder(id: string | number) {
  void action.run(() => collectionService.cancel(id), {
    confirm: { content: '确认取消该收款码？', danger: true },
    successMessage: '已取消收款',
    onSuccess: () => {
      dialogShow.value = false
    },
  })
}

const channelLabels = COLLECTION_PAY_CHANNELS.map((c) => PAY_CHANNEL[c].label).join(' / ')
</script>

<template>
  <div>
    <PageHeader
      title="线下聚合收款"
      description="创建固定金额动态收款码，支持微信 / 支付宝收单"
    />

    <div class="collect-form ic-band">
      <NForm label-placement="top">
        <NFormItem
          label="收款金额（元）"
          required
        >
          <NInputNumber
            v-model:value="form.amountYuan"
            :min="0"
            :precision="2"
            style="width: 100%"
          />
        </NFormItem>
        <NFormItem
          label="收款事由"
          required
        >
          <NInput
            v-model:value="form.subject"
            placeholder="如：桌台 A3 结账"
            maxlength="60"
          />
        </NFormItem>
        <NFormItem label="业务类型">
          <NSelect
            v-model:value="form.businessType"
            :options="businessTypeOptions"
          />
        </NFormItem>
        <NFormItem label="有效期（分钟）">
          <NInputNumber
            v-model:value="form.expiresInMinutes"
            :min="1"
            :max="60"
            :precision="0"
            style="width: 100%"
          />
        </NFormItem>
        <NFormItem label="会员手机号（可选，仅用于匹配会员）">
          <NInput
            v-model:value="form.memberPhone"
            placeholder="用于关联会员权益，服务端返回掩码"
            maxlength="11"
          />
        </NFormItem>

        <p class="collect-form__hint ic-muted">
          支持渠道：{{ channelLabels }}。仅创建固定金额码，不生成无金额通用静态码。
        </p>

        <NButton
          type="primary"
          size="large"
          block
          :loading="action.running.value"
          @click="submit"
        >
          生成收款码
        </NButton>
      </NForm>
    </div>

    <CollectionCodeDialog
      v-model:show="dialogShow"
      :order="currentOrder"
      :loading="action.running.value"
      @cancel="onCancelOrder"
    />
  </div>
</template>

<style scoped>
.collect-form {
  max-width: 480px;
  padding: var(--ic-space-5);
}
.collect-form__hint {
  font-size: var(--ic-font-xs);
  line-height: 1.6;
  margin: 0 0 var(--ic-space-4);
}
</style>
