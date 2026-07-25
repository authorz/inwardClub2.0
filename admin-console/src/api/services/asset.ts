/**
 * 资产直传服务（七牛）。
 *
 * 两步式直传，与服务端 asset 模块契约一致：
 *  1. POST /admin/assets/upload-credentials 申请上传凭证（服务端登记 pending 资产、生成 objectKey、签发 token）。
 *  2. 用 multipart/form-data 直传到七牛 uploadUrl。
 *
 * 业务表保存 objectKey（对象路径/key），不保存完整域名 URL 或 assetId；预览用服务端解析后的公开地址。
 */
import { http } from '@/api/http'
import { API_PATHS } from '@/constants/api-paths'

/** 服务端返回的上传凭证（对齐后端 asset.UploadCredential） */
export interface UploadCredential {
  assetId: number
  objectKey: string
  uploadToken: string
  uploadUrl: string
  expiresAt: string
  maxSizeBytes: number
}

/** 直传成功后的结果：objectKey 用于写入业务字段（path），publicUrl 仅用于预览 */
export interface UploadedAsset {
  assetId: number
  objectKey: string
  publicUrl: string
}

const publicDomain = import.meta.env.VITE_ASSET_PUBLIC_DOMAIN || ''

/** 由 objectKey 拼出公开访问地址（未配置公开域名时返回空，交由占位展示）。 */
function resolvePublicUrl(objectKey: string): string {
  if (!publicDomain) return ''
  return `${publicDomain.replace(/\/$/, '')}/${objectKey}`
}

export const assetService = {
  /**
   * 图片直传：申请凭证 → multipart 直传七牛。
   * 返回 objectKey（写入 bannerPath 等业务字段）与解析后的预览地址。
   * 直传请求走原生 fetch，绕开总后台 axios（不注入 Authorization / baseURL，目标是七牛域名）。
   */
  async uploadImage(purpose: string, file: File): Promise<UploadedAsset> {
    const cred = await http.post<UploadCredential>(API_PATHS.assets.uploadCredentials, {
      purpose,
      filename: file.name,
      contentType: file.type,
      sizeBytes: file.size,
    })

    const form = new FormData()
    form.append('token', cred.uploadToken)
    form.append('key', cred.objectKey)
    // 回调体引用 $(x:assetId)，直传必须回传该自定义变量，服务端回调才能标记对应 pending 资产。
    form.append('x:assetId', String(cred.assetId))
    form.append('file', file)

    const res = await fetch(cred.uploadUrl, { method: 'POST', body: form })
    if (!res.ok) {
      throw new Error(`图片上传失败（${res.status}）`)
    }

    return { assetId: cred.assetId, objectKey: cred.objectKey, publicUrl: resolvePublicUrl(cred.objectKey) }
  },
}
