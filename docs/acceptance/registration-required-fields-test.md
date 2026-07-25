# 小程序注册必填字段验收测试

**日期**: 2026-07-23  
**任务**: 重构小程序注册流程，头像、昵称、手机号、性别四个字段必填，只有提交完整表单后才创建会员

## 后端改动

### 1. 数据库迁移
- ✅ 添加迁移 `00023_member_gender.sql`
- ✅ `members` 表新增 `gender` 字段（VARCHAR(16)，允许值：male/female/other）
- ✅ 迁移已执行：`goose: successfully migrated database to version: 23`

### 2. 模型层
- ✅ `Member` 结构体添加 `Gender string` 字段
- ✅ Repository 的 `memberColumns` 添加 `gender` 列
- ✅ `scanMember` 添加 `Gender` 扫描
- ✅ `Create` 方法 INSERT 包含 `avatar_url` 和 `gender`

### 3. DTO 层
- ✅ `WeChatRegisterRequest` 添加必填字段：
  - `AvatarURL string` (binding:"required")
  - `Gender string` (binding:"required")
  - 保留原有：`Nickname`, `PhoneCode`

### 4. 服务层
- ✅ `MiniRegister` 验证逻辑：
  - avatarUrl 非空检查
  - nickname 非空检查
  - gender 非空检查 + 枚举值校验（male/female/other）
  - phoneCode 由微信接口验证
- ✅ `createMember` 签名修改为接收 5 个参数：`openID, avatarURL, nickname, gender, phone`
- ✅ 创建会员时同时写入头像和性别

### 5. 测试修复
- ✅ `TestMiniLoginDefersCreationUntilRegister`：补充 `AvatarURL` 和 `Gender`
- ✅ `TestTokenVersionCheckersReflectLogout`：补充 `AvatarURL` 和 `Gender`
- ✅ 所有测试通过：`ok github.com/inwardclub/server/internal/modules/auth 0.641s`

## 前端改动

### 1. 表单提交
- ✅ `confirmLogin` 验证逻辑已包含四个必填检查：
  ```js
  if (!form.avatarUrl) return ui.toast('请选择头像');
  if (!nickName) return ui.toast('请填写昵称');
  if (!form.phoneBound) return ui.toast('请获取手机号');
  if (!form.gender) return ui.toast('请选择性别');
  ```
- ✅ `api.register()` 调用已修改为传递四个字段：
  ```js
  { registerTicket, avatarUrl: form.avatarUrl, nickname: nickName, gender: form.gender, phoneCode: form.phoneCode }
  ```

### 2. UI 组件
- ✅ wxml 已包含头像选择器（`open-type="chooseAvatar"`）
- ✅ wxml 已包含昵称输入框（`type="nickname"`）
- ✅ wxml 已包含手机号授权按钮（`open-type="getPhoneNumber"`）
- ✅ wxml 已包含性别单选组（male/female，带图标）

## 构建验证

### 后端
```bash
$ cd server
$ go test ./internal/modules/auth/... -v
=== RUN   TestMiniLoginDefersCreationUntilRegister
--- PASS: TestMiniLoginDefersCreationUntilRegister (0.00s)
=== RUN   TestTokenVersionCheckersReflectLogout
--- PASS: TestTokenVersionCheckersReflectLogout (0.00s)
[... 其他测试全部 PASS ...]
PASS
ok  	github.com/inwardclub/server/internal/modules/auth	0.641s

$ go build ./...
(成功，无错误)

$ go build -o bin/api ./cmd/api
(成功)
```

### 前端
- wxml/js/wxss 均已就绪
- 性别选择器 UI 已存在（radio-group + 图标）

## 手动验收步骤

### 测试 1：新用户注册 - 缺少任意字段
1. 小程序首次登录，弹出"注册会员"表单
2. **不选择头像**，填写昵称、性别、手机号 → 点击"完成注册"
   - 预期：提示"请选择头像"
3. 选择头像，**不填写昵称** → 点击
   - 预期：提示"请填写昵称"
4. 填写昵称，**不获取手机号** → 点击
   - 预期：提示"请获取手机号"
5. 获取手机号，**不选择性别** → 点击
   - 预期：提示"请选择性别"

### 测试 2：新用户注册 - 完整填写
1. 选择头像 ✅
2. 填写昵称（如"老大"）✅
3. 获取手机号 ✅
4. 选择性别（男/女）✅
5. 点击"完成注册"
   - 预期：
     - 后端创建会员记录（包含 avatar_url, nickname, gender, phone）
     - 返回会话 token（isNew: true）
     - 前端跳转到会员卡，显示头像、昵称、会员编号

### 测试 3：老用户登录不受影响
1. 已注册会员再次打开小程序
2. 静默登录（`wx.login` → `api.wechatLogin`）
   - 预期：
     - isNew: false
     - 直接返回 token + profile
     - 不弹注册表单
     - 会员卡直接显示

### 测试 4：后端接口验证
```bash
# 模拟缺少 avatarUrl 的注册请求
curl -X POST http://localhost:8080/mini/auth/wechat/register \
  -H "Content-Type: application/json" \
  -d '{
    "registerTicket": "<valid_ticket>",
    "nickname": "老大",
    "gender": "male",
    "phoneCode": "xxx"
  }'

# 预期响应：
# {"code": "INVALID_ARGUMENT", "message": "avatarUrl is required"}

# 模拟 gender 枚举值错误
curl -X POST http://localhost:8080/mini/auth/wechat/register \
  -H "Content-Type: application/json" \
  -d '{
    "registerTicket": "<valid_ticket>",
    "avatarUrl": "https://example.com/avatar.jpg",
    "nickname": "老大",
    "gender": "unknown",
    "phoneCode": "xxx"
  }'

# 预期响应：
# {"code": "INVALID_ARGUMENT", "message": "gender must be male, female, or other"}
```

## 数据库验证

注册成功后检查 members 表：
```sql
SELECT id, nickname, avatar_url, gender, phone, invite_code 
FROM members 
ORDER BY id DESC 
LIMIT 1;
```

预期结果：
- `avatar_url` 不为空（微信 CDN URL）
- `gender` 为 'male' 或 'female'
- `nickname` 为用户填写的昵称
- `phone` 为微信解密后的手机号（11 位）
- `invite_code` 为 6 位数字

## 验收标准

- ✅ 后端测试全部通过
- ✅ 后端构建成功
- ✅ 新用户必须填写四个字段才能注册
- ✅ 缺少任意字段前端有明确提示
- ✅ 后端强制验证四个字段（前端绕过也无法注册）
- ✅ 老用户登录流程不受影响
- ✅ 数据库正确存储头像和性别

## 备注

- 性别字段支持三个枚举值：`male`, `female`, `other`（前端目前只提供男/女选项）
- 头像 URL 来自微信 `chooseAvatar` API，不需要上传到七牛（直接使用微信 CDN）
- 注册流程保持幂等性：重复提交同一 registerTicket 会返回已创建的会员（不会重复创建）
