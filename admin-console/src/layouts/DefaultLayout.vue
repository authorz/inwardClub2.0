<script setup lang="ts">
/**
 * 总后台主布局：侧边菜单 + 顶部用户信息 + 面包屑 + 内容区。
 * 菜单从 MENU 配置派生，并按权限过滤；点击叶子节点导航。
 */
import { computed, h, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  NBreadcrumb,
  NBreadcrumbItem,
  NButton,
  NDropdown,
  NDrawer,
  NDrawerContent,
  NIcon,
  NLayout,
  NLayoutContent,
  NLayoutHeader,
  NLayoutSider,
  NMenu,
  type MenuOption,
} from 'naive-ui'
import { MENU, type MenuNode } from './menu'
import { useAuthStore } from '@/stores/auth'
import { usePermissionStore } from '@/stores/permission'
import { ADMIN_ROLE_LABELS, type AdminRole } from '@/constants/roles'
import { readStorage, STORAGE_KEYS, writeStorage } from '@/utils/storage'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const permission = usePermissionStore()

const collapsed = ref<boolean>(readStorage<boolean>(STORAGE_KEYS.SIDER_COLLAPSED) ?? false)
const mobileQuery = window.matchMedia('(max-width: 768px)')
const isMobile = ref(mobileQuery.matches)
const mobileMenuOpen = ref(false)

function syncViewport(): void {
  isMobile.value = mobileQuery.matches
  mobileMenuOpen.value = false
}

onMounted(() => mobileQuery.addEventListener('change', syncViewport))
onBeforeUnmount(() => mobileQuery.removeEventListener('change', syncViewport))
watch(() => route.fullPath, () => { mobileMenuOpen.value = false })

function toggleCollapsed(): void {
  if (isMobile.value) {
    mobileMenuOpen.value = !mobileMenuOpen.value
    return
  }
  collapsed.value = !collapsed.value
  writeStorage(STORAGE_KEYS.SIDER_COLLAPSED, collapsed.value)
}

/** 将 MENU 转为 NMenu 选项，按权限过滤空分组 */
function toMenuOptions(nodes: MenuNode[]): MenuOption[] {
  const result: MenuOption[] = []
  for (const node of nodes) {
    if (node.children) {
      const children = toMenuOptions(node.children)
      if (children.length) {
        result.push({
          key: node.key,
          label: node.label,
          icon: node.icon ? () => h(NIcon, null, { default: () => h(node.icon!) }) : undefined,
          children,
        })
      }
    } else if (node.path && permission.has(node.permission)) {
      result.push({
        key: node.path,
        label: node.label,
        icon: node.icon ? () => h(NIcon, null, { default: () => h(node.icon!) }) : undefined,
      })
    }
  }
  return result
}

const menuOptions = computed(() => toMenuOptions(MENU))
const activeKey = computed(() => route.path)

function handleMenuSelect(key: string): void {
  mobileMenuOpen.value = false
  if (key !== route.path) void router.push(key)
}

const breadcrumb = computed<string[]>(() => {
  const meta = route.meta as { breadcrumb?: string[]; title?: string }
  return meta.breadcrumb ?? (meta.title ? [meta.title] : [])
})

const roleLabel = computed(() => {
  const role = auth.user?.role as AdminRole | undefined
  return role ? (ADMIN_ROLE_LABELS[role] ?? role) : ''
})

const userDropdownOptions = [
  { label: '退出登录', key: 'logout' },
]

async function handleUserAction(key: string): Promise<void> {
  if (key === 'logout') {
    await auth.logout()
    void router.replace('/login')
  }
}

function renderCollapseIcon(): ReturnType<typeof h> {
  return h('span', { class: 'layout__collapse-icon' }, collapsed.value ? '»' : '«')
}
</script>

