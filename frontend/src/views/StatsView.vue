<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { motion, AnimatePresence } from 'motion-v'
import { useWorkHoursStore } from '../stores/workHours'
import { getMonthName, getMonthDays, formatHours } from '../features/date'
import StatCard from '../components/StatCard.vue'
import MonthCalendar from '../components/MonthCalendar.vue'

const store = useWorkHoursStore()

const today = new Date()
const currentYear = ref(today.getFullYear())
const currentMonth = ref(today.getMonth())

const monthName = computed(() => {
  return getMonthName(new Date(currentYear.value, currentMonth.value))
})

const days = computed(() => {
  return getMonthDays(currentYear.value, currentMonth.value)
})

const totalMonthHours = computed(() => {
  return days.value.reduce((sum, d) => {
    return sum + (d.date ? store.getHoursForDate(d.date) : 0)
  }, 0)
})

const daysWorked = computed(() => {
  return days.value.filter(d => d.day > 0 && d.date && store.getHoursForDate(d.date) > 0).length
})

function prevMonth() {
  if (currentMonth.value === 0) {
    currentMonth.value = 11
    currentYear.value--
  } else {
    currentMonth.value--
  }
}

function nextMonth() {
  if (currentMonth.value === 11) {
    currentMonth.value = 0
    currentYear.value++
  } else {
    currentMonth.value++
  }
}

async function loadStats() {
  await store.fetchMonthStats(currentYear.value, currentMonth.value + 1)
}

onMounted(() => {
  loadStats()
})

watch([currentYear, currentMonth], () => {
  loadStats()
})
</script>

<template>
  <div class="bg-bg-primary">
    <div class="max-w-md mx-auto p-4">
      <!-- Header -->
      <motion.div
        class="flex items-center justify-between mb-4"
        :initial="{ opacity: 0, y: -20 }"
        :animate="{ opacity: 1, y: 0 }"
        :transition="{ duration: 0.3, ease: 'easeOut' }"
      >
        <motion.button
          @click="prevMonth"
          class="p-2 rounded-full hover:bg-bg-surface-secondary transition-colors"
          :while-hover="{ scale: 1.15, x: -3 }"
          :while-tap="{ scale: 0.85 }"
          :transition="{ type: 'spring', stiffness: 400, damping: 17 }"
        >
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
          </svg>
        </motion.button>

        <motion.h2
          class="text-xl font-semibold capitalize"
          :key="`${currentYear}-${currentMonth}`"
          :initial="{ opacity: 0, y: -10, scale: 0.95 }"
          :animate="{ opacity: 1, y: 0, scale: 1 }"
          :transition="{ duration: 0.2 }"
        >
          {{ monthName }} {{ currentYear }}
        </motion.h2>

        <motion.button
          @click="nextMonth"
          class="p-2 rounded-full hover:bg-bg-surface-secondary transition-colors"
          :while-hover="{ scale: 1.15, x: 3 }"
          :while-tap="{ scale: 0.85 }"
          :transition="{ type: 'spring', stiffness: 400, damping: 17 }"
        >
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
          </svg>
        </motion.button>
      </motion.div>

      <!-- Stats Summary -->
      <div class="grid grid-cols-2 gap-3 mb-4">
        <StatCard
          :value="formatHours(totalMonthHours)"
          label="часов за месяц"
          color="blue"
        />
        <StatCard
          :value="String(daysWorked)"
          label="дней работал"
          color="purple"
        />
      </div>

      <!-- Calendar -->
      <AnimatePresence mode="wait">
        <MonthCalendar
          :key="`${currentYear}-${currentMonth}`"
          :year="currentYear"
          :month="currentMonth"
        />
      </AnimatePresence>
    </div>
  </div>
</template>
