<script setup lang="ts">
/** 门店管理员账号：展示总后台和门店后台创建的全部管理员。 */
import AccountListView from './AccountListView.vue'
import { storeAdminAccountService } from '@/api/services'
import { API_PATHS } from '@/constants/api-paths'
</script>

<template>
  <AccountListView
    title="门店管理员"
    description="统一管理各门店的超级管理员与普通管理员账号"
    variant="store-admin"
    :service="storeAdminAccountService"
    :disable-path="API_PATHS.storeAdminAccounts.disable"
  />
</template>
