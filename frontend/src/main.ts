import { createApp } from 'vue'
import './style.css'
import App from './App.vue'
import { router } from './router/router.ts'
import { createPinia } from 'pinia'
import { init, miniApp, swipeBehavior, closingBehavior, viewport } from '@tma.js/sdk-vue'

// Инициализируем SDK
init()

// Настраиваем поведение Telegram Mini App
try {
  miniApp.mount()
  swipeBehavior.mount()
  closingBehavior.mount()
  viewport.mount()

  swipeBehavior.disableVertical()
  closingBehavior.enableConfirmation()
  viewport.expand()
  miniApp.ready()
} catch (e) {
  console.warn('Telegram SDK setup error:', e)
}

const pinia = createPinia()

const app = createApp(App)
app.use(router)
app.use(pinia)
app.mount('#app')
