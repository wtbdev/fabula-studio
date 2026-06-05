/**
 * AG-UI Protocol Node.js Test Client
 * 
 * 使用方法:
 *   node test/test_agui_node.js [command]
 * 
 * 命令:
 *   health    - 健康检查
 *   session   - 创建并测试会话
 *   chat      - 普通聊天
 *   stream    - SSE 流式聊天
 *   full      - 完整测试流程
 */

const API_BASE = 'http://localhost:8080';

// 颜色输出
const colors = {
  green: '\x1b[32m',
  red: '\x1b[31m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  reset: '\x1b[0m'
};

function log(color, prefix, message) {
  console.log(`${color}${colors.reset} ${message}`);
}

async function request(method, path, body = null) {
  const options = {
    method,
    headers: { 'Content-Type': 'application/json' }
  };
  if (body) {
    options.body = JSON.stringify(body);
  }
  
  const resp = await fetch(`${API_BASE}${path}`, options);
  return resp.json();
}

async function testHealth() {
  log(colors.yellow, '🏥', '测试健康检查...');
  try {
    const data = await request('GET', '/api/health');
    log(colors.green, '✅', `服务状态: ${data.status}`);
    return true;
  } catch (e) {
    log(colors.red, '❌', `连接失败: ${e.message}`);
    return false;
  }
}

async function createSession() {
  log(colors.yellow, '🆕', '创建新会话...');
  const data = await request('POST', '/api/sessions', { user_id: 'test-user' });
  log(colors.green, '✅', `会话创建成功: ${data.id}`);
  return data.id;
}

async function listSessions() {
  log(colors.yellow, '📋', '列出所有会话...');
  const data = await request('GET', '/api/sessions');
  log(colors.blue, '📊', `共 ${data.length} 个会话`);
  data.forEach(s => console.log(`   - ${s.id} (${s.user_id})`));
}

async function testChat(sessionId) {
  log(colors.yellow, '💬', '测试普通聊天...');
  const data = await request('POST', '/api/chat', {
    session_id: sessionId,
    message: '帮我分析一下这个故事的主要冲突'
  });
  
  if (data.error) {
    log(colors.red, '❌', `错误: ${data.error}`);
  } else {
    log(colors.green, '✅', `响应 (${data.run_id}):`);
    console.log(data.content);
  }
}

async function testStream(sessionId) {
  log(colors.yellow, '📡', '测试 SSE 流式聊天...');
  
  const resp = await fetch(`${API_BASE}/api/chat/stream`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      session_id: sessionId,
      message: '简要介绍一下这个故事的角色关系'
    })
  });

  const reader = resp.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';
  let messageContent = '';
  let toolCalls = [];

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;

    buffer += decoder.decode(value, { stream: true });
    const lines = buffer.split('\n');
    buffer = lines.pop();

    for (const line of lines) {
      if (line.startsWith('data: ')) {
        try {
          const event = JSON.parse(line.slice(6));
          handleStreamEvent(event, messageContent, toolCalls);
        } catch (e) {
          // Skip invalid JSON
        }
      }
    }
  }
  
  log(colors.green, '🏁', '流结束');
  
  if (messageContent) {
    console.log('\n📝 完整响应:');
    console.log(messageContent);
  }
  
  if (toolCalls.length > 0) {
    console.log('\n🔧 工具调用:');
    toolCalls.forEach(tc => {
      console.log(`   - ${tc.name}: ${JSON.stringify(tc.args)}`);
    });
  }
}

function handleStreamEvent(event, messageContent, toolCalls) {
  switch (event.type) {
    case 'RUN_START':
      log(colors.blue, '🚀', `运行开始: ${event.run_id}`);
      break;
    case 'TEXT_MESSAGE_START':
      log(colors.blue, '💬', `消息开始: ${event.message_id}`);
      break;
    case 'TEXT_MESSAGE_CONTENT':
      process.stdout.write(event.content);
      messageContent += event.content;
      break;
    case 'TEXT_MESSAGE_END':
      console.log(); // New line
      log(colors.green, '✅', '消息结束');
      break;
    case 'TOOL_CALL_START':
      log(colors.yellow, '🔧', `工具调用: ${event.tool_name}`);
      toolCalls.push({ name: event.tool_name, id: event.tool_id, args: null });
      break;
    case 'TOOL_CALL_ARGS':
      if (toolCalls.length > 0) {
        toolCalls[toolCalls.length - 1].args = event.args;
      }
      log(colors.yellow, '📋', `参数: ${JSON.stringify(event.args)}`);
      break;
    case 'TOOL_CALL_END':
      log(colors.green, '✔️', '工具完成');
      break;
    case 'RUN_FINISHED':
      log(colors.green, '🏁', '运行完成');
      break;
    case 'RUN_ERROR':
      log(colors.red, '❌', `错误: ${event.error}`);
      break;
    case 'CUSTOM':
      log(colors.blue, '📊', `自定义: ${JSON.stringify(event.meta)}`);
      break;
    default:
      log(colors.yellow, '❓', `未知事件: ${event.type}`);
  }
}

async function fullTest() {
  console.log('🧪 AG-UI Protocol 完整测试');
  console.log('========================\n');
  
  // 1. 健康检查
  const healthy = await testHealth();
  if (!healthy) {
    log(colors.red, '❌', '服务未启动，请先运行: go run ./cmd/server');
    process.exit(1);
  }
  console.log('');
  
  // 2. 创建会话
  const sessionId = await createSession();
  console.log('');
  
  // 3. 列出会话
  await listSessions();
  console.log('');
  
  // 4. 普通聊天
  await testChat(sessionId);
  console.log('');
  
  // 5. 流式聊天
  await testStream(sessionId);
  console.log('');
  
  log(colors.green, '🎉', '所有测试完成！');
}

// 主程序
const command = process.argv[2] || 'full';

(async () => {
  try {
    switch (command) {
      case 'health':
        await testHealth();
        break;
      case 'session':
        await createSession();
        break;
      case 'chat':
        const s1 = await createSession();
        await testChat(s1);
        break;
      case 'stream':
        const s2 = await createSession();
        await testStream(s2);
        break;
      case 'full':
        await fullTest();
        break;
      default:
        console.log('用法: node test/test_agui_node.js [health|session|chat|stream|full]');
    }
  } catch (e) {
    log(colors.red, '❌', `错误: ${e.message}`);
    process.exit(1);
  }
})();
