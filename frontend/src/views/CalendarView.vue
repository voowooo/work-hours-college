<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { motion, AnimatePresence } from 'motion-v'
import { useWorkHoursStore } from '../stores/workHours'
import { formatDateLocal, getMonday, addDays, formatDateFull, formatHours } from '../features/date'
import { useSwipe } from '../features/swipe'
import WeekSelector from '../components/WeekSelector.vue'
import TimeEntry from '../components/TimeEntry.vue'

const store = useWorkHoursStore()

const today = new Date()
const currentWeekStart = ref(getMonday(new Date()))
const selectedDate = ref(formatDateLocal(today))
const showMenu = ref(false)
const menuRef = ref<HTMLElement | null>(null)

const weekDays = computed(() => {
  const days = []
  for (let i = 0; i < 7; i++) {
    days.push(addDays(currentWeekStart.value, i))
  }
  return days
})

const isCurrentWeek = computed(() => {
  const currentMonday = getMonday(new Date())
  return currentWeekStart.value.getTime() === currentMonday.getTime()
})

const selectedEntry = computed(() => {
  return store.getEntryByDate(selectedDate.value)
})

const arrivedAt = computed(() => selectedEntry.value?.arrivedAt || null)
const leftAt = computed(() => selectedEntry.value?.leftAt || null)
const hoursWorked = computed(() => selectedEntry.value?.hoursWorked || 0)

function selectDate(date: string) {
  selectedDate.value = date
}

function prevWeek() {
  currentWeekStart.value = addDays(currentWeekStart.value, -7)
  loadWeekData()
}

function nextWeek() {
  currentWeekStart.value = addDays(currentWeekStart.value, 7)
  loadWeekData()
}

function goToCurrentWeek() {
  currentWeekStart.value = getMonday(new Date())
  selectedDate.value = formatDateLocal(today)
  loadWeekData()
}

async function loadWeekData() {
  const days = weekDays.value
  if (days.length >= 7 && days[0] && days[6]) {
    const startDate = formatDateLocal(days[0])
    const endDate = formatDateLocal(days[6])
    await store.fetchEntries(startDate, endDate)
  }
}

async function updateArrivedAt(time: string | null) {
  await store.upsertEntry(selectedDate.value, time, leftAt.value)
}

async function updateLeftAt(time: string | null) {
  await store.upsertEntry(selectedDate.value, arrivedAt.value, time)
}

function toggleMenu() {
  showMenu.value = !showMenu.value
}

async function deleteDay() {
  showMenu.value = false
  await store.deleteEntry(selectedDate.value)
}

function handleClickOutside(e: Event) {
  if (menuRef.value && !menuRef.value.contains(e.target as Node)) {
    showMenu.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
  loadWeekData()
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})

const { handleTouchStart, handleTouchEnd } = useSwipe({
  onSwipeLeft: nextWeek,
  onSwipeRight: prevWeek
})
</script>

