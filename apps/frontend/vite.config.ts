import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue(), tailwindcss()],
  build: {
    rolldownOptions: {
      output: {
        codeSplitting: {
          groups: [
            {
              name: 'vendor-vue',
              test: /node_modules[\\/](vue|@vue|vue-router)[\\/]/,
              priority: 30,
            },
            {
              name: 'vendor-naive',
              test: /node_modules[\\/](naive-ui|vdirs|vooks|vueuc|date-fns|css-render|evtd|treemate|seemly)[\\/]/,
              priority: 20,
            },
            {
              name: 'vendor-editor',
              test: /node_modules[\\/](@codemirror|@lezer|crelt|style-mod|w3c-keyname)[\\/]/,
              priority: 20,
            },
          ],
        },
      },
    },
  },
})
