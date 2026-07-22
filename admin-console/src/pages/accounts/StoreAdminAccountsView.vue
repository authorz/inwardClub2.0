<script setup lang="ts">
/** 门店管理员账号：复用 AccountListView，展示绑定门店。 */
import AccountListView from './AccountListView.vue'
import { storeAdminAccountService } from '@/api/services'
import { API_PATHS } from '@/constants/api-paths'
</script>

<template>
  <AccountListView
    title="门店管理员"
    description="门店后台管理员账号，绑定唯一门店"
    variant="store-admin"
    :service="storeAdminAccountService"
    :disable-path="API_PATHS.storeAdminAccounts.disable"
  />
</template>
