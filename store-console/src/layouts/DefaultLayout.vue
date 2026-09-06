<script setup lang="ts">
/**
 * 门店后台主布局：侧边菜单 + 顶部账号信息 + 当前门店只读展示 + 内容区。
 *
 * 侧边菜单来自 MENU 配置并按权限过滤。顶部展示当前门店（只读，来自 token scope/me），
 * 不提供门店选择器或切换。
 */
import { computed, h, onBeforeUnmount, onMounted, ref, watch, type Component } from 'vue'
import { RouterView, useRoute, useRouter } from 'vue-router'
import {
  NAvatar,
  NButton,
  NDropdown,
  NDrawer,
  NDrawerContent,
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
const mobileNavigationOpen = ref(false)
const mobileMedia = window.matchMedia('(max-width: 768px)')
const isMobile = ref(mobileMedia.matches)

function updateViewport() {
  isMobile.value = mobileMedia.matches
  if (!isMobile.value) mobileNavigationOpen.value = false
}

onMounted(() => mobileMedia.addEventListener('change', updateViewport))
onBeforeUnmount(() => mobileMedia.removeEventListener('change', updateViewport))
watch(() => route.fullPath, () => { mobileNavigationOpen.value = false })

function renderIcon(name: string) {
  return () => h(AppIcon as Component, { name })
}

/** 按权限过滤后的菜单，多入口分组收起为二级导航。 */
const menuOptions = computed<MenuOption[]>(() => {
  const options: MenuOption[] = []
  for (const group of MENU) {
    const items = group.items.filter((item) => {
      const roleAllowed = !item.roles?.length || (auth.account?.role && item.roles.includes(auth.account.role))
      return roleAllowed && auth.hasPermission(item.permissions)
    })
    if (items.length === 0) continue
    const childOptions = items.map((item) => ({
      label: item.title,
      key: item.name,
      icon: renderIcon(item.icon),
    }))
    if (group.name && group.title) {
      options.push({
        label: group.title,
        key: `section:${group.name}`,
        icon: group.icon ? renderIcon(group.icon) : undefined,
        children: childOptions,
      })
    } else {
      options.push(...childOptions)
    }
  }
  return options
})

const activeKey = computed(() => (typeof route.name === 'string' ? route.name : ''))
const expandedKeys = ref<Array<string | number>>([])

function sectionKeyForRoute(routeName: string) {
  const group = MENU.find((entry) => (
    entry.name
    && entry.title
    && entry.items.some((item) => item.name === routeName)
  ))
  return group?.name ? `section:${group.name}` : undefined
}

watch(activeKey, (key) => {
  const sectionKey = sectionKeyForRoute(key)
  expandedKeys.value = sectionKey ? [sectionKey] : []
}, { immediate: true })

function onMenuExpanded(keys: Array<string | number>) {
  const newlyExpanded = keys.find((key) => !expandedKeys.value.includes(key))
  expandedKeys.value = newlyExpanded ? [newlyExpanded] : []
}

function onMenuSelect(key: string) {
  mobileNavigationOpen.value = false
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
      return '超级管理员'
    case 'cashier':
      return '门店管理员'
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
    :has-sider="!isMobile"
    class="app-layout"
  >
    <NLayoutSider
      v-if="!isMobile"
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
      <div class="menu-scroll">
        <NMenu
          class="side-menu"
          :options="menuOptions"
          :value="activeKey"
          :expanded-keys="expandedKeys"
          :collapsed="collapsed"
          :collapsed-width="64"
          :indent="18"
          :root-indent="16"
          @update:value="onMenuSelect"
          @update:expanded-keys="onMenuExpanded"
        />
      </div>
    </NLayoutSider>

    <NLayout class="main-layout">
      <NLayoutHeader
        bordered
        class="topbar"
      >
        <div class="topbar__store">
          <NButton
            v-if="isMobile"
            quaternary
            class="topbar__menu-toggle"
            aria-label="打开门店导航"
            aria-controls="mobile-store-navigation"
            :aria-expanded="mobileNavigationOpen"
            @click="mobileNavigationOpen = true"
          >
            <svg
              width="22"
              height="22"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="1.7"
              aria-hidden="true"
            >
              <path d="M4 6h16M4 12h16M4 18h16" />
            </svg>
          </NButton>
          <AppIcon
            v-else
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
            :aria-label="`${accountName}，${roleLabel}，账号菜单`"
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
  <NDrawer
    v-model:show="mobileNavigationOpen"
    placement="left"
    :width="296"
    :style="{ maxWidth: 'calc(100vw - 32px)' }"
  >
    <NDrawerContent
      title="门店导航"
      closable
      :native-scrollbar="false"
      :body-content-style="{ padding: '8px 8px 24px' }"
    >
      <nav
        id="mobile-store-navigation"
        aria-label="门店后台导航"
      >
        <p class="mobile-navigation__store">
          {{ storeName }}
        </p>
        <NMenu
          class="side-menu"
          :options="menuOptions"
          :value="activeKey"
          :expanded-keys="expandedKeys"
          :indent="18"
          :root-indent="16"
          @update:value="onMenuSelect"
          @update:expanded-keys="onMenuExpanded"
        />
      </nav>
    </NDrawerContent>
  </NDrawer>
</template>

<style scoped>
.app-layout {
  height: 100vh;
  height: 100dvh;
}
.main-layout {
  min-width: 0;
}
.mobile-navigation__store {
  margin: 8px 16px 16px;
  color: var(--ic-color-text-secondary);
  overflow-wrap: anywhere;
}
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
.menu-scroll {
  height: calc(100vh - var(--ic-header-height));
  overflow-y: auto;
  padding: var(--ic-space-2) var(--ic-space-2) var(--ic-space-5);
}
.side-menu {
  --n-item-height: 42px !important;
}
.side-menu :deep(.n-menu-item-content) {
  border-radius: var(--ic-radius-md);
}
.side-menu :deep(.n-submenu-children) {
  padding-bottom: var(--ic-space-1);
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
  height: calc(100dvh - var(--ic-header-height));
  background: var(--ic-color-bg);
}

@media (max-width: 768px) {
  .topbar {
    gap: var(--ic-space-2);
    padding: 0 var(--ic-space-3);
  }
  .topbar__store {
    min-width: 0;
  }
  .topbar__menu-toggle {
    flex-shrink: 0;
    width: 44px;
    height: 44px;
  }
  .topbar__store-name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .topbar__store-tag,
  .topbar__account-text {
    display: none;
  }
  .topbar__account {
    flex-shrink: 0;
    min-width: 44px;
    min-height: 44px;
  }
  .content {
    padding: var(--ic-space-4);
    padding-bottom: max(var(--ic-space-5), env(safe-area-inset-bottom));
  }
  .side-menu {
    --n-item-height: 44px !important;
  }
}
</style>
