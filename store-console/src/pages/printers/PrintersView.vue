<script setup lang="ts">
/**
 * 打印机管理：本店打印设备增删改与启用状态。
 * 服务端设备模型为 name/provider(默认 xpyun)/deviceSn/deviceKey/status(active|disabled)；
 * deviceKey 仅写入不回显。启用开关切换 status，写操作带 Idempotency-Key。
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
const editDeviceSn = ref(false)
const editDeviceKey = ref(false)
const form = reactive<{
  id: string | number | null
  name: string
  deviceSn: string
  provider: string
  deviceKey: string
}>({ id: null, name: '', deviceSn: '', provider: '', deviceKey: '' })

function openCreate() {
  form.id = null
  form.name = ''
  form.deviceSn = ''
  form.provider = ''
  form.deviceKey = ''
  editDeviceSn.value = true
  editDeviceKey.value = true
  editShow.value = true
}

function openEdit(row: PrinterDevice) {
  form.id = row.id
  form.name = row.name
  form.deviceSn = row.deviceSn ?? ''
  form.provider = row.provider ?? ''
  form.deviceKey = ''
  editDeviceSn.value = false
  editDeviceKey.value = false
  editShow.value = true
}

function save() {
  void action.run(
    () =>
      form.id == null
        ? printerService.create({
            name: form.name,
            deviceSn: form.deviceSn,
            provider: form.provider || undefined,
            deviceKey: form.deviceKey || undefined,
          })
        : // DevicePatch 不含 provider；provider 仅创建时设置。
          printerService.update(form.id, {
            name: form.name,
            ...(editDeviceSn.value ? { deviceSn: form.deviceSn } : {}),
            ...(editDeviceKey.value && form.deviceKey ? { deviceKey: form.deviceKey } : {}),
          }),
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
  textColumn<PrinterDevice>('名称', (r) => r.name),
  textColumn<PrinterDevice>('SN', (r) => r.deviceSn),
  textColumn<PrinterDevice>('提供商', (r) => r.provider),
  statusColumn<PrinterDevice>('状态', ACTIVE_STATUS, (r) => r.status, { width: 100 }),
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
            { permissions: [PERM.printerWrite], type: 'primary', text: true, onClick: () => openEdit(row) },
            { default: () => '编辑' },
          ),
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
      :title="form.id == null ? '新增打印机' : '编辑打印机'"
      style="width: 400px"
    >
      <form
        class="printer-form"
        autocomplete="off"
        @submit.prevent="save"
      >
        <label>
          <span class="ic-muted">名称</span>
          <NInput
            v-model:value="form.name"
            placeholder="如：前台小票机"
          />
        </label>
        <label>
          <span class="ic-muted">设备 SN</span>
          <NInput
            v-model:value="form.deviceSn"
            :input-props="{ name: 'printer-device-sn', autocomplete: 'off' }"
            :readonly="form.id != null && !editDeviceSn"
            placeholder="设备序列号"
          >
            <template
              v-if="form.id != null && !editDeviceSn"
              #suffix
            >
              <NButton
                text
                type="primary"
                attr-type="button"
                @click="editDeviceSn = true"
              >
                修改 SN
              </NButton>
            </template>
          </NInput>
        </label>
        <label v-if="form.id == null">
          <span class="ic-muted">提供商</span>
          <NInput
            v-model:value="form.provider"
            placeholder="留空默认 xpyun"
          />
        </label>
        <label>
          <span class="ic-muted">设备密钥</span>
          <NInput
            v-if="form.id == null || editDeviceKey"
            v-model:value="form.deviceKey"
            type="password"
            :input-props="{ name: 'printer-device-key', autocomplete: 'new-password' }"
            show-password-on="click"
            :placeholder="form.id == null ? '设备密钥（可选）' : '留空则不修改'"
          />
          <NButton
            v-else
            attr-type="button"
            @click="editDeviceKey = true"
          >
            修改设备密钥
          </NButton>
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
            :disabled="!form.name || !form.deviceSn"
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
