import { createRouter, createWebHistory } from 'vue-router'
import { useAuth } from '../composables/useAuth'

export const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      redirect: '/home',
    },
    {
      path: '/home',
      name: 'home',
      component: () => import('../views/HomeView.vue'),
      meta: {
        title: '叙幕工作室',
        subtitle: '把小说快速整理成可编辑、可继续打磨的剧本工程',
        publicLayout: true,
      },
    },
    {
      path: '/login',
      name: 'login',
      component: () => import('../views/LoginView.vue'),
      meta: {
        title: '登录',
        subtitle: '使用邮箱和密码继续管理剧本项目',
        guestOnly: true,
        publicLayout: true,
      },
    },
    {
      path: '/register',
      name: 'register',
      component: () => import('../views/RegisterView.vue'),
      meta: {
        title: '注册',
        subtitle: '创建叙幕工作室账号并获得初始 AI 点数',
        guestOnly: true,
        publicLayout: true,
      },
    },
    {
      path: '/projects',
      name: 'projects',
      component: () => import('../views/Main.vue'),
      meta: {
        title: '项目列表',
        subtitle: '管理小说改编工程、AI 生成状态和最近编辑入口',
        requiresAuth: true,
      },
    },
    {
      path: '/projects/new',
      name: 'project-create',
      component: () => import('../views/ProjectCreateView.vue'),
      meta: {
        title: '创建项目',
        subtitle: '填写小说文本、改编参数和高级提示词',
        requiresAuth: true,
      },
    },
    {
      path: '/projects/:id/edit',
      name: 'project-editor',
      component: () => import('../views/ProjectEditorView.vue'),
      meta: {
        title: '剧本编辑器',
        subtitle: '逐场修改 AI 生成的结构化剧本初稿',
        requiresAuth: true,
        workbenchLayout: true,
      },
    },
    {
      path: '/account',
      name: 'account',
      component: () => import('../views/AccountView.vue'),
      meta: {
        title: '个人账户',
        subtitle: '查看登录用户和基础账户资料',
        requiresAuth: true,
      },
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/home',
    },
  ],
})

router.beforeEach(async (to) => {
  const { authState, fetchMe } = useAuth()

  if (authState.token && !authState.user) {
    try {
      await fetchMe()
    } catch {
      if (to.meta.requiresAuth) {
        return {
          path: '/login',
          query: {
            redirect: to.fullPath,
          },
        }
      }
    }
  }

  if (to.meta.requiresAuth && !authState.user) {
    return {
      path: '/login',
      query: {
        redirect: to.fullPath,
      },
    }
  }

  if (to.meta.guestOnly && authState.user) {
    return '/projects'
  }

  return true
})
