# 05. Models Schema 数据结构与数据库设计

## 1. 设计目标

今日 MVP 数据结构需要支撑以下能力：

1. 用户注册登录；
2. 用户 AI 点数扣除；
3. 项目增删改查；
4. 小说文本保存；
5. AI 生成场次保存；
6. 项目编辑页的场次列表和内容保存。

MVP 阶段不单独设计角色表、关系表、版本表。后续可从 `scenes.raw_json` 或 AI 生成结果中扩展。

---

## 2. 数据库表结构

以下 SQL 以通用 SQL/SQLite/MySQL 风格书写，实际字段类型可根据数据库调整。

---

## 2.1 users 用户表

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

### 字段说明

| 字段 | 类型 | 说明 |
|---|---|---|
| id | VARCHAR(64) | 用户 ID，建议使用 UUID 或 nanoid |
| email | VARCHAR(255) | 用户邮箱，唯一 |
| password_hash | VARCHAR(255) | 密码哈希 |
| nickname | VARCHAR(100) | 用户昵称 |
| ai_points | INTEGER | AI 点数，默认 1000 |
| created_at | DATETIME | 创建时间 |
| updated_at | DATETIME | 更新时间 |

### 索引建议

```sql
CREATE UNIQUE INDEX idx_users_email ON users(email);
```

---

## 2.2 projects 项目表

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

### 字段说明

| 字段 | 类型 | 说明 |
|---|---|---|
| id | VARCHAR(64) | 项目 ID |
| user_id | VARCHAR(64) | 所属用户 ID |
| title | VARCHAR(255) | 项目名称 |
| novel_title | VARCHAR(255) | 小说名称 |
| source_text | TEXT | 小说原文 |
| config_json | TEXT | 改编参数 JSON 字符串 |
| status | VARCHAR(32) | draft/generating/completed/failed |
| error_message | TEXT | 生成失败原因 |
| created_at | DATETIME | 创建时间 |
| updated_at | DATETIME | 更新时间 |

### 索引建议

```sql
CREATE INDEX idx_projects_user_id ON projects(user_id);
CREATE INDEX idx_projects_status ON projects(status);
```

---

## 2.3 scenes 场次表

```sql
CREATE TABLE scenes (
  id VARCHAR(64) PRIMARY KEY,
  project_id VARCHAR(64) NOT NULL,
  scene_no INTEGER NOT NULL,
  title VARCHAR(255) NOT NULL,
  location VARCHAR(255),
  time_text VARCHAR(255),
  summary TEXT,
  content TEXT NOT NULL,
  raw_json TEXT,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);
```

### 字段说明

| 字段 | 类型 | 说明 |
|---|---|---|
| id | VARCHAR(64) | 场次 ID |
| project_id | VARCHAR(64) | 所属项目 ID |
| scene_no | INTEGER | 场次序号 |
| title | VARCHAR(255) | 场次标题 |
| location | VARCHAR(255) | 场景地点 |
| time_text | VARCHAR(255) | 场景时间 |
| summary | TEXT | 场景概要 |
| content | TEXT | 可编辑剧本文本 |
| raw_json | TEXT | AI 原始结构化数据 JSON 字符串 |
| created_at | DATETIME | 创建时间 |
| updated_at | DATETIME | 更新时间 |

### 索引建议

```sql
CREATE INDEX idx_scenes_project_id ON scenes(project_id);
CREATE INDEX idx_scenes_project_scene_no ON scenes(project_id, scene_no);
```

---

## 2.4 ai_logs AI 调用日志表，可选

今日可不实现，但建议预留。

```sql
CREATE TABLE ai_logs (
  id VARCHAR(64) PRIMARY KEY,
  user_id VARCHAR(64) NOT NULL,
  project_id VARCHAR(64),
  type VARCHAR(64) NOT NULL,
  prompt TEXT,
  result TEXT,
  cost_points INTEGER NOT NULL DEFAULT 0,
  status VARCHAR(32) NOT NULL,
  error_message TEXT,
  created_at DATETIME NOT NULL
);
```

### 字段说明

| 字段 | 说明 |
|---|---|
| type | AI 调用类型，如 generate_script |
| prompt | 实际发送给 AI 的提示词 |
| result | AI 返回原始结果 |
| cost_points | 本次消耗点数 |
| status | success/failed |
| error_message | 错误信息 |

---

# 3. TypeScript 类型定义

## 3.1 ApiResponse

```ts
export interface ApiResponse<T> {
  code: number
  message: string
  data: T
}
```

## 3.2 PageResult

```ts
export interface PageResult<T> {
  list: T[]
  total: number
  page: number
  pageSize: number
}
```

## 3.3 User

```ts
export interface User {
  id: string
  email: string
  nickname: string
  aiPoints: number
  createdAt?: string
  updatedAt?: string
}
```

## 3.4 AdaptConfig

```ts
export interface AdaptConfig {
  style: '影视剧' | '短剧' | '舞台剧' | '广播剧'
  dialogueLevel: '简略' | '适中' | '详细'
  adaptationMode: '忠实原文' | '适度改编' | '大胆改编'
  sceneGranularity?: '少量大场' | '适中' | '较多小场'
  narrationLevel?: '少量保留' | '适中保留' | '大量保留'
  customPrompt?: string
}
```

## 3.5 Project

```ts
export type ProjectStatus = 'draft' | 'generating' | 'completed' | 'failed'

export interface Project {
  id: string
  userId: string
  title: string
  novelTitle?: string
  sourceText?: string
  config: AdaptConfig
  status: ProjectStatus
  errorMessage?: string | null
  sceneCount?: number
  createdAt: string
  updatedAt: string
}
```

## 3.6 Scene

```ts
export interface Scene {
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

## 3.7 SceneRawJson

```ts
export interface SceneRawJson {
  characters?: string[]
  script?: ScriptBlock[]
  source?: {
    chapters?: string[]
    summary?: string
  }
}
```

## 3.8 ScriptBlock

```ts
export interface ScriptBlock {
  type: 'action' | 'dialogue' | 'narration' | 'voice_over' | 'transition'
  character?: string
  content: string
}
```

---

# 4. 数据关系

```text
users 1 ─── n projects 1 ─── n scenes
```

说明：

1. 一个用户可以有多个项目。
2. 一个项目可以有多个场次。
3. 场次必须属于某一个项目。
4. 用户访问场次时，需要通过 scene → project → user 校验权限。

---

# 5. 今日简化点

1. 不单独建 characters 表。
2. 不单独建 relationships 表。
3. 不单独建 versions 表。
4. 场次编辑只修改 content，不反向解析结构化 script。
5. AI 输出原始结构保存在 raw_json，方便后续扩展。
