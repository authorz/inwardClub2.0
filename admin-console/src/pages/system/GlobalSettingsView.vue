<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import {
  NButton,
  NDynamicTags,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NSpin,
  NSwitch,
  NText,
} from 'naive-ui'
import PageHeader from '@/components/PageHeader.vue'
import { systemService } from '@/api/services'
import { toastError, toastSuccess } from '@/utils/feedback'

const loading = ref(false)
const saving = ref(false)
const defaultRechargeNotice = '新用户首充积分赠送双倍，充值一千及以上都赠送双倍积分，不与新用户首充赠送双倍同享。'
const form = reactive({
  firstRechargeDoublePointsEnabled: false,
  rechargeDoublePointsThresholdAmount: 1000,
  rechargeNotice: defaultRechargeNotice,
  franchiseInquirySources: [] as string[],
  franchiseHotline: '',
  phoneChangeIntervalDays: 30,
})

onMounted(load)

async function load(): Promise<void> {
  loading.value = true
  try {
    const settings = await systemService.getGlobalSettings()
    form.firstRechargeDoublePointsEnabled = Boolean(
      settings.firstRechargeDoublePointsEnabled,
    )
    form.rechargeDoublePointsThresholdAmount =
      settings.rechargeDoublePointsThresholdAmount || 1000
    form.rechargeNotice = settings.rechargeNotice ?? defaultRechargeNotice
    form.franchiseInquirySources = settings.franchiseInquirySources || []
    form.franchiseHotline = settings.franchiseHotline || ''
    form.phoneChangeIntervalDays = settings.phoneChangeIntervalDays || 30
  } catch (e) {
    toastError((e as { message?: string }).message ?? '读取全局设置失败')
  } finally {
    loading.value = false
  }
}

