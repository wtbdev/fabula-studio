# Feature Spec 01：用户系统

## 1. 功能名称

用户注册与登录系统

## 2. 功能目标

实现叙幕工作室 MVP 的基础用户系统，使用户可以完成注册、登录、获取当前用户信息和退出登录。用户登录后才能访问项目列表、创建项目、生成剧本和编辑剧本。

## 3. 功能范围

### 包含

1. 用户注册；
2. 用户登录；
3. 获取当前登录用户；
4. 退出登录；
5. 登录态校验；
6. 前端路由鉴权；
7. 用户 AI 点数展示。

### 不包含

邮箱验证码、找回密码、第三方登录、复杂用户资料编辑、权限角色系统。

## 4. 用户故事

- 作为新用户，我希望可以通过邮箱、密码和昵称注册账号，以便开始使用叙幕工作室创建剧本项目。
- 作为已注册用户，我希望可以使用邮箱和密码登录系统，以便继续管理我的剧本项目。
- 作为用户，我希望刷新页面后仍然保持登录状态，以便不用频繁重新登录。
- 作为用户，我希望可以退出当前账号，以便保护我的项目数据。

## 5. 页面需求

### 5.1 注册页

路由：

```txt
/register
```

字段：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| 邮箱 | input | 是 | 用户登录邮箱 |
| 昵称 | input | 是 | 用户展示名称 |
| 密码 | password | 是 | 登录密码 |
| 确认密码 | password | 是 | 二次确认密码 |

交互要求：

1. 校验必填字段；
2. 校验两次密码一致；
3. 点击注册后调用注册接口；
4. 注册成功后保存 token；
5. 注册成功后跳转项目列表页。

### 5.2 登录页

路由：

```txt
/login
```

字段：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| 邮箱 | input | 是 | 用户登录邮箱 |
| 密码 | password | 是 | 用户登录密码 |

交互要求：

1. 校验邮箱和密码；
2. 点击登录后调用登录接口；
3. 登录成功后保存 token；
4. 登录成功后获取当前用户信息；
5. 登录成功后跳转项目列表页；
6. 登录失败时显示错误提示。

### 5.3 顶栏用户信息

登录后页面右上角展示：

```txt
创作者用户｜AI 点数：700｜退出
```

## 6. API 依赖

### 注册

```http
POST /api/auth/register
```

请求体：

```json
{
  "email": "user@example.com",
  "password": "123456",
  "nickname": "创作者用户"
}
```

响应：

```json
{
  "code": 0,
  "message": "注册成功",
  "data": {
    "token": "jwt_token",
    "user": {
      "id": "user_001",
      "email": "user@example.com",
      "nickname": "创作者用户",
      "aiPoints": 1000
    }
  }
}
```

### 登录

```http
POST /api/auth/login
```

请求体：

```json
{
  "email": "user@example.com",
  "password": "123456"
}
```

响应：

```json
{
  "code": 0,
  "message": "登录成功",
  "data": {
    "token": "jwt_token",
    "user": {
      "id": "user_001",
      "email": "user@example.com",
      "nickname": "创作者用户",
      "aiPoints": 1000
    }
  }
}
```

### 获取当前用户

```http
GET /api/auth/me
Authorization: Bearer jwt_token
```

响应：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "user_001",
    "email": "user@example.com",
    "nickname": "创作者用户",
    "aiPoints": 700
  }
}
```

### 退出登录

```http
POST /api/auth/logout
```

MVP 阶段前端删除本地 token 即可。

## 7. 前端状态设计

```ts
interface AuthState {
  token: string | null
  user: User | null
  isLoggedIn: boolean
}

interface User {
  id: string
  email: string
  nickname: string
  aiPoints: number
}
```

主要方法：

```ts
login(email: string, password: string): Promise<void>
register(payload: RegisterPayload): Promise<void>
fetchMe(): Promise<void>
logout(): void
```

## 8. 路由鉴权规则

需要登录后访问：

```txt
/projects
/projects/new
/projects/:id/edit
```

未登录访问时跳转到：

```txt
/login
```

## 9. 数据库存储

```sql
CREATE TABLE users (
  id VARCHAR(64) PRIMARY KEY,
  email VARCHAR(255) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  nickname VARCHAR(100) NOT NULL,
  ai_points INTEGER NOT NULL DEFAULT 1000,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);
```

## 10. 验收标准

1. 用户可以注册账号；
2. 注册成功后可以自动登录；
3. 用户可以通过邮箱和密码登录；
4. 登录成功后可以进入项目列表页；
5. 用户刷新页面后仍能保持登录状态；
6. 用户可以看到自己的昵称和 AI 点数；
7. 用户可以退出登录；
8. 退出登录后无法访问项目页面；
9. 未登录访问项目页面时自动跳转登录页；
10. 登录失败时页面显示错误信息。
