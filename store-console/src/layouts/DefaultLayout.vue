<script setup lang="ts">
/**
 * 门店后台主布局：侧边菜单 + 顶部账号信息 + 当前门店只读展示 + 内容区。
 *
 * 侧边菜单来自 MENU 配置并按权限过滤。顶部展示当前门店（只读，来自 token scope/me），
 * 不提供门店选择器或切换。
 */
import { computed, h, ref, type Component } from 'vue'
import { RouterView, useRoute, useRouter } from 'vue-router'
import {
  NAvatar,
  NButton,
  NDropdown,
  NLayout,
  NLayoutContent,
  NLayoutHeader,
  NLayoutSider,
  NMenu,
  type MenuOption,
} from 'naive-ui'
import { MENU } from '@/constants/menu'
import { useAuthStore } from '@/stores/auth'
import { confirm } from '@/composables/useConfirm'
import { AppIcon } from '@/components/common'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()
const collapsed = ref(false)

function renderIcon(name: string) {
  return () => h(AppIcon as Component, { name })
}

/** 按权限过滤后的菜单，分组渲染。 */
const menuOptions = computed<MenuOption[]>(() => {
  const options: MenuOption[] = []
  for (const group of MENU) {
    const items = group.items.filter((item) => auth.hasPermission(item.permissions))
    if (items.length === 0) continue
    if (group.title) {
      options.push({ type: 'group', label: group.title, key: `group:${group.title}`, children: [] })
    }
    for (const item of items) {
      options.push({
        label: item.title,
        key: item.name,
        icon: renderIcon(item.icon),
      })
    }
  }
  return options
})

const activeKey = computed(() => route.name as string)

function onMenuSelect(key: string) {
  router.push({ name: key })
}

const accountOptions = [
  { label: '退出登录', key: 'logout' },
]

async function onAccountAction(key: string) {
  if (key === 'logout') {
    const ok = await confirm({ content: '确认退出门店后台登录？', danger: true })
    if (!ok) return
    await auth.logout()
    router.replace({ name: 'login' })
  }
}

const accountName = computed(() => auth.account?.name ?? '门店账号')
const roleLabel = computed(() => {
  switch (auth.account?.role) {
    case 'store_admin':
      return '门店管理员'
    case 'cashier':
      return '收银员'
    case 'store_operator':
      return '门店运营'
    default:
      return '门店账号'
  }
})
const storeName = computed(() => auth.store?.name ?? '当前门店')
</script>

<template>
  <NLayout
    has-sider
    style="height: 100vh"
  >
    <NLayoutSider
      bordered
      collapse-mode="width"
      :collapsed-width="64"
      :width="232"
      :collapsed="collapsed"
      show-trigger
      @collapse="collapsed = true"
      @expand="collapsed = false"
    >
      <div
        class="brand"
        :class="{ 'brand--collapsed': collapsed }"
      >
        <div class="brand__logo">
          IC
        </div>
        <div
          v-if="!collapsed"
          class="brand__text"
        >
          <div class="brand__name">
            门店后台
          </div>
          <div class="brand__store">
            {{ storeName }}
          </div>
        </div>
      </div>
      <NMenu
        :options="menuOptions"
        :value="activeKey"
        :collapsed="collapsed"
        :collapsed-width="64"
        :indent="18"
        @update:value="onMenuSelect"
      />
    </NLayoutSider>

    <NLayout>
      <NLayoutHeader
        bordered
        class="topbar"
      >
        <div class="topbar__store">
          <AppIcon
            name="store"
            :size="16"
          />
          <span class="topbar__store-name">{{ storeName }}</span>
          <span class="topbar__store-tag">当前门店 · 不可切换</span>
        </div>

        <NDropdown
          :options="accountOptions"
          trigger="click"
          @select="onAccountAction"
        >
          <NButton
            text
            class="topbar__account"
          >
            <NAvatar
              round
              :size="28"
              :src="auth.account?.avatarUrl"
            >
              {{ accountName.slice(0, 1) }}
            </NAvatar>
            <span class="topbar__account-text">
              <span class="topbar__account-name">{{ accountName }}</span>
              <span class="topbar__account-role">{{ roleLabel }}</span>
            </span>
          </NButton>
        </NDropdown>
      </NLayoutHeader>

      <NLayoutContent class="content">
        <RouterView />
      </NLayoutContent>
    </NLayout>
  </NLayout>
</template>

<style scoped>
.brand {
  display: flex;
  align-items: center;
  gap: var(--ic-space-3);
  height: var(--ic-header-height);
  padding: 0 var(--ic-space-4);
  border-bottom: var(--ic-divider);
}
.brand--collapsed {
  justify-content: center;
  padding: 0;
}
.brand__logo {
  width: 32px;
  height: 32px;
  border-radius: var(--ic-radius-sm);
  background: var(--ic-color-primary);
  color: #fff;
  font-weight: 700;
  font-size: var(--ic-font-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.brand__name {
  font-weight: 600;
  font-size: var(--ic-font-base);
}
.brand__store {
  font-size: var(--ic-font-xs);
  color: var(--ic-color-text-tertiary);
}
.topbar {
  height: var(--ic-header-height);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 var(--ic-space-5);
  background: var(--ic-color-surface);
}
.topbar__store {
  display: flex;
  align-items: center;
  gap: var(--ic-space-2);
  color: var(--ic-color-text-secondary);
}
.topbar__store-name {
  font-weight: 600;
  color: var(--ic-color-text);
}
.topbar__store-tag {
  font-size: var(--ic-font-xs);
  color: var(--ic-color-text-tertiary);
  padding-left: var(--ic-space-2);
  border-left: var(--ic-divider);
}
.topbar__account {
  display: flex;
  align-items: center;
  gap: var(--ic-space-2);
}
.topbar__account-text {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  line-height: 1.2;
}
.topbar__account-name {
  font-size: var(--ic-font-sm);
  font-weight: 600;
}
.topbar__account-role {
  font-size: var(--ic-font-xs);
  color: var(--ic-color-text-tertiary);
}
.content {
  padding: var(--ic-space-5);
  overflow: auto;
  height: calc(100vh - var(--ic-header-height));
  background: var(--ic-color-bg);
}
</style>
