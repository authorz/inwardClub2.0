# InwardClub 2.0

本仓库按独立交付边界拆分：

```text
admin-console/   总后台独立站点
store-console/   门店后台独立站点
server/          Go 2.0 服务端
mini-program/    微信小程序端
design/          设计稿、原型和视觉规范
docs/            产品、接口、架构和迁移文档
tasks/           历史任务交接与验收记录
```

协作模式：

- Codex 全权负责分析、开发实现、审查、验证和交付。
- 确有独立并行价值时，由 Codex 调用自身的多 Agent 能力并负责结果整合与最终验收。
- 总后台和门店后台必须独立站点、独立账号、独立登录态、独立 token audience。
- 服务端是权限最终裁决方，前端权限只用于体验和提示。
