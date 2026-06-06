import { createApp } from 'vue'
import naive from 'naive-ui'
import { router } from './router'
import './style.css'
import App from './App.vue'

const bootstrap = async () => {
  if (import.meta.env.DEV) {
    await import('./mock')
  }

  createApp(App).use(router).use(naive).mount('#app')
}

void bootstrap()
