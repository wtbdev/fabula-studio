# Fabula Studio — 题目三：AI 小说转剧本工具

AI 辅助剧本创作工具，帮助小说作者将自己的作品改编成剧本，降低改编门槛，提升创作效率。

输入 3 个章节以上的小说文本，系统自动转换为结构化剧本（YAML 格式），让作者可以快速获得可编辑、可进一步打磨的剧本初稿。

> 演示视频：[百度网盘](https://pan.baidu.com/s/1NHJmnF76Zv5Wdxf6lcJ0Yg)

## 功能

- **小说自动分析** — 输入完整小说文本，自动提取角色、场景、情节结构
- **结构化剧本生成** — 基于故事节拍逐步生成剧本，支持大规模并行
- **原文依据映射** — 每场剧本都标注了引用的原文章节，可对照阅读
- **在线编辑工作台** — 浏览器内直接编辑剧本、管理场次、实时预览
- **多格式导出** — 支持 YAML / TXT / Markdown / DOCX 导出
- **全链路可观测** — Jaeger 集成，生成过程实时追踪

## 技术栈

| 层 | 技术 |
|---|---|
| 前端 | Vue 3 + TypeScript + Vite 8 + Naive UI + CodeMirror + Tailwind CSS 4 |
| 后端 | Go 1.26 + Chi router + PostgreSQL + sqlc |
| AI | LLM Agent (OpenAI-compatible) + trpc-agent-go |
| 观测 | OpenTelemetry + Jaeger |
| 打包 | Docker Compose (PostgreSQL + Jaeger + 前后端) |

## 快速开始

### 前置条件

- Docker
- Go 1.26+
- Node.js 22+
- PostgreSQL 17
- LLM API key (OpenAI-compatible)

### 开发环境

```bash
# 启动依赖（PostgreSQL + Jaeger）
docker compose up -d postgres jaeger

# 启动后端
cd apps/backend
cp ../../.env.example .env   # 配置 API key
go run ./cmd/server/

# 启动前端（另一个终端）
cd apps/frontend
npm install
npm run dev
```

### 生产部署

```bash
# 启动全部服务
docker compose up -d

# 或使用启动脚本（开发时同步启动后端子进程）
./start.sh
```

打开 http://localhost 访问。Jaeger UI 在 http://localhost:16686/jaeger/。

## 项目结构

```
├── apps/
│   ├── backend/                # Go 后端
│   │   ├── cmd/server/         # 入口
│   │   ├── internal/
│   │   │   ├── agent/          # AI Agent 定义（分析、编剧本、场景写作等）
│   │   │   ├── converter/      # 小说→剧本转换器（旧批量流程）
│   │   │   ├── graph/          # 动态人物关系图
│   │   │   ├── pipeline/       # 生成流水线（新核心流程）
│   │   │   ├── repo/           # 数据访问层
│   │   │   ├── schema/         # 数据模型定义
│   │   │   ├── scene/          # 场景编排
│   │   │   ├── server/         # HTTP API + SSE 事件流
│   │   │   └── util/           # 工具库（JSON/YAML 修复等）
│   │   ├── db/                 # 数据库迁移 + sqlc 查询
│   │   └── Dockerfile
│   └── frontend/               # Vue 3 前端
│       ├── src/
│       │   ├── api/            # API 客户端
│       │   ├── components/     # 组件（编辑器、场景列表等）
│       │   ├── composables/    # 组合式函数
│       │   ├── views/          # 页面（Main、ProjectEditorView 等）
│       │   └── mock/           # 开发模式 mock
│       ├── nginx.conf          # 生产 nginx 配置
│       └── Dockerfile
├── docs/                       # 架构文档
├── docker-compose.yml          # 编排文件
└── start.sh                    # 开发启动脚本
```

## 架构概览

生成流水线分为 6 步：

1. **提取故事节拍** — 分析原文提取节拍（Story Beats）
2. **规划场景** — 将节拍聚类为场景计划（Scene Plan）
3. **索引原文** — 为场景建立原文引用索引
4. **生成场景** — 并行调用场景写作 Agent 生成每个场景（JSON）
5. **合并剧本** — 按时间线合并所有场景
6. **校验** — 检查一致性和完整性

每步之间通过动态关系图传递角色状态，保证叙事连贯。

详细架构参见 [`docs/novel-to-screenplay-architecture.md`](docs/novel-to-screenplay-architecture.md)。

## 场景数据格式

场景在系统内部以 JSON 存储，导出时转换为 YAML。以下为最终输出的完整结构。

### 顶层结构

```yaml
metadata:
  title: "剧本标题"              # 必需
  author: "作者"                 # 必需
  version: "1.0"                 # 必需
  created_at: "2026-06-07"       # 必需
  original_novel: "小说名称"      # 必需
  logline: "一句话梗概"            # 必需
  genre: ["类型1", "类型2"]       # 必需
  source_chapters: [1, 2, 3]     # 可选，来源章节索引

characters:
  - id: "char_001"               # 必需，角色唯一标识
    name: "角色名"                # 必需
    intro: "角色介绍"              # 可选
    gender: "男/女"               # 可选
    age: 30                      # 可选
    personality: ["关键词"]       # 可选
    relationships:               # 可选
      - target: "char_002"       # 必需（关系存在时）
        type: "朋友/敌人"         # 必需
        description: "描述"       # 可选

scenes:
  - id: "scene_001"              # 必需，场景唯一标识
    sequence: 1                  # 必需，全剧顺序号
    heading: "外景/内景 地点 - 时间" # 必需，场景标题
    setting:
      location: "地点"            # 必需
      time: "时间"               # 必需
      interior: true             # 必需，true=内景 false=外景
    synopsis: "一句话概要"         # 必需
    characters_present:           # 必需，至少 1 个
      - "char_001"
    content:                     # 必需，至少 1 条
      - type: action             # 必需，"action" | "dialogue" | "transition" | "shot"
        text: "动作描述"          # 必需
      - type: dialogue
        character: "char_001"    # 必需（dialogue 类型时）
        parenthetical: "(轻声)"   # 可选，仅 dialogue
        text: "对白内容"          # 必需
      - type: transition
        text: "切至："            # 必需
```

### 元素必要性说明

| 路径 | 必要性 | 说明 |
|---|---|---|
| `metadata.title` | **必需** | 剧本标题 |
| `metadata.author` | **必需** | 作者名 |
| `metadata.version` | **必需** | 固定 `"1.0"` |
| `metadata.created_at` | **必需** | 生成日期 |
| `metadata.original_novel` | **必需** | 来源小说名称 |
| `metadata.logline` | **必需** | 剧情梗概 |
| `metadata.genre` | **必需** | 类型列表，至少 1 项 |
| `metadata.source_chapters` | 可选 | 参考的原文章节索引 |
| `characters[].id` | **必需** | 角色 ID，被 scenes 引用 |
| `characters[].name` | **必需** | 角色全名 |
| `characters[].intro` | 可选 | 角色外貌/性格简介 |
| `characters[].gender` | 可选 | |
| `characters[].age` | 可选 | |
| `characters[].personality` | 可选 | 性格关键词 |
| `characters[].relationships` | 可选 | 角色关系列表 |
| `scenes[].id` | **必需** | 场景 ID，格式 `scene_NNN` |
| `scenes[].sequence` | **必需** | 全剧顺序号，从 1 递增 |
| `scenes[].heading` | **必需** | 标准场景标题 |
| `scenes[].setting.location` | **必需** | 场景地点 |
| `scenes[].setting.time` | **必需** | 场景时间 |
| `scenes[].setting.interior` | **必需** | 内景/外景标识 |
| `scenes[].synopsis` | **必需** | 一句话场景概要 |
| `scenes[].characters_present` | **必需** | 出场角色 ID 列表 |
| `scenes[].content[].type` | **必需** | 取值 `action` / `dialogue` / `transition` / `shot` |
| `scenes[].content[].text` | **必需** | 内容正文 |
| `scenes[].content[].character` | **dialogue 时必需** | 对白角色 ID |
| `scenes[].content[].parenthetical` | 可选 | 对白括号提示，仅用于 `dialogue` |

## 环境变量

| 变量 | 默认值 | 说明 |
|---|---|---|
| `LLM_MODEL` | `deepseek-v4-flash` | 使用的 LLM 模型 |
| `OPENAI_API_KEY` | — | API Key |
| `OPENAI_BASE_URL` | `https://api.openai.com/v1` | API 地址 |
| `DATABASE_URL` | `postgres://fabula:fabula@localhost:5432/fabula?sslmode=disable` | PostgreSQL 连接 |
| `JWT_SECRET` | `fabula-local-development-secret` | JWT 签名密钥 |
| `JWT_TTL` | `24h` | Token 有效期 |
| `LISTEN_ADDR` | `:8080` | 后端监听地址 |
| `OTLP_ENDPOINT` | `localhost:4317` | OpenTelemetry 端点 |
| `PIPELINE_MAX_CONCURRENCY` | `3000` | 生成并发上限 |

## API

API 文档参见 [`docs/api/`](docs/api/)。

## License

MIT
