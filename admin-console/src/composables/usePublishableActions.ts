/**
 * 可发布资源的通用操作（公共状态）。
 *
 * 商品、活动都遵循「发布 / 下架」同一套动作语义。服务端没有独立的
 * publish/unpublish 接口，发布/下架通过在整体 PUT 更新里改写 status 完成。
 * 由于更新 DTO 含多个 binding:required 字段，这里先 GET 详情补齐必填字段，
 * 再以整体覆盖的方式 PUT 回写 status，避免 422 与字段被清空。
 */
import type { ResourceService } from '@/api/resource'
import { runAudited } from './useAuditedAction'

export interface StatusToggle {
  /** 表示「已发布 / 上架」的 status 值 */
  publishedStatus: string
  /** 「下架」时写回的 status 值 */
  unpublishedStatus: string
}

export function usePublishableActions<T extends { id: string; status?: string }>(
  service: ResourceService<T>,
  toggle: StatusToggle,
  onDone: () => void,
) {
  async function setStatus(id: string, targetStatus: string): Promise<void> {
    const detail = await service.get(id)
    await service.update(id, { ...detail, status: targetStatus })
  }

  async function publish(row: T, name: string): Promise<void> {
    const ok = await runAudited({
      title: '发布',
      content: `确认发布「${name}」？发布为跨店可见的全局资源，将写入审计日志。`,
      highRisk: true,
      positiveText: '确认发布',
      execute: () => setStatus(row.id, toggle.publishedStatus),
      successText: '已发布',
    })
    if (ok) onDone()
  }

  async function unpublish(row: T, name: string): Promise<void> {
    const ok = await runAudited({
      title: '下架',
      content: `确认下架「${name}」？下架后门店端将不可见，将写入审计日志。`,
      highRisk: true,
      positiveText: '确认下架',
      execute: () => setStatus(row.id, toggle.unpublishedStatus),
      successText: '已下架',
    })
    if (ok) onDone()
  }

  return { publish, unpublish }
}
