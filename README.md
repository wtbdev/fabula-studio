# Fabula Studio

AI 驱动的长篇小说转剧本工具。

## 功能

- **小说分析** — 自动提取角色、场景、情节结构
- **剧本生成** — 基于故事节拍逐步生成结构化剧本
- **递归故事树** — 超长文本分片理解，避免过载
- **动态人物关系图** — 随剧情推进维护角色状态与关系
- **场景上下文包** — 每场景只看到当前时间点之前的信息，防止剧透
- **并行生成** — 场景级并行，支持大量场景快速产出
- **可编辑工作台** — 在线编辑剧本、管理场次、导出多种格式
- **原文阅读** — 对照原文查看改编依据
- **全链路追踪** — Jaeger 集成，支持生成过程调试

## 技术栈

| 层 | 技术 |
|---|---|
| 前端 | Vue 3 + TypeScript + Vite 8 + Naive UI + CodeMirror + Tailwind CSS 4 |
| 后端 | Go 1.26 + Chi router + PostgreSQL + sqlc |
| AI | LLM Agent (OpenAI-compatible) + trpc-agent-go |
| 观测 | OpenTelemetry + Jaeger |
| 打包 | Docker Compose (PostgreSQL + Jaeger + 前后端) |

| AI | LLM Agent (OpenAI-compatible) + trpc-agent-go |

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
