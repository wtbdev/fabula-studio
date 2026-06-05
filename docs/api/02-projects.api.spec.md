# 02. Projects API Spec 项目管理接口

## 1. 模块说明

项目是叙幕工作室的核心业务对象。一个项目对应一部小说的剧本改编工程。

今日 MVP 需要实现项目的增删改查，以及为后续 AI 生成保存小说文本和改编参数。

---

## 2. 数据模型

### ProjectStatus

```ts
type ProjectStatus = 'draft' | 'generating' | 'completed' | 'failed'
```

### AdaptConfig

```ts
interface AdaptConfig {
  style: '影视剧' | '短剧' | '舞台剧' | '广播剧'
  dialogueLevel: '简略' | '适中' | '详细'
  adaptationMode: '忠实原文' | '适度改编' | '大胆改编'
  sceneGranularity?: '少量大场' | '适中' | '较多小场'
  narrationLevel?: '少量保留' | '适中保留' | '大量保留'
  customPrompt?: string
}
```

### ProjectDTO

```ts
interface ProjectDTO {
  id: string
  userId: string
  title: string
  novelTitle?: string
  sourceText?: string
  config: AdaptConfig
  status: ProjectStatus
  errorMessage?: string
  sceneCount?: number
  createdAt: string
  updatedAt: string
}
```

---

## 3. 创建项目

### Endpoint

```http
POST /api/projects
```

### Auth

需要登录。

### Request Body

```json
{
  "title": "雨夜重逢剧本",
  "novelTitle": "雨夜重逢",
  "sourceText": "第一章......第二章......第三章......",
  "config": {
    "style": "影视剧",
    "dialogueLevel": "适中",
    "adaptationMode": "忠实原文",
    "sceneGranularity": "适中",
    "narrationLevel": "少量保留",
    "customPrompt": "减少旁白，增强对白冲突。"
  }
}
```

### Request Schema

| 字段 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| title | string | 是 | 项目名称 |
| novelTitle | string | 否 | 小说标题 |
| sourceText | string | 是 | 小说原文 |
| config | object | 是 | 改编参数 |

### Success Response

```json
{
  "code": 0,
  "message": "项目创建成功",
  "data": {
    "id": "project_001",
    "title": "雨夜重逢剧本",
    "novelTitle": "雨夜重逢",
    "status": "draft",
    "createdAt": "2026-06-06T10:00:00.000Z",
    "updatedAt": "2026-06-06T10:00:00.000Z"
  }
}
```

### Error Responses

| code | message | 说明 |
|---:|---|---|
| 40001 | 未登录或登录已过期 | Token 无效 |
| 40002 | 参数校验失败 | 缺少 title/sourceText/config |
| 41001 | 小说文本过短 | 不满足 3 章或最低字数要求 |

### Acceptance Criteria

- 项目创建后状态为 `draft`。
- 项目必须归属于当前登录用户。
- 小说原文保存到 `source_text`。
- config 以 JSON 字符串保存到 `config_json`。

---

## 4. 获取项目列表

### Endpoint

```http
GET /api/projects
```

### Auth

需要登录。

### Query Params

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|---|---|---:|---:|---|
| page | number | 否 | 1 | 页码 |
| pageSize | number | 否 | 10 | 每页数量 |
| keyword | string | 否 | - | 按项目名/小说名搜索 |

### Success Response

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": "project_001",
        "title": "雨夜重逢剧本",
        "novelTitle": "雨夜重逢",
        "status": "completed",
        "sceneCount": 8,
        "createdAt": "2026-06-06T10:00:00.000Z",
        "updatedAt": "2026-06-06T11:00:00.000Z"
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 10
  }
}
```

### Acceptance Criteria

- 只返回当前用户自己的项目。
- 列表不需要返回完整 `sourceText`，避免数据过大。
- `sceneCount` 可通过 scenes 表统计，也可以后续优化缓存。

---

## 5. 获取项目详情

### Endpoint

```http
GET /api/projects/{projectId}
```

### Auth

需要登录。

### Path Params

| 参数 | 类型 | 说明 |
|---|---|---|
| projectId | string | 项目 ID |

### Success Response

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "project_001",
    "userId": "user_001",
    "title": "雨夜重逢剧本",
    "novelTitle": "雨夜重逢",
    "sourceText": "第一章......",
    "config": {
      "style": "影视剧",
      "dialogueLevel": "适中",
      "adaptationMode": "忠实原文",
      "sceneGranularity": "适中",
      "narrationLevel": "少量保留",
      "customPrompt": "减少旁白，增强对白冲突。"
    },
    "status": "completed",
    "errorMessage": null,
    "createdAt": "2026-06-06T10:00:00.000Z",
    "updatedAt": "2026-06-06T11:00:00.000Z"
  }
}
```

### Error Responses

| code | message | 说明 |
|---:|---|---|
| 40001 | 未登录或登录已过期 | Token 无效 |
| 40401 | 项目不存在 | 项目不存在或不属于当前用户 |

---

## 6. 修改项目基础信息

### Endpoint

```http
PATCH /api/projects/{projectId}
```

### Auth

需要登录。

### Request Body

```json
{
  "title": "新的项目名称",
  "novelTitle": "新的小说名称"
}
```

### Success Response

```json
{
  "code": 0,
  "message": "项目更新成功",
  "data": {
    "id": "project_001",
    "title": "新的项目名称",
    "novelTitle": "新的小说名称",
    "updatedAt": "2026-06-06T11:10:00.000Z"
  }
}
```

### Acceptance Criteria

- 今日只允许修改 `title` 和 `novelTitle`。
- 不在此接口中修改小说原文和 config。

---

## 7. 删除项目

### Endpoint

```http
DELETE /api/projects/{projectId}
```

### Auth

需要登录。

### Success Response

```json
{
  "code": 0,
  "message": "项目删除成功",
  "data": true
}
```

### Acceptance Criteria

- 删除项目时一并删除该项目下所有 scenes。
- MVP 阶段允许物理删除。
- 如果项目不存在或不属于当前用户，返回 `40401`。
