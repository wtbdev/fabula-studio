<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import {
  ArrowRight,
  AtSign,
  LockKeyhole,
  PenLine,
  UserPlus,
} from 'lucide-vue-next'
import { useAuth } from '../composables/useAuth'
import type { FormInst, FormRules } from 'naive-ui'

const router = useRouter()
const message = useMessage()
const { register } = useAuth()

const formRef = ref<FormInst | null>(null)
const loading = ref(false)

const formModel = reactive({
  email: '',
  nickname: '',
  password: '',
  confirmPassword: '',
})

const emailInputProps = { autocomplete: 'email', inputmode: 'email' }
const passwordInputProps = { autocomplete: 'new-password' }
const authCardContentStyle = { padding: '0' }

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
  nickname: [{ required: true, message: '请输入昵称', trigger: ['blur', 'input'] }],
  password: [
    { required: true, message: '请输入密码', trigger: ['blur', 'input'] },
    {
      validator: (_rule: unknown, value: string | undefined) => {
        return !value || value.length >= 6
      },
      message: '密码至少 6 位',
      trigger: ['blur', 'input'],
    },
  ],
  confirmPassword: [
    { required: true, message: '请再次输入密码', trigger: ['blur', 'input'] },
    {
      validator: (_rule: unknown, value: string | undefined) => {
        return !value || value === formModel.password
      },
      message: '两次输入的密码不一致',
      trigger: ['blur', 'input'],
    },
  ],
}

const getErrorMessage = (error: unknown) => {
  return error instanceof Error ? error.message : '注册失败，请稍后重试'
}

const handleRegister = async () => {
  try {
    await formRef.value?.validate()
    loading.value = true
    await register({
      email: formModel.email.trim(),
      password: formModel.password,
      nickname: formModel.nickname.trim(),
    })
    message.success('注册成功')
    await router.push('/projects')
  } catch (error) {
    if (Array.isArray(error)) return
    message.error(getErrorMessage(error))
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <main class="auth-page auth-login-page" aria-labelledby="register-title">
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
          <span class="auth-kicker">创作从这里开始</span>
          <h2>
            把小说变成
            <br />
            一份能继续打磨的剧本
          </h2>
          <p>注册后即可创建项目，体验 AI 小说改编剧本的完整流程。</p>
        </div>
      </aside>

      <section class="auth-form-panel" aria-labelledby="register-title">
        <n-card
          :bordered="false"
          class="auth-card auth-login-card"
          :content-style="authCardContentStyle"
        >
          <div class="auth-card-header">
            <n-tag size="small" :bordered="false" type="success">邮箱注册</n-tag>
            <h1 id="register-title">创建账号</h1>
            <p>注册即开始你的剧本改编之旅。</p>
          </div>

          <div class="auth-card-body">
            <n-form ref="formRef" :model="formModel" :rules="rules" label-placement="top">
              <n-form-item first label="邮箱" path="email">
                <n-input
                  v-model:value="formModel.email"
                  clearable
                  placeholder="user@example.com"
                  :disabled="loading"
                  :input-props="emailInputProps"
                >
                  <template #prefix>
                    <n-icon><AtSign /></n-icon>
                  </template>
                </n-input>
              </n-form-item>
              <n-form-item first label="昵称" path="nickname">
                <n-input
                  v-model:value="formModel.nickname"
                  clearable
                  placeholder="你的创作者名称"
                  :disabled="loading"
                >
                  <template #prefix>
                    <n-icon><PenLine /></n-icon>
                  </template>
                </n-input>
              </n-form-item>
              <n-form-item first label="密码" path="password">
                <n-input
                  v-model:value="formModel.password"
                  type="password"
                  show-password-on="click"
                  placeholder="至少 6 位"
                  :disabled="loading"
                  :input-props="passwordInputProps"
                >
                  <template #prefix>
                    <n-icon><LockKeyhole /></n-icon>
                  </template>
                </n-input>
              </n-form-item>
              <n-form-item first label="确认密码" path="confirmPassword">
                <n-input
                  v-model:value="formModel.confirmPassword"
                  type="password"
                  show-password-on="click"
                  placeholder="再次输入密码"
                  :disabled="loading"
                  @keydown.enter="handleRegister"
                >
                  <template #prefix>
                    <n-icon><LockKeyhole /></n-icon>
                  </template>
                </n-input>
              </n-form-item>

              <n-button
                class="auth-submit"
                block
                type="primary"
                :loading="loading"
                @click="handleRegister"
              >
                <template #icon>
                  <n-icon><UserPlus /></n-icon>
                </template>
                注册并登录
              </n-button>
            </n-form>

            <div class="auth-switch">
              <span>已有账号？</span>
              <n-button text type="primary" @click="router.push('/login')">
                <template #icon>
                  <n-icon><ArrowRight /></n-icon>
                </template>
                登录
              </n-button>
            </div>
          </div>
        </n-card>
      </section>
    </section>
  </main>
</template>
