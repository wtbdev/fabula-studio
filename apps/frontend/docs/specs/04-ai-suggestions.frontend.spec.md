# Frontend Feature Spec 04：AI 场景建议

## 1. 功能名称

AI 场景建议面板

## 2. 功能目标

在项目编辑页中为当前场次提供 AI 创作建议。用户可以点击按钮让 AI 分析当前场次内容，返回具体、可操作的优化建议，帮助用户从对白、冲突、节奏、人物动机和结构等角度打磨剧本。

该功能用于增强产品的 AI 辅助创作属性，让叙幕工作室不只是一次性生成器，而是持续辅助编辑的创作工作台。

---

## 3. 功能范围

### 本阶段实现

1. 在编辑页增加右侧 AI 建议面板；
2. 展示当前场次的建议列表；
3. 支持点击“生成 AI 建议”；
4. 生成建议前扣除 AI 点数；
5. 展示建议标题、问题、原因和修改建议；
6. 支持刷新建议；
7. 支持忽略建议；
8. 支持复制建议内容。

### 本阶段不实现

1. 自动应用修改；
2. 多建议批量应用；
3. AI 建议与编辑器 diff 对比；
4. 建议版本管理；
5. 根据角色表深度分析人物关系。

---

## 4. 页面入口

该功能集成在项目编辑页：

```txt
/projects/:id/edit
```

编辑页从左右栏升级为三栏：

```txt
┌──────────────┬────────────────────────┬──────────────────┐
│ 场次列表       │ 剧本编辑器               │ AI 建议面板         │
└──────────────┴────────────────────────┴──────────────────┘
```

如果屏幕宽度较小，右侧 AI 建议面板可以折叠为抽屉。

---

## 5. UI 布局

### 5.1 右侧建议面板

面板标题：

```txt
AI 场景建议
```

面板内容：

1. 当前场次标题；
2. “生成建议”按钮；
3. 点数消耗提示；
4. 建议列表；
5. 空状态；
6. 加载状态；
7. 错误状态。

---

## 6. 交互流程

### 6.1 首次进入场次

1. 用户进入编辑页；
2. 默认选中第一场；
3. 右侧建议面板显示空状态；
4. 空状态文案：

```txt
当前场次暂无 AI 建议。
点击生成建议，让 AI 帮你检查对白、节奏和人物动机。
```

---

### 6.2 生成 AI 建议

1. 用户点击“生成 AI 建议”；
2. 前端检查当前场次是否存在内容；
3. 前端弹出确认提示：

```txt
本次生成建议将消耗 30 AI 点数，是否继续？
```

4. 用户确认后调用接口；
5. 按钮进入 loading 状态；
6. 请求成功后展示建议列表；
7. 更新用户 AI 点数；
8. 请求失败时展示错误提示。

---

### 6.3 切换场次

1. 用户点击左侧其他场次；
2. 编辑器切换内容；
3. AI 建议面板切换到对应场次的建议；
4. 若该场次没有建议，则展示空状态。

---

### 6.4 忽略建议

1. 用户点击某条建议上的“忽略”；
2. 前端调用更新建议状态接口；
3. 建议从当前列表中隐藏，或标记为已忽略。

---

### 6.5 复制建议

1. 用户点击“复制”；
2. 将建议内容复制到剪贴板；
3. 显示提示：

```txt
已复制建议内容
```

---

## 7. 组件设计

建议组件结构：

```txt
components/editor/
  AiSuggestionPanel.vue
  AiSuggestionCard.vue
```

### AiSuggestionPanel.vue

职责：

1. 接收当前 sceneId；
2. 获取当前场次建议；
3. 触发生成建议；
4. 管理 loading 和 error 状态；
5. 渲染建议列表。

Props：

```ts
interface Props {
  projectId: string
  sceneId: string
  sceneContent: string
}
```

Emits：

```ts
interface Emits {
  refreshUser: []
}
```

### AiSuggestionCard.vue

职责：

1. 展示单条建议；
2. 处理复制；
3. 处理忽略。

Props：

```ts
interface Props {
  suggestion: SceneSuggestion
}
```

---

## 8. 前端状态设计

建议在 `editorStore` 或独立 `suggestionStore` 中管理。

```ts
interface SuggestionState {
  suggestionsBySceneId: Record<string, SceneSuggestion[]>
  loadingSceneIds: string[]
}
```

主要方法：

```ts
fetchSuggestions(sceneId: string): Promise<void>
generateSuggestions(sceneId: string, content: string): Promise<void>
dismissSuggestion(suggestionId: string): Promise<void>
```

---

## 9. API 依赖

```http
GET /api/scenes/:sceneId/suggestions
POST /api/scenes/:sceneId/suggestions
PATCH /api/suggestions/:suggestionId
```

---

## 10. 错误处理

| 场景 | 前端提示 |
|---|---|
| 当前场次内容为空 | 当前场次没有可分析内容 |
| 点数不足 | AI 点数不足，无法生成建议 |
| 接口超时 | AI 分析时间较长，请稍后重试 |
| 生成失败 | 建议生成失败，请重新尝试 |
| 未登录 | 登录状态已失效，请重新登录 |

---

## 11. 验收标准

1. 编辑页右侧可以看到 AI 建议面板；
2. 用户可以为当前场次生成建议；
3. 生成建议时按钮显示 loading；
4. 生成成功后能展示至少 3 条建议；
5. 每条建议包含标题、问题、原因和建议；
6. 生成建议后用户 AI 点数更新；
7. 切换场次时建议面板同步切换；
8. 用户可以忽略建议；
9. 用户可以复制建议内容；
10. 点数不足时无法生成建议，并显示提示。
