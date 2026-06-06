# API Spec 07：场次版本历史

## 1. 功能说明

场次版本历史接口用于记录、查看和恢复场次内容的历史版本。每次用户保存场次、应用 AI 重新生成结果或恢复版本时，后端可以创建一条版本记录。

该功能用于降低编辑风险，让用户可以回退到过去版本。

---

## 2. 数据模型

```ts
export interface SceneVersion {
  id: string
  sceneId: string
  projectId: string
  versionNo: number
  title?: string
  content: string
  source: 'manual_save' | 'ai_regenerate' | 'restore'
  createdAt: string
}
```

---

## 3. 数据库设计

```sql
CREATE TABLE scene_versions (
  id VARCHAR(64) PRIMARY KEY,
  scene_id VARCHAR(64) NOT NULL,
  project_id VARCHAR(64) NOT NULL,
  version_no INTEGER NOT NULL,
  title VARCHAR(255),
  content TEXT NOT NULL,
  source VARCHAR(64) NOT NULL DEFAULT 'manual_save',
  created_at DATETIME NOT NULL
);
```

建议索引：

```sql
CREATE INDEX idx_scene_versions_scene_id ON scene_versions(scene_id);
CREATE INDEX idx_scene_versions_project_id ON scene_versions(project_id);
```

---

## 4. 版本创建规则

后端在以下场景自动创建版本：

1. 用户保存场次内容；
2. 用户应用 AI 重新生成内容并保存；
3. 用户恢复某个历史版本。

建议更新 `PATCH /api/scenes/:sceneId` 接口，使其支持 `versionSource` 字段。

### 保存场次请求扩展

```http
PATCH /api/scenes/:sceneId
```

请求体：

```json
{
  "content": "新的场次内容",
  "versionSource": "manual_save"
}
```

`versionSource` 可选值：

```txt
manual_save | ai_regenerate | restore
```

如果未传，默认为 `manual_save`。

---

## 5. 获取场次版本列表

### 接口

```http
GET /api/scenes/:sceneId/versions
```

### 鉴权

需要登录。

### 查询参数

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|
| page | number | 否 | 1 | 页码 |
| pageSize | number | 否 | 20 | 每页数量 |

### 成功响应

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": "version_005",
        "sceneId": "scene_001",
        "projectId": "project_001",
        "versionNo": 5,
        "title": "v5 手动保存",
        "source": "manual_save",
        "createdAt": "2026-06-07T11:00:00.000Z"
      },
      {
        "id": "version_004",
        "sceneId": "scene_001",
        "projectId": "project_001",
        "versionNo": 4,
        "title": "v4 AI 重生成",
        "source": "ai_regenerate",
        "createdAt": "2026-06-07T10:40:00.000Z"
      }
    ],
    "total": 5
  }
}
```

注意：列表接口可以不返回完整 content，避免数据过大。

---

## 6. 获取单个版本详情

### 接口

```http
GET /api/scene-versions/:versionId
```

### 成功响应

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "version_005",
    "sceneId": "scene_001",
    "projectId": "project_001",
    "versionNo": 5,
    "title": "v5 手动保存",
    "content": "【第 1 场】旧书店收到来信\n\n地点：雾港旧书店\n时间：雨夜\n\n陈伯：你终于来了。",
    "source": "manual_save",
    "createdAt": "2026-06-07T11:00:00.000Z"
  }
}
```

---

## 7. 恢复历史版本

### 接口

```http
POST /api/scenes/:sceneId/restore/:versionId
```

### 处理流程

1. 校验用户登录；
2. 校验场次存在；
3. 校验版本存在；
4. 校验场次和版本属于同一项目；
5. 校验项目属于当前用户；
6. 将版本 content 写回 scenes.content；
7. 创建一条新的 `restore` 类型版本；
8. 返回更新后的场次内容。

### 成功响应

```json
{
  "code": 0,
  "message": "历史版本已恢复",
  "data": {
    "sceneId": "scene_001",
    "content": "【第 1 场】旧书店收到来信\n\n地点：雾港旧书店\n时间：雨夜\n\n陈伯：你终于来了。",
    "restoredFromVersionId": "version_003",
    "newVersionId": "version_006",
    "updatedAt": "2026-06-07T11:20:00.000Z"
  }
}
```

---

## 8. 版本号生成规则

建议按场次递增：

```txt
同一个 sceneId 下 versionNo 从 1 开始递增
```

创建新版本时：

```sql
SELECT MAX(version_no) FROM scene_versions WHERE scene_id = ?
```

新版本号为最大值 + 1。

---

## 9. 错误码

| code | message | 说明 |
|---:|---|---|
| 40001 | 未登录 | token 不存在或失效 |
| 40301 | 无权访问该场次 | 场次不属于当前用户 |
| 40401 | 场次不存在 | sceneId 无效 |
| 40403 | 历史版本不存在 | versionId 无效 |
| 40903 | 版本与场次不匹配 | 版本不属于该场次 |
| 50013 | 版本恢复失败 | 数据库或服务异常 |
