<script setup lang="ts">
import { useRouter } from 'vue-router'
import {
  ArrowRight,
  BookOpenText,
  Bot,
  Clapperboard,
  FileText,
  GitBranch,
  Layers3,
  MessageSquareText,
  Sparkles,
  WandSparkles,
} from 'lucide-vue-next'

const router = useRouter()

const featureCards = [
  {
    icon: BookOpenText,
    label: '解析层',
    title: '小说结构解析',
    copy: '章节、人物、地点和关键事件先被拆成清晰索引，改编不再只靠一段长文本硬扛。',
  },
  {
    icon: Clapperboard,
    label: '生成层',
    title: '剧本场次生成',
    copy: '把剧情转成场景、动作、对白和转场，作者拿到的是能继续修改的剧本初稿。',
  },
  {
    icon: GitBranch,
    label: '依据层',
    title: '原文依据映射',
    copy: '每场都能回看对应小说段落，AI 改编从哪里来、改了什么，一眼能查。',
  },
  {
    icon: MessageSquareText,
    label: '打磨层',
    title: '创作式 AI 修改',
    copy: '润色对白、增强冲突、减少旁白、重写本场，都以可预览的建议呈现。',
  },
]

const proofNotes = [
  {
    icon: FileText,
    label: '原文',
    title: '章节来源清晰保留',
    copy: '每个场次都能回到小说片段，不怕 AI 把细节改得无处可查。',
  },
  {
    icon: GitBranch,
    label: '结构',
    title: '角色与事件自动成线',
    copy: '人物、地点、冲突和时间点先形成索引，再进入剧本生成。',
  },
  {
    icon: WandSparkles,
    label: '建议',
    title: '改写动作可预览',
    copy: '对白增强、冲突补足、旁白削减都以建议呈现，作者决定是否采用。',
  },
]

const flowSteps = [
  '上传三章以上小说',
  '设置改编风格',
  'AI 生成场次与角色线索',
  '进入编辑器继续打磨',
]
</script>

<template>
  <main class="home-page">
    <nav class="home-nav" aria-label="首页导航">
      <button class="home-brand" @click="router.push('/home')">
        <span class="brand-mark">叙</span>
        <span>
          <strong>叙幕工作室</strong>
          <small>Fabula Studio</small>
        </span>
      </button>

      <div class="home-nav-actions">
        <n-button text @click="router.push('/login')">登录</n-button>
        <n-button type="primary" @click="router.push('/register')">开始创作</n-button>
      </div>
    </nav>

    <section class="home-hero">
      <div class="hero-copy">
        <n-tag :bordered="false" type="success">
          <template #icon>
            <n-icon><Sparkles /></n-icon>
          </template>
          小说改编剧本工作台
        </n-tag>
        <h1>把小说整理成一份能继续创作的剧本工程</h1>
        <p>
          叙幕工作室面向小说作者、编剧和内容创作者，把章节解析、角色提取、场景拆分和剧本初稿放进同一个清爽工作流里。
        </p>
        <div class="hero-actions">
          <n-button size="large" type="primary" @click="router.push('/register')">
            <template #icon>
              <n-icon><WandSparkles /></n-icon>
            </template>
            创建第一个项目
          </n-button>
          <n-button size="large" secondary @click="router.push('/login')">
            查看项目列表
            <template #icon>
              <n-icon><ArrowRight /></n-icon>
            </template>
          </n-button>
        </div>
      </div>

      <div class="hero-script-stage" aria-hidden="true">
        <div class="script-paper">
          <!-- <div class="paper-grip">
            <span></span>
            <span></span>
          </div> -->

          <div class="script-editor-bar">
            <span>剧本编辑器</span>
            <Clapperboard />
          </div>

          <div class="script-editor-body">
            <div class="scene-kicker">
              <span>第一章 雾港</span>
              <span>依据 86%</span>
            </div>
            <strong>内景 旧书店 - 雨夜</strong>
            <p class="script-action">雨水敲打旧书店的玻璃窗，灯管轻微闪烁。林晚站在门口，手里攥着那封被水汽浸软的信。</p>

            <div class="script-dialogue">
              <span>陈伯</span>
              <p>你终于来了。</p>
            </div>

            <div class="script-dialogue">
              <span>林晚</span>
              <p>这是哪里来的？为什么信封上有我父亲的字？</p>
            </div>

            <div class="script-note">
              <Bot />
              <div>
                <span>AI 建议</span>
                <p>陈伯交信过快，可以增加一次犹豫和追问，让冲突慢半拍出现。</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- <section class="home-metrics">
      <article>
        <strong>3+</strong>
        <span>章节即可启动解析</span>
      </article>
      <article>
        <strong>4</strong>
        <span>核心工作区贯通</span>
      </article>
      <article>
        <strong>300</strong>
        <span>点数生成 MVP 初稿</span>
      </article>
      <article>
        <strong>YAML</strong>
        <span>面向结构化扩展</span>
      </article>
    </section> -->

    <section class="home-proof-band">
      <div class="home-proof-inner">
        <div class="proof-heading">
          <span>改编证据链</span>
          <h2>从原文到场次，线索不丢</h2>
          <p>真正好用的 AI 改编不是黑盒生成，而是把每一次拆解、生成和修改都放回创作者能判断的上下文里。</p>
        </div>

        <div class="proof-track">
          <article
            v-for="(note, index) in proofNotes"
            :key="note.title"
            class="proof-card"
            :style="{ '--entry-delay': `${index * 90 + 100}ms` }"
          >
            <span class="proof-icon">
              <component :is="note.icon" />
            </span>
            <small>{{ note.label }}</small>
            <h3>{{ note.title }}</h3>
            <p>{{ note.copy }}</p>
          </article>
        </div>
      </div>
    </section>

    <section class="home-section">
      <div class="section-heading">
        <n-tag :bordered="false">项目特色</n-tag>
        <h2>不是替作者写完，而是把改编的第一步铺平</h2>
      </div>

      <div class="feature-grid">
        <article
          v-for="(feature, index) in featureCards"
          :key="feature.title"
          class="feature-card"
          :style="{ '--entry-delay': `${index * 80 + 80}ms` }"
        >
          <span class="feature-icon">
            <component :is="feature.icon" />
          </span>
          <small>{{ feature.label }}</small>
          <h3>{{ feature.title }}</h3>
          <p>{{ feature.copy }}</p>
        </article>
      </div>
    </section>

    <section class="home-section workflow-band">
      <div class="section-heading">
        <n-tag :bordered="false" type="success">创作流程</n-tag>
        <h2>从小说文本到剧本工作台，一条线走完</h2>
      </div>

      <div class="flow-list">
        <article
          v-for="(step, index) in flowSteps"
          :key="step"
          :style="{ '--entry-delay': `${index * 70 + 80}ms` }"
        >
          <span>{{ index + 1 }}</span>
          <p>{{ step }}</p>
        </article>
      </div>
    </section>

    <section class="home-section home-cta">
      <div>
        <n-icon><Layers3 /></n-icon>
        <h2>让第一版剧本先站起来</h2>
        <p>把小说、参数、场次和编辑器放进一个工程里，后续角色工作台、Agent 心流和导出能力都能顺势接上。</p>
      </div>
      <n-button size="large" type="primary" @click="router.push('/register')">
        <template #icon>
          <n-icon><FileText /></n-icon>
        </template>
        新建剧本项目
      </n-button>
    </section>
  </main>
</template>
