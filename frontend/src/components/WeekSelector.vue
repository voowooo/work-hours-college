<script setup lang="ts">
import { computed } from 'vue'
import { motion } from 'motion-v'
import { formatDateLocal, isToday, getDayName } from '../features/date'
import { useScheduleStore } from '../stores/schedule'

const props = defineProps<{
  days: Date[]
  selectedDate: string
}>()

const emit = defineEmits<{
  select: [date: string]
}>()

const store = useScheduleStore()

const weekRange = computed(() => {
  if (props.days.length < 7) return ''
  const start = props.days[0]?.toLocaleDateString('ru', { day: 'numeric', month: 'short' })
  const end = props.days[6]?.toLocaleDateString('ru', { day: 'numeric', month: 'short' })
  return `${start} - ${end}`
})

function selectDate(date: Date) {
  emit('select', formatDateLocal(date))
}

function isSelected(date: Date): boolean {
  return formatDateLocal(date) === props.selectedDate
}

function lessonLabel(date: Date): string {
  const key = formatDateLocal(date)
  const total = store.lessonCount(key)
  if (!total) return ''
  return `${store.conductedCount(key)}/${total}`
}
</script>

<template>
  <div>
    <motion.div
      class="text-center mb-3"
      :initial="{ opacity: 0, y: -10 }"
      :animate="{ opacity: 1, y: 0 }"
      :transition="{ duration: 0.3 }"
    >
      <h2 class="text-lg font-semibold text-text-secondary">
        {{ weekRange }}
      </h2>
    </motion.div>

    <div class="grid grid-cols-7 gap-1">
      <motion.button
        v-for="(day, index) in days"
        :key="day.toISOString()"
        @click="selectDate(day)"
        :class="[
          'flex flex-col items-center py-2 rounded-lg',
          isSelected(day)
            ? 'bg-bg-accent text-text-on-accent'
            : isToday(day)
              ? 'bg-bg-accent/10'
              : 'hover:bg-bg-surface-secondary'
        ]"
        :initial="{ opacity: 0, y: 20 }"
        :animate="{ opacity: 1, y: 0 }"
        :transition="{ delay: index * 0.04, type: 'spring', stiffness: 300, damping: 20 }"
        :while-hover="{ scale: 1.08, y: -2 }"
        :while-tap="{ scale: 0.92 }"
      >
        <span
          class="text-xs font-medium"
          :class="isSelected(day) ? 'text-text-on-accent/70' : 'text-text-secondary'"
        >
          {{ getDayName(day) }}
        </span>
        <span class="text-lg font-bold">
          {{ day.getDate() }}
        </span>
        <motion.span
          v-if="lessonLabel(day)"
          :class="[
            'text-[10px] mt-0.5',
            isSelected(day) ? 'text-text-on-accent/70' : 'text-text-muted'
          ]"
          :initial="{ opacity: 0, scale: 0.5 }"
          :animate="{ opacity: 1, scale: 1 }"
          :transition="{ delay: index * 0.04 + 0.15, type: 'spring', stiffness: 400, damping: 15 }"
        >
          {{ lessonLabel(day) }}
        </motion.span>
      </motion.button>
    </div>
  </div>
</template>
