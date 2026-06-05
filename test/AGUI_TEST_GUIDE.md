# AG-UI Protocol 测试指南

## 快速开始

### 1. 启动后端服务

```bash
cd apps/backend
go run ./cmd/server
```

服务将在 `http://localhost:8080` 启动。

### 2. 测试方式

#### 方式一：HTML 可视化客户端（推荐）

直接在浏览器中打开：

```bash
open test/agui-test.html
```

功能：
- ✅ 实时事件流显示
- ✅ 工具调用可视化
- ✅ 会话管理
- ✅ 统计信息
- ✅ 错误提示

#### 方式二：命令行测试脚本

```bash
# 添加执行权限
chmod +x test/test_agui.sh

# 运行完整测试
./test/test_agui.sh
```

#### 方式三：Node.js 测试客户端

```bash
# 完整测试
node test/test_agui_node.js full

# 单独测试
node test/test_agui_node.js health    # 健康检查
node test/test_agui_node.js session   # 创建会话
node test/test_agui_node.js chat      # 普通聊天
node test/test_agui_node.js stream    # SSE 流式聊天
```

#### 方式四：curl 手动测试

```bash
# 健康检查
curl http://localhost:8080/api/health

# 创建会话
curl -X POST http://localhost:8080/api/sessions \
  -H "Content-Type: application/json" \
  -d '{"user_id": "test-user"}'

# 普通聊天
curl -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "帮我分析一下这个故事"}'

# SSE 流式聊天
curl -N -X POST http://localhost:8080/api/chat/stream \
  -H "Content-Type: application/json" \
  -d '{"message": "简要介绍一下角色关系"}'
```

## API 端点

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/health` | 健康检查 |
| GET | `/api/sessions` | 列出所有会话 |
| POST | `/api/sessions` | 创建新会话 |
| GET | `/api/sessions/{id}` | 获取会话详情 |
| POST | `/api/chat` | 普通聊天（非流式） |
| POST | `/api/chat/stream` | SSE 流式聊天 |

## AG-UI 事件类型

| 事件 | 描述 |
|------|------|
| `RUN_START` | 运行开始 |
| `TEXT_MESSAGE_START` | 文本消息开始 |
| `TEXT_MESSAGE_CONTENT` | 文本内容片段 |
| `TEXT_MESSAGE_END` | 文本消息结束 |
| `TOOL_CALL_START` | 工具调用开始 |
| `TOOL_CALL_ARGS` | 工具调用参数 |
| `TOOL_CALL_END` | 工具调用结束 |
| `RUN_FINISHED` | 运行完成 |
| `RUN_ERROR` | 运行错误 |
| `CUSTOM` | 自定义事件（Pipeline 进度） |

## 示例响应

### 普通聊天响应

```json
{
  "run_id": "run-1234567890",
  "content": "这个故事的主要冲突在于..."
}
```

### SSE 流式事件

```
data: {"type":"RUN_START","run_id":"run-123","timestamp":1234567890}

data: {"type":"TEXT_MESSAGE_START","run_id":"run-123","message_id":"msg-456"}

data: {"type":"TEXT_MESSAGE_CONTENT","run_id":"run-123","message_id":"msg-456","content":"这个"}

data: {"type":"TEXT_MESSAGE_CONTENT","run_id":"run-123","message_id":"msg-456","content":"故事"}

data: {"type":"TOOL_CALL_START","run_id":"run-123","tool_name":"validate_output","tool_id":"tc-789"}

data: {"type":"TOOL_CALL_ARGS","run_id":"run-123","tool_id":"tc-789","args":{"format":"json","content":"..."}}

data: {"type":"TOOL_CALL_END","run_id":"run-123","tool_id":"tc-789"}

data: {"type":"TEXT_MESSAGE_END","run_id":"run-123","message_id":"msg-456"}

data: {"type":"RUN_FINISHED","run_id":"run-123"}
```

## 故障排查

### 1. 连接失败

检查服务是否启动：
```bash
curl http://localhost:8080/api/health
```

### 2. CORS 错误

服务已配置 CORS，如果仍有问题，检查浏览器控制台。

### 3. SSE 流不工作

确保使用正确的请求头：
```bash
curl -N -X POST http://localhost:8080/api/chat/stream \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{"message": "test"}'
```

## 官方 AG-UI SDK

如果你需要在前端项目中集成 AG-UI，可以使用官方 SDK：

```bash
npm install @ag-ui/client
```

示例代码：

```typescript
import { AGUIClient } from '@ag-ui/client';

const client = new AGUIClient({
  baseUrl: 'http://localhost:8080',
});

// 流式聊天
const stream = await client.chatStream({
  message: '帮我分析故事',
  sessionId: 'my-session',
});

for await (const event of stream) {
  switch (event.type) {
    case 'TEXT_MESSAGE_CONTENT':
      console.log(event.content);
      break;
    case 'TOOL_CALL_START':
      console.log('Tool:', event.toolName);
      break;
    // ...
  }
}
```

更多信息: https://docs.ag-ui.com
