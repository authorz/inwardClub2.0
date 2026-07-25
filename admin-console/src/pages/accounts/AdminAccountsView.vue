<script setup lang="ts">
/** 总后台账号：复用 AccountListView。 */
import AccountListView from './AccountListView.vue'
import { adminAccountService } from '@/api/services'
import { API_PATHS } from '@/constants/api-paths'
</script>

<template>
  <AccountListView
    title="总后台账号"
    description="总部管理员账号；系统管理员不可删除或禁用"
    variant="super"
    :service="adminAccountService"
    :disable-path="API_PATHS.adminAccounts.disable"
  />
</template>
