<script setup lang="ts">
/**
 * 总后台独立登录页（独立入口、独立账号、独立 token audience=admin）。
 * 登录成功后由 auth store 校验 token audience；非 admin 会被拒绝并清空登录态。
 */
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NButton, NForm, NFormItem, NInput, type FormInst, type FormRules } from 'naive-ui'
import { useAuthStore } from '@/stores/auth'
import { toastError } from '@/utils/feedback'
import type { NormalizedError } from '@/api/types'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()

const formRef = ref<FormInst | null>(null)
const model = reactive({ username: '', password: '' })
const submitting = ref(false)

const rules: FormRules = {
  username: [{ required: true, message: '请输入账号', trigger: ['blur', 'input'] }],
  password: [{ required: true, message: '请输入密码', trigger: ['blur', 'input'] }],
}

async function handleSubmit(): Promise<void> {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }
  submitting.value = true
  try {
    const ok = await auth.login({ username: model.username, password: model.password })
    if (ok) {
      const redirect = (route.query.redirect as string) || '/dashboard'
      void router.replace(redirect)
    }
  } catch (e) {
    const err = e as NormalizedError
    toastError(err.message || '登录失败')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="login">
    <div class="login__panel">
      <div class="login__brand">
        <span class="login__mark">IC</span>
        <div>
          <h1 class="login__title">
            InwardClub 总后台
          </h1>
          <p class="login__sub">
            系统级管理站点 · 独立账号登录
          </p>
        </div>
      </div>

      <NForm
        ref="formRef"
        :model="model"
        :rules="rules"
        class="login__form"
        @keyup.enter="handleSubmit"
      >
        <NFormItem
          label="账号"
          path="username"
        >
          <NInput
            v-model:value="model.username"
            placeholder="请输入总后台账号"
          />
        </NFormItem>
        <NFormItem
          label="密码"
          path="password"
        >
          <NInput
            v-model:value="model.password"
            type="password"
            show-password-on="click"
            placeholder="请输入密码"
          />
        </NFormItem>
        <NButton
          type="primary"
          block
          :loading="submitting"
          class="login__submit"
          @click="handleSubmit"
        >
          登录
        </NButton>
      </NForm>

      <p class="login__notice">
        仅限总部 / 总管理角色登录；门店后台账号无法登录本站点。
      </p>
    </div>
  </div>
</template>

<style scoped>
.login {
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--ic-color-bg);
}
.login__panel {
  width: 380px;
  padding: var(--ic-space-xl);
  background: var(--ic-color-surface);
  border: 1px solid var(--ic-color-border);
  border-radius: var(--ic-radius-lg);
}
.login__brand {
  display: flex;
  align-items: center;
  gap: var(--ic-space-md);
  margin-bottom: var(--ic-space-xl);
}
.login__mark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border-radius: var(--ic-radius-md);
  background: var(--ic-color-primary);
  color: var(--ic-color-text-inverse);
  font-weight: 700;
  font-size: var(--ic-font-lg);
}
.login__title {
  margin: 0;
  font-size: var(--ic-font-xl);
  font-weight: 600;
}
.login__sub {
  margin: var(--ic-space-xs) 0 0;
  font-size: var(--ic-font-sm);
  color: var(--ic-color-text-tertiary);
}
.login__submit {
  margin-top: var(--ic-space-sm);
}
.login__notice {
  margin: var(--ic-space-lg) 0 0;
  font-size: var(--ic-font-xs);
  color: var(--ic-color-text-tertiary);
  line-height: 1.5;
}
</style>
