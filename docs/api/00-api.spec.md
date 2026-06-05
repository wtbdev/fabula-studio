# 00. Fabula Studio API Spec 总览

## 1. 元信息

| 项目 | 内容 |
|---|---|
| API 名称 | 叙幕工作室（Fabula Studio）MVP API |
| 当前版本 | v0.1.0 |
| Base URL | `/api` |
| 鉴权方式 | Bearer Token / JWT |
| 数据格式 | JSON |
| 目标阶段 | 两天 MVP |

---

## 2. API 设计目标

本 API 面向叙幕工作室 MVP，实现以下核心业务闭环：

```text
用户注册登录 → 创建小说项目 → AI 生成剧本场次 → 进入项目编辑器 → 编辑并保存场次内容
```

API 设计遵循以下原则：

1. **先保证主链路可用**：优先支持用户、项目、生成、编辑四大模块。
2. **结构简单，便于开发**：MVP 阶段不引入复杂异步任务队列，生成接口可同步返回结果。
3. **前后端契约清晰**：每个接口明确请求参数、响应结构、错误码。
4. **后续可扩展**：保留项目状态、AI 日志、场次 raw_json 字段，为后续 YAML 导出、角色表、Agent 功能扩展做准备。

---

## 3. 统一请求规范

### 3.1 Content-Type

```http
Content-Type: application/json
```

文件上传暂不设计为 multipart。今日 MVP 中小说文本可以由前端读取 txt 文件后，以 `sourceText` 字段提交。

### 3.2 鉴权请求头

除注册、登录接口外，其余接口都需要携带 Token：

```http
Authorization: Bearer <jwt_token>
```

---

## 4. 统一响应格式

所有接口返回统一格式：

```ts
interface ApiResponse<T> {
  code: number
  message: string
  data: T
}
```

### 成功响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

### 失败响应示例

```json
{
  "code": 40001,
  "message": "未登录或登录已过期",
  "data": null
}
```

---

## 5. 分页规范

列表接口统一使用如下分页参数：

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|---|---|---:|---:|---|
| page | number | 否 | 1 | 当前页码 |
| pageSize | number | 否 | 10 | 每页数量 |
| keyword | string | 否 | - | 搜索关键词 |

分页响应格式：

```ts
interface PageResult<T> {
  list: T[]
  total: number
  page: number
  pageSize: number
}
```

---

## 6. 时间格式

所有时间字段使用 ISO 8601 字符串：

```text
2026-06-06T10:00:00.000Z
```

前端可根据用户时区自行格式化展示。

---

## 7. 状态枚举

### 7.1 项目状态 ProjectStatus

| 状态 | 说明 |
|---|---|
| draft | 草稿，已创建但未生成剧本 |
| generating | 正在生成剧本 |
| completed | 剧本生成完成 |
| failed | 剧本生成失败 |

---

## 8. 今日 MVP 接口清单

### 用户系统

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/auth/register` | 用户注册 |
| POST | `/auth/login` | 用户登录 |
| GET | `/auth/me` | 获取当前用户 |
| POST | `/auth/logout` | 退出登录 |

### 项目管理

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/projects` | 创建项目 |
| GET | `/projects` | 获取项目列表 |
| GET | `/projects/{projectId}` | 获取项目详情 |
| PATCH | `/projects/{projectId}` | 修改项目基础信息 |
| DELETE | `/projects/{projectId}` | 删除项目 |

### AI 生成

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/projects/{projectId}/generate` | AI 生成剧本场次 |
| GET | `/projects/{projectId}/generate/status` | 获取生成状态，可选 |

### 场次编辑

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/projects/{projectId}/scenes` | 获取项目场次列表 |
| GET | `/scenes/{sceneId}` | 获取单个场次详情 |
| PATCH | `/scenes/{sceneId}` | 保存场次内容 |

---

## 9. 鉴权与权限规则

1. 用户只能访问自己创建的项目。
2. 用户只能编辑自己项目下的场次。
3. 项目生成时需要检查用户 AI 点数是否足够。
4. 项目删除后，其下场次应一并删除。
5. 未登录请求受保护接口时返回 `40001`。

---

## 10. AI 点数规则

MVP 阶段使用固定扣点逻辑：

| 操作 | 消耗点数 |
|---|---:|
| AI 生成剧本 | 300 |
| 查询项目/场次 | 0 |
| 编辑保存场次 | 0 |
| 创建项目 | 0 |

点数不足时返回错误码 `50001`。

---

## 11. MVP 暂不实现

以下能力暂不纳入今日 API：

1. 邮箱验证码；
2. 找回密码；
3. 真实支付；
4. DOCX/PDF 导出；
5. Agent 对话；
6. AI 建议；
7. 角色工作台；
8. 复杂异步任务队列；
9. 多人协作；
10. 项目版本历史。
