<script setup lang="ts">
import { motion } from 'motion-v'
import TimeInput from './TimeInput.vue'

defineProps<{
  label: string
  time: string | null
  placeholder: string
}>()

const emit = defineEmits<{
  update: [time: string | null]
}>()
</script>

<template>
  <motion.div
    class="flex p-3 bg-bg-surface-secondary rounded-lg"
    :class="time ? 'flex-row items-center justify-between' : 'flex-col justify-center items-start gap-3'"
    :initial="{ opacity: 0, x: -20 }"
    :animate="{ opacity: 1, x: 0 }"
    :transition="{ type: 'spring', stiffness: 300, damping: 24 }"
    :while-hover="{ scale: 1.01, backgroundColor: 'rgba(245, 245, 245, 0.9)' }"
    layout
  >
    <div class="flex flex-col w-full">
      <p class="text-sm text-text-secondary">{{ label }}</p>
      <motion.p
        v-if="time"
        class="text-xl font-semibold"
        :initial="{ opacity: 0, scale: 0.8 }"
        :animate="{ opacity: 1, scale: 1 }"
        :transition="{ type: 'spring', stiffness: 400, damping: 15 }"
        :key="time"
      >
        {{ time || '--:--' }}
      </motion.p>
    </div>
    <TimeInput
      :model-value="time"
      @update:model-value="emit('update', $event)"
      :placeholder="placeholder"
    />
  </motion.div>
</template>
