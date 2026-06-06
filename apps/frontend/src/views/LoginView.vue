<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import {
  ArrowRight,
  AtSign,
  KeyRound,
  LockKeyhole,
  LogIn,
} from 'lucide-vue-next'
import { useAuth } from '../composables/useAuth'
import { getFormValidationMessage } from '../utils/formErrors'
import type { FormInst, FormRules } from 'naive-ui'

const route = useRoute()
const router = useRouter()
const message = useMessage()
const { login } = useAuth()

const formRef = ref<FormInst | null>(null)
const loading = ref(false)

const formModel = reactive({
  email: '',
  password: '',
})

const emailInputProps = { autocomplete: 'email', inputmode: 'email' }
const passwordInputProps = { autocomplete: 'current-password' }
const authCardContentStyle = { padding: '0' }
const demoAccount = {
  email: 'writer@example.com',
  password: '123456',
}
const showDemoAccount = import.meta.env.DEV

const rules: FormRules = {
  email: [
    { required: true, message: '请输入邮箱', trigger: ['blur', 'input'] },
    {
      validator: (_rule: unknown, value: string | undefined) => {
        const normalizedValue = value?.trim() ?? ''
        return normalizedValue === '' || /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(normalizedValue)
      },
      message: '请输入正确的邮箱格式',
      trigger: ['blur', 'input'],
    },
  ],
  password: [{ required: true, message: '请输入密码', trigger: ['blur', 'input'] }],
}

const getErrorMessage = (error: unknown) => {
  return error instanceof Error ? error.message : '登录失败，请稍后重试'
}

const resolveRedirectTarget = () => {
  const redirect = route.query.redirect

  if (typeof redirect !== 'string' || !redirect.startsWith('/') || redirect.startsWith('//')) {
    return '/projects'
  }

  const pathname = redirect.split(/[?#]/)[0]
  return pathname === '/login' || pathname === '/register' ? '/projects' : redirect
}

const fillDemoAccount = () => {
  formModel.email = demoAccount.email
  formModel.password = demoAccount.password
  formRef.value?.restoreValidation()
}

const handleLogin = async () => {
  if (loading.value) return

  try {
    await formRef.value?.validate()
    loading.value = true
    await login({
      email: formModel.email.trim(),
      password: formModel.password,
    })
    message.success('登录成功')
    await router.push(resolveRedirectTarget())
  } catch (error) {
    const validationMessage = getFormValidationMessage(error)
    message.error(validationMessage || getErrorMessage(error))
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <main class="auth-page auth-login-page" aria-labelledby="login-title">
    <section class="auth-panel auth-login-panel">
      <aside class="auth-copy">
        <button class="auth-home-link" type="button" aria-label="返回首页" @click="router.push('/home')">
          <span class="brand-mark">叙</span>
          <span>
            <strong>叙幕工作室</strong>
            <small>Fabula Studio</small>
          </span>
        </button>

        <div class="auth-copy-main">
          <span class="auth-kicker">创作从这里继续</span>
          <h2>
            把剧本工程
            <br />
            重新带回桌面
          </h2>
          <p>登录以继续你的创作之旅。</p>
        </div>

        <!-- <div class="auth-proof-list" aria-label="登录后保留的内容">
          <span>
            <n-icon><ShieldCheck /></n-icon>
            登录后下次免登录
          </span>
          <span>
            <n-icon><FileText /></n-icon>
            项目列表同步
          </span>
          <span>
            <n-icon><Sparkles /></n-icon>
            AI 点数保留
          </span>
        </div> -->

        <!-- <div class="auth-script-preview" aria-hidden="true">
          <div class="auth-script-topline">
            <span>雾港来信</span>
          </div>
          <strong>雾港来信</strong>
          <p>INT. 旧书店 - 雨夜</p>
          <div>
            <span>场次 01</span>
            <span>旧书店收到来信</span>
          </div>
          <div>
            <span>场次 02</span>
            <span>林家老宅重启谜团</span>
          </div>
        </div> -->
      </aside>

      <section class="auth-form-panel" aria-labelledby="login-title">
        <n-card
          :bordered="false"
          class="auth-card auth-login-card"
          :content-style="authCardContentStyle"
        >
          <div class="auth-card-header">
            <n-tag size="small" :bordered="false" type="success">邮箱登录</n-tag>
            <h1 id="login-title">欢迎回来</h1>
            <p>登录以打磨你的小说改编项目。</p>
          </div>

          <div class="auth-card-body">
            <n-form ref="formRef" :model="formModel" :rules="rules" label-placement="top">
              <n-form-item first label="邮箱" path="email">
                <n-input
                  v-model:value="formModel.email"
                  clearable
                  placeholder="writer@example.com"
                  :disabled="loading"
                  :input-props="emailInputProps"
                  @keydown.enter.prevent="handleLogin"
                >
                  <template #prefix>
                    <n-icon><AtSign /></n-icon>
                  </template>
                </n-input>
              </n-form-item>
              <n-form-item first label="密码" path="password">
                <n-input
                  v-model:value="formModel.password"
                  type="password"
                  show-password-on="click"
                  placeholder="请输入密码"
                  :disabled="loading"
                  :input-props="passwordInputProps"
                  @keydown.enter.prevent="handleLogin"
                >
                  <template #prefix>
                    <n-icon><LockKeyhole /></n-icon>
                  </template>
                </n-input>
              </n-form-item>

              <div v-if="showDemoAccount" class="auth-demo">
                <div>
                  <span>本地试用账号</span>
                  <strong>{{ demoAccount.email }}</strong>
                </div>
                <n-button size="small" tertiary :disabled="loading" @click="fillDemoAccount">
                  <template #icon>
                    <n-icon><KeyRound /></n-icon>
                  </template>
                  填入
                </n-button>
              </div>

              <n-button
                class="auth-submit"
                block
                type="primary"
                :loading="loading"
                @click="handleLogin"
              >
                <template #icon>
                  <n-icon><LogIn /></n-icon>
                </template>
                登录
              </n-button>
            </n-form>

            <div class="auth-switch">
              <span>还没有账号？</span>
              <n-button text type="primary" @click="router.push('/register')">
                <template #icon>
                  <n-icon><ArrowRight /></n-icon>
                </template>
                创建账号
              </n-button>
            </div>
          </div>
        </n-card>
      </section>
    </section>
  </main>
</template>
