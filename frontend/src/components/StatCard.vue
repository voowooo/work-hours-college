<script setup lang="ts">
import { motion } from 'motion-v'

defineProps<{
  value: string | number
  label: string
  color?: 'blue' | 'purple' | 'green' | 'red'
}>()
</script>

<template>
  <motion.div
    class="bg-bg-surface rounded-xl p-4 text-center shadow-sm"
    :initial="{ opacity: 0, scale: 0.8, y: 20 }"
    :animate="{ opacity: 1, scale: 1, y: 0 }"
    :transition="{ type: 'spring', stiffness: 300, damping: 20 }"
    :while-hover="{ scale: 1.05, y: -2 }"
    :while-tap="{ scale: 0.97 }"
  >
    <motion.p
      class="text-2xl font-bold"
      :class="{
        'text-hours-high': color === 'blue',
        'text-hours-medium': color === 'purple',
        'text-hours-good': color === 'green',
        'text-hours-low': color === 'red',
        'text-text-accent': !color
      }"
      :initial="{ opacity: 0, y: 10 }"
      :animate="{ opacity: 1, y: 0 }"
      :transition="{ delay: 0.15, duration: 0.3 }"
    >
      {{ value }}
    </motion.p>
    <motion.p
      class="text-xs text-text-secondary"
      :initial="{ opacity: 0 }"
      :animate="{ opacity: 1 }"
      :transition="{ delay: 0.25, duration: 0.3 }"
    >
      {{ label }}
    </motion.p>
  </motion.div>
</template>
