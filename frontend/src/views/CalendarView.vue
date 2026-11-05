<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { motion, AnimatePresence } from 'motion-v'
import { useScheduleStore, type Lesson } from '../stores/schedule'
import { lessonBell } from '../features/bells'
import { formatDateLocal, getMonday, addDays, formatDateFull } from '../features/date'
import { useSwipe } from '../features/swipe'
import WeekSelector from '../components/WeekSelector.vue'

const store = useScheduleStore()

const today = new Date()
const currentWeekStart = ref(getMonday(new Date()))
const selectedDate = ref(formatDateLocal(today))

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

const selectedDay = computed(() => store.days[selectedDate.value])
const lessons = computed(() => store.lessonsFor(selectedDate.value))
const conductedCount = computed(() => store.conductedCount(selectedDate.value))

function selectDate(date: string) {
  selectedDate.value = date
}

function prevWeek() {
  currentWeekStart.value = addDays(currentWeekStart.value, -7)
}

function nextWeek() {
  currentWeekStart.value = addDays(currentWeekStart.value, 7)
}

function goToCurrentWeek() {
  currentWeekStart.value = getMonday(new Date())
  selectedDate.value = formatDateLocal(today)
}

function loadWeek() {
  const dates = weekDays.value.map(day => formatDateLocal(day))
  void store.fetchRange(dates)
}

function bellLabel(period: number): string {
  const bell = lessonBell(selectedDate.value, period)
  return bell ? `${bell.start}–${bell.end}` : ''
}

function isDone(lesson: Lesson): boolean {
  return store.isConducted(selectedDate.value, lesson.period)
}

function phaseLabel(lesson: Lesson): string {
  if (isDone(lesson)) return 'проведён'
  const phase = store.phase(selectedDate.value, lesson.period)
  if (phase === 'ongoing') return 'идёт'
  if (phase === 'upcoming') return 'ожидается'
  if (phase === 'done') return 'не проведён'
  return ''
}

// Ручная отметка: урок можно пометить проведённым (или снять отметку),
// например при переносе или отмене занятия.
function toggleLesson(lesson: Lesson) {
  void store.setConducted(selectedDate.value, lesson, !isDone(lesson)).catch(() => undefined)
}

onMounted(loadWeek)
watch(currentWeekStart, loadWeek)

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

      <WeekSelector
        :days="weekDays"
        :selected-date="selectedDate"
        @select="selectDate"
      />

      <motion.div
        class="mt-6 bg-bg-surface rounded-xl shadow-md p-4"
        :initial="{ opacity: 0, y: 30 }"
        :animate="{ opacity: 1, y: 0 }"
        :transition="{ delay: 0.15, type: 'spring', stiffness: 250, damping: 22 }"
      >
        <div class="flex items-start justify-between gap-3 mb-4">
          <div>
            <motion.h3
              class="text-lg font-semibold capitalize"
              :key="selectedDate"
              :initial="{ opacity: 0, x: -15 }"
              :animate="{ opacity: 1, x: 0 }"
              :transition="{ delay: 0.25, duration: 0.3 }"
            >
              {{ formatDateFull(selectedDate) }}
            </motion.h3>
            <p v-if="lessons.length" class="text-sm text-text-secondary mt-0.5">
              {{ conductedCount }} из {{ lessons.length }} проведено
            </p>
          </div>
        </div>

        <div v-if="selectedDay?.status === 'loading'" class="space-y-2">
          <div v-for="n in 4" :key="n" class="h-16 rounded-lg skeleton-shimmer bg-bg-surface-secondary" />
        </div>

        <div v-else-if="selectedDay?.status === 'error'" class="text-center py-8">
          <p class="text-text-secondary">{{ selectedDay.error }}</p>
          <button
            class="mt-3 text-sm text-text-accent"
            @click="store.fetchDay(selectedDate)"
          >
            Повторить
          </button>
        </div>

        <div v-else-if="selectedDay?.status === 'missing'" class="text-center py-8 text-text-secondary">
          <p>На эту дату расписания нет</p>
        </div>

        <div v-else-if="lessons.length === 0" class="text-center py-8 text-text-secondary">
          <p>В этот день уроков нет</p>
        </div>

        <div v-else class="space-y-2">
          <p class="text-[11px] text-text-muted px-1 -mb-1">
            Нажмите на урок, чтобы отметить его вручную
          </p>
          <AnimatePresence>
            <motion.div
              v-for="(lesson, index) in lessons"
              :key="`${selectedDate}-${lesson.period}`"
              role="button"
              tabindex="0"
              class="w-full flex items-center gap-3 p-3 rounded-lg text-left cursor-pointer select-none active:scale-[0.99] transition-transform"
              :class="isDone(lesson) ? 'bg-bg-accent/10' : store.phase(selectedDate, lesson.period) === 'ongoing' ? 'bg-hours-medium/15' : 'bg-bg-surface-secondary'"
              :initial="{ opacity: 0, y: 12 }"
              :animate="{ opacity: 1, y: 0 }"
              :transition="{ delay: index * 0.04, type: 'spring', stiffness: 320, damping: 24 }"
              @click="toggleLesson(lesson)"
              @keydown.enter.prevent="toggleLesson(lesson)"
              @keydown.space.prevent="toggleLesson(lesson)"
            >
              <div
                class="w-9 h-9 shrink-0 rounded-full flex items-center justify-center text-sm font-bold"
                :class="isDone(lesson) ? 'bg-bg-accent text-text-on-accent' : 'bg-bg-surface text-text-secondary'"
              >
                {{ lesson.period }}
              </div>
              <div class="min-w-0 flex-1">
                <p class="font-semibold truncate">
                  {{ lesson.subject || 'Без предмета' }}
                  <span class="font-medium text-text-secondary">· {{ lesson.group || 'без группы' }}</span>
                </p>
                <p class="text-xs text-text-secondary mt-0.5">
                  {{ bellLabel(lesson.period) }}
                  <template v-if="bellLabel(lesson.period) && lesson.room"> · </template>
                  {{ lesson.room ? `ауд. ${lesson.room}` : '' }}
                </p>
              </div>
              <span
                class="text-[11px] font-medium shrink-0"
                :class="isDone(lesson) ? 'text-text-accent' : store.phase(selectedDate, lesson.period) === 'ongoing' ? 'text-hours-medium' : 'text-text-muted'"
              >
                {{ phaseLabel(lesson) }}
              </span>
            </motion.div>
          </AnimatePresence>
        </div>
      </motion.div>
    </div>
  </div>
</template>

<style scoped>
@keyframes shimmer {
  0% { background-position: -200% 0; }
  100% { background-position: 200% 0; }
}

.skeleton-shimmer {
  background: linear-gradient(
    90deg,
    var(--color-bg-surface-secondary) 25%,
    var(--color-bg-surface) 50%,
    var(--color-bg-surface-secondary) 75%
  );
  background-size: 200% 100%;
  animation: shimmer 1.5s ease-in-out infinite;
}
</style>
