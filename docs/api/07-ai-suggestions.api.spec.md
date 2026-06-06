# API Spec 04：AI 场景建议

## 1. 功能说明

AI 场景建议接口用于分析当前场次剧本文本，并返回针对对白、冲突、节奏、人物动机和结构的优化建议。

该接口需要消耗 AI 点数。

---

## 2. 数据模型

### SceneSuggestion

```ts
export interface SceneSuggestion {
  id: string
  projectId: string
  sceneId: string
  type: 'dialogue' | 'conflict' | 'rhythm' | 'character' | 'structure' | 'visual'
  title: string
  problem: string
  reason: string
  suggestion: string
  applyText?: string
  status: 'pending' | 'accepted' | 'dismissed'
  createdAt: string
  updatedAt: string
}
```

---

## 3. 数据库设计

```sql
CREATE TABLE scene_suggestions (
  id VARCHAR(64) PRIMARY KEY,
  project_id VARCHAR(64) NOT NULL,
  scene_id VARCHAR(64) NOT NULL,
  type VARCHAR(64) NOT NULL,
  title VARCHAR(255) NOT NULL,
  problem TEXT NOT NULL,
  reason TEXT NOT NULL,
  suggestion TEXT NOT NULL,
  apply_text TEXT,
  status VARCHAR(32) NOT NULL DEFAULT 'pending',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);
```

建议索引：

```sql
CREATE INDEX idx_scene_suggestions_scene_id ON scene_suggestions(scene_id);
CREATE INDEX idx_scene_suggestions_project_id ON scene_suggestions(project_id);
```

---

## 4. 获取场次建议列表

### 接口

```http
GET /api/scenes/:sceneId/suggestions
```

### 鉴权

需要登录。

### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|---|---|---|---|
| sceneId | string | 是 | 场次 ID |

### 查询参数

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|
| status | string | 否 | pending | pending / accepted / dismissed / all |

### 成功响应

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": "suggestion_001",
      "projectId": "project_001",
      "sceneId": "scene_001",
      "type": "conflict",
      "title": "增强场景冲突",
      "problem": "当前场景冲突不足",
      "reason": "两位角色的对白较平，没有体现旧事带来的压力。",
      "suggestion": "建议增加一句带有试探意味的对白，让许晚主动提到过去事件。",
      "applyText": "许晚：你真的只是为了那封信而来吗？",
      "status": "pending",
      "createdAt": "2026-06-07T10:00:00.000Z",
      "updatedAt": "2026-06-07T10:00:00.000Z"
    }
  ]
}
```

---

## 5. 生成场次建议

### 接口

```http
POST /api/scenes/:sceneId/suggestions
```

### 鉴权

需要登录。

### 点数消耗

```txt
30 AI 点数
```

### 请求体

```json
{
  "content": "【第 1 场】旧书店收到来信\n\n地点：雾港旧书店\n时间：雨夜\n\n陈伯：你终于来了。",
  "count": 3
}
```

字段说明：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| content | string | 是 | 当前场次剧本文本 |
| count | number | 否 | 生成建议数量，默认 3 |

### 处理流程

1. 校验用户登录；
2. 校验场次存在；
3. 校验场次所属项目属于当前用户；
4. 校验用户 AI 点数不少于 30；
5. 扣除 30 点；
6. 调用 AI 分析当前场次；
7. 保存建议到 `scene_suggestions` 表；
8. 返回建议列表和剩余点数。

### 成功响应

```json
{
  "code": 0,
  "message": "AI 建议生成成功",
  "data": {
    "costPoints": 30,
    "remainingPoints": 670,
    "suggestions": [
      {
        "id": "suggestion_001",
        "projectId": "project_001",
        "sceneId": "scene_001",
        "type": "conflict",
        "title": "增强场景冲突",
        "problem": "当前场景冲突不足",
        "reason": "角色之间的对白比较平，没有体现隐藏矛盾。",
        "suggestion": "可以增加一句试探性对白，让许晚主动提到过去事件。",
        "applyText": "许晚：你真的只是为了那封信而来吗？",
        "status": "pending",
        "createdAt": "2026-06-07T10:00:00.000Z",
        "updatedAt": "2026-06-07T10:00:00.000Z"
      }
    ]
  }
}
```

---

## 6. 更新建议状态

### 接口

```http
PATCH /api/suggestions/:suggestionId
```

### 请求体

```json
{
  "status": "dismissed"
}
```

字段说明：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| status | string | 是 | accepted / dismissed |

### 成功响应

```json
{
  "code": 0,
  "message": "建议状态已更新",
  "data": {
    "id": "suggestion_001",
    "status": "dismissed",
    "updatedAt": "2026-06-07T10:10:00.000Z"
  }
}
```

---

## 7. AI Prompt 建议

```txt
你是专业编剧顾问。请分析以下剧本场次，从对白、冲突、节奏、人物动机、结构清晰度等角度给出 3 条具体建议。

要求：
1. 输出 JSON；
2. 每条建议包含 type、title、problem、reason、suggestion、applyText；
3. 建议必须具体可操作；
4. 不要输出泛泛而谈的建议。

剧本场次：
{{content}}
```

---

## 8. 错误码

| code | message | 说明 |
|---:|---|---|
| 40001 | 未登录 | token 不存在或失效 |
| 40301 | 无权访问该场次 | 场次不属于当前用户 |
| 40401 | 场次不存在 | sceneId 无效 |
| 40901 | 当前场次内容为空 | 无法生成建议 |
| 40201 | AI 点数不足 | 点数少于 30 |
| 50011 | AI 建议生成失败 | AI 调用失败 |
