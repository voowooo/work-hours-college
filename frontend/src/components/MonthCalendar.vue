<script setup lang="ts">
import { computed } from 'vue'
import { motion } from 'motion-v'
import { getMonthDays, formatHours } from '../features/date'
import { useWorkHoursStore } from '../stores/workHours'

const props = defineProps<{
  year: number
  month: number
}>()

const store = useWorkHoursStore()

const days = computed(() => {
  const monthDays = getMonthDays(props.year, props.month)
  return monthDays.map(d => ({
    ...d,
    hours: d.date ? store.getHoursForDate(d.date) : 0
  }))
})

function getHoursColor(hours: number): string {
  if (hours === 0) return ''
  if (hours < 4) return 'bg-hours-low/10 text-hours-low'
  if (hours < 6) return 'bg-hours-medium/10 text-hours-medium'
  if (hours < 8) return 'bg-hours-good/10 text-hours-good'
  return 'bg-hours-high/10 text-hours-high'
}
</script>

<template>
  <motion.div
    class="bg-bg-surface rounded-xl shadow-sm overflow-hidden"
    :initial="{ opacity: 0, y: 20 }"
    :animate="{ opacity: 1, y: 0 }"
    :transition="{ duration: 0.35, ease: 'easeOut' }"
  >
    <div class="grid grid-cols-7 border-b border-border">
      <motion.div
        v-for="(day, index) in ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс']"
        :key="day"
        class="text-center text-xs font-medium text-text-secondary py-2"
        :initial="{ opacity: 0, y: -10 }"
        :animate="{ opacity: 1, y: 0 }"
        :transition="{ delay: index * 0.03, duration: 0.25 }"
      >
        {{ day }}
      </motion.div>
    </div>

    <div class="grid grid-cols-7">
      <motion.div
        v-for="(item, index) in days"
        :key="`${year}-${month}-${index}`"
        class="p-2 text-center min-h-[50px] flex flex-col items-center justify-center border-b border-border"
        :initial="{ opacity: 0, scale: 0.7 }"
        :animate="{ opacity: 1, scale: 1 }"
        :transition="{
          delay: index * 0.012,
          type: 'spring',
          stiffness: 350,
          damping: 22
        }"
        :while-hover="item.day ? { scale: 1.15, backgroundColor: 'rgba(113, 119, 68, 0.06)' } : {}"
      >
        <span
          v-if="item.day"
          class="text-sm font-medium"
        >
          {{ item.day }}
        </span>
        <motion.span
          v-if="item.day && item.hours > 0"
          :class="['text-[10px] mt-1 px-1.5 py-0.5 rounded-full', getHoursColor(item.hours)]"
          :initial="{ opacity: 0, scale: 0 }"
          :animate="{ opacity: 1, scale: 1 }"
          :transition="{ delay: index * 0.012 + 0.1, type: 'spring', stiffness: 500, damping: 15 }"
        >
          {{ formatHours(item.hours) }}
        </motion.span>
      </motion.div>
    </div>
  </motion.div>
</template>
