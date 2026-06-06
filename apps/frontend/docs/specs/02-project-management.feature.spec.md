# Feature Spec 02：项目管理与 AI 剧本生成

## 1. 功能名称

项目管理与 AI 剧本生成

## 2. 功能目标

实现用户对剧本项目的基础管理能力，包括项目创建、项目列表、项目详情、项目修改、项目删除，以及基于小说文本调用 AI 生成剧本场次。

一个项目对应一部小说的剧本改编工程。

## 3. 功能范围

### 包含

1. 创建项目；
2. 查看项目列表；
3. 查看项目详情；
4. 修改项目基础信息；
5. 删除项目；
6. 上传或粘贴小说文本；
7. 设置改编参数；
8. 调用 AI 生成剧本；
9. 扣除固定 AI 点数；
10. 保存生成后的场次数据。

### 不包含

项目分享、多人协作、项目归档、高级搜索筛选、异步任务队列、复杂生成进度、生成历史版本。

## 4. 用户故事

- 作为小说作者，我希望可以创建剧本项目，并上传小说文本，以便让 AI 根据小说生成剧本初稿。
- 作为用户，我希望可以查看自己创建过的项目，以便继续编辑之前的剧本。
- 作为用户，我希望可以修改项目名称和小说名称，以便更好地管理项目。
- 作为用户，我希望可以删除不需要的项目，以便保持项目列表整洁。
- 作为用户，我希望可以点击按钮让 AI 根据小说文本生成剧本场次。

## 5. 页面需求

### 5.1 项目列表页

路由：

```txt
/projects
```

页面内容：

1. 顶栏用户信息；
2. 创建项目按钮；
3. 项目列表；
4. 空状态提示；
5. 删除项目按钮；
6. 进入编辑按钮。

项目卡片展示：

| 信息 | 说明 |
|---|---|
| 项目名称 | 剧本项目名称 |
| 小说名称 | 原小说名称 |
| 状态 | draft / generating / completed / failed |
| 场次数量 | 已生成场次数 |
| 创建时间 | 项目创建时间 |
| 更新时间 | 最近修改时间 |

### 5.2 创建项目页

路由：

```txt
/projects/new
```

页面字段：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| 项目名称 | input | 是 | 当前剧本项目名称 |
| 小说名称 | input | 否 | 原小说标题 |
| 小说文本 | textarea / txt 上传 | 是 | 小说原文 |
| 改编风格 | select | 是 | 影视剧、短剧、舞台剧、广播剧 |
| 对白详细度 | radio | 是 | 简略、适中、详细 |
| 改编方式 | radio | 是 | 忠实原文、适度改编、大胆改编 |
| 场景拆分粒度 | radio | 否 | 少量大场、适中、较多小场 |
| 旁白保留程度 | radio | 否 | 少量保留、适中保留、大量保留 |
| 高级提示词 | textarea | 否 | 用户补充要求 |

按钮：

1. 保存为草稿；
2. 创建并生成剧本；
3. 返回项目列表。

## 6. 项目状态

```ts
type ProjectStatus = 'draft' | 'generating' | 'completed' | 'failed'
```

| 状态 | 说明 |
|---|---|
| draft | 项目已创建，但尚未生成剧本 |
| generating | 正在调用 AI 生成剧本 |
| completed | 剧本生成成功 |
| failed | 剧本生成失败 |

## 7. 点数规则

MVP 阶段固定规则：

```txt
每次生成剧本扣除 300 AI 点数
```

生成前需要检查：

1. 用户是否登录；
2. 项目是否属于当前用户；
3. 用户点数是否大于等于 300；
4. 项目是否存在小说文本。

点数不足时提示：

```txt
AI 点数不足，无法生成剧本。
```

## 8. API 依赖

### 创建项目

```http
POST /api/projects
```

请求体：

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

响应：

```json
{
  "code": 0,
  "message": "项目创建成功",
  "data": {
    "id": "project_001",
    "title": "雨夜重逢剧本",
    "status": "draft",
    "createdAt": "2026-06-06T10:00:00.000Z"
  }
}
```

### 获取项目列表

```http
GET /api/projects?page=1&pageSize=10
```

响应：

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
    "total": 1
  }
}
```

### 获取项目详情

```http
GET /api/projects/:projectId
```

### 修改项目

```http
PATCH /api/projects/:projectId
```

请求体：

```json
{
  "title": "新的项目名称",
  "novelTitle": "新的小说名称"
}
```

### 删除项目

```http
DELETE /api/projects/:projectId
```

### 生成剧本

```http
POST /api/projects/:projectId/generate
```

响应：

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
        "createdAt": "2026-06-06T10:10:00.000Z",
        "updatedAt": "2026-06-06T10:10:00.000Z"
      }
    ]
  }
}
```

## 9. 数据结构

```ts
interface AdaptConfig {
  style: '影视剧' | '短剧' | '舞台剧' | '广播剧'
  dialogueLevel: '简略' | '适中' | '详细'
  adaptationMode: '忠实原文' | '适度改编' | '大胆改编'
  sceneGranularity?: '少量大场' | '适中' | '较多小场'
  narrationLevel?: '少量保留' | '适中保留' | '大量保留'
  customPrompt?: string
}

interface Project {
  id: string
  userId: string
  title: string
  novelTitle?: string
  sourceText: string
  config: AdaptConfig
  status: 'draft' | 'generating' | 'completed' | 'failed'
  errorMessage?: string
  createdAt: string
  updatedAt: string
}
```

## 10. 数据库存储

```sql
CREATE TABLE projects (
  id VARCHAR(64) PRIMARY KEY,
  user_id VARCHAR(64) NOT NULL,
  title VARCHAR(255) NOT NULL,
  novel_title VARCHAR(255),
  source_text TEXT NOT NULL,
  config_json TEXT,
  status VARCHAR(32) NOT NULL DEFAULT 'draft',
  error_message TEXT,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);
```

## 11. AI 生成要求

AI 返回建议使用 JSON，后端解析后保存为 scenes。

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

后端将 script 转换为可编辑文本 content。

## 12. 验收标准

1. 用户可以进入项目列表页；
2. 用户可以创建项目；
3. 创建项目时可以输入小说文本；
4. 创建项目时可以设置改编参数；
5. 用户可以查看自己创建的项目；
6. 用户可以修改项目名称；
7. 用户可以删除项目；
8. 用户可以点击生成剧本；
9. 生成剧本前会校验 AI 点数；
10. 生成剧本后扣除 300 点；
11. 生成成功后项目状态变为 completed；
12. 生成失败后项目状态变为 failed；
13. 生成成功后数据库中产生 scenes 记录；
14. 用户可以从项目列表进入编辑页。
