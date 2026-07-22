<script setup lang="ts">
/**
 * 门店后台独立登录页。
 * 独立登录入口，不复用总后台登录态；登录成功后按 redirect 或工作台跳转。
 */
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NButton, NForm, NFormItem, NInput } from 'naive-ui'
import { useAuthStore } from '@/stores/auth'
import { feedback } from '@/utils/feedback'
import { appConfig } from '@/config'
import { ApiError } from '@/api/error'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()

const model = reactive({ username: '', password: '' })
const loading = ref(false)

async function onSubmit() {
  if (!model.username || !model.password) {
    feedback.message.warning('请输入账号和密码')
    return
  }
  loading.value = true
  try {
    await auth.login({ username: model.username.trim(), password: model.password })
    feedback.message.success('登录成功')
    const redirect = (route.query.redirect as string) || '/dashboard'
    router.replace(redirect)
  } catch (err) {
    const msg = err instanceof ApiError ? err.message : (err as Error).message
    feedback.message.error(msg || '登录失败')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login">
    <div class="login__panel ic-band">
      <div class="login__brand">
        <div class="login__logo">
          IC
        </div>
        <div>
          <h1 class="login__title">
            InwardClub 门店后台
          </h1>
          <p class="login__subtitle ic-muted">
            单店运营工作台 · {{ appConfig.appId }}
          </p>
        </div>
      </div>

      <NForm
        :show-label="false"
        @keyup.enter="onSubmit"
      >
        <NFormItem>
          <NInput
            v-model:value="model.username"
            placeholder="门店账号 / 收银员账号"
            size="large"
            :input-props="{ autocomplete: 'username' }"
          />
        </NFormItem>
        <NFormItem>
          <NInput
            v-model:value="model.password"
            type="password"
            show-password-on="click"
            placeholder="登录密码"
            size="large"
            :input-props="{ autocomplete: 'current-password' }"
          />
        </NFormItem>
        <NButton
          type="primary"
          block
          size="large"
          :loading="loading"
          @click="onSubmit"
        >
          登录
        </NButton>
      </NForm>

      <p class="login__note ic-muted">
        本站点仅供门店工作人员登录，账号与总后台独立，不可跨店操作。
      </p>
    </div>
  </div>
</template>

<style scoped>
.login {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: var(--ic-space-4);
}
.login__panel {
  width: 100%;
  max-width: 400px;
  padding: var(--ic-space-6);
}
.login__brand {
  display: flex;
  align-items: center;
  gap: var(--ic-space-3);
  margin-bottom: var(--ic-space-6);
}
.login__logo {
  width: 44px;
  height: 44px;
  border-radius: var(--ic-radius-md);
  background: var(--ic-color-primary);
  color: #fff;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
}
.login__title {
  font-size: var(--ic-font-md);
  margin: 0;
}
.login__subtitle {
  font-size: var(--ic-font-xs);
  margin: 2px 0 0;
}
.login__note {
  font-size: var(--ic-font-xs);
  margin-top: var(--ic-space-5);
  line-height: 1.6;
}
</style>
