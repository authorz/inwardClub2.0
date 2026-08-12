<script setup lang="ts">
/**
 * 活动核销：扫码/输入核销码核销活动票。
 * 核销为高风险写操作，服务端带 Idempotency-Key。
 *
 * 说明：服务端无「待核销活动票」列表读端点（GET /store/tickets 不存在，仅 /store/tickets/verify），
 * 故待核销列表退化为空状态，仅保留扫码即核销的动作。
 */
import { ref } from 'vue'
import { verificationService } from '@/api/services'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { PERM } from '@/constants/permissions'
import { EmptyState, PageHeader, PermissionButton, VerifyDialog } from '@/components/common'
import { feedback } from '@/utils/feedback'

const action = useAsyncAction()
const dialogShow = ref(false)

function openScan() {
  dialogShow.value = true
}

function onConfirm(code: string) {
  if (!/^\d{6}$/.test(code.trim())) {
    feedback.message.warning('请输入6位数字核销码')
    return
  }
  void action.run(() => verificationService.verifyTicket(code), {
    successMessage: '核销成功',
    onSuccess: () => {
      dialogShow.value = false
    },
  })
}
</script>

<template>
  <div>
    <PageHeader
      title="活动核销"
      description="扫码或输入核销码核销本店活动票"
    >
      <template #actions>
        <PermissionButton
          :permissions="[PERM.ticketVerify]"
          type="primary"
          @click="openScan"
        >
          扫码核销
        </PermissionButton>
      </template>
    </PageHeader>

    <EmptyState description="暂无待核销活动票（活动票列表功能待后端支持），请使用扫码核销" />

    <VerifyDialog
      v-model:show="dialogShow"
      title="活动票核销"
      placeholder="扫描或输入6位数字核销码"
      :loading="action.running.value"
      @confirm="onConfirm"
    >
      <p class="ic-muted">
        请核对会员与活动信息后核销。
      </p>
    </VerifyDialog>
  </div>
</template>
