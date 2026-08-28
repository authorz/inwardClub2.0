<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import {
  NButton,
  NDynamicTags,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NSelect,
  NSpin,
  NSwitch,
  NText,
} from 'naive-ui'
import PageHeader from '@/components/PageHeader.vue'
import { couponCategoryService, systemService } from '@/api/services'
import type { CouponCategory } from '@/api/models'
import { toastError, toastSuccess } from '@/utils/feedback'

const loading = ref(false)
const saving = ref(false)
const couponCategories = ref<CouponCategory[]>([])
const defaultRechargeNotice = '新用户全平台首次成功充值低于 1000 元，按到账金币数赠送 2 倍积分。'
type GiftCouponUsageMode = 'unlimited' | 'limited'
interface GiftCouponUsageRuleForm {
  couponCategoryId: number | string | null
  mode: GiftCouponUsageMode
  dailyLimit: number
}
const giftCouponUsageModeOptions = [
  { label: '无限制', value: 'unlimited' },
  { label: '每日限用', value: 'limited' },
]
const form = reactive({
  firstRechargeDoublePointsEnabled: false,
  rechargeDoublePointsThresholdAmount: 1000,
  rechargeNotice: defaultRechargeNotice,
  franchiseInquirySources: [] as string[],
  franchiseHotline: '',
  phoneChangeIntervalDays: 30,
  printerDeveloperAccount: '',
  printerDeveloperKey: '',
  printerDeveloperKeyConfigured: false,
  printerApiUrl: 'https://open.xpyun.net/api/openapi/xprinter',
  giftCouponUsageRules: [] as GiftCouponUsageRuleForm[],
})

const couponCategoryOptions = computed(() => couponCategories.value.map((category) => ({
  label: category.name,
  value: category.id,
  disabled: category.status !== 'active',
})))

function categoryOptionsFor(index: number) {
  const selected = new Set(form.giftCouponUsageRules
    .filter((_, ruleIndex) => ruleIndex !== index)
    .map((rule) => String(rule.couponCategoryId)))
  return couponCategoryOptions.value.map((option) => ({
    ...option,
    disabled: option.disabled || selected.has(String(option.value)),
  }))
}

function addGiftCouponUsageRule(): void {
  const used = new Set(form.giftCouponUsageRules.map((rule) => String(rule.couponCategoryId)))
  const category = couponCategories.value.find(
    (item) => item.status === 'active' && !used.has(String(item.id)),
  )
  if (!category) return toastError('没有可继续配置的券种')
  form.giftCouponUsageRules.push({
    couponCategoryId: category.id,
    mode: 'unlimited',
    dailyLimit: 1,
  })
}

function removeGiftCouponUsageRule(index: number): void {
  form.giftCouponUsageRules.splice(index, 1)
}

onMounted(load)

