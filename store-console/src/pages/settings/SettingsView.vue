<script setup lang="ts">
/**
 * 设置：门店资料维护与营业状态。
 * 门店 Logo 只提交 assetId（此处展示当前 Logo，上传组件待资产服务接入）。
 * 门店范围来自 token scope，页面不出现门店选择器。
 */
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { NButton, NForm, NFormItem, NInput, NInputNumber, NSpin, NSwitch, NTimePicker } from 'naive-ui'
import { profileService, type StoreProfile } from '@/api/services/profile'
import { lowSpendRuleService } from '@/api/services/lowSpendRule'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { ApiError } from '@/api/error'
import { useAuthStore } from '@/stores/auth'
import { AssetImage, AssetUpload, PageHeader } from '@/components/common'

const auth = useAuthStore()
const action = useAsyncAction()
const loading = ref(false)
const errorMsg = ref<string | null>(null)
const savedBusinessHours = ref('')
let statusTimer: number | undefined

const ruleForm = reactive({
  enabled: true,
  reservationCutoff: '20:00',
  consumptionCutoff: '20:30',
  minimumAmount: 88,
  rewardPoints: 2000,
})

const form = reactive<Partial<StoreProfile>>({
  name: '',
  address: '',
  phone: '',
  customerServiceQrAssetId: null,
  customerServiceQrUrl: '',
  businessHours: '',
  latitude: null,
  longitude: null,
  status: 'open',
  logoUrl: '',
})

async function load() {
  loading.value = true
  errorMsg.value = null
  try {
    const profile = await profileService.get()
    Object.assign(form, profile)
    savedBusinessHours.value = profile.businessHours ?? ''
  } catch (err) {
    errorMsg.value = err instanceof ApiError ? err.message : '门店资料加载失败'
    // 退化：至少展示 token scope 中的门店名。
    form.name = auth.store?.name ?? ''
  } finally {
    loading.value = false
  }
}

async function loadRule() {
  try {
    const rule = await lowSpendRuleService.get()
    Object.assign(ruleForm, {
      enabled: rule.enabled,
      reservationCutoff: rule.reservationCutoff || '20:00',
      consumptionCutoff: rule.consumptionCutoff || '20:30',
      minimumAmount: rule.minimumAmount || 88,
      rewardPoints: rule.rewardPoints || 2000,
    })
  } catch (err) {
    errorMsg.value = err instanceof ApiError ? err.message : '奖励规则加载失败'
  }
}

function saveProfile() {
  void action.run(
    () =>
      profileService.update({
        name: form.name,
        address: form.address,
        phone: form.phone,
        customerServiceQrAssetId: form.customerServiceQrAssetId
          ? Number(form.customerServiceQrAssetId)
          : null,
        businessHours: form.businessHours,
        latitude: form.latitude ?? null,
        longitude: form.longitude ?? null,
      }),
    { successMessage: '门店资料已保存', onSuccess: () => load() },
  )
}

function toggleStatus(open: boolean) {
  const status: 'open' | 'closed' = open ? 'open' : 'closed'
  void action.run(() => profileService.updateStatus(status), {
    confirm: {
      content: `${open ? '确认设置为营业中' : '确认设置为休息中'}？人工状态将在下一个营业时间边界自动恢复。`,
    },
    successMessage: '营业状态已更新',
    onSuccess: (profile) => Object.assign(form, profile),
  })
}

function restoreAutomaticStatus() {
  void action.run(() => profileService.updateStatus('auto'), {
    successMessage: '已恢复按营业时间自动判断',
    onSuccess: (profile) => Object.assign(form, profile),
  })
}

function formatBoundary(value?: string | null) {
  if (!value) return ''
  return new Intl.DateTimeFormat('zh-CN', {
    timeZone: 'Asia/Shanghai',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }).format(new Date(value))
}

