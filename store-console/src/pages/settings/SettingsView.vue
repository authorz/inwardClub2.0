<script setup lang="ts">
/**
 * 设置：门店资料维护与营业状态。
 * 门店 Logo 只提交 assetId（此处展示当前 Logo，上传组件待资产服务接入）。
 * 门店范围来自 token scope，页面不出现门店选择器。
 */
import { onMounted, reactive, ref } from 'vue'
import { NButton, NDivider, NForm, NFormItem, NInput, NInputNumber, NSpin, NSwitch } from 'naive-ui'
import { profileService, type StoreProfile } from '@/api/services/profile'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { ApiError } from '@/api/error'
import { useAuthStore } from '@/stores/auth'
import { AssetImage, PageHeader } from '@/components/common'

const auth = useAuthStore()
const action = useAsyncAction()
const settingsAction = useAsyncAction()
const loading = ref(false)
const errorMsg = ref<string | null>(null)
const settingsLoading = ref(false)
const settingsErrorMsg = ref<string | null>(null)
const settingsText = ref('{}')

const form = reactive<Partial<StoreProfile>>({
  name: '',
  address: '',
  phone: '',
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
  } catch (err) {
    errorMsg.value = err instanceof ApiError ? err.message : '门店资料加载失败'
    // 退化：至少展示 token scope 中的门店名。
    form.name = auth.store?.name ?? ''
  } finally {
    loading.value = false
  }
}

async function loadSettings() {
  settingsLoading.value = true
  settingsErrorMsg.value = null
  try {
    const view = (await profileService.getSettings()) as { settings?: Record<string, unknown> }
    settingsText.value = JSON.stringify(view.settings ?? {}, null, 2)
  } catch (err) {
    settingsErrorMsg.value = err instanceof ApiError ? err.message : '门店设置加载失败'
  } finally {
    settingsLoading.value = false
  }
}

function saveSettings() {
  let parsed: Record<string, unknown>
  try {
    parsed = JSON.parse(settingsText.value)
  } catch {
    settingsErrorMsg.value = 'JSON 格式有误，请检查后重试'
    return
  }
  settingsErrorMsg.value = null
  void settingsAction.run(() => profileService.updateSettings({ settings: parsed }), {
    successMessage: '门店设置已保存',
    onSuccess: () => loadSettings(),
  })
}

function saveProfile() {
  void action.run(
    () =>
      profileService.update({
        name: form.name,
        address: form.address,
        phone: form.phone,
        businessHours: form.businessHours,
        latitude: form.latitude ?? null,
        longitude: form.longitude ?? null,
      }),
    { successMessage: '门店资料已保存', onSuccess: () => load() },
  )
}

function toggleStatus(open: boolean) {
  const status = open ? 'open' : 'closed'
  void action.run(() => profileService.updateStatus(status), {
    confirm: { content: open ? '确认设置为营业中？' : '确认设置为休息中？' },
    successMessage: '营业状态已更新',
    onSuccess: () => {
      form.status = status
    },
  })
}

onMounted(() => {
  load()
  loadSettings()
})
</script>

<template>
  <div>
    <PageHeader
      title="设置"
      description="门店资料与营业状态（门店范围固定，不可切换）"
    />

    <NSpin :show="loading">
      <div class="settings ic-band">
        <div class="settings__logo">
          <AssetImage
            :url="form.logoUrl"
            :size="72"
          />
          <div>
            <div class="settings__store-name">
              {{ form.name || auth.store?.name || '当前门店' }}
            </div>
            <p class="ic-muted settings__logo-hint">
              Logo 仅提交 assetId，上传组件待资产服务接入
            </p>
          </div>
        </div>

        <NForm
          label-placement="top"
          class="settings__form"
        >
          <NFormItem label="门店名称">
            <NInput
              v-model:value="form.name"
              placeholder="门店名称"
            />
          </NFormItem>
          <NFormItem label="门店地址">
            <NInput
              v-model:value="form.address"
              placeholder="门店地址"
            />
          </NFormItem>
          <NFormItem label="联系电话">
            <NInput
              v-model:value="form.phone"
              placeholder="门店联系电话"
            />
          </NFormItem>
          <NFormItem label="营业时间">
            <NInput
              v-model:value="form.businessHours"
              placeholder="如：10:00 - 22:00"
            />
          </NFormItem>
          <NFormItem label="纬度 latitude">
            <NInputNumber
              v-model:value="form.latitude"
              :show-button="false"
              :precision="6"
              :min="-90"
              :max="90"
              placeholder="如：31.230416（小程序据此算距离/导航）"
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
              placeholder="如：121.473701（小程序据此算距离/导航）"
              style="width: 100%"
            />
          </NFormItem>

          <NFormItem label="营业状态">
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
          </NFormItem>

          <NButton
            type="primary"
            :loading="action.running.value"
            @click="saveProfile"
          >
            保存资料
          </NButton>
        </NForm>

        <p
          v-if="errorMsg"
          class="ic-muted settings__error"
        >
          {{ errorMsg }}（等待服务端接口就绪）
        </p>
      </div>
    </NSpin>

    <NDivider />

    <NSpin :show="settingsLoading">
      <div class="settings ic-band">
        <NFormItem label="门店设置（JSON）">
          <NInput
            v-model:value="settingsText"
            type="textarea"
            :rows="10"
            placeholder="{}"
          />
        </NFormItem>

        <NButton
          type="primary"
          :loading="settingsAction.running.value"
          @click="saveSettings"
        >
          保存设置
        </NButton>

        <p
          v-if="settingsErrorMsg"
          class="ic-muted settings__error"
        >
          {{ settingsErrorMsg }}
        </p>
      </div>
    </NSpin>
  </div>
</template>

<style scoped>
.settings {
  max-width: 520px;
  padding: var(--ic-space-5);
}
.settings__logo {
  display: flex;
  align-items: center;
  gap: var(--ic-space-4);
  padding-bottom: var(--ic-space-4);
  border-bottom: var(--ic-divider);
  margin-bottom: var(--ic-space-4);
}
.settings__store-name {
  font-size: var(--ic-font-md);
  font-weight: 600;
}
.settings__logo-hint {
  font-size: var(--ic-font-xs);
  margin: 4px 0 0;
}
.settings__error {
  font-size: var(--ic-font-xs);
  margin-top: var(--ic-space-3);
}
</style>
