# API Spec 05：角色管理

## 1. 功能说明

角色管理接口用于管理某个剧本项目中的角色数据，包括获取角色列表、创建角色、查看角色详情、修改角色和删除角色。

角色数据可以来源于 AI 生成结果，也可以由用户手动创建和维护。

---

## 2. 数据模型

```ts
export interface Character {
  id: string
  projectId: string
  name: string
  role: string
  description: string
  personality?: string
  background?: string
  motivation?: string
  firstSceneNo?: number
  sceneCount?: number
  note?: string
  createdAt: string
  updatedAt: string
}
```

---

## 3. 数据库设计

```sql
CREATE TABLE characters (
  id VARCHAR(64) PRIMARY KEY,
  project_id VARCHAR(64) NOT NULL,
  name VARCHAR(255) NOT NULL,
  role VARCHAR(255) NOT NULL,
  description TEXT NOT NULL,
  personality TEXT,
  background TEXT,
  motivation TEXT,
  first_scene_no INTEGER,
  scene_count INTEGER NOT NULL DEFAULT 0,
  note TEXT,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);
```

建议索引：

```sql
CREATE INDEX idx_characters_project_id ON characters(project_id);
CREATE UNIQUE INDEX idx_characters_project_name ON characters(project_id, name);
```

---

## 4. 获取项目角色列表

### 接口

```http
GET /api/projects/:projectId/characters
```

### 鉴权

需要登录。

### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|---|---|---|---|
| projectId | string | 是 | 项目 ID |

### 成功响应

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": "char_001",
      "projectId": "project_001",
      "name": "林晚",
      "role": "女主角",
      "description": "继承旧书店的年轻小说作者。",
      "personality": "敏感、执着、观察力强",
      "background": "父亲失踪多年，留下神秘信件。",
      "motivation": "查明父亲失踪真相。",
      "firstSceneNo": 1,
      "sceneCount": 8,
      "note": "后续需要强化她的主动性。",
      "createdAt": "2026-06-07T10:00:00.000Z",
      "updatedAt": "2026-06-07T10:00:00.000Z"
    }
  ]
}
```

---

## 5. 创建角色

### 接口

```http
POST /api/projects/:projectId/characters
```

### 请求体

```json
{
  "name": "陈伯",
  "role": "配角",
  "description": "旧书店的守店老人。",
  "personality": "沉稳、克制、像是知道很多秘密",
  "background": "曾是林晚父亲的朋友。",
  "motivation": "保护林晚，同时隐瞒部分真相。",
  "note": "可以作为悬疑信息的引导者。"
}
```

### 字段说明

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| name | string | 是 | 角色姓名 |
| role | string | 是 | 角色身份 |
| description | string | 是 | 角色简介 |
| personality | string | 否 | 性格特点 |
| background | string | 否 | 背景故事 |
| motivation | string | 否 | 角色动机 |
| note | string | 否 | 用户备注 |

### 成功响应

```json
{
  "code": 0,
  "message": "角色创建成功",
  "data": {
    "id": "char_002",
    "projectId": "project_001",
    "name": "陈伯",
    "role": "配角",
    "description": "旧书店的守店老人。",
    "personality": "沉稳、克制、像是知道很多秘密",
    "background": "曾是林晚父亲的朋友。",
    "motivation": "保护林晚，同时隐瞒部分真相。",
    "firstSceneNo": null,
    "sceneCount": 0,
    "note": "可以作为悬疑信息的引导者。",
    "createdAt": "2026-06-07T10:10:00.000Z",
    "updatedAt": "2026-06-07T10:10:00.000Z"
  }
}
```

---

## 6. 获取角色详情

### 接口

```http
GET /api/characters/:characterId
```

### 成功响应

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "char_001",
    "projectId": "project_001",
    "name": "林晚",
    "role": "女主角",
    "description": "继承旧书店的年轻小说作者。",
    "personality": "敏感、执着、观察力强",
    "background": "父亲失踪多年，留下神秘信件。",
    "motivation": "查明父亲失踪真相。",
    "firstSceneNo": 1,
    "sceneCount": 8,
    "note": "后续需要强化她的主动性。",
    "createdAt": "2026-06-07T10:00:00.000Z",
    "updatedAt": "2026-06-07T10:00:00.000Z"
  }
}
```

---

## 7. 修改角色

### 接口

```http
PATCH /api/characters/:characterId
```

### 请求体

```json
{
  "role": "女主角",
  "description": "继承旧书店的年轻小说作者，正在调查父亲失踪真相。",
  "personality": "敏感、执着、外冷内热",
  "background": "父亲七年前失踪，只留下了一封未寄出的信。",
  "motivation": "找到父亲留下的线索。",
  "note": "需要在前几场中加强她的主动调查行为。"
}
```

### 成功响应

```json
{
  "code": 0,
  "message": "角色更新成功",
  "data": {
    "id": "char_001",
    "updatedAt": "2026-06-07T10:20:00.000Z"
  }
}
```

---

## 8. 删除角色

### 接口

```http
DELETE /api/characters/:characterId
```

### 成功响应

```json
{
  "code": 0,
  "message": "角色删除成功",
  "data": true
}
```

### 注意

删除角色只删除角色表中的记录，不自动删除剧本文本中已出现的角色姓名。

---

## 9. AI 生成时的角色入库建议

如果 AI 生成剧本时返回 characters 字段，后端可以在生成项目 scenes 后同步写入 characters 表。

AI 返回示例：

```json
{
  "characters": [
    {
      "name": "林晚",
      "role": "女主角",
      "description": "继承旧书店的年轻小说作者。",
      "personality": "敏感、执着、观察力强"
    }
  ],
  "scenes": []
}
```

入库规则：

1. 同一项目内角色名唯一；
2. 如果角色名已存在，则更新描述和场次数量；
3. 如果角色名不存在，则创建新角色。

---

## 10. 错误码

| code | message | 说明 |
|---:|---|---|
| 40001 | 未登录 | token 不存在或失效 |
| 40301 | 无权访问该项目 | 项目不属于当前用户 |
| 40401 | 项目不存在 | projectId 无效 |
| 40402 | 角色不存在 | characterId 无效 |
| 40902 | 角色名已存在 | 同一项目下重复角色名 |
| 42201 | 角色姓名不能为空 | 参数错误 |
| 42202 | 角色身份不能为空 | 参数错误 |
