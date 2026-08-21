<script setup lang="ts">
/**
 * 打印机管理：门店只提交设备 SN，开发者账号由总后台统一配置。
 * 新增成功以芯烨云注册接口成功为前提；启用开关切换本地使用状态。
 */
import { computed, h, reactive, ref } from 'vue'
import { NButton, NInput, NModal, NSpace, NSwitch, type DataTableColumns } from 'naive-ui'
import { printerService } from '@/api/services'
import { useAsyncList } from '@/composables/useAsyncList'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { ACTIVE_STATUS } from '@/constants/enums'
import { PERM } from '@/constants/permissions'
import { statusColumn, textColumn } from '@/utils/columns'
import { DataTable, PageHeader, PermissionButton } from '@/components/common'
import type { PrinterDevice } from '@/types/models'

const list = useAsyncList<PrinterDevice>((params) => printerService.list(params))
const action = useAsyncAction()

const editShow = ref(false)
const form = reactive<{
  deviceSn: string
}>({ deviceSn: '' })

const providerStatusLabels: Record<PrinterDevice['providerStatus'], string> = {
  offline: '离线',
  online: '在线正常',
  abnormal: '在线异常',
  unknown: '查询失败',
  unconfigured: '账号未配置',
}

function openCreate() {
  form.deviceSn = ''
  editShow.value = true
}

function save() {
  void action.run(
    () =>
      printerService.create({ deviceSn: form.deviceSn.trim() }),
    {
      successMessage: '已保存',
      onSuccess: () => {
        editShow.value = false
        list.refresh()
      },
    },
  )
}

function toggleStatus(row: PrinterDevice, enabled: boolean) {
  void action.run(() => printerService.update(row.id, { status: enabled ? 'active' : 'disabled' }), {
    successMessage: enabled ? '已启用' : '已停用',
    onSuccess: () => list.refresh(),
  })
}

function remove(row: PrinterDevice) {
  void action.run(() => printerService.remove(row.id), {
    confirm: { content: `确认删除打印机「${row.name}」？`, danger: true },
    successMessage: '已删除',
    onSuccess: () => list.refresh(),
  })
}

const columns = computed<DataTableColumns<PrinterDevice>>(() => [
  textColumn<PrinterDevice>('SN', (r) => r.deviceSn),
  textColumn<PrinterDevice>('设备状态', (r) => providerStatusLabels[r.providerStatus] ?? '查询失败'),
  statusColumn<PrinterDevice>('启用状态', ACTIVE_STATUS, (r) => r.status, { width: 100 }),
  {
    title: '启用',
    key: 'enabled',
    width: 90,
    render: (row: PrinterDevice) =>
      h(NSwitch, {
        value: row.status === 'active',
        onUpdateValue: (v: boolean) => toggleStatus(row, v),
      }),
  },
  {
    title: '操作',
    key: 'actions',
    width: 140,
    fixed: 'right',
    render: (row: PrinterDevice) =>
      h(NSpace, { size: 4 }, {
        default: () => [
          h(
            PermissionButton,
            { permissions: [PERM.printerWrite], type: 'error', text: true, onClick: () => remove(row) },
            { default: () => '删除' },
          ),
        ],
      }),
  },
])
</script>

<template>
  <div>
    <PageHeader
      title="打印机管理"
      description="维护本店打印设备"
    >
      <template #actions>
        <PermissionButton
          :permissions="[PERM.printerWrite]"
          type="primary"
          @click="openCreate"
        >
          新增打印机
        </PermissionButton>
      </template>
    </PageHeader>

    <DataTable
      :columns="columns"
      :data="list.rows.value"
      :loading="list.loading.value"
      :page="list.page.value"
      :page-size="list.pageSize.value"
      :total="list.total.value"
      empty-text="暂无打印机"
      @update:page="list.setPage"
      @update:page-size="list.setPageSize"
    />

    <NModal
      v-model:show="editShow"
      preset="card"
      title="新增打印机"
      style="width: 400px"
    >
      <form
        class="printer-form"
        autocomplete="off"
        @submit.prevent="save"
      >
        <label>
          <span class="ic-muted">设备 SN</span>
          <NInput
            v-model:value="form.deviceSn"
            :input-props="{ name: 'printer-device-sn', autocomplete: 'off' }"
            placeholder="设备序列号"
          />
        </label>
      </form>
      <template #footer>
        <div class="printer-form__footer">
          <NButton @click="editShow = false">
            取消
          </NButton>
          <NButton
            type="primary"
            :loading="action.running.value"
            :disabled="!form.deviceSn.trim()"
            @click="save"
          >
            保存
          </NButton>
        </div>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.printer-form {
  display: flex;
  flex-direction: column;
  gap: var(--ic-space-4);
}
.printer-form label {
  display: flex;
  flex-direction: column;
  gap: var(--ic-space-2);
  font-size: var(--ic-font-sm);
}
.printer-form__footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--ic-space-2);
}
</style>
