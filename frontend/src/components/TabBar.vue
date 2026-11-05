<script setup lang="ts">
import { RouterLink, useRoute } from 'vue-router'
import { motion } from 'motion-v'

const route = useRoute()

const tabs = [
  { path: '/', label: 'Календарь', icon: 'calendar' },
  { path: '/stats', label: 'Статистика', icon: 'chart' },
  { path: '/alt', label: 'Alt', icon: 'image' }
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
