/**
 * 打印机设备管理服务。
 */

import { del, getPaged, patch, post } from '../request'
import { API_PATHS } from '@/constants/apiPaths'
import type { PageQuery } from '@/types/api'
import type { PrinterDevice } from '@/types/models'

/** 新增打印机：门店提交设备位置/名称和 SN，开发者账号由总后台统一配置。 */
export interface PrinterCreatePayload {
  name: string
  deviceSn: string
}

/** 编辑打印机：nil 字段保持不变。 */
export interface PrinterPatchPayload {
  name?: string
  status?: PrinterDevice['status']
}

export const printerService = {
  list(params?: PageQuery) {
    return getPaged<PrinterDevice>(API_PATHS.printers.list, params)
  },
  create(body: PrinterCreatePayload) {
    return post<PrinterDevice>(API_PATHS.printers.list, body, { idempotent: true })
  },
  update(id: string | number, body: PrinterPatchPayload) {
    return patch<PrinterDevice>(API_PATHS.printers.device(id), body, { idempotent: true })
  },
  remove(id: string | number) {
    return del<unknown>(API_PATHS.printers.device(id), { idempotent: true })
  },
}
