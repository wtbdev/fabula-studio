import { createApp } from 'vue'
import { registerNaiveComponents } from './plugins/naive'
import { router } from './router'
import './style.css'
import App from './App.vue'

const bootstrap = async () => {
  if (import.meta.env.DEV) {
    await import('./mock')
  }

  const app = createApp(App)
  registerNaiveComponents(app)
  app.use(router).mount('#app')
}

void bootstrap()
