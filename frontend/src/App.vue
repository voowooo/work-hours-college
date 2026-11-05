<script setup lang="ts">
import { RouterView, useRoute } from 'vue-router'
import { motion, AnimatePresence } from 'motion-v'
import TabBar from './components/TabBar.vue'
import { useTelegramUser } from './composables/useTelegramUser'

const route = useRoute()
useTelegramUser()
</script>

<template>
  <div class="min-h-screen bg-bg-primary pb-20">
    <AnimatePresence mode="wait">
      <motion.main
        :key="route.path"
        :initial="{ opacity: 0, y: 12 }"
        :animate="{ opacity: 1, y: 0 }"
        :exit="{ opacity: 0, y: -12 }"
        :transition="{ duration: 0.25, ease: 'easeInOut' }"
      >
        <RouterView />
      </motion.main>
    </AnimatePresence>
    <TabBar />
  </div>
</template>
