# 01. Auth API Spec 用户系统接口

## 1. 模块说明

用户系统负责注册、登录、获取当前用户信息与退出登录。

MVP 阶段采用邮箱 + 密码登录，登录成功后返回 JWT Token。用户注册成功后默认赠送 `1000` AI 点数。

---

## 2. 数据模型

### UserDTO

```ts
interface UserDTO {
  id: string
  email: string
  nickname: string
  aiPoints: number
  createdAt?: string
  updatedAt?: string
}
```

### AuthTokenDTO

```ts
interface AuthTokenDTO {
  token: string
  user: UserDTO
}
```

---

## 3. 用户注册

### Endpoint

```http
POST /api/auth/register
```

### Auth

不需要登录。

### Request Body

```json
{
  "email": "user@example.com",
  "password": "123456",
  "nickname": "创作者用户"
}
```

### Request Schema

| 字段 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| email | string | 是 | 用户邮箱，唯一 |
| password | string | 是 | 密码，长度建议不少于 6 位 |
| nickname | string | 是 | 用户昵称 |

### Success Response

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
      "aiPoints": 1000,
      "createdAt": "2026-06-06T10:00:00.000Z",
      "updatedAt": "2026-06-06T10:00:00.000Z"
    }
  }
}
```

### Error Responses

| code | message | 说明 |
|---:|---|---|
| 40002 | 参数校验失败 | 邮箱、密码或昵称为空 |
| 40003 | 邮箱已被注册 | email 已存在 |
| 50000 | 服务器内部错误 | 未知异常 |

### Acceptance Criteria

- 注册成功后用户入库。
- 密码必须加密保存，不允许明文存储。
- 注册成功后默认 `ai_points = 1000`。
- 注册成功后可以直接返回 token，让用户自动登录。

---

## 4. 用户登录

### Endpoint

```http
POST /api/auth/login
```

### Auth

不需要登录。

### Request Body

```json
{
  "email": "user@example.com",
  "password": "123456"
}
```

### Request Schema

| 字段 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| email | string | 是 | 用户邮箱 |
| password | string | 是 | 用户密码 |

### Success Response

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

### Error Responses

| code | message | 说明 |
|---:|---|---|
| 40002 | 参数校验失败 | 邮箱或密码为空 |
| 40004 | 邮箱或密码错误 | 登录失败 |
| 50000 | 服务器内部错误 | 未知异常 |

### Acceptance Criteria

- 登录成功返回 JWT Token。
- 登录失败不能提示具体是邮箱不存在还是密码错误，统一返回“邮箱或密码错误”。
- 前端将 token 保存到 localStorage 或状态管理中。

---

## 5. 获取当前用户

### Endpoint

```http
GET /api/auth/me
```

### Auth

需要登录。

### Headers

```http
Authorization: Bearer <jwt_token>
```

### Success Response

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "user_001",
    "email": "user@example.com",
    "nickname": "创作者用户",
    "aiPoints": 700,
    "createdAt": "2026-06-06T10:00:00.000Z",
    "updatedAt": "2026-06-06T11:00:00.000Z"
  }
}
```

### Error Responses

| code | message | 说明 |
|---:|---|---|
| 40001 | 未登录或登录已过期 | Token 缺失、错误或过期 |

### Acceptance Criteria

- 可用于前端刷新页面后恢复登录态。
- 不返回 `password_hash`。

---

## 6. 退出登录

### Endpoint

```http
POST /api/auth/logout
```

### Auth

需要登录。

### Success Response

```json
{
  "code": 0,
  "message": "退出登录成功",
  "data": true
}
```

### MVP 实现说明

MVP 阶段后端可直接返回成功，前端删除本地 token 即可。无需实现 token 黑名单。
