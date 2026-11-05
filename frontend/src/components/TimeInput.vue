<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'
import { motion, AnimatePresence } from 'motion-v'

const props = defineProps<{
  modelValue: string | null
  placeholder?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string | null]
}>()

const ITEM_HEIGHT = 48

const selectedHour = ref(0)
const selectedMinute = ref(0)
const showPicker = ref(false)
const hoursScroll = ref<HTMLElement | null>(null)
const minutesScroll = ref<HTMLElement | null>(null)

const hours = Array.from({ length: 24 }, (_, i) => i)
const minutes = Array.from({ length: 6 }, (_, i) => i * 10)

watch(() => props.modelValue, (val) => {
  if (val) {
    const [h, m] = val.split(':')
    selectedHour.value = h ? parseInt(h) : 0
    const min = m ? parseInt(m) : 0
    selectedMinute.value = Math.min(50, Math.floor(min / 10) * 10)
  }
}, { immediate: true })

function openPicker() {
  showPicker.value = true
  if (!props.modelValue) {
    const now = new Date()
    let h = now.getHours()
    let m = Math.round(now.getMinutes() / 10) * 10
    if (m === 60) {
      m = 0
      h = (h + 1) % 24
    }
    selectedHour.value = h
    selectedMinute.value = m
  }
  nextTick(() => {
    scrollTo(hoursScroll.value, selectedHour.value)
    scrollTo(minutesScroll.value, minutes.indexOf(selectedMinute.value))
  })
}

function closePicker() {
  showPicker.value = false
}

function scrollTo(el: HTMLElement | null, index: number) {
  if (!el || index < 0) return
  el.scrollTop = index * ITEM_HEIGHT
}

function handleHoursScroll() {
  if (!hoursScroll.value) return
  const index = Math.round(hoursScroll.value.scrollTop / ITEM_HEIGHT)
  selectedHour.value = Math.max(0, Math.min(hours.length - 1, index))
}

function handleMinutesScroll() {
  if (!minutesScroll.value) return
  const index = Math.round(minutesScroll.value.scrollTop / ITEM_HEIGHT)
  selectedMinute.value = minutes[Math.max(0, Math.min(minutes.length - 1, index))] ?? 0
}

function handleHoursScrollEnd() {
  hoursScroll.value?.scrollTo({ top: selectedHour.value * ITEM_HEIGHT, behavior: 'smooth' })
}

function handleMinutesScrollEnd() {
  const index = Math.max(0, minutes.indexOf(selectedMinute.value))
  minutesScroll.value?.scrollTo({ top: index * ITEM_HEIGHT, behavior: 'smooth' })
}

function save() {
  const hh = String(selectedHour.value).padStart(2, '0')
  const mm = String(selectedMinute.value).padStart(2, '0')
  emit('update:modelValue', `${hh}:${mm}`)
  closePicker()
}


</script>

