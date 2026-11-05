import { ref, onMounted } from 'vue'
import { initData } from '@tma.js/sdk-vue'

const telegramUserId = ref<string | null>(localStorage.getItem('telegram_user_id'))
const telegramFirstName = ref<string | null>(localStorage.getItem('telegram_first_name'))
const telegramUsername = ref<string | null>(localStorage.getItem('telegram_username'))
const isInitialized = ref(false)

export function useTelegramUser() {
  onMounted(() => {
    if (isInitialized.value) return

    let user
    try {
      initData.restore()
      user = initData.user()
    } catch {
      return
    }

    if (user) {
      telegramUserId.value = String(user.id)
      telegramFirstName.value = user.first_name || null
      telegramUsername.value = user.username || null
      localStorage.setItem('telegram_user_id', telegramUserId.value)
      if (telegramFirstName.value) {
        localStorage.setItem('telegram_first_name', telegramFirstName.value)
      }
      if (telegramUsername.value) {
        localStorage.setItem('telegram_username', telegramUsername.value)
      } else {
        localStorage.removeItem('telegram_username')
      }
      isInitialized.value = true
    }
  })

  return {
    userId: telegramUserId,
    firstName: telegramFirstName,
    username: telegramUsername,
    isInitialized
  }
}
