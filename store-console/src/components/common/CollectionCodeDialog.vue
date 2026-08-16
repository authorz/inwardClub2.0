<script setup lang="ts">
/**
 * 收款码弹窗：将微信 Native code_url 渲染为二维码，并展示金额、状态、
 * 有效期倒计时与匹配到的掩码会员。
 *
 * 二维码内容由后端向微信支付下单后返回；前端只将本次 code_url 转为二维码，
 * 不生成无金额的静态通用码。
 * 会员手机号仅用于匹配，展示为掩码，前端不留存原始号码。
 */
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { NButton, NModal, NResult, NSpin, NTag } from 'naive-ui'
import QRCode from 'qrcode'
import { formatCent, formatCountdown } from '@/utils/format'
import { COLLECTION_ORDER_STATUS, PAY_CHANNEL, resolveEnum } from '@/constants/enums'
import type { CollectionOrder } from '@/types/models'

const props = withDefaults(
  defineProps<{
    show: boolean
    order?: CollectionOrder | null
    loading?: boolean
    polling?: boolean
  }>(),
  { order: null, loading: false, polling: false },
)

const emit = defineEmits<{
  'update:show': [value: boolean]
  cancel: [id: string | number]
  expired: []
}>()

const remaining = ref(0)
const qrDataUrl = ref('')
const qrError = ref(false)
let timer: ReturnType<typeof setInterval> | null = null
let qrRenderVersion = 0

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
  [() => props.show, () => props.order?.expiresAt, () => props.order?.status],
  ([open, , status]) => {
    stop()
    if (open && props.order && status === 'pending') {
      tick()
      timer = setInterval(tick, 1000)
    }
  },
)

watch(
  [() => props.show, () => props.order?.qrContent, () => props.order?.status],
  async ([open, content, status]) => {
    const version = ++qrRenderVersion
    qrDataUrl.value = ''
    qrError.value = false
    if (!open || status !== 'pending') return
    if (!content || !content.startsWith('weixin://wxpay/')) {
      qrError.value = true
      return
    }
    try {
      const dataUrl = await QRCode.toDataURL(content, {
        width: 240,
        margin: 1,
        errorCorrectionLevel: 'M',
        color: { dark: '#111111', light: '#ffffff' },
      })
      if (version === qrRenderVersion) qrDataUrl.value = dataUrl
    } catch {
      if (version === qrRenderVersion) qrError.value = true
    }
  },
  { immediate: true },
)

onBeforeUnmount(stop)

const channelLabels = computed(() => PAY_CHANNEL.wechat.label)

const statusMeta = computed(() => resolveEnum(COLLECTION_ORDER_STATUS, props.order?.status))
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    title="微信收款码"
    style="width: 460px"
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

        <NTag
          :type="statusMeta.tone === 'default' ? 'default' : statusMeta.tone"
          :bordered="false"
          size="small"
        >
          {{ statusMeta.label }}
        </NTag>

        <div
          v-if="order.status === 'pending'"
          class="collect__qr"
        >
          <img
            v-if="qrDataUrl"
            :src="qrDataUrl"
            alt="微信收款二维码"
          >
          <NSpin
            v-else-if="!qrError"
            size="large"
          />
          <NResult
            v-else
            status="error"
            title="收款码生成失败"
            description="请关闭弹窗后重新生成"
          />
        </div>

        <NResult
          v-else-if="order.status === 'paid'"
          status="success"
          title="收款成功"
          :description="`已收到 ${formatCent(order.amountCent)}`"
        />
        <NResult
          v-else
          :status="order.status === 'expired' ? 'warning' : 'info'"
          :title="statusMeta.label"
          description="此收款码已停止使用"
        />

        <div
          v-if="order.status === 'pending'"
          class="collect__meta"
        >
          <span>请使用{{ channelLabels }}扫一扫</span>
          <span
            v-if="order.expiresAt"
            class="ic-mono"
          >剩余 {{ formatCountdown(remaining) }}</span>
        </div>
        <div
          v-if="order.status === 'pending'"
          class="collect__polling"
        >
          <span class="collect__polling-dot" />
          {{ polling ? '正在确认支付结果' : '等待顾客支付' }}
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
  width: 256px;
  height: 256px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #fff;
  border: 1px solid var(--ic-color-border);
  padding: 8px;
}
.collect__qr img {
  width: 240px;
  height: 240px;
  object-fit: contain;
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
.collect__polling {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--ic-color-text-tertiary);
  font-size: var(--ic-font-xs);
}
.collect__polling-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--ic-color-success);
  animation: collect-pulse 1.5s ease-in-out infinite;
}
@keyframes collect-pulse {
  0%, 100% { opacity: 0.35; }
  50% { opacity: 1; }
}
.collect__footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--ic-space-2);
}
</style>