<template>
  <NLayout
    has-sider
    class="layout"
  >
    <NLayoutSider
      v-if="!isMobile"
      bordered
      collapse-mode="width"
      :collapsed="collapsed"
      :collapsed-width="64"
      :width="232"
      :native-scrollbar="false"
    >
      <div class="layout__brand">
        <span class="layout__brand-mark">IC</span>
        <span
          v-if="!collapsed"
          class="layout__brand-text"
        >InwardClub 总后台</span>
      </div>
      <NMenu
        class="layout__menu"
        :value="activeKey"
        :options="menuOptions"
        :collapsed="collapsed"
        :collapsed-width="64"
        :indent="20"
        accordion
        @update:value="handleMenuSelect"
      />
    </NLayoutSider>

    <NLayout class="layout__main">
      <NLayoutHeader
        bordered
        class="layout__header"
      >
        <div class="layout__header-left">
          <NButton
            quaternary
            size="small"
            class="layout__menu-toggle"
            :aria-label="isMobile ? '打开导航菜单' : (collapsed ? '展开导航菜单' : '收起导航菜单')"
            :aria-expanded="isMobile ? mobileMenuOpen : !collapsed"
            @click="toggleCollapsed"
          >
            <span v-if="isMobile">☰ 菜单</span>
            <component
              :is="renderCollapseIcon"
              v-else
            />
          </NButton>
          <NBreadcrumb class="layout__breadcrumb">
            <NBreadcrumbItem
              v-for="(c, i) in breadcrumb"
              :key="i"
            >
              {{ c }}
            </NBreadcrumbItem>
          </NBreadcrumb>
        </div>
        <div class="layout__header-right">
          <NDropdown
            trigger="click"
            :options="userDropdownOptions"
            @select="handleUserAction"
          >
            <div class="layout__user">
              <span class="layout__user-name">{{ auth.user?.displayName ?? '未登录' }}</span>
              <span
                v-if="roleLabel"
                class="layout__user-role"
              >{{ roleLabel }}</span>
            </div>
          </NDropdown>
        </div>
      </NLayoutHeader>

      <NLayoutContent
        class="layout__content"
        :native-scrollbar="false"
      >
        <router-view />
      </NLayoutContent>
    </NLayout>
  </NLayout>
  <NDrawer
    v-model:show="mobileMenuOpen"
    placement="left"
    :width="280"
    style="max-width: 85vw"
  >
    <NDrawerContent
      title="InwardClub 总后台"
      closable
      :native-scrollbar="false"
    >
      <NMenu
        class="layout__menu"
        :value="activeKey"
        :options="menuOptions"
        :indent="20"
        accordion
        @update:value="handleMenuSelect"
      />
    </NDrawerContent>
  </NDrawer>
</template>

<style scoped>
.layout {
  height: 100vh;
  height: 100dvh;
}
.layout__main {
  min-width: 0;
}
.layout__brand {
  display: flex;
  align-items: center;
  gap: var(--ic-space-sm);
  height: var(--ic-header-height);
  padding: 0 var(--ic-space-md);
  border-bottom: 1px solid var(--ic-color-border);
}
.layout__brand-mark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border-radius: var(--ic-radius-sm);
  background: var(--ic-color-primary);
  color: var(--ic-color-text-inverse);
  font-weight: 700;
  font-size: var(--ic-font-sm);
}
.layout__brand-text {
  font-weight: 600;
  font-size: var(--ic-font-md);
  white-space: nowrap;
}
.layout__menu {
  padding: var(--ic-space-sm);
}
:deep(.layout__menu .n-menu-item) {
  margin-top: 2px;
}
:deep(.layout__menu .n-menu-item-content) {
  height: 44px;
  border-radius: var(--ic-radius-md);
}
:deep(.layout__menu .n-menu-item-content-header) {
  font-size: 15px;
  font-weight: 500;
}
:deep(.layout__menu .n-menu-item-content__icon) {
  font-size: 21px;
}
:deep(.layout__menu .n-submenu-children) {
  padding: 2px 0 6px;
}
.layout__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: var(--ic-header-height);
  padding: 0 var(--ic-space-lg);
}
.layout__header-left {
  display: flex;
  align-items: center;
  gap: var(--ic-space-md);
}
.layout__collapse-icon {
  font-size: var(--ic-font-lg);
  line-height: 1;
}
.layout__breadcrumb {
  font-size: var(--ic-font-sm);
}
.layout__user {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  cursor: pointer;
  line-height: 1.2;
}
.layout__user-name {
  font-size: var(--ic-font-sm);
  font-weight: 600;
}
.layout__user-role {
  font-size: var(--ic-font-xs);
  color: var(--ic-color-text-tertiary);
}
.layout__content {
  padding: var(--ic-space-lg);
  background: var(--ic-color-bg);
  height: calc(100% - var(--ic-header-height));
}
@media (max-width: 768px) {
  .layout__header {
    padding-inline: 12px;
    gap: 8px;
  }
  .layout__header-left {
    min-width: 0;
    gap: 8px;
  }
  .layout__menu-toggle {
    min-height: 44px;
    flex-shrink: 0;
  }
  .layout__breadcrumb {
    overflow: hidden;
    white-space: nowrap;
  }
  .layout__user-name {
    max-width: 96px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .layout__content {
    padding: 16px;
  }
}
</style>