async function save(): Promise<void> {
  if (
    !Number.isInteger(form.rechargeDoublePointsThresholdAmount)
    || form.rechargeDoublePointsThresholdAmount <= 0
  ) {
    return toastError('满额双倍积分门槛必须是大于 0 的整数')
  }
  if (
    !Number.isInteger(form.phoneChangeIntervalDays)
    || form.phoneChangeIntervalDays < 1
    || form.phoneChangeIntervalDays > 3650
  ) {
    return toastError('手机号更换间隔必须是 1 到 3650 天的整数')
  }
  const sources = form.franchiseInquirySources.map((item) => item.trim()).filter(Boolean)
  if (!sources.length) return toastError('加盟咨询信息渠道至少保留一项')
  const rechargeNotice = form.rechargeNotice.trim()
  if (Array.from(rechargeNotice).length > 200) {
    return toastError('充值弹窗提示不能超过 200 个字')
  }
  const hotline = form.franchiseHotline.trim()
  if (hotline && (hotline.length < 6 || hotline.length > 32)) {
    return toastError('请输入有效的加盟热线')
  }
  saving.value = true
  try {
    const settings = await systemService.updateGlobalSettings({
      firstRechargeDoublePointsEnabled: form.firstRechargeDoublePointsEnabled,
      rechargeDoublePointsThresholdAmount: form.rechargeDoublePointsThresholdAmount,
      rechargeNotice,
      franchiseInquirySources: sources,
      franchiseHotline: hotline,
      phoneChangeIntervalDays: form.phoneChangeIntervalDays,
    })
    form.firstRechargeDoublePointsEnabled = Boolean(
      settings.firstRechargeDoublePointsEnabled,
    )
    form.rechargeDoublePointsThresholdAmount = settings.rechargeDoublePointsThresholdAmount
    form.rechargeNotice = settings.rechargeNotice ?? ''
    form.franchiseInquirySources = settings.franchiseInquirySources || []
    form.franchiseHotline = settings.franchiseHotline || ''
    form.phoneChangeIntervalDays = settings.phoneChangeIntervalDays || 30
    toastSuccess('全局设置已保存')
  } catch (e) {
    toastError((e as { message?: string }).message ?? '保存全局设置失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <section>
    <PageHeader
      title="全局设置"
      description="配置全局展示、充值奖励与加盟咨询信息"
      :breadcrumb="['系统设置', '全局设置']"
    />

    <div class="settings-panel">
      <NSpin :show="loading">
        <NForm
          label-placement="left"
          label-width="160"
        >
          <NFormItem label="新用户首充奖励">
            <div class="reward-field">
              <NSwitch v-model:value="form.firstRechargeDoublePointsEnabled">
                <template #checked>
                  已开启
                </template>
                <template #unchecked>
                  已关闭
                </template>
              </NSwitch>
              <NText depth="3">
                开启后，会员全平台首次成功充值按到账金币数赠送 2 倍积分。
                首次充值达到下方门槛时，不再享受首充奖励。
              </NText>
            </div>
          </NFormItem>
          <NFormItem label="满额双倍积分门槛">
            <div class="threshold-field">
              <NInputNumber
                v-model:value="form.rechargeDoublePointsThresholdAmount"
                :min="1"
                :precision="0"
              >
                <template #suffix>
                  元
                </template>
              </NInputNumber>
              <NText depth="3">
                所有单次充值达到此金额时赠送 2 倍积分；若同时属于首次充值，
                只发放满额奖励，不再叠加首充奖励。
              </NText>
            </div>
          </NFormItem>
          <NFormItem label="充值弹窗提示">
            <div class="notice-field">
              <NInput
                v-model:value="form.rechargeNotice"
                type="textarea"
                :autosize="{ minRows: 3, maxRows: 5 }"
                maxlength="200"
                show-count
                clearable
                placeholder="留空则不在小程序充值弹窗中显示提示"
              />
              <NText depth="3">
                文案会随快捷充值列表接口返回，并显示在个人中心充值弹窗中；调整奖励门槛后请同步更新。
              </NText>
            </div>
          </NFormItem>
          <NFormItem label="加盟咨询信息渠道">
            <div class="source-field">
              <NDynamicTags v-model:value="form.franchiseInquirySources" />
              <NText depth="3">
                点击“+”新增渠道，点击标签右侧可删除；小程序加盟咨询表单会按当前顺序展示。
              </NText>
            </div>
          </NFormItem>
          <NFormItem label="全国加盟热线">
            <div class="hotline-field">
              <NInput
                v-model:value="form.franchiseHotline"
                clearable
                maxlength="32"
                placeholder="请输入加盟联系电话"
              />
              <NText depth="3">
                将显示在小程序加盟咨询页面顶部，用户点击后可复制号码。
              </NText>
            </div>
          </NFormItem>
          <NFormItem label="手机号更换间隔">
            <div class="threshold-field">
              <NInputNumber
                v-model:value="form.phoneChangeIntervalDays"
                :min="1"
                :max="3650"
                :precision="0"
              >
                <template #suffix>
                  天
                </template>
              </NInputNumber>
              <NText depth="3">
                会员更换绑定手机号后，需要等待此间隔才能再次更换；首次绑定不受限制。
              </NText>
            </div>
          </NFormItem>
          <NFormItem label=" ">
            <NButton
              type="primary"
              :loading="saving"
              @click="save"
            >
              保存配置
            </NButton>
          </NFormItem>
        </NForm>
      </NSpin>
    </div>
  </section>
</template>

<style scoped>
.settings-panel {
  max-width: 820px;
  padding: var(--ic-space-lg);
  background: var(--ic-color-surface);
  border: 1px solid var(--ic-color-border);
  border-radius: var(--ic-radius-md);
}

.reward-field {
  display: flex;
  flex-direction: column;
  gap: var(--ic-space-sm);
}

.threshold-field {
  display: flex;
  width: min(100%, 360px);
  flex-direction: column;
  gap: var(--ic-space-sm);
}

.source-field {
  display: flex;
  width: min(100%, 560px);
  flex-direction: column;
  gap: var(--ic-space-sm);
}

.notice-field {
  display: flex;
  width: min(100%, 560px);
  flex-direction: column;
  gap: var(--ic-space-sm);
}

.hotline-field {
  display: flex;
  width: min(100%, 360px);
  flex-direction: column;
  gap: var(--ic-space-sm);
}

.time-field {
  display: grid;
  width: min(100%, 560px);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--ic-space-md);
}

.time-field__item {
  display: flex;
  flex-direction: column;
  gap: var(--ic-space-xs);
}

@media (max-width: 720px) {
  .time-field {
    grid-template-columns: 1fr;
  }
}
</style>
