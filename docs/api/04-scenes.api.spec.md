# 04. Scenes API Spec 场次编辑接口

## 1. 模块说明

场次编辑接口用于支撑项目编辑页面。

今日 MVP 编辑页面采用左右栏布局：

- 左栏：当前项目的场次列表；
- 右栏：当前选中场次的剧本文本编辑器。

今日不做富文本编辑，不做复杂结构化反向解析，只保存 `content` 字段。

---

## 2. 数据模型

### SceneDTO

```ts
interface SceneDTO {
  id: string
  projectId: string
  sceneNo: number
  title: string
  location?: string
  timeText?: string
  summary?: string
  content: string
  rawJson?: SceneRawJson
  createdAt: string
  updatedAt: string
}
```

### SceneRawJson

```ts
interface SceneRawJson {
  characters?: string[]
  script?: ScriptBlock[]
  source?: {
    chapters?: string[]
    summary?: string
  }
}
```

---

## 3. 获取项目场次列表

### Endpoint

```http
GET /api/projects/{projectId}/scenes
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
  "data": [
    {
      "id": "scene_001",
      "projectId": "project_001",
      "sceneNo": 1,
      "title": "雨夜重逢",
      "location": "旧车站",
      "timeText": "夜晚",
      "summary": "林舟与许晚在雨夜重逢。",
      "content": "【第 1 场】雨夜重逢\n\n地点：旧车站\n时间：夜晚\n\n动作：雨水敲打着玻璃。\n\n许晚：你还是来了。",
      "updatedAt": "2026-06-06T10:10:00.000Z"
    }
  ]
}
```

### Error Responses

| code | message | 说明 |
|---:|---|---|
| 40001 | 未登录或登录已过期 | Token 无效 |
| 40401 | 项目不存在 | 项目不存在或不属于当前用户 |

### Acceptance Criteria

- 只允许获取当前用户自己的项目场次。
- 按 `scene_no ASC` 排序。
- 今日可以直接返回 content，方便前端切换场次时快速展示。

---

## 4. 获取单个场次详情

### Endpoint

```http
GET /api/scenes/{sceneId}
```

### Auth

需要登录。

### Path Params

| 参数 | 类型 | 说明 |
|---|---|---|
| sceneId | string | 场次 ID |

### Success Response

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "scene_001",
    "projectId": "project_001",
    "sceneNo": 1,
    "title": "雨夜重逢",
    "location": "旧车站",
    "timeText": "夜晚",
    "summary": "林舟与许晚在雨夜重逢。",
    "content": "【第 1 场】雨夜重逢\n\n地点：旧车站\n时间：夜晚\n\n动作：雨水敲打着玻璃。\n\n许晚：你还是来了。",
    "rawJson": {
      "characters": ["林舟", "许晚"],
      "script": [
        {
          "type": "action",
          "content": "雨水敲打着玻璃。"
        },
        {
          "type": "dialogue",
          "character": "许晚",
          "content": "你还是来了。"
        }
      ]
    },
    "createdAt": "2026-06-06T10:10:00.000Z",
    "updatedAt": "2026-06-06T10:10:00.000Z"
  }
}
```

### Error Responses

| code | message | 说明 |
|---:|---|---|
| 40001 | 未登录或登录已过期 | Token 无效 |
| 40402 | 场次不存在 | 场次不存在或不属于当前用户项目 |

---

## 5. 保存场次内容

### Endpoint

```http
PATCH /api/scenes/{sceneId}
```

### Auth

需要登录。

### Request Body

今日最小实现只需要传 `content`。

```json
{
  "content": "【第 1 场】雨夜车站重逢\n\n地点：废弃车站\n时间：深夜\n\n动作：雨水顺着站台边缘流下。\n\n许晚：你终于肯来了。"
}
```

如果前端已经提供标题/地点/时间编辑，也可以传完整字段：

```json
{
  "title": "雨夜车站重逢",
  "location": "废弃车站",
  "timeText": "深夜",
  "summary": "林舟和许晚在废弃车站重新见面。",
  "content": "【第 1 场】雨夜车站重逢\n\n地点：废弃车站\n时间：深夜\n\n动作：雨水顺着站台边缘流下。\n\n许晚：你终于肯来了。"
}
```

### Success Response

```json
{
  "code": 0,
  "message": "场次保存成功",
  "data": {
    "id": "scene_001",
    "updatedAt": "2026-06-06T11:00:00.000Z"
  }
}
```

### Error Responses

| code | message | 说明 |
|---:|---|---|
| 40001 | 未登录或登录已过期 | Token 无效 |
| 40002 | 参数校验失败 | content 为空或格式错误 |
| 40402 | 场次不存在 | 场次不存在或不属于当前用户项目 |

### Acceptance Criteria

- 保存时只更新当前场次。
- 不需要反向解析 `content` 到 `rawJson`。
- 保存成功后更新 `updated_at`。
- 切换场次前，前端应调用此接口保存当前场次内容。

---

## 6. 前端交互建议

### 编辑页加载流程

```text
1. 进入 /projects/:projectId/edit
2. 调用 GET /api/projects/:projectId/scenes
3. 默认选中第一个场次
4. 右侧 textarea 显示该场次 content
5. 用户点击左侧其他场次
6. 如果当前 content 有改动，先 PATCH 保存
7. 切换到新场次
```

### 保存策略

今日推荐使用手动保存 + 切换自动保存：

- 点击“保存”按钮：保存当前场次；
- 切换场次前：自动保存当前场次；
- 页面刷新前：可以提示用户保存。

暂不做实时自动保存，降低复杂度。
