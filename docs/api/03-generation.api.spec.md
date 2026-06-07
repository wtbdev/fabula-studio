# 03. Generation API Spec AI 剧本生成接口

## 1. 模块说明

生成模块负责将项目中的小说文本转换为可编辑的剧本场次。

今日 MVP 建议采用同步生成方式：前端点击生成后，后端扣除 AI 点数、调用模型、解析结果、保存 scenes，然后返回生成结果。

后续可以扩展为异步任务队列。

---

## 2. 生成剧本

### Endpoint

```http
POST /api/projects/{projectId}/generate
```

### Auth

需要登录。

### Path Params

| 参数 | 类型 | 说明 |
|---|---|---|
| projectId | string | 项目 ID |

### Request Body

MVP 阶段不需要额外参数，直接使用项目中已保存的 `sourceText` 和 `config`。

```json
{}
```

### Cost Rule

固定扣除：`300` AI 点数。

### Process

```text
1. 校验登录态
2. 校验项目归属
3. 校验项目 sourceText 是否存在
4. 校验用户 aiPoints >= 300
5. 扣除 300 点
6. 将项目状态更新为 generating
7. 组装 AI Prompt
8. 调用 AI 生成结构化 JSON
9. 解析 AI 返回结果
10. 将每个 scene 写入 scenes 表
11. 将项目状态更新为 completed
12. 返回 scenes 和剩余点数
```

### Success Response

```json
{
  "code": 0,
  "message": "剧本生成成功",
  "data": {
    "projectId": "project_001",
    "status": "completed",
    "costPoints": 300,
    "remainingPoints": 700,
    "scenes": [
      {
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
    ],
    "artifacts": {
      "sourceIndex": {
        "sentences": [
          {
            "id": "s_000001",
            "index": 1,
            "chapter": 1,
            "chapterIndex": 1,
            "text": "雨水敲打着玻璃。"
          }
        ]
      },
      "storyBeats": [
        {
          "id": "beat_001",
          "sequence": 1,
          "summary": "林舟与许晚在雨夜重逢。",
          "sourceRefs": [{ "sentenceId": "s_000001", "startIndex": 1, "endIndex": 1 }]
        }
      ],
      "scenePlan": {
        "id": "scene_plan",
        "scenes": [
          {
            "id": "plan_001",
            "sequence": 1,
            "purpose": "建立重逢冲突",
            "location": "旧车站",
            "timeFrame": "夜晚",
            "characters": ["林舟", "许晚"]
          }
        ]
      },
      "warnings": []
    }
  }
}
```

`artifacts` 为只读生成过程产物；当前不要求持久化，客户端必须按可选字段处理。

### Error Responses

| code | message | 说明 |
|---:|---|---|
| 40001 | 未登录或登录已过期 | Token 无效 |
| 40401 | 项目不存在 | 项目不存在或不属于当前用户 |
| 41002 | 项目缺少小说文本 | sourceText 为空 |
| 50001 | AI 点数不足 | 用户点数不足 300 |
| 51001 | AI 生成失败 | 模型调用失败 |
| 51002 | AI 返回格式解析失败 | 返回内容不是合法 JSON |

### Acceptance Criteria

- 生成成功后，项目状态必须变成 `completed`。
- 生成失败后，项目状态必须变成 `failed`，并保存 `error_message`。
- 扣点操作需要和生成操作保持一致性。MVP 可先简化处理；如果实现事务，建议扣点、场次写入、项目状态更新放在事务中。
- 生成前如果项目下已有 scenes，可以先删除旧 scenes，再写入新 scenes。

---

## 3. 获取生成状态，可选

### Endpoint

```http
GET /api/projects/{projectId}/generate/status
```

### Auth

需要登录。

### Success Response

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "projectId": "project_001",
    "status": "generating",
    "progress": 60,
    "currentStep": "正在拆分剧本场次",
    "artifacts": null
  }
}
```

`artifacts` 字段可选；仅当当前后端进程仍保留最近一次生成结果时返回。

### MVP 说明

今日如果采用同步生成，可以不实现此接口。前端用 loading 状态即可。

---

## 4. 推荐 AI 返回 JSON 格式

为了降低开发风险，今日不建议直接让 AI 返回 YAML。建议让 AI 返回 JSON，后端解析后入库，后续导出时再转换为 YAML。

### AI 输出格式

```json
{
  "scenes": [
    {
      "sceneNo": 1,
      "title": "雨夜重逢",
      "location": "旧车站",
      "timeText": "夜晚",
      "summary": "林舟与许晚在雨夜重逢。",
      "characters": ["林舟", "许晚"],
      "script": [
        {
          "type": "action",
          "content": "雨水敲打着旧车站的玻璃。"
        },
        {
          "type": "dialogue",
          "character": "许晚",
          "content": "你还是来了。"
        }
      ]
    }
  ]
}
```

### AI Prompt 模板

```text
你是一个专业编剧助手。请根据用户提供的小说文本，将其改编为结构化剧本 JSON。

要求：
1. 只输出 JSON，不要输出 Markdown，不要输出解释。
2. JSON 顶层必须包含 scenes 数组。
3. 每个 scene 必须包含 sceneNo、title、location、timeText、summary、characters、script。
4. script 中每一项必须包含 type 和 content。
5. 如果 type 是 dialogue，必须包含 character。
6. type 只能是 action、dialogue、narration、voice_over、transition 之一。
7. 尽量保留原小说主要剧情。
8. 根据用户改编参数调整风格。

改编参数：
- 改编风格：{{style}}
- 对白详细度：{{dialogueLevel}}
- 改编方式：{{adaptationMode}}
- 场景拆分粒度：{{sceneGranularity}}
- 旁白保留程度：{{narrationLevel}}
- 用户补充要求：{{customPrompt}}

小说文本：
{{sourceText}}
```

---

## 5. 剧本文本转换规则

后端将 AI 返回的 `script` 转换为可编辑的 `content`。

### 转换规则

| type | 转换文本 |
|---|---|
| action | `动作：{content}` |
| dialogue | `{character}：{content}` |
| narration | `旁白：{content}` |
| voice_over | `画外音：{content}` |
| transition | `转场：{content}` |

### 输出格式

```text
【第 {sceneNo} 场】{title}

地点：{location}
时间：{timeText}

动作：...

角色名：对白内容
```

---

## 6. Mock 兜底策略

为了保证演示稳定，如果 AI 调用失败或返回解析失败，可以使用后端内置 mock scenes。

建议仅在开发环境开启 mock 兜底：

```env
AI_MOCK_FALLBACK=true
```

生产环境应返回错误，不应静默使用 mock。
