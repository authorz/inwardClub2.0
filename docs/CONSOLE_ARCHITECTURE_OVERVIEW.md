# InwardClub 2.0 后台站点架构总览

## 1. 核心决策

InwardClub 2.0 采用两个独立后台站点：

- 总后台：`admin-web`
- 门店后台：`store-web`

二者必须独立部署、独立登录、独立账号、独立 token audience、独立 API client、独立菜单和独立权限码。

这样设计的主要目的：

- 防止总后台和门店后台登录态混用。
- 降低水平越权风险。
- 避免门店后台误出现跨店筛选和全局操作。
- 让安全审计、监控、错误追踪和发布回滚边界更清晰。

## 2. 文档入口

- 总后台架构：`docs/ADMIN_CONSOLE_ARCHITECTURE.md`
- 门店后台架构：`docs/STORE_CONSOLE_ARCHITECTURE.md`
- v2 Go API 执行规格：`docs/CLAUDE_GO_2_0_IMPLEMENTATION_SPEC.md`
- v1 接口覆盖映射：`docs/V1_API_INVENTORY_AND_V2_MAPPING.md`

## 3. 禁止事项

1. 禁止总后台和门店后台共享账号登录入口。
2. 禁止总后台和门店后台共享 token audience。
3. 禁止门店后台调用 `/api/v2/admin/*`。
4. 禁止总后台前端复用门店后台登录态。
5. 禁止门店后台出现门店选择器来决定数据范围。
6. 禁止前端通过传 `storeId` 控制门店后台数据范围。
7. 禁止把权限隔离只做在前端；最终权限必须由 Go API 强制。

## 4. 允许复用

允许复用开源底座、设计规范、复制后的基础组件、表格经验和表单模式。

如果复用代码，必须满足：

- 构建产物独立。
- 环境变量独立。
- API client 独立。
- auth store 独立。
- permission store 独立。
- 错误监控项目独立。

推荐两个站点都以 `Vue Vben Admin 5` 为底座分别初始化，而不是做一个运行时多租户控制台。
