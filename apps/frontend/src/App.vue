<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { dateZhCN, zhCN } from 'naive-ui'
import { CircleUserRound, LogOut, Plus } from 'lucide-vue-next'
import type { GlobalThemeOverrides } from 'naive-ui'
import { useAuth } from './composables/useAuth'
import AppSidebarView from './views/AppSidebarView.vue'

const route = useRoute()
const router = useRouter()
const { authState, logout } = useAuth()

const themeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#2f7664',
    primaryColorHover: '#398973',
    primaryColorPressed: '#245f50',
    primaryColorSuppl: '#3c9a80',
    borderRadius: '8px',
    fontFamily:
      'Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
  },
  Layout: {
    color: '#f4f6f4',
    siderColor: '#17211f',
    headerColor: '#ffffff',
  },
  Card: {
    borderRadius: '8px',
  },
}

const pageTitle = computed(() => {
  return typeof route.meta.title === 'string' ? route.meta.title : '叙幕工作室控制台'
})

const pageSubtitle = computed(() => {
  return typeof route.meta.subtitle === 'string'
    ? route.meta.subtitle
    : 'MVP 主线：注册登录、项目管理、AI 生成与剧本编辑'
})

const useStandaloneLayout = computed(() =>
  Boolean(route.meta.publicLayout || route.meta.workbenchLayout),
)

const handleLogout = async () => {
  await logout()
  await router.push('/login')
}
</script>

<template>
  <n-config-provider :locale="zhCN" :date-locale="dateZhCN" :theme-overrides="themeOverrides">
    <n-message-provider>
      <n-dialog-provider>
        <router-view v-if="useStandaloneLayout" />

        <n-layout v-else class="app-shell" has-sider>
          <AppSidebarView />

          <n-layout class="app-main">
            <n-layout-header bordered class="app-header">
              <div>
                <h1>{{ pageTitle }}</h1>
                <p>{{ pageSubtitle }}</p>
              </div>
              <n-space align="center" class="app-header-actions">
                <n-tag :bordered="false" type="success">
                  <template #icon>
                    <n-icon><CircleUserRound /></n-icon>
                  </template>
                  {{ authState.user?.nickname ?? '创作者用户' }}
                </n-tag>
                <n-button secondary @click="handleLogout">
                  <template #icon>
                    <n-icon><LogOut /></n-icon>
                  </template>
                  退出
                </n-button>
                <n-button type="primary" @click="router.push('/projects/new')">
                  <template #icon>
                    <n-icon><Plus /></n-icon>
                  </template>
                  创建项目
                </n-button>
              </n-space>
            </n-layout-header>

            <n-layout-content class="app-content">
              <router-view />
            </n-layout-content>
          </n-layout>
        </n-layout>
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>
