<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import {
  ArrowLeft,
  BookOpenText,
  FileUp,
  ListChecks,
  Save,
  WandSparkles,
} from 'lucide-vue-next'
import { generationApi, projectsApi } from '../api'
import { useAuth } from '../composables/useAuth'
import type { FormInst, FormRules } from 'naive-ui'
import type {
  AdaptationMode,
  AdaptStyle,
  CreateProjectRequest,
  DialogueLevel,
  NarrationLevel,
  SceneGranularity,
} from '../api'

const router = useRouter()
const message = useMessage()
const { authState, fetchMe } = useAuth()

const formRef = ref<FormInst | null>(null)
const activeAction = ref<'draft' | 'generate' | ''>('')

const formModel = reactive({
  title: '',
  novelTitle: '',
  sourceText: '',
  style: '影视剧' as AdaptStyle,
  dialogueLevel: '适中' as DialogueLevel,
  adaptationMode: '忠实原文' as AdaptationMode,
  sceneGranularity: '适中' as SceneGranularity,
  narrationLevel: '少量保留' as NarrationLevel,
  customPrompt: '',
})

const styleOptions = [
  { label: '影视剧', value: '影视剧' },
  { label: '短剧', value: '短剧' },
  { label: '舞台剧', value: '舞台剧' },
  { label: '广播剧', value: '广播剧' },
]

const dialogueOptions: DialogueLevel[] = ['简略', '适中', '详细']
const adaptationOptions: AdaptationMode[] = ['忠实原文', '适度改编', '大胆改编']
const granularityOptions: SceneGranularity[] = ['少量大场', '适中', '较多小场']
const narrationOptions: NarrationLevel[] = ['少量保留', '适中保留', '大量保留']

const sourceStats = computed(() => {
  const chapterCount = formModel.sourceText.match(/第[一二三四五六七八九十百千万\d]+章/g)?.length ?? 0
  const characterCount = formModel.sourceText.trim().length

  return {
    chapterCount,
    characterCount,
  }
})

const canGenerate = computed(() => (authState.user?.aiPoints ?? 0) >= 300)

const rules: FormRules = {
  title: [{ required: true, message: '请输入项目名称', trigger: ['blur', 'input'] }],
  style: [
    {
      validator: (_rule: unknown, value: string) =>
        typeof value === 'string' && value.trim().length > 0,
      message: '请选择或输入改编风格',
      trigger: ['blur', 'change'],
    },
  ],
  sourceText: [
    { required: true, message: '请粘贴或导入小说文本', trigger: ['blur', 'input'] },
    {
      validator: (_rule: unknown, value: string) => {
        const chapterCount = value.match(/第[一二三四五六七八九十百千万\d]+章/g)?.length ?? 0
        return chapterCount >= 3
      },
      message: '请至少提供 3 章文本，便于拆分剧本场次',
      trigger: ['blur', 'input'],
    },
  ],
}

const getErrorMessage = (error: unknown) => {
  return error instanceof Error ? error.message : '操作失败，请稍后重试'
}

const buildPayload = (): CreateProjectRequest => ({
  title: formModel.title.trim(),
  novelTitle: formModel.novelTitle.trim() || undefined,
  sourceText: formModel.sourceText.trim(),
  config: {
    style: formModel.style.trim() as AdaptStyle,
    dialogueLevel: formModel.dialogueLevel,
    adaptationMode: formModel.adaptationMode,
    sceneGranularity: formModel.sceneGranularity,
    narrationLevel: formModel.narrationLevel,
    customPrompt: formModel.customPrompt.trim() || undefined,
  },
})

const createProject = async () => {
  await formRef.value?.validate()
  return projectsApi.create(buildPayload())
}

const handleTextFileChange = async (event: Event) => {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''

  if (!file) return

  if (!file.name.toLowerCase().endsWith('.txt')) {
    message.warning('请导入 txt 文本文件')
    return
  }

  try {
    const text = await file.text()
    formModel.sourceText = text

    const filename = file.name.replace(/\.txt$/i, '')
    if (!formModel.title.trim()) formModel.title = filename
    if (!formModel.novelTitle.trim()) formModel.novelTitle = filename

    message.success('小说文本已导入')
    formRef.value?.restoreValidation()
  } catch {
    message.error('文本读取失败，请重新选择文件')
  }
}

const handleSaveDraft = async () => {
  activeAction.value = 'draft'

  try {
    await createProject()
    message.success('项目草稿已保存')
    await router.push('/projects')
  } catch (error) {
    if (Array.isArray(error)) return
    message.error(getErrorMessage(error))
  } finally {
    activeAction.value = ''
  }
}

const handleCreateAndGenerate = async () => {
  if (!canGenerate.value) {
    message.warning('AI 点数不足，无法生成剧本。')
    return
  }

  activeAction.value = 'generate'

  try {
    const project = await createProject()
    await generationApi.generate(project.id)
    await fetchMe()
    message.success('剧本生成成功，已扣除 300 点')
    await router.push(`/projects/${project.id}/edit`)
  } catch (error) {
    if (Array.isArray(error)) return
    message.error(getErrorMessage(error))
  } finally {
    activeAction.value = ''
  }
}
</script>

