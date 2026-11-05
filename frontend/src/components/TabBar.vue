<script setup lang="ts">
import { RouterLink, useRoute } from 'vue-router'
import { motion } from 'motion-v'

const route = useRoute()

const tabs = [
  { path: '/', label: 'Календарь', icon: 'calendar' },
  { path: '/stats', label: 'Статистика', icon: 'chart' },
  { path: '/settings', label: 'Настройки', icon: 'settings' },
  // { path: '/alt', label: 'Alt', icon: 'image' }
]

function isActive(path: string): boolean {
  return route.path === path
}
</script>

<template>
  <nav class="fixed bottom-0 inset-x-0 z-50 pb-safe">
    <motion.div
      class="max-w-md mx-auto px-4 pb-2"
      :initial="{ y: 80, opacity: 0 }"
      :animate="{ y: 0, opacity: 1 }"
      :transition="{ type: 'spring', stiffness: 300, damping: 30, delay: 0.2 }"
    >
      <div class="bg-bg-surface/80 backdrop-blur-xl rounded-2xl shadow-lg border border-border/50">
        <div class="flex items-center justify-around h-16">
          <RouterLink
            v-for="tab in tabs"
            :key="tab.path"
            :to="tab.path"
            class="relative flex flex-col items-center justify-center gap-0.5 w-full h-full"
            :class="isActive(tab.path) ? 'text-text-accent' : 'text-text-secondary'"
          >
            <motion.div
              class="flex flex-col items-center justify-center gap-0.5"
              :while-hover="{ scale: 1.1 }"
              :while-tap="{ scale: 0.9 }"
              :transition="{ type: 'spring', stiffness: 400, damping: 17 }"
            >
              <!-- Calendar Icon -->
              <svg
                v-if="tab.icon === 'calendar'"
                class="w-6 h-6"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"
                />
              </svg>

              <!-- Chart Icon -->
              <svg
                v-if="tab.icon === 'chart'"
                class="w-6 h-6"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"
                />
              </svg>

              <svg
                v-if="tab.icon === 'settings'"
                class="w-6 h-6"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
                />
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
              </svg>

              <!-- Image Icon -->
              <svg
                v-if="tab.icon === 'image'"
                class="w-6 h-6"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
                />
              </svg>

              <span class="text-[10px] font-medium">
                {{ tab.label }}
              </span>
            </motion.div>

            <!-- Active Indicator -->
            <motion.div
              v-if="isActive(tab.path)"
              class="absolute bottom-1 w-1 h-1 rounded-full bg-text-accent"
              :layout-id="'tab-indicator'"
              :transition="{ type: 'spring', stiffness: 500, damping: 30 }"
            />
          </RouterLink>
        </div>
      </div>
    </motion.div>
  </nav>
</template>

<style scoped>
.pb-safe {
  padding-bottom: env(safe-area-inset-bottom, 0px);
}
</style>
