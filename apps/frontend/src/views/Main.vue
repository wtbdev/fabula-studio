<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useDialog, useMessage } from 'naive-ui'
import {
  CalendarClock,
  CheckCircle2,
  Clapperboard,
  Coins,
  Edit3,
  FileText,
  FolderKanban,
  LoaderCircle,
  Plus,
  RefreshCw,
  Search,
  Save,
  Trash2,
  WandSparkles,
} from 'lucide-vue-next'
import { generationApi, projectsApi } from '../api'
import { useAuth } from '../composables/useAuth'
import { getFormValidationMessage } from '../utils/formErrors'
import type { FormInst, FormRules } from 'naive-ui'
import type { ProjectDTO, ProjectStatus } from '../api'

const router = useRouter()
const message = useMessage()
const dialog = useDialog()
const { authState, fetchMe } = useAuth()

const projects = ref<ProjectDTO[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const loading = ref(false)
const generatingProjectId = ref('')
const deletingProjectId = ref('')
const searchKeyword = ref('')
let generationPollingTimer: number | null = null
let pollingGenerationStatuses = false


const editVisible = ref(false)
const editLoading = ref(false)
const editFormRef = ref<FormInst | null>(null)
const editingProject = ref<ProjectDTO | null>(null)
const editModel = reactive({
  title: '',
  novelTitle: '',
})

const editRules: FormRules = {
  title: [{ required: true, message: '请输入项目名称', trigger: ['blur', 'input'] }],
}

const statusMeta: Record<ProjectStatus, { label: string }> = {
  draft: { label: '草稿' },
  generating: { label: '生成中' },
  completed: { label: '已完成' },
  failed: { label: '失败' },
}

const projectSummary = computed(() => {
  return {
    total: total.value,
    completed: projects.value.filter((project) => project.status === 'completed').length,
    generating: projects.value.filter((project) => project.status === 'generating').length,
    scenes: projects.value.reduce((sum, project) => sum + (project.sceneCount ?? 0), 0),
  }
})

const isLastProjectPage = computed(() => {
  return total.value > 0 && page.value * pageSize.value >= total.value
})

const projectDescription = (project: ProjectDTO) => {
  if (project.errorMessage) return project.errorMessage
  if (project.status === 'generating') return 'AI 正在拆解章节、人物和场次线索。'
  if (project.status === 'draft') return '还没有生成剧本，可以先完善原文与改编参数。'
  return '剧本工程已生成，可进入编辑器继续打磨场次与对白。'
}

const projectPrimaryActionLabel = (project: ProjectDTO) => {
  if (project.status === 'generating') return '查看进度'
  if (project.status === 'completed') return '编辑剧本'
  if (project.status === 'failed') return '重新生成'
  return '生成剧本'
}

const formatDate = (value?: string) => {
  if (!value) return '未记录'

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value

  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}

const getErrorMessage = (error: unknown) => {
  return error instanceof Error ? error.message : '操作失败，请稍后重试'
}
const isGenerationActive = (status?: string) => status === 'generating' || status === 'queued' || status === 'running'

const stopGenerationPolling = () => {
  if (!generationPollingTimer) return
  window.clearInterval(generationPollingTimer)
  generationPollingTimer = null
}

const pollGenerationStatuses = async () => {
  if (pollingGenerationStatuses) return
  const generatingProjects = projects.value.filter((project) => isGenerationActive(project.status))
  if (generatingProjects.length === 0) {
    stopGenerationPolling()
    return
  }

  pollingGenerationStatuses = true
  try {
    const statuses = await Promise.all(
      generatingProjects.map((project) => generationApi.status(project.id).catch(() => null)),
    )
    const settled = statuses.some((status) => status?.status === 'completed' || status?.status === 'failed')
    if (settled) {
      await Promise.all([fetchProjects(), fetchMe()])
    }
  } finally {
    pollingGenerationStatuses = false
  }
}

const syncGenerationPolling = () => {
  if (!projects.value.some((project) => isGenerationActive(project.status))) {
    stopGenerationPolling()
    return
  }

  if (!generationPollingTimer) {
    generationPollingTimer = window.setInterval(() => {
      void pollGenerationStatuses()
    }, 2500)
  }
}


const fetchProjects = async () => {
  loading.value = true

  try {
    const result = await projectsApi.list({
      page: page.value,
      pageSize: pageSize.value,
      keyword: searchKeyword.value.trim() || undefined,
    })
    projects.value = result.list
    total.value = result.total
    syncGenerationPolling()
  } catch (error) {
    message.error(getErrorMessage(error))
  } finally {
    loading.value = false
  }
}

const handleSearch = async () => {
  page.value = 1
  await fetchProjects()
}

const handleClearSearch = async () => {
  searchKeyword.value = ''
  page.value = 1
  await fetchProjects()
}

const handlePageChange = async (nextPage: number) => {
  page.value = nextPage
  await fetchProjects()
}

const handlePageSizeChange = async (nextPageSize: number) => {
  pageSize.value = nextPageSize
  page.value = 1
  await fetchProjects()
}

const openEditModal = (project: ProjectDTO) => {
  editingProject.value = project
  editModel.title = project.title
  editModel.novelTitle = project.novelTitle ?? ''
  editVisible.value = true
}

const handleUpdateProject = async () => {
  if (!editingProject.value) return

  try {
    await editFormRef.value?.validate()
    editLoading.value = true
    await projectsApi.update(editingProject.value.id, {
      title: editModel.title.trim(),
      novelTitle: editModel.novelTitle.trim() || undefined,
    })
    message.success('项目信息已更新')
    editVisible.value = false
    await fetchProjects()
  } catch (error) {
    const validationMessage = getFormValidationMessage(error)
    message.error(validationMessage || getErrorMessage(error))
  } finally {
    editLoading.value = false
  }
}

const handleDeleteProject = (project: ProjectDTO) => {
  dialog.warning({
    title: '删除项目',
    content: `确定删除「${project.title}」吗？已生成的场次也会一起移除。`,
    positiveText: '删除',
    negativeText: '取消',
    positiveButtonProps: {
      type: 'error',
    },
    onPositiveClick: async () => {
      deletingProjectId.value = project.id

      try {
        await projectsApi.remove(project.id)
        message.success('项目已删除')
        await fetchProjects()
      } catch (error) {
        message.error(getErrorMessage(error))
      } finally {
        deletingProjectId.value = ''
      }
    },
  })
}

const handleGenerateProject = async (project: ProjectDTO) => {
  if (authState.user && authState.user.aiPoints < 1) {
    message.warning('AI 点数不足，无法生成剧本。')
    return
  }

  generatingProjectId.value = project.id

  try {
    await generationApi.generate(project.id, {
      config: project.config,
      adaptationProfile: project.adaptationProfile,
    })
    project.status = 'generating'
    project.errorMessage = null
    project.updatedAt = new Date().toISOString()
    message.success('生成任务已启动')
    void fetchMe()
    await fetchProjects()
    syncGenerationPolling()
  } catch (error) {
    message.error(getErrorMessage(error))
    await fetchProjects()
  } finally {
    generatingProjectId.value = ''
  }
}

const handlePrimaryProjectAction = async (project: ProjectDTO) => {
  if (project.status === 'completed' || project.status === 'generating') {
    await router.push(`/projects/${project.id}/edit`)
    return
  }

  await handleGenerateProject(project)
}

onMounted(() => {
  void fetchProjects()
})

onUnmounted(() => {
  stopGenerationPolling()
})
</script>

<template>
  <section class="project-dashboard">
    <div class="metric-strip">
      <article class="metric-card" data-accent="total">
        <div class="metric-icon">
          <FolderKanban />
        </div>
        <div class="metric-body">
          <span class="metric-label">项目总数</span>
          <strong class="metric-value">{{ projectSummary.total }}</strong>
        </div>
      </article>
      <article class="metric-card" data-accent="completed">
        <div class="metric-icon">
          <CheckCircle2 />
        </div>
        <div class="metric-body">
          <span class="metric-label">已完成</span>
          <strong class="metric-value">{{ projectSummary.completed }}</strong>
        </div>
      </article>
      <article class="metric-card" data-accent="generating">
        <div class="metric-icon">
          <LoaderCircle :class="{ 'project-loading-icon': projectSummary.generating > 0 }" />
        </div>
        <div class="metric-body">
          <span class="metric-label">生成中</span>
          <strong class="metric-value">{{ projectSummary.generating }}</strong>
        </div>
      </article>
      <article class="metric-card" data-accent="scenes">
        <div class="metric-icon">
          <Clapperboard />
        </div>
        <div class="metric-body">
          <span class="metric-label">生成场次</span>
          <strong class="metric-value">{{ projectSummary.scenes }}</strong>
        </div>
      </article>
      <article class="metric-card" data-accent="points">
        <div class="metric-icon">
          <Coins />
        </div>
        <div class="metric-body">
          <span class="metric-label">AI 点数</span>
          <strong class="metric-value">{{ authState.user?.aiPoints ?? 0 }}</strong>
        </div>
      </article>
    </div>

    <n-card class="main-grid project-list-panel" :bordered="false">
      <template #header>
        <div class="panel-heading">
          <span>剧本工程档案</span>
          <small>从原文到场次，保留每个剧本工程的状态线索。</small>
        </div>
      </template>
      <template #header-extra>
        <n-space>
          <n-button size="small" secondary :loading="loading" @click="fetchProjects">
            <template #icon>
              <n-icon><RefreshCw /></n-icon>
            </template>
            刷新
          </n-button>
          <n-button size="small" type="primary" @click="router.push('/projects/new')">
            <template #icon>
              <n-icon><Plus /></n-icon>
            </template>
            新建
          </n-button>
        </n-space>
      </template>

      <div class="project-list-toolbar">
        <n-input
          v-model:value="searchKeyword"
          clearable
          class="project-search"
          placeholder="搜索项目名或小说名"
          @clear="handleClearSearch"
          @keyup.enter="handleSearch"
        >
          <template #prefix>
            <n-icon><Search /></n-icon>
          </template>
        </n-input>
        <n-button type="primary" secondary :loading="loading" @click="handleSearch">
          <template #icon>
            <n-icon><Search /></n-icon>
          </template>
          搜索
        </n-button>
      </div>

      <n-spin :show="loading">
        <n-empty
          v-if="projects.length === 0"
          class="project-empty"
          :description="searchKeyword.trim() ? '没有匹配的剧本工程' : '还没有剧本项目'"
        >
          <template #icon>
            <n-icon><FileText /></n-icon>
          </template>
          <template #extra>
            <n-button type="primary" @click="router.push('/projects/new')">
              <template #icon>
                <n-icon><Plus /></n-icon>
              </template>
              创建第一个项目
            </n-button>
          </template>
        </n-empty>

        <div v-else class="project-card-grid">
          <article
            v-for="project in projects"
            :key="project.id"
            class="project-card"
            :data-status="project.status"
          >
            <div class="project-card-accent" aria-hidden="true" />

            <div class="project-card-body">
              <div class="project-card-head">
                <div class="project-card-title-copy">
                  <div class="project-kicker-row">
                    <span class="project-kicker">剧本工程</span>
                    <span class="project-status-chip" :data-status="project.status">
                      {{ statusMeta[project.status].label }}
                    </span>
                  </div>
                  <h3>{{ project.title }}</h3>
                  <p>原小说：{{ project.novelTitle || '未填写小说名称' }}</p>
                </div>
              </div>

              <p class="project-card-summary" :class="{ error: Boolean(project.errorMessage) }">
                {{ projectDescription(project) }}
              </p>

              <div class="project-meta">
                <span>
                  <n-icon><Clapperboard /></n-icon>
                  {{ project.sceneCount ?? 0 }} 场
                </span>
                <span>
                  <n-icon><CalendarClock /></n-icon>
                  创建：{{ formatDate(project.createdAt) }}
                </span>
                <span>
                  <n-icon><Save /></n-icon>
                  更新：{{ formatDate(project.updatedAt) }}
                </span>
              </div>
            </div>

            <div class="project-card-footer">
              <n-button
                block
                class="project-card-edit-action"
                size="small"
                type="primary"
                :loading="generatingProjectId === project.id"
                @click="handlePrimaryProjectAction(project)"
              >
                <template #icon>
                  <n-icon>
                    <Clapperboard v-if="project.status === 'completed'" />
                    <LoaderCircle v-else-if="project.status === 'generating'" class="project-loading-icon" />
                    <WandSparkles v-else />
                  </n-icon>
                </template>
                {{ projectPrimaryActionLabel(project) }}
              </n-button>

              <div class="project-card-tools">
                <n-button size="tiny" secondary @click="openEditModal(project)">
                  <template #icon>
                    <n-icon><Edit3 /></n-icon>
                  </template>
                  信息
                </n-button>
                <n-button
                  v-if="project.status === 'completed'"
                  size="tiny"
                  secondary
                  :loading="generatingProjectId === project.id"
                  @click="handleGenerateProject(project)"
                >
                  <template #icon>
                    <n-icon><WandSparkles /></n-icon>
                  </template>
                  重新生成
                </n-button>
                <n-button
                  circle
                  class="project-delete-action"
                  size="tiny"
                  tertiary
                  type="error"
                  aria-label="删除项目"
                  :loading="deletingProjectId === project.id"
                  @click="handleDeleteProject(project)"
                >
                  <template #icon>
                    <n-icon><Trash2 /></n-icon>
                  </template>
                </n-button>
              </div>
            </div>
          </article>

          <button
            v-if="isLastProjectPage"
            type="button"
            class="project-create-card"
            @click="router.push('/projects/new')"
          >
            <span class="project-create-card-copy">
              <span class="project-create-card-icon">
                <n-icon><Plus /></n-icon>
              </span>
              <span>
                <span class="project-create-card-title">继续创建新的剧本工程</span>
                <span class="project-create-card-desc">导入新的小说文本，开始整理下一份场次稿。</span>
              </span>
            </span>
            <span class="project-create-card-action">新建项目</span>
          </button>
        </div>
      </n-spin>

      <div v-if="total > pageSize" class="project-pagination">
        <n-pagination
          :page="page"
          :page-size="pageSize"
          :page-sizes="[5, 10, 20]"
          :item-count="total"
          show-size-picker
          @update:page="handlePageChange"
          @update:page-size="handlePageSizeChange"
        />
      </div>
    </n-card>

    <n-modal
      v-model:show="editVisible"
      preset="card"
      title="修改项目信息"
      class="project-edit-modal"
      :bordered="false"
    >
      <n-form ref="editFormRef" :model="editModel" :rules="editRules" label-placement="top">
        <n-form-item label="项目名称" path="title">
          <n-input v-model:value="editModel.title" placeholder="输入剧本项目名称" />
        </n-form-item>
        <n-form-item label="小说名称" path="novelTitle">
          <n-input v-model:value="editModel.novelTitle" placeholder="输入原小说名称" />
        </n-form-item>
      </n-form>

      <template #footer>
        <n-space justify="end">
          <n-button @click="editVisible = false">取消</n-button>
          <n-button type="primary" :loading="editLoading" @click="handleUpdateProject">
            <template #icon>
              <n-icon><Save /></n-icon>
            </template>
            保存
          </n-button>
        </n-space>
      </template>
    </n-modal>
  </section>
</template>