<template>
  <div class="relative flex w-full">
    <motion.button
      v-if="!showPicker"
      @click="openPicker"
      class="w-full flex rounded-lg transition-colors"
      :class="modelValue ? 'bg-transparent text-text-accent py-2 justify-end' : 'bg-bg-accent text-text-on-accent px-4 py-2 hover:bg-bg-accent-hover justify-center'"
      :while-hover="{ scale: 1.03 }"
      :while-tap="{ scale: 0.96 }"
      :transition="{ type: 'spring', stiffness: 400, damping: 17 }"
    >
      {{ modelValue ? 'Изменить' : 'Установить' }}
    </motion.button>

    <!-- iOS Style Picker -->
    <Teleport to="body">
      <AnimatePresence>
        <motion.div
          v-if="showPicker"
          class="fixed inset-0 z-50 flex items-end justify-center"
          :initial="{ opacity: 0 }"
          :animate="{ opacity: 1 }"
          :exit="{ opacity: 0 }"
          :transition="{ duration: 0.2 }"
        >
          <!-- Backdrop -->
          <motion.div
            class="absolute inset-0 bg-black/40"
            @click="closePicker"
            :initial="{ opacity: 0 }"
            :animate="{ opacity: 1 }"
            :exit="{ opacity: 0 }"
          />

          <!-- Picker Panel -->
          <motion.div
            class="relative w-full max-w-md bg-bg-surface rounded-t-2xl shadow-xl"
            :initial="{ y: '100%' }"
            :animate="{ y: 0 }"
            :exit="{ y: '100%' }"
            :transition="{ type: 'spring', stiffness: 300, damping: 30 }"
          >
            <!-- Header -->
            <div class="flex items-center justify-between px-4 py-3 border-b border-border">
              <motion.button
                @click="closePicker"
                class="text-text-accent font-medium"
                :while-hover="{ scale: 1.05 }"
                :while-tap="{ scale: 0.95 }"
              >
                Отмена
              </motion.button>
              <span class="font-semibold text-text-primary">
                {{ placeholder || 'Выберите время' }}
              </span>
              <motion.button
                @click="save"
                class="text-text-accent font-medium"
                :while-hover="{ scale: 1.05 }"
                :while-tap="{ scale: 0.95 }"
              >
                Сохранить
              </motion.button>
            </div>

            <!-- Scroll Picker -->
            <div class="relative h-60 overflow-hidden">
              <!-- Selection Indicator -->
              <div class="absolute inset-x-0 top-1/2 w-[90%] mx-auto border-2 border-border-accent rounded-full -translate-y-1/2 h-12 bg-bg-accent/10 pointer-events-none z-10" />

              <!-- Gradient Masks -->
              <div
                class="absolute inset-x-0 top-0 h-20 bg-gradient-to-b from-bg-surface to-transparent pointer-events-none z-10" />
              <div
                class="absolute inset-x-0 bottom-0 h-20 bg-gradient-to-t from-bg-surface to-transparent pointer-events-none z-10" />

              <div class="flex h-full items-stretch justify-center gap-1 px-10">
                <!-- Scrollable Hours -->
                <div ref="hoursScroll" @scroll="handleHoursScroll" @scrollend="handleHoursScrollEnd"
                  class="w-1/2 h-full overflow-y-scroll snap-y snap-mandatory scrollbar-hide"
                  style="-ms-overflow-style: none; scrollbar-width: none;">
                  <!-- Spacer for centering -->
                  <div class="h-24" />

                  <motion.div
                    v-for="(hour, index) in hours"
                    :key="hour"
                    class="h-12 flex items-center justify-center snap-center"
                    :initial="{ opacity: 0, x: -20 }"
                    :animate="{ opacity: 1, x: 0 }"
                    :transition="{ delay: index * 0.02, duration: 0.2 }"
                  >
                    <span :class="[
                      'text-2xl font-medium transition-all',
                      selectedHour === hour
                        ? 'text-text-primary scale-110'
                        : 'text-text-muted'
                    ]">
                      {{ String(hour).padStart(2, '0') }}
                    </span>
                  </motion.div>

                  <!-- Spacer for centering -->
                  <div class="h-24" />
                </div>

                <!-- Colon Separator -->
                <div class="flex items-center pointer-events-none">
                  <span class="text-2xl font-medium text-text-primary">:</span>
                </div>

                <!-- Scrollable Minutes -->
                <div ref="minutesScroll" @scroll="handleMinutesScroll" @scrollend="handleMinutesScrollEnd"
                  class="w-1/2 h-full overflow-y-scroll snap-y snap-mandatory scrollbar-hide"
                  style="-ms-overflow-style: none; scrollbar-width: none;">
                  <!-- Spacer for centering -->
                  <div class="h-24" />

                  <motion.div
                    v-for="(minute, index) in minutes"
                    :key="minute"
                    class="h-12 flex items-center justify-center snap-center"
                    :initial="{ opacity: 0, x: 20 }"
                    :animate="{ opacity: 1, x: 0 }"
                    :transition="{ delay: index * 0.03, duration: 0.2 }"
                  >
                    <span :class="[
                      'text-2xl font-medium transition-all',
                      selectedMinute === minute
                        ? 'text-text-primary scale-110'
                        : 'text-text-muted'
                    ]">
                      {{ String(minute).padStart(2, '0') }}
                    </span>
                  </motion.div>

                  <!-- Spacer for centering -->
                  <div class="h-24" />
                </div>
              </div>
            </div>

            <!-- Quick Actions -->
            <div class="px-4 py-3 border-t border-border space-y-2">
              <motion.button
                @click="save"
                class="w-full py-2.5 bg-bg-accent text-text-on-accent rounded-xl hover:opacity-90 transition-opacity font-medium"
                :while-hover="{ scale: 1.02 }"
                :while-tap="{ scale: 0.97 }"
                :transition="{ type: 'spring', stiffness: 400, damping: 17 }"
              >
                Сохранить
              </motion.button>
            </div>
          </motion.div>
        </motion.div>
      </AnimatePresence>
    </Teleport>
  </div>
</template>

<style scoped>
.scrollbar-hide::-webkit-scrollbar {
  display: none;
}
</style>