<template>
  <div
    class="bg-bg-primary"
    @touchstart="handleTouchStart"
    @touchend="handleTouchEnd"
  >
    <div class="max-w-md mx-auto p-4">
      <!-- Week Navigation -->
      <motion.div
        class="flex items-center justify-between mb-4"
        :initial="{ opacity: 0, y: -20 }"
        :animate="{ opacity: 1, y: 0 }"
        :transition="{ duration: 0.3, ease: 'easeOut' }"
      >
        <motion.button
          @click="prevWeek"
          class="p-2 rounded-full hover:bg-bg-surface-secondary transition-colors"
          :while-hover="{ scale: 1.15, x: -3 }"
          :while-tap="{ scale: 0.85 }"
          :transition="{ type: 'spring', stiffness: 400, damping: 17 }"
        >
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
          </svg>
        </motion.button>

        <motion.button
          v-if="!isCurrentWeek"
          @click="goToCurrentWeek"
          class="text-xs text-text-accent hover:text-bg-accent-hover transition-colors"
          :initial="{ opacity: 0, scale: 0.8 }"
          :animate="{ opacity: 1, scale: 1 }"
          :while-hover="{ scale: 1.1 }"
          :while-tap="{ scale: 0.9 }"
        >
          Текущая неделя
        </motion.button>
        <div v-else class="w-10" />

        <motion.button
          @click="nextWeek"
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

      <!-- Week Days -->
      <WeekSelector
        :days="weekDays"
        :selected-date="selectedDate"
        @select="selectDate"
      />

      <!-- Selected Date Info -->
      <motion.div
        class="mt-6 bg-bg-surface rounded-xl shadow-md p-4"
        :initial="{ opacity: 0, y: 30 }"
        :animate="{ opacity: 1, y: 0 }"
        :transition="{ delay: 0.15, type: 'spring', stiffness: 250, damping: 22 }"
      >
        <div class="flex items-center justify-between mb-4">
          <motion.h3
            class="text-lg font-semibold capitalize"
            :initial="{ opacity: 0, x: -15 }"
            :animate="{ opacity: 1, x: 0 }"
            :transition="{ delay: 0.25, duration: 0.3 }"
            :key="selectedDate"
          >
            {{ formatDateFull(selectedDate) }}
          </motion.h3>

          <div v-if="selectedEntry" ref="menuRef" class="relative">
            <motion.button
              @click.stop="toggleMenu"
              class="p-1.5 rounded-full hover:bg-bg-surface-secondary"
              :while-hover="{ scale: 1.1 }"
              :while-tap="{ scale: 0.9 }"
            >
              <svg class="w-5 h-5 text-text-secondary" fill="currentColor" viewBox="0 0 20 20">
                <path d="M10 6a2 2 0 110-4 2 2 0 010 4zM10 12a2 2 0 110-4 2 2 0 010 4zM10 18a2 2 0 110-4 2 2 0 010 4z" />
              </svg>
            </motion.button>

            <AnimatePresence>
              <motion.div
                v-if="showMenu"
                class="absolute right-0 top-full mt-1 bg-bg-surface rounded-xl shadow-lg border border-border/50 overflow-hidden z-20 min-w-[160px]"
                :initial="{ opacity: 0, scale: 0.9, y: -8 }"
                :animate="{ opacity: 1, scale: 1, y: 0 }"
                :exit="{ opacity: 0, scale: 0.9, y: -8 }"
                :transition="{ type: 'spring', stiffness: 400, damping: 25 }"
              >
                <button
                  @click="deleteDay"
                  class="w-full px-4 py-2.5 text-left text-sm text-bg-danger hover:bg-bg-danger/10 transition-colors flex items-center gap-2"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                  Удалить день
                </button>
              </motion.div>
            </AnimatePresence>
          </div>
        </div>

        <div class="space-y-4">
          <TimeEntry
            label="Пришел"
            :time="arrivedAt"
            placeholder="Время прихода"
            @update="updateArrivedAt"
          />

          <TimeEntry
            label="Ушел"
            :time="leftAt"
            placeholder="Время ухода"
            @update="updateLeftAt"
          />

          <!-- Hours Worked -->
          <motion.div
            class="p-3 bg-bg-accent/10 rounded-lg"
            :initial="{ opacity: 0, scale: 0.9 }"
            :animate="{ opacity: 1, scale: 1 }"
            :transition="{ delay: 0.3, type: 'spring', stiffness: 300, damping: 20 }"
            :key="hoursWorked"
          >
            <p class="text-sm text-text-accent">Отработано</p>
            <motion.p
              class="text-2xl font-bold text-text-primary"
              :initial="{ opacity: 0, y: 8 }"
              :animate="{ opacity: 1, y: 0 }"
              :transition="{ delay: 0.4, duration: 0.25 }"
            >
              {{ formatHours(hoursWorked) }}
            </motion.p>
          </motion.div>
        </div>
      </motion.div>
    </div>
  </div>
</template>
