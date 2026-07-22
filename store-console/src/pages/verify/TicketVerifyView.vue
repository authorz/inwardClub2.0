<script setup lang="ts">
/**
 * 票券核销：扫码/输入券号核销本店券。
 * 核销为高风险写操作，服务端带 Idempotency-Key。
 *
 * 说明：服务端无「待核销券实例」列表读端点（GET /store/coupon-entitlements 不存在），
 * 故待核销列表退化为空状态，仅保留扫码即核销的动作。
 */
import { ref } from 'vue'
import { verificationService } from '@/api/services'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { PERM } from '@/constants/permissions'
import { EmptyState, PageHeader, PermissionButton, VerifyDialog } from '@/components/common'

const action = useAsyncAction()
const dialogShow = ref(false)

function openScan() {
  dialogShow.value = true
}

function onConfirm(code: string) {
  void action.run(() => verificationService.verifyCoupon({ entitlementNo: code }), {
    successMessage: '券核销成功',
    onSuccess: () => {
      dialogShow.value = false
    },
  })
}
</script>

<template>
  <div>
    <PageHeader
      title="票券核销"
      description="扫码或输入券号核销本店券"
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

    <EmptyState description="暂无待核销券（券实例列表功能待后端支持），请使用扫码核销" />

    <VerifyDialog
      v-model:show="dialogShow"
      title="券核销"
      placeholder="扫描或输入券号"
      :loading="action.running.value"
      @confirm="onConfirm"
    />
  </div>
</template>
