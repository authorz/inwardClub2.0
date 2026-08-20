/**
 * 门店后台路由。
 *
 * 业务路由由菜单配置（FLAT_MENU）驱动，component 从 pageComponents 映射取，
 * 保证菜单、路由、权限三者共用同一份 name/permission 定义，不各写一套。
 */

import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { FLAT_MENU } from '@/constants/menu'
import { pageComponents } from './pages'
import { installRouterGuards } from './guards'

const businessRoutes: RouteRecordRaw[] = FLAT_MENU.map((item) => ({
  path: item.path,
  name: item.name,
  component: pageComponents[item.name],
  meta: {
    title: item.title,
    permissions: item.permissions ?? [],
    roles: item.roles ?? [],
    requiresAuth: true,
  },
}))

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'login',
    component: () => import('@/pages/LoginView.vue'),
    meta: { title: '门店后台登录', public: true },
  },
  {
    path: '/',
    component: () => import('@/layouts/DefaultLayout.vue'),
    redirect: '/dashboard',
    children: businessRoutes,
  },
  {
    path: '/403',
    name: 'forbidden',
    component: () => import('@/pages/error/ForbiddenView.vue'),
    meta: { title: '无权限', requiresAuth: true },
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'not-found',
    component: () => import('@/pages/error/NotFoundView.vue'),
    meta: { title: '页面不存在' },
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

installRouterGuards(router)

export default router
