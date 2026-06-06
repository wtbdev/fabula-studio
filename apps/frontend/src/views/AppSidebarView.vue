<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  CircleUserRound,
  FilePlus2,
  FolderKanban,
  Sparkles,
} from 'lucide-vue-next'
import { useAuth } from '../composables/useAuth'

const route = useRoute()
const router = useRouter()
const { authState } = useAuth()

const navItems = [
  { label: '项目列表', key: 'projects', icon: FolderKanban, route: '/projects' },
  { label: '创建项目', key: 'project-create', icon: FilePlus2, route: '/projects/new' },
  { label: '个人账户', key: 'account', icon: CircleUserRound, route: '/account' },
]

const activeMenuKey = computed(() => {
  return typeof route.name === 'string' ? route.name : 'projects'
})

const aiPoints = computed(() => authState.user?.aiPoints ?? 0)

const handleNav = (item: typeof navItems[number]) => {
  void router.push(item.route)
}
</script>

<template>
  <n-layout-sider
    bordered
    collapse-mode="width"
    :collapsed-width="72"
    :width="248"
    class="app-sider"
  >
    <div class="brand">
      <div class="brand-mark">叙</div>
      <div class="brand-copy">
        <strong>叙幕工作室</strong>
        <span>Fabula Studio</span>
      </div>
    </div>

    <nav class="sider-nav" aria-label="主导航">
      <button
        v-for="item in navItems"
        :key="item.key"
        class="sider-nav-item"
        :class="{ 'is-active': activeMenuKey === item.key }"
        type="button"
        @click="handleNav(item)"
      >
        <span class="sider-nav-icon">
          <component :is="item.icon" />
        </span>
        <span class="sider-nav-label">{{ item.label }}</span>
      </button>
    </nav>

    <div class="sider-footer">
      <div class="sider-footer-header">
        <span class="sider-footer-icon">
          <Sparkles />
        </span>
        <span class="sider-footer-label">AI 点数</span>
      </div>
      <strong class="sider-footer-value">
        {{ aiPoints }}
        <small>点</small>
      </strong>
    </div>
  </n-layout-sider>
</template>