const statusHint = computed(() => {
  if (form.statusMode === 'manual') {
    return `人工调整，至 ${formatBoundary(form.statusOverrideUntil)} 自动恢复`
  }
  return `按营业时间 ${savedBusinessHours.value || '未设置'} 自动判断`
})

async function refreshStatus() {
  if (document.visibilityState !== 'visible') return
  try {
    const profile = await profileService.get()
    savedBusinessHours.value = profile.businessHours ?? ''
    form.status = profile.status
    form.statusMode = profile.statusMode
    form.scheduledOpen = profile.scheduledOpen
    form.statusOverrideUntil = profile.statusOverrideUntil
  } catch {
    // 定时刷新失败不打断当前资料编辑，下一轮继续尝试。
  }
}

function handleVisibilityChange() {
  if (document.visibilityState === 'visible') void refreshStatus()
}

function saveRule() {
  if (!ruleForm.reservationCutoff || !ruleForm.consumptionCutoff) {
    errorMsg.value = '请选择完整的规则时间'
    return
  }
  if (ruleForm.reservationCutoff >= ruleForm.consumptionCutoff) {
    errorMsg.value = '低消截止时间必须晚于预约或候桌截止时间'
    return
  }
  if (!Number.isInteger(ruleForm.minimumAmount) || ruleForm.minimumAmount <= 0) {
    errorMsg.value = '低消金额必须是大于 0 的整数'
    return
  }
  if (!Number.isInteger(ruleForm.rewardPoints) || ruleForm.rewardPoints <= 0) {
    errorMsg.value = '赠送积分必须是大于 0 的整数'
    return
  }
  errorMsg.value = null
  void action.run(() => lowSpendRuleService.update({ ...ruleForm }), {
    successMessage: '预约低消奖励规则已保存',
    onSuccess: () => loadRule(),
  })
}

onMounted(() => {
  load()
  loadRule()
  statusTimer = window.setInterval(() => void refreshStatus(), 60_000)
  document.addEventListener('visibilitychange', handleVisibilityChange)
})

onUnmounted(() => {
  if (statusTimer !== undefined) window.clearInterval(statusTimer)
  document.removeEventListener('visibilitychange', handleVisibilityChange)
})
</script>

