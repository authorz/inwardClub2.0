import { http } from '@/api/http'
import { API_PATHS } from '@/constants/api-paths'

interface PasswordEncryptionKey {
  keyId: string
  algorithm: 'RSA-OAEP-256'
  publicKey: string
}

export interface EncryptedPassword {
  passwordKeyId: string
  passwordCiphertext: string
}

function fromBase64(value: string): Uint8Array {
  const binary = window.atob(value)
  return Uint8Array.from(binary, (char) => char.charCodeAt(0))
}

function toBase64(value: ArrayBuffer): string {
  const bytes = new Uint8Array(value)
  let binary = ''
  for (let offset = 0; offset < bytes.length; offset += 0x8000) {
    binary += String.fromCharCode(...bytes.subarray(offset, offset + 0x8000))
  }
  return window.btoa(binary)
}

// Fetches the authenticated server key immediately before use so a rotated key
// never causes the password to fall back to plaintext transport.
export async function encryptPassword(password: string): Promise<EncryptedPassword> {
  if (!window.isSecureContext || !window.crypto?.subtle) {
    throw new Error('当前页面不是安全连接，无法提交管理员密码')
  }
  const keyInfo = await http.get<PasswordEncryptionKey>(API_PATHS.auth.passwordEncryptionKey)
  if (keyInfo.algorithm !== 'RSA-OAEP-256') {
    throw new Error('服务端密码加密算法不受支持')
  }
  const publicKey = await window.crypto.subtle.importKey(
    'spki',
    fromBase64(keyInfo.publicKey),
    { name: 'RSA-OAEP', hash: 'SHA-256' },
    false,
    ['encrypt'],
  )
  const ciphertext = await window.crypto.subtle.encrypt(
    { name: 'RSA-OAEP' },
    publicKey,
    new TextEncoder().encode(password),
  )
  return {
    passwordKeyId: keyInfo.keyId,
    passwordCiphertext: toBase64(ciphertext),
  }
}
