import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import zhCn from 'element-plus/dist/locale/zh-cn.mjs'
import 'element-plus/dist/index.css'
import App from './App.vue'
import router from './router'
import { setUnauthorizedHandler } from './api'
import { normalizeRedirect } from './utils/navigation'
import './style.css'

const app = createApp(App)

setUnauthorizedHandler(() => {
  if (router.currentRoute.value.path !== '/login') {
    router.push({
      path: '/login',
      query: { redirect: normalizeRedirect(router.currentRoute.value.fullPath) },
    })
  }
})

app.use(router).use(ElementPlus, { locale: zhCn })
app.mount('#app')
