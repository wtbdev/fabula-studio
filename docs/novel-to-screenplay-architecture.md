# Fabula Studio — 架构设计

## 1. 项目目标

将长篇小说自动转化为可编辑的结构化剧本。核心挑战是长篇小说的上下文过长，直接交给 AI 生成容易遗漏、混乱或泄露后文信息。

最终目标：

- 支持超长小说输入；
- 自动拆解为可处理的剧情单元；
- 维护随剧情推进的人物关系与故事状态；
- 基于剧情单元生成结构化剧本；
- 输出可编辑的剧本初稿（JSON 存储，YAML/多种格式导出）；
- 保持最终 Schema 简洁，不暴露内部分析结构。

## 2. 核心设计思想

整体方案采用"故事节拍 → 场景计划 → 场景上下文包 + 动态关系图"的流水线架构。

### 2.1 故事节拍（Story Beats）

小说文本不是直接交给 AI 写剧本的。系统先让分析 Agent 提取**故事节拍**——每个节拍是一个语义完整的剧情单元（一个事件、一段冲突、一次人物互动），带有顺序、摘要、人物、地点等信息。

节拍是剧本生成的原子单位。

### 2.2 场景规划（Scene Plan）

节拍不直接等于剧本场景。系统根据以下条件将节拍聚类为**场景计划**：

- 同一地点/时间线的节拍合并为一场；
- 冲突阶段变化时拆分；
- 短节拍合并，长节拍独立。

场景计划说明每场的戏剧目的、出场人物、关键情节点。

### 2.3 场景上下文包（Scene Context）

每个场景生成前，系统根据场景计划和动态图组装上下文包，包含：

- 场景规划（目的、地点、时间、人物、关键情节点）
- 相关原文或摘要
- 出场人物信息与当前关系
- 已知事实和未解决的冲突
- 必须保留的剧情点
- 禁止使用的后文信息

上下文包确保每场只看到当前时间点之前的信息。

### 2.4 动态人物关系图

人物关系不是一次性抽取的静态结果，而是随着剧情推进不断更新的动态图。每生成一批场景后，关系图分析 Agent 会更新出场人物状态、关系变化、未解决的冲突。

动态图的关键作用：让每个场景只看到当前时间点之前已经发生的信息。

## 3. 生成流水线

流水线分为 6 步，每步由 pipeline 编排器调度：

### Step 1: 提取故事节拍（Story Beating）

输入完整小说文本，调用 StoryBeatExtractor Agent 提取全部故事节拍。每个节拍包含：

- 节拍 ID
- 摘要
- 出场人物
- 地点
- 原文句子引用

输出：节拍列表（长度通常在数百级别）。

### Step 2: 规划场景（Scene Planning）

将节拍聚类为场景计划。ScenePlanner Agent 根据人物、地点、时间线、冲突阶段对节拍分组，决定：

- 哪些节拍合并为一场
- 每场的戏剧目的
- 每场的人物和地点
- 每场的关键情节点

支持按地点/时间聚类并行加速。

### Step 3: 索引原文（Source Indexing）

建立原文句子级别的索引（SourceIndex），记录每个句子的章节号、文本内容。用于后续场景的原文依据追溯。

### Step 4: 生成场景（Scene Writing）

对每个场景计划，调用 SceneWriter Agent 独立生成场景。支持**大规模并行**（并发数由 `PIPELINE_MAX_CONCURRENCY` 控制）。

每个场景写作 Agent 接收独立的上下文包，输出 JSON 格式场景（2026年6月从 YAML 改为 JSON，消除引号嵌套问题）。

场景结构：

```json
{
  "id": "scene_001",
  "sequence": 1,
  "heading": "外景/内景 地点 - 时间",
  "setting": { "location": "...", "time": "...", "interior": true },
  "synopsis": "一句话概要",
  "characters_present": ["char_001", "char_002"],
  "content": [
    { "type": "action", "text": "..." },
    { "type": "dialogue", "character": "char_001", "parenthetical": "(轻声)", "text": "..." },
    { "type": "transition", "text": "切至：" }
  ]
}
```

### Step 5: 合并剧本（Screenplay Assemble）

按场景顺序合并所有场景为完整剧本。同时更新动态关系图。

### Step 6: 校验（Validation）

对完整剧本进行程序级校验：

- 场景顺序是否连续
- 对白角色是否在 characters_present 中
- characters_present 是否引用合法角色 ID
- 场景内容是否为空

校验失败时系统尝试修复一次，仍失败则返回错误和原始输出。

### 生成产物

生成过程中，流水线通过 SSE 事件流推送实时进度（步骤变更、节拍/场景/角色计数等）。生成完成后，以下产物持久化到数据库：

