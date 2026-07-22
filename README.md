# InwardClub 2.0

本仓库按独立交付边界拆分：

```text
admin-console/   总后台独立站点
store-console/   门店后台独立站点
server/          Go 2.0 服务端
mini-program/    微信小程序端
design/          设计稿、原型和视觉规范
docs/            产品、接口、架构和迁移文档
tasks/           Claude 任务分发与 Codex 验收记录
```

协作模式：

- Claude 负责具体开发实现。
- Codex 负责监工、拆任务、分发任务、审查结果和验收。
- 总后台和门店后台必须独立站点、独立账号、独立登录态、独立 token audience。
- 服务端是权限最终裁决方，前端权限只用于体验和提示。
