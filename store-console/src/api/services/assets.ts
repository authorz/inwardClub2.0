import { post } from '../request'
import { API_PATHS } from '@/constants/apiPaths'

interface UploadCredential {
  assetId: number
  objectKey: string
  uploadToken: string
  uploadUrl: string
  publicUrl: string
}

export interface UploadedAsset {
  assetId: number
  objectKey: string
  publicUrl: string
}

const publicDomain = import.meta.env.VITE_ASSET_PUBLIC_DOMAIN || ''

export const assetService = {
  async uploadImage(purpose: string, file: File): Promise<UploadedAsset> {
    const credential = await post<UploadCredential>(API_PATHS.assets.uploadCredentials, {
      purpose,
      filename: file.name,
      contentType: file.type,
      sizeBytes: file.size,
    })
    const form = new FormData()
    form.append('token', credential.uploadToken)
    form.append('key', credential.objectKey)
    form.append('x:assetId', String(credential.assetId))
    form.append('file', file)
    const response = await fetch(credential.uploadUrl, { method: 'POST', body: form })
    if (!response.ok) throw new Error(`图片上传失败（${response.status}）`)
    const publicUrl = credential.publicUrl || (publicDomain
      ? `${publicDomain.replace(/\/$/, '')}/${credential.objectKey}`
      : '')
    return { assetId: credential.assetId, objectKey: credential.objectKey, publicUrl }
  },
}