- 完整剧本（场景列表，每个场景带 rawJson）
- 故事节拍列表
- 场景计划
- 原文索引
- 动态关系图快照

## 4. 当前实现的 Agent 分工

| Agent | 职责 | 输入/输出格式 |
|---|---|---|
| StoryBeatExtractor | 提取故事节拍 | JSON |
| ScenePlanner | 节拍聚类为场景计划 | JSON |
| SceneWriter | 根据上下文包写单场景 | JSON（2026年6月前为 YAML） |
| GraphAnalyzer | 更新关系图状态 | JSON |
| 旧流水线 Analyzer/Writer | 一次性批量转换（已废弃） | JSON（2026年6月前为 YAML） |

## 5. 数据结构

### 持久化层

```text
projects          → 项目主表（标题、原文、配置、状态）
scenes            → 场景表（场景号、标题、地点、时间、概要、rawJson）
generation_jobs   → 生成任务（状态、步骤、产物、错误信息）
users             → 用户表
```

### rawJson 结构

每个场景的 `rawJson` 字段存储完整场景内容（JSON 格式），包含：

- `characters` — 出场角色 ID 列表
- `script` — 场景内容（action/dialogue/transition 数组）
- `source`（新增于 2026年6月）— 原文章节引用（`chapters[]`, `summary`）

### 原文依据

每个场景在生成时从 ScenePlan 的 `SourceRefs` 和 SourceIndex 的 `Sentence` 数据中提取引用的章节号，写入 `rawJson.source.chapters`。前端「原文依据」tab 显示这些引用，可跳转到「原文阅读」tab 查看原始文本。

## 6. 前端架构

前端是 Vue 3 单页应用，主要页面：

- **首页/项目列表**（Main.vue）— 项目 CRUD、生成入口、状态展示
- **编辑器**（ProjectEditorView.vue）— 剧本编辑工作台、场次管理、扩展面板
- **项目创建**（ProjectCreateView.vue）— 新建项目、设置改编参数

编辑器扩展面板包含 4 个 tab：

| Tab | 内容 |
|---|---|
| 场次信息 | 当前场景的概要、地点、时间等元数据 |
| 原文依据 | 当前场景引用的原文章节索引，可点击跳转 |
| 原文阅读 | 项目原始文本，按章节分开展示 |
| 生成日志 | 实时生成进度、事件流（sessionStorage 持久化） |

### 生成状态

- 项目列表页显示每项目的生成状态（`draft` / `generating` / `completed` / `failed`）
- 编辑器顶部显示「正在生成」（加载动画）或「已生成」（绿色勾）
- 生成失败时显示截断的错误信息 + 复制全部按钮
- 生成日志通过 sessionStorage 持久化（按 projectId 隔离），刷新后恢复

## 7. 关键设计决策

### 7.1 为什么用 JSON 而不是 YAML 做 LLM 输出

LLM 输出 YAML 时，字符串值中包含双引号（如中文对话 `"我"`）会被 YAML 解析器错误截断。JSON 没有这个问题——字符串统一用 `"` 包裹，内嵌引号自动转义。

前端导出时再从 JSON 转为 YAML/其他格式。

### 7.2 为什么场景级并行

每个场景的上下文包是独立的——不依赖其他场景的生成结果（只依赖之前的关系图状态）。因此可以大规模并行。并发数由环境变量控制，可根据 API 限频调整。

### 7.3 为什么项目创建后不允许修改

当前实现为一次性转换，创建时设置的改编参数（风格、详细度、粒度等）在生成时固定。后续版本可能支持参数调整后重新生成。

### 7.4 为什么动态图不是核心

动态图是辅助上下文控制的手段，不是项目主体。当前实现中图更新在场景生成后批量执行，不阻塞场景生成。重点始终是：剧情不乱、人物不乱、上下文可控、输出可编辑。

## 8. 技术栈

| 层 | 技术 |
|---|---|
| 前端 | Vue 3 + TypeScript + Vite 8 + Naive UI + CodeMirror 6 + Tailwind CSS 4 |
| 后端 | Go 1.26 + Chi + PostgreSQL 17 + sqlc |
| AI | OpenAI-compatible API + trpc-agent-go |
| 观测 | OpenTelemetry + Jaeger |
| 打包 | Docker Compose |

## 9. 部署

docker-compose.yml 定义 4 个服务：

- **postgres** — 数据库
- **jaeger** — 分布式追踪
- **backend** — Go API 服务
- **frontend** — Nginx + 静态文件

start.sh 提供开发模式一键启动（docker 依赖 + 后端热重载）。
