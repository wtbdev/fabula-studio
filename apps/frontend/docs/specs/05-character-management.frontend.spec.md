# Frontend Feature Spec 05：角色管理

## 1. 功能名称

角色管理工作台

## 2. 功能目标

为每个剧本项目提供角色查看和编辑能力。用户可以查看 AI 从小说与剧本中提取出的角色列表，并编辑角色身份、简介、性格、背景故事和备注。

该功能用于提升项目的结构化程度，让用户在修改剧本时能够同时维护人物设定。

---

## 3. 功能范围

### 本阶段实现

1. 新增角色管理页面；
2. 展示当前项目的角色列表；
3. 查看角色详情；
4. 编辑角色基础信息；
5. 新增角色；
6. 删除角色；
7. 显示角色出现的场次数量；
8. 从项目编辑页跳转到角色管理页。

### 本阶段不实现

1. 复杂人物关系图；
2. AI 自动更新角色关系；
3. 角色头像生成；
4. 角色台词风格分析；
5. 角色与场次的复杂双向绑定编辑。

---

## 4. 页面入口

路由：

```txt
/projects/:id/characters
```

从以下位置进入：

1. 侧边栏菜单“角色管理”；
2. 项目编辑页顶部“角色管理”按钮；
3. 后续可从场次角色卡片跳转。

---

## 5. 页面布局

角色管理页采用左右布局：

```txt
┌────────────────────┬──────────────────────────────┐
│ 左侧：角色列表        │ 右侧：角色详情编辑器             │
└────────────────────┴──────────────────────────────┘
```

左侧宽度建议 320px，右侧自适应。

---

## 6. 左侧角色列表

### 6.1 展示内容

每个角色卡片展示：

1. 角色姓名；
2. 角色身份；
3. 简短描述；
4. 出现场次数。

示例：

```txt
林晚
女主角 · 出现 8 场
继承旧书店的年轻小说作者
```

### 6.2 操作

1. 点击角色后右侧展示详情；
2. 当前选中角色高亮；
3. 顶部提供“新增角色”按钮；
4. 支持空状态。

---

## 7. 右侧角色详情编辑器

### 7.1 字段

| 字段 | 类型 | 是否必填 | 说明 |
|---|---|---|---|
| 角色姓名 | input | 是 | 角色名称 |
| 角色身份 | input/select | 是 | 主角、配角、反派等 |
| 角色简介 | textarea | 是 | 简要说明角色定位 |
| 性格特点 | textarea | 否 | 角色性格与行为风格 |
| 背景故事 | textarea | 否 | 角色过去经历 |
| 角色动机 | textarea | 否 | 角色目标或核心欲望 |
| 作者备注 | textarea | 否 | 用户自由记录 |

### 7.2 操作

1. 保存修改；
2. 删除角色；
3. 返回项目编辑器。

删除角色前需要二次确认。

---

## 8. 新增角色流程

1. 用户点击“新增角色”；
2. 右侧出现空表单；
3. 用户填写姓名和身份；
4. 点击保存；
5. 调用创建角色接口；
6. 创建成功后刷新角色列表并选中新角色。

---

## 9. 删除角色流程

1. 用户点击“删除角色”；
2. 弹出确认框：

```txt
确定删除角色「林晚」吗？该操作不会删除已有剧本文本中的角色名。
```

3. 用户确认；
4. 调用删除接口；
5. 删除成功后刷新列表；
6. 默认选中列表第一项。

---

## 10. 前端状态设计

建议新增 `characterStore`。

```ts
interface CharacterState {
  characters: Character[]
  currentCharacterId: string | null
  loading: boolean
  saving: boolean
}
```

主要方法：

```ts
fetchCharacters(projectId: string): Promise<void>
createCharacter(projectId: string, payload: CharacterCreatePayload): Promise<void>
updateCharacter(characterId: string, payload: CharacterUpdatePayload): Promise<void>
deleteCharacter(characterId: string): Promise<void>
setCurrentCharacter(id: string): void
```

---

## 11. API 依赖

```http
GET /api/projects/:projectId/characters
POST /api/projects/:projectId/characters
GET /api/characters/:characterId
PATCH /api/characters/:characterId
DELETE /api/characters/:characterId
```

---

## 12. 空状态

当项目暂无角色时，显示：

```txt
当前项目暂无角色。
你可以手动创建角色，也可以在重新生成剧本时让 AI 自动提取角色。
```

按钮：

```txt
新增角色
```

---

## 13. 验收标准

1. 用户可以从项目进入角色管理页；
2. 页面能展示当前项目角色列表；
3. 点击角色后能展示角色详情；
4. 用户可以修改角色信息；
5. 修改后刷新页面不丢失；
6. 用户可以新增角色；
7. 用户可以删除角色；
8. 删除角色前有确认提示；
9. 无角色时展示空状态；
10. 角色数量和场次数量展示正确或有兜底值。
