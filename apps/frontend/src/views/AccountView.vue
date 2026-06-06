<script setup lang="ts">
import { computed, ref } from 'vue'
import { useMessage } from 'naive-ui'
import {
  CalendarDays,
  Clock3,
  Fingerprint,
  Mail,
  RefreshCw,
  ShieldCheck,
  User,
} from 'lucide-vue-next'
import { useAuth } from '../composables/useAuth'

const message = useMessage()
const { authState, fetchMe } = useAuth()

const refreshing = ref(false)
const aiPoints = computed(() => authState.user?.aiPoints ?? 0)

const formatDate = (value?: string) => {
  if (!value) return '未记录'

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value

  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(date)
}

const formatDateTime = (value?: string) => {
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
  return error instanceof Error ? error.message : '账户信息刷新失败，请稍后重试'
}

const accountFacts = computed(() => [
  { label: '登录邮箱', value: authState.user?.email ?? '-', icon: Mail },
  { label: '注册时间', value: formatDate(authState.user?.createdAt), icon: CalendarDays },
  { label: '账户状态', value: authState.user ? '已登录' : '未登录', icon: ShieldCheck },
])

const profileDetails = computed(() => [
  { label: '昵称', value: authState.user?.nickname ?? '-', icon: User },
  { label: '邮箱', value: authState.user?.email ?? '-', icon: Mail },
  { label: '用户 ID', value: authState.user?.id ?? '-', icon: Fingerprint },
  { label: '注册时间', value: formatDate(authState.user?.createdAt), icon: CalendarDays },
  { label: '资料更新时间', value: formatDateTime(authState.user?.updatedAt), icon: Clock3 },
  { label: '账户状态', value: authState.user ? '已登录' : '未登录', icon: ShieldCheck },
])

const handleRefreshUser = async () => {
  refreshing.value = true

  try {
    await fetchMe()
    message.success('账户信息已刷新')
  } catch (error) {
    message.error(getErrorMessage(error))
  } finally {
    refreshing.value = false
  }
}
</script>

<template>
  <section class="account-page">
    <div class="account-hero-card">
      <div class="account-hero-accent" />
      <div class="account-hero-body">
        <div class="account-avatar">
          <User />
        </div>
        <div class="account-identity">
          <h2>{{ authState.user?.nickname ?? '未登录用户' }}</h2>
          <div class="account-meta-list">
            <span v-for="fact in accountFacts" :key="fact.label" class="account-meta-item">
              <component :is="fact.icon" />
              {{ fact.value }}
            </span>
          </div>
        </div>
      </div>

      <n-button secondary :loading="refreshing" class="account-refresh-btn" @click="handleRefreshUser">
        <template #icon>
          <n-icon><RefreshCw /></n-icon>
        </template>
        刷新账户信息
      </n-button>
    </div>

    <div class="account-grid">
      <n-card class="account-panel account-profile-panel" :bordered="false">
        <template #header>
          <div class="panel-heading">
            <span>个人信息</span>
            <small>基础账户资料。</small>
          </div>
        </template>

        <div class="profile-detail-list">
          <article
            v-for="detail in profileDetails"
            :key="detail.label"
            class="profile-detail-item"
          >
            <span class="profile-detail-icon">
              <component :is="detail.icon" />
            </span>
            <div>
              <span>{{ detail.label }}</span>
              <strong>{{ detail.value }}</strong>
            </div>
          </article>
        </div>
      </n-card>

      <n-card class="account-panel account-points-panel" :bordered="false">
        <template #header>
          <div class="panel-heading">
            <span>AI 点数</span>
            <small>当前账户可用余额。</small>
          </div>
        </template>

        <div class="points-display">
          <div class="points-current">
            <span class="points-label">当前余额</span>
            <strong class="points-value">{{ aiPoints }}</strong>
            <span class="points-unit">点</span>
          </div>
        </div>
      </n-card>
    </div>
  </section>
</template>
