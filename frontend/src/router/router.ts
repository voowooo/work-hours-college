import { createWebHistory, createRouter } from 'vue-router'
import CalendarView from '../views/CalendarView.vue'
import StatsView from '../views/StatsView.vue'
import SettingsView from '../views/SettingsView.vue'
import AltView from '../views/AltView.vue'

const routes = [
  { path: '/', component: CalendarView },
  { path: '/stats', component: StatsView },
  { path: '/settings', component: SettingsView },
  { path: '/alt', component: AltView },
]

export const router = createRouter({
  history: createWebHistory(),
  routes,
})