<template>
  <div>
    <PageHeader
      title="设置"
      description="门店资料与营业状态（门店范围固定，不可切换）"
    />

    <NSpin :show="loading">
      <section class="settings ic-band">
        <header class="settings__header">
          <div class="settings__logo">
            <AssetImage
              :url="form.logoUrl"
              :size="64"
            />
            <div>
              <div class="settings__store-name">
                {{ form.name || auth.store?.name || '当前门店' }}
              </div>
              <p class="ic-muted settings__logo-hint">
                维护门店展示资料、地址及客服联系方式
              </p>
            </div>
          </div>

          <div class="settings__status">
            <div>
              <div class="settings__status-label">
                营业状态
              </div>
              <div class="ic-muted settings__status-hint">
                {{ statusHint }}
                <NButton
                  v-if="form.statusMode === 'manual'"
                  text
                  size="tiny"
                  class="settings__status-auto"
                  :disabled="action.running.value"
                  @click="restoreAutomaticStatus"
                >
                  恢复自动
                </NButton>
              </div>
            </div>
            <NSwitch
              :value="form.status === 'open'"
              :loading="action.running.value"
              @update:value="toggleStatus"
            >
              <template #checked>
                营业中
              </template>
              <template #unchecked>
                休息中
              </template>
            </NSwitch>
          </div>
        </header>

        <NForm
          label-placement="top"
          class="settings__form"
        >
          <div class="settings__grid">
            <section class="settings__section">
              <div class="settings__section-head">
                <h2 class="settings__section-title">
                  基础资料
                </h2>
                <p class="ic-muted settings__section-desc">
                  用于小程序门店列表、导航和营业信息展示
                </p>
              </div>

              <div class="settings__fields">
                <NFormItem label="门店名称">
                  <NInput
                    v-model:value="form.name"
                    placeholder="门店名称"
                  />
                </NFormItem>
                <NFormItem label="营业时间">
                  <NInput
                    v-model:value="form.businessHours"
                    placeholder="如：15:00-02:00"
                  />
                </NFormItem>
                <NFormItem
                  label="门店地址"
                  class="settings__field--wide"
                >
                  <NInput
                    v-model:value="form.address"
                    placeholder="请输入完整门店地址"
                  />
                </NFormItem>
                <NFormItem label="纬度 latitude">
                  <NInputNumber
                    v-model:value="form.latitude"
                    :show-button="false"
                    :precision="6"
                    :min="-90"
                    :max="90"
                    placeholder="如：31.230416"
                    style="width: 100%"
                  />
                </NFormItem>
                <NFormItem label="经度 longitude">
                  <NInputNumber
                    v-model:value="form.longitude"
                    :show-button="false"
                    :precision="6"
                    :min="-180"
                    :max="180"
                    placeholder="如：121.473701"
                    style="width: 100%"
                  />
                </NFormItem>
              </div>
            </section>

            <section class="settings__section settings__contact">
              <div class="settings__section-head">
                <h2 class="settings__section-title">
                  客服联系方式
                </h2>
                <p class="ic-muted settings__section-desc">
                  用户可通过电话或微信二维码联系当前门店
                </p>
              </div>

              <NFormItem label="客服电话 / 联系方式">
                <NInput
                  v-model:value="form.phone"
                  placeholder="请输入客服电话或门店联系电话"
                />
              </NFormItem>
              <NFormItem label="客服微信二维码">
                <AssetUpload
                  v-model:asset-id="form.customerServiceQrAssetId"
                  v-model:preview-url="form.customerServiceQrUrl"
                  purpose="store_contact_qr"
                  :width="150"
                  :height="150"
                />
              </NFormItem>
            </section>
          </div>

          <div class="settings__actions">
            <p
              v-if="errorMsg"
              class="ic-muted settings__error"
            >
              {{ errorMsg }}（等待服务端接口就绪）
            </p>
            <NButton
              type="primary"
              :loading="action.running.value"
              @click="saveProfile"
            >
              保存资料
            </NButton>
          </div>
        </NForm>
      </section>
    </NSpin>

    <section class="reward-settings ic-band">
      <div class="reward-settings__head">
        <div>
          <h2 class="settings__section-title">
            预约低消奖励
          </h2>
          <p class="ic-muted settings__section-desc">
            按时预约座位或候桌，并在截止前完成微信或金币点餐低消后，自动标记已到店并发放积分
          </p>
        </div>
        <NSwitch v-model:value="ruleForm.enabled">
          <template #checked>
            已开启
          </template>
          <template #unchecked>
            已关闭
          </template>
        </NSwitch>
      </div>

      <NForm label-placement="top">
        <div class="reward-settings__grid">
          <NFormItem label="预约 / 候桌截止">
            <NTimePicker
              v-model:formatted-value="ruleForm.reservationCutoff"
              format="HH:mm"
              :clearable="false"
              style="width: 100%"
            />
          </NFormItem>
          <NFormItem label="完成低消截止">
            <NTimePicker
              v-model:formatted-value="ruleForm.consumptionCutoff"
              format="HH:mm"
              :clearable="false"
              style="width: 100%"
            />
          </NFormItem>
          <NFormItem label="累计低消金额">
            <NInputNumber
              v-model:value="ruleForm.minimumAmount"
              :min="1"
              :precision="0"
              style="width: 100%"
            >
              <template #suffix>
                元
              </template>
            </NInputNumber>
          </NFormItem>
          <NFormItem label="达标赠送积分">
            <NInputNumber
              v-model:value="ruleForm.rewardPoints"
              :min="1"
              :precision="0"
              style="width: 100%"
            >
              <template #suffix>
                积分
              </template>
            </NInputNumber>
          </NFormItem>
        </div>
        <div class="reward-settings__actions">
          <span class="ic-muted">同一会员在本门店每天最多获得一次</span>
          <NButton
            type="primary"
            :loading="action.running.value"
            @click="saveRule"
          >
            保存奖励规则
          </NButton>
        </div>
      </NForm>
    </section>
  </div>
