import {
  NAlert,
  NButton,
  NCard,
  NCollapse,
  NCollapseItem,
  NConfigProvider,
  NDialogProvider,
  NEmpty,
  NForm,
  NFormItem,
  NIcon,
  NInput,
  NLayout,
  NLayoutContent,
  NLayoutHeader,
  NLayoutSider,
  NMessageProvider,
  NModal,
  NPagination,
  NPopconfirm,
  NProgress,
  NRadioButton,
  NRadioGroup,
  NScrollbar,
  NSelect,
  NSpace,
  NSpin,
  NTag,
} from 'naive-ui'
import type { App } from 'vue'

export function registerNaiveComponents(app: App) {
  app.component('NAlert', NAlert)
  app.component('NButton', NButton)
  app.component('NCard', NCard)
  app.component('NCollapse', NCollapse)
  app.component('NCollapseItem', NCollapseItem)
  app.component('NConfigProvider', NConfigProvider)
  app.component('NDialogProvider', NDialogProvider)
  app.component('NEmpty', NEmpty)
  app.component('NForm', NForm)
  app.component('NFormItem', NFormItem)
  app.component('NIcon', NIcon)
  app.component('NInput', NInput)
  app.component('NLayout', NLayout)
  app.component('NLayoutContent', NLayoutContent)
  app.component('NLayoutHeader', NLayoutHeader)
  app.component('NLayoutSider', NLayoutSider)
  app.component('NMessageProvider', NMessageProvider)
  app.component('NModal', NModal)
  app.component('NPagination', NPagination)
  app.component('NPopconfirm', NPopconfirm)
  app.component('NProgress', NProgress)
  app.component('NRadioButton', NRadioButton)
  app.component('NRadioGroup', NRadioGroup)
  app.component('NScrollbar', NScrollbar)
  app.component('NSelect', NSelect)
  app.component('NSpace', NSpace)
  app.component('NSpin', NSpin)
  app.component('NTag', NTag)
}
