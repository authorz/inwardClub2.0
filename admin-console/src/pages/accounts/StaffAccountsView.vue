<script setup lang="ts">
/** 员工账号：复用 AccountListView，每行展示一条员工与门店的绑定关系。 */
import AccountListView from './AccountListView.vue'
import { staffAccountService } from '@/api/services'
import { API_PATHS } from '@/constants/api-paths'
</script>

<template>
  <AccountListView
    title="员工"
    description="门店工作人员账号；同一微信用户可绑定多个门店"
    variant="staff"
    :service="staffAccountService"
    :disable-path="API_PATHS.staffAccounts.disable"
  />
</template>