</template>

<style scoped>
.settings {
  width: 100%;
  box-sizing: border-box;
  padding: var(--ic-space-5) var(--ic-space-6);
}
.settings__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--ic-space-4);
  padding-bottom: var(--ic-space-5);
  border-bottom: var(--ic-divider);
  margin-bottom: var(--ic-space-5);
}
.settings__logo,
.settings__status {
  display: flex;
  align-items: center;
}
.settings__logo {
  gap: var(--ic-space-4);
}
.settings__status {
  justify-content: flex-end;
  gap: var(--ic-space-4);
  text-align: right;
}
.settings__store-name {
  font-size: var(--ic-font-md);
  font-weight: 600;
}
.settings__logo-hint {
  font-size: var(--ic-font-xs);
  margin: 4px 0 0;
}
.settings__status-label {
  font-size: var(--ic-font-base);
  font-weight: 600;
}
.settings__status-hint {
  font-size: var(--ic-font-xs);
  margin-top: 2px;
}
.settings__status-auto {
  margin-left: var(--ic-space-2);
}
.settings__form {
  width: 100%;
}
.settings__grid {
  display: grid;
  grid-template-columns: minmax(0, 1.45fr) minmax(300px, 0.75fr);
  align-items: start;
  gap: var(--ic-space-7);
}
.settings__section-head {
  margin-bottom: var(--ic-space-4);
}
.settings__section-title {
  margin: 0;
  color: var(--ic-color-text);
  font-size: var(--ic-font-md);
  font-weight: 600;
}
.settings__section-desc {
  margin: 4px 0 0;
  font-size: var(--ic-font-xs);
}
.settings__fields {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  column-gap: var(--ic-space-4);
}
.settings__field--wide {
  grid-column: 1 / -1;
}
.settings__actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: var(--ic-space-3);
  margin-top: var(--ic-space-4);
  padding-top: var(--ic-space-4);
  border-top: var(--ic-divider);
}
.settings__error {
  font-size: var(--ic-font-xs);
  margin: 0 auto 0 0;
}
.reward-settings {
  margin-top: var(--ic-space-5);
  padding: var(--ic-space-5) var(--ic-space-6);
}
.reward-settings__head,
.reward-settings__actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--ic-space-4);
}
.reward-settings__head {
  padding-bottom: var(--ic-space-4);
  border-bottom: var(--ic-divider);
  margin-bottom: var(--ic-space-4);
}
.reward-settings__grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: var(--ic-space-4);
}
.reward-settings__actions {
  padding-top: var(--ic-space-3);
  border-top: var(--ic-divider);
  font-size: var(--ic-font-xs);
}

@media (max-width: 1080px) {
  .settings__grid {
    grid-template-columns: 1fr;
    gap: var(--ic-space-6);
  }
  .reward-settings__grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 680px) {
  .settings {
    padding: var(--ic-space-4);
  }
  .settings__header {
    align-items: flex-start;
    flex-direction: column;
  }
  .settings__status {
    width: 100%;
    justify-content: space-between;
    text-align: left;
  }
  .settings__fields {
    grid-template-columns: 1fr;
  }
  .settings__field--wide {
    grid-column: auto;
  }
  .settings__actions {
    align-items: stretch;
    flex-direction: column;
  }
  .settings__actions :deep(.n-button) {
    width: 100%;
  }
  .reward-settings {
    padding: var(--ic-space-4);
  }
  .reward-settings__head,
  .reward-settings__actions {
    align-items: stretch;
    flex-direction: column;
  }
  .reward-settings__grid {
    grid-template-columns: 1fr;
  }
}
</style>
