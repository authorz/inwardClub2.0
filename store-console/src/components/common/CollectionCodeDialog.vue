<script setup lang="ts">
/**
 * 收款码弹窗：展示动态聚合收款码、金额、有效期倒计时与匹配到的掩码会员。
 *
 * 二维码内容/展示 URL 由收单服务商通过后端返回；前端只展示，不生成静态通用码。
 * 会员手机号仅用于匹配，展示为掩码，前端不留存原始号码。
 */
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { NButton, NModal, NSpin } from 'naive-ui'
import { formatCent, formatCountdown } from '@/utils/format'
import { PAY_CHANNEL } from '@/constants/enums'
import type { CollectionOrder } from '@/types/models'

const props = withDefaults(
  defineProps<{
    show: boolean
    order?: CollectionOrder | null
    loading?: boolean
  }>(),
  { order: null, loading: false },
)

const emit = defineEmits<{
  'update:show': [value: boolean]
  cancel: [id: string | number]
  expired: []
}>()

const remaining = ref(0)
let timer: ReturnType<typeof setInterval> | null = null

function stop() {
  if (timer) {
    clearInterval(timer)
    timer = null
  }
}

function tick() {
  if (!props.order?.expiresAt) {
    remaining.value = 0
    return
  }
  const diff = Math.floor((new Date(props.order.expiresAt).getTime() - Date.now()) / 1000)
  remaining.value = Math.max(0, diff)
  if (remaining.value <= 0) {
    stop()
    emit('expired')
  }
}

watch(
  () => props.show,
  (open) => {
    stop()
    if (open && props.order) {
      tick()
      timer = setInterval(tick, 1000)
    }
  },
)

onBeforeUnmount(stop)

const channelLabels = computed(() =>
  PAY_CHANNEL.wechat.label + ' / ' + PAY_CHANNEL.alipay.label,
)
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    title="聚合收款码"
    style="width: 380px"
    @update:show="emit('update:show', $event)"
  >
    <NSpin :show="loading">
      <div
        v-if="order"
        class="collect"
      >
        <div class="collect__amount ic-mono">
          {{ formatCent(order.amountCent) }}
        </div>
        <div class="collect__subject ic-muted">
          {{ order.subject }}
        </div>

        <div class="collect__qr">
          <img
            v-if="order.qrDisplayUrl"
            :src="order.qrDisplayUrl"
            alt="收款码"
          >
          <div
            v-else
            class="collect__qr-placeholder ic-mono"
          >
            {{ order.qrContent || '等待收单服务商返回收款码' }}
          </div>
        </div>

        <div class="collect__meta">
          <span>支持渠道：{{ channelLabels }}</span>
          <span
            v-if="order.expiresAt"
            class="ic-mono"
          >剩余 {{ formatCountdown(remaining) }}</span>
        </div>
        <div
          v-if="order.memberNickname"
          class="collect__member ic-muted"
        >
          已匹配会员：{{ order.memberNickname }} · {{ order.memberPhoneMasked }}
        </div>
      </div>
    </NSpin>

    <template #footer>
      <div class="collect__footer">
        <NButton
          v-if="order && order.status === 'pending'"
          type="error"
          ghost
          @click="emit('cancel', order.id)"
        >
          取消收款
        </NButton>
        <NButton @click="emit('update:show', false)">
          关闭
        </NButton>
      </div>
    </template>
  </NModal>
</template>

<style scoped>
.collect {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--ic-space-2);
  padding: var(--ic-space-2) 0;
}
.collect__amount {
  font-size: var(--ic-font-xl);
  font-weight: 700;
}
.collect__subject {
  font-size: var(--ic-font-sm);
}
.collect__qr {
  margin: var(--ic-space-3) 0;
}
.collect__qr img {
  width: 200px;
  height: 200px;
  object-fit: contain;
}
.collect__qr-placeholder {
  width: 200px;
  height: 200px;
  display: flex;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: var(--ic-space-3);
  border: 1px dashed var(--ic-color-border-strong);
  border-radius: var(--ic-radius-md);
  color: var(--ic-color-text-tertiary);
  font-size: var(--ic-font-xs);
  word-break: break-all;
}
.collect__meta {
  display: flex;
  justify-content: space-between;
  width: 100%;
  font-size: var(--ic-font-xs);
  color: var(--ic-color-text-secondary);
}
.collect__member {
  font-size: var(--ic-font-xs);
}
.collect__footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--ic-space-2);
}
</style>