async function load(): Promise<void> {
  loading.value = true
  try {
    const [settings, categories] = await Promise.all([
      systemService.getGlobalSettings(),
      couponCategoryService.list({ page: 1, pageSize: 100 }),
    ])
    couponCategories.value = categories.items
    form.firstRechargeDoublePointsEnabled = Boolean(
      settings.firstRechargeDoublePointsEnabled,
    )
    form.rechargeDoublePointsThresholdAmount =
      settings.rechargeDoublePointsThresholdAmount || 1000
    form.rechargeNotice = settings.rechargeNotice ?? defaultRechargeNotice
    form.franchiseInquirySources = settings.franchiseInquirySources || []
    form.franchiseHotline = settings.franchiseHotline || ''
    form.phoneChangeIntervalDays = settings.phoneChangeIntervalDays || 30
    form.printerDeveloperAccount = settings.printerDeveloperAccount || ''
    form.printerDeveloperKey = ''
    form.printerDeveloperKeyConfigured = Boolean(settings.printerDeveloperKeyConfigured)
    form.printerApiUrl = settings.printerApiUrl || 'https://open.xpyun.net/api/openapi/xprinter'
    form.giftCouponUsageRules = (settings.giftCouponUsageRules ?? []).map((rule) => ({
      couponCategoryId: rule.couponCategoryId,
      mode: rule.dailyLimit == null ? 'unlimited' : 'limited',
      dailyLimit: rule.dailyLimit ?? 1,
    }))
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
    return toastError('首充双倍积分上限必须是大于 0 的整数')
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
  const printerDeveloperAccount = form.printerDeveloperAccount.trim()
  const printerDeveloperKey = form.printerDeveloperKey.trim()
  const printerApiUrl = form.printerApiUrl.trim()
  if ((printerDeveloperAccount && !form.printerDeveloperKeyConfigured && !printerDeveloperKey)
    || (!printerDeveloperAccount && (form.printerDeveloperKeyConfigured || printerDeveloperKey))) {
    return toastError('打印机开发者账号和开发者密钥必须同时配置')
  }
  try {
    const parsed = new URL(printerApiUrl)
    if (!['https:', 'http:'].includes(parsed.protocol)) throw new Error('invalid protocol')
  } catch {
    return toastError('请输入有效的打印机接口 URL')
  }
  const configuredCategories = new Set<string>()
  for (const rule of form.giftCouponUsageRules) {
    if (rule.couponCategoryId == null) return toastError('请选择赠送券规则适用的券种')
    const categoryKey = String(rule.couponCategoryId)
    if (configuredCategories.has(categoryKey)) return toastError('同一券种不能重复配置')
    configuredCategories.add(categoryKey)
    if (rule.mode === 'limited'
      && (!Number.isInteger(rule.dailyLimit) || rule.dailyLimit < 1 || rule.dailyLimit > 999)) {
      return toastError('赠送券每日使用上限必须是 1 到 999 的整数')
    }
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
      printerDeveloperAccount,
      ...(printerDeveloperKey ? { printerDeveloperKey } : {}),
      printerApiUrl,
      giftCouponUsageRules: form.giftCouponUsageRules.map((rule) => ({
        couponCategoryId: rule.couponCategoryId!,
        dailyLimit: rule.mode === 'unlimited' ? null : rule.dailyLimit,
      })),
    })
    form.firstRechargeDoublePointsEnabled = Boolean(
      settings.firstRechargeDoublePointsEnabled,
    )
    form.rechargeDoublePointsThresholdAmount = settings.rechargeDoublePointsThresholdAmount
    form.rechargeNotice = settings.rechargeNotice ?? ''
    form.franchiseInquirySources = settings.franchiseInquirySources || []
    form.franchiseHotline = settings.franchiseHotline || ''
    form.phoneChangeIntervalDays = settings.phoneChangeIntervalDays || 30
    form.printerDeveloperAccount = settings.printerDeveloperAccount || ''
    form.printerDeveloperKey = ''
    form.printerDeveloperKeyConfigured = Boolean(settings.printerDeveloperKeyConfigured)
    form.printerApiUrl = settings.printerApiUrl || 'https://open.xpyun.net/api/openapi/xprinter'
    form.giftCouponUsageRules = (settings.giftCouponUsageRules ?? []).map((rule) => ({
      couponCategoryId: rule.couponCategoryId,
      mode: rule.dailyLimit == null ? 'unlimited' : 'limited',
      dailyLimit: rule.dailyLimit ?? 1,
    }))
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
      description="配置充值奖励、赠送券使用规则、加盟咨询与打印机开放平台账号"
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
                开启后，仅会员全平台首次成功充值且实付金额低于下方上限时，
                按到账金币数赠送 2 倍积分；后续充值不赠送。
              </NText>
            </div>
          </NFormItem>
          <NFormItem label="首充双倍积分上限">
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
                首次充值实付金额必须严格低于此金额才赠送 2 倍积分；
                等于或高于此金额均不赠送。
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
          <div class="settings-section-title">
            赠送券单日使用规则
          </div>
          <NFormItem label="赠送券使用限制">
            <div class="gift-coupon-rules">
              <NText depth="3">
                规则仅作用于 VIP、充值、人工发放等赠送来源。购买券商品获得的券始终不受限制；未配置的券种默认无限制。
              </NText>
              <div
                v-for="(rule, index) in form.giftCouponUsageRules"
                :key="`${rule.couponCategoryId}-${index}`"
                class="gift-coupon-rule"
              >
                <NSelect
                  v-model:value="rule.couponCategoryId"
                  :options="categoryOptionsFor(index)"
                  placeholder="选择券种"
                />
                <NSelect
                  v-model:value="rule.mode"
                  :options="giftCouponUsageModeOptions"
                />
                <NInputNumber
                  v-if="rule.mode === 'limited'"
                  v-model:value="rule.dailyLimit"
                  :min="1"
                  :max="999"
                  :precision="0"
                >
                  <template #suffix>
                    张/日
                  </template>
                </NInputNumber>
                <NText
                  v-else
                  depth="3"
                  class="unlimited-text"
                >
                  不限制每日张数
                </NText>
                <NButton
                  type="error"
                  text
                  @click="removeGiftCouponUsageRule(index)"
                >
                  移除
                </NButton>
              </div>
              <NButton
                secondary
                @click="addGiftCouponUsageRule"
              >
                添加券种规则
              </NButton>
            </div>
          </NFormItem>
          <div class="settings-section-title">
            芯烨云打印机开放平台
          </div>
          <NFormItem label="开发者账号">
            <div class="printer-field">
              <NInput
                v-model:value="form.printerDeveloperAccount"
                clearable
                maxlength="128"
                placeholder="请输入芯烨云开发者 ID"
              />
              <NText depth="3">
                所有门店共用此开发者账号；门店添加打印机时只需填写设备 SN。
              </NText>
            </div>
          </NFormItem>
          <NFormItem label="开发者密钥">
            <div class="printer-field">
              <NInput
                v-model:value="form.printerDeveloperKey"
                type="password"
                show-password-on="click"
                :input-props="{ name: 'printer-developer-key', autocomplete: 'new-password' }"
                :placeholder="form.printerDeveloperKeyConfigured ? '已配置，留空则不修改' : '请输入 UserKEY'"
              />
              <NText depth="3">
                密钥仅用于服务端签名，保存后不会回显。
              </NText>
            </div>
          </NFormItem>
          <NFormItem label="接口 URL">
            <div class="printer-field">
              <NInput
                v-model:value="form.printerApiUrl"
                clearable
                placeholder="https://open.xpyun.net/api/openapi/xprinter"
              />
              <NText depth="3">
                填写芯烨云 xprinter 接口根地址，不包含 addPrinters 等具体接口名。
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

.printer-field {
  display: flex;
  width: min(100%, 560px);
  flex-direction: column;
  gap: var(--ic-space-sm);
}

.gift-coupon-rules {
  display: flex;
  width: min(100%, 720px);
  flex-direction: column;
  gap: var(--ic-space-sm);
}

.gift-coupon-rule {
  display: grid;
  grid-template-columns: minmax(160px, 1fr) 120px minmax(150px, 180px) auto;
  align-items: center;
  gap: var(--ic-space-sm);
  padding: var(--ic-space-sm) 0;
  border-bottom: 1px solid var(--ic-color-border);
}

.unlimited-text {
  display: flex;
  align-items: center;
  min-height: 34px;
}

.settings-section-title {
  margin: var(--ic-space-lg) 0 var(--ic-space-md) 160px;
  font-size: var(--ic-font-md);
  font-weight: 600;
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

  .gift-coupon-rule {
    grid-template-columns: 1fr;
  }
}
</style>