<template>
  <section class="project-create-page">
    <div class="create-toolbar">
      <n-button secondary @click="router.push('/projects')">
        <template #icon>
          <n-icon><ArrowLeft /></n-icon>
        </template>
        返回项目列表
      </n-button>
      <n-space align="center">
        <n-tag :bordered="false" type="success">AI 点数 {{ authState.user?.aiPoints ?? 0 }}</n-tag>
      </n-space>
    </div>

    <div class="project-create-layout">
      <n-card class="project-create-form" :bordered="false">
        <template #header>
          <div class="panel-heading">
            <span>项目与原文</span>
            <small>把小说文本整理成可继续编辑的剧本工程。</small>
          </div>
        </template>

        <n-form ref="formRef" :model="formModel" :rules="rules" label-placement="top">
          <div class="create-form-grid">
            <div class="create-form-field">
              <n-form-item label="项目名称" path="title">
                <n-input v-model:value="formModel.title" placeholder="输入剧本项目名称" />
              </n-form-item>
            </div>
            <div class="create-form-field">
              <n-form-item label="小说名称" path="novelTitle">
                <n-input v-model:value="formModel.novelTitle" placeholder="输入原小说名称" />
              </n-form-item>
            </div>
            <div class="create-form-field">
              <n-form-item label="改编风格" path="style">
                <n-select
                  v-model:value="formModel.style"
                  :options="styleOptions"
                  filterable
                  tag
                  placeholder="选择或输入改编风格"
                />
              </n-form-item>
            </div>
            <div class="create-form-field">
              <n-form-item label="对白详细度" path="dialogueLevel">
                <n-radio-group
                  v-model:value="formModel.dialogueLevel"
                  class="create-choice-group"
                  name="dialogueLevel"
                  button-style="solid"
                >
                  <n-radio-button
                    v-for="option in dialogueOptions"
                    :key="option"
                    :value="option"
                    :label="option"
                  />
                </n-radio-group>
              </n-form-item>
            </div>
            <div class="create-form-field">
              <n-form-item label="改编方式" path="adaptationMode">
                <n-radio-group
                  v-model:value="formModel.adaptationMode"
                  class="create-choice-group"
                  name="adaptationMode"
                  button-style="solid"
                >
                  <n-radio-button
                    v-for="option in adaptationOptions"
                    :key="option"
                    :value="option"
                    :label="option"
                  />
                </n-radio-group>
              </n-form-item>
            </div>
            <div class="create-form-field">
              <n-form-item label="场景拆分粒度" path="sceneGranularity">
                <n-radio-group
                  v-model:value="formModel.sceneGranularity"
                  class="create-choice-group"
                  name="sceneGranularity"
                  button-style="solid"
                >
                  <n-radio-button
                    v-for="option in granularityOptions"
                    :key="option"
                    :value="option"
                    :label="option"
                  />
                </n-radio-group>
              </n-form-item>
            </div>
            <div class="create-form-field">
              <n-form-item label="旁白保留程度" path="narrationLevel">
                <n-radio-group
                  v-model:value="formModel.narrationLevel"
                  class="create-choice-group"
                  name="narrationLevel"
                  button-style="solid"
                >
                  <n-radio-button
                    v-for="option in narrationOptions"
                    :key="option"
                    :value="option"
                    :label="option"
                  />
                </n-radio-group>
              </n-form-item>
            </div>
            <div class="create-form-field">
              <n-form-item label="TXT 导入">
                <label class="text-file-control">
                  <input type="file" accept=".txt,text/plain" @change="handleTextFileChange" />
                  <span>
                    <n-icon><FileUp /></n-icon>
                    选择文本文件
                  </span>
                </label>
              </n-form-item>
            </div>
            <div class="create-form-field full">
              <n-form-item label="小说文本" path="sourceText">
                <n-input
                  v-model:value="formModel.sourceText"
                  type="textarea"
                  :autosize="{ minRows: 10, maxRows: 18 }"
                  placeholder="第一章 ......&#10;&#10;第二章 ......&#10;&#10;第三章 ......"
                />
              </n-form-item>
            </div>
            <div class="create-form-field full">
              <n-form-item label="高级提示词" path="customPrompt">
                <n-input
                  v-model:value="formModel.customPrompt"
                  type="textarea"
                  :autosize="{ minRows: 4, maxRows: 8 }"
                  placeholder="例如：减少旁白，增强对白冲突。每场控制在 3 分钟以内。"
                />
              </n-form-item>
            </div>
          </div>

          <n-space justify="end" class="create-actions">
            <n-button @click="router.push('/projects')">取消</n-button>
            <n-button :loading="activeAction === 'draft'" @click="handleSaveDraft">
              <template #icon>
                <n-icon><Save /></n-icon>
              </template>
              保存草稿
            </n-button>
            <n-button
              type="primary"
              :loading="activeAction === 'generate'"
              :disabled="activeAction === 'draft'"
              class="create-generate-action"
              @click="handleCreateAndGenerate"
            >
              <template #icon>
                <n-icon><WandSparkles /></n-icon>
              </template>
              <span class="create-generate-copy">
                <strong>创建并生成</strong>
                <small>消耗 300 点</small>
              </span>
            </n-button>
          </n-space>
        </n-form>
      </n-card>

      <aside class="create-side-panel">
        <div class="create-side-block">
          <span class="side-icon"><n-icon><BookOpenText /></n-icon></span>
          <strong>原文概览</strong>
          <dl>
            <div>
              <dt>章节</dt>
              <dd>{{ sourceStats.chapterCount }}</dd>
            </div>
            <div>
              <dt>字数</dt>
              <dd>{{ sourceStats.characterCount }}</dd>
            </div>
          </dl>
        </div>

        <div class="create-side-block">
          <span class="side-icon brick"><n-icon><ListChecks /></n-icon></span>
          <strong>改编参数</strong>
          <ul>
            <li>{{ formModel.style }} / {{ formModel.adaptationMode }}</li>
            <li>对白：{{ formModel.dialogueLevel }}</li>
            <li>拆分：{{ formModel.sceneGranularity }}</li>
            <li>旁白：{{ formModel.narrationLevel }}</li>
          </ul>
        </div>
      </aside>
    </div>
  </section>
</template>
