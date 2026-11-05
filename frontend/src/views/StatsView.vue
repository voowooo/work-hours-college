<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { motion } from 'motion-v'
import { useScheduleStore, type SubjectStat } from '../stores/schedule'
import StatCard from '../components/StatCard.vue'

const store = useScheduleStore()

const groups = computed(() => {
  const map = new Map<string, SubjectStat[]>()
  for (const item of store.stats) {
    const list = map.get(item.group) ?? []
    list.push(item)
    map.set(item.group, list)
  }
  return [...map.entries()]
})

function conductedWidth(item: SubjectStat): string {
  const scale = Math.max(item.plannedHours, item.conducted, 1)
  return `${(item.conducted / scale) * 100}%`
}

function remainingWidth(item: SubjectStat): string {
  if (item.plannedHours <= item.conducted) return '0%'
  const scale = Math.max(item.plannedHours, item.conducted, 1)
  return `${((item.plannedHours - item.conducted) / scale) * 100}%`
}
</script>

<template>
  <div class="bg-bg-primary">
    <div class="max-w-md mx-auto p-4">
      <motion.h2
        class="text-xl font-semibold mb-4"
        :initial="{ opacity: 0, y: -12 }"
        :animate="{ opacity: 1, y: 0 }"
      >
        Статистика
      </motion.h2>

      <div class="grid grid-cols-2 gap-3 mb-4">
        <StatCard :value="store.totalConducted" label="проведено уроков" color="green" />
        <StatCard :value="store.totalRemaining" label="осталось по плану" color="purple" />
      </div>

      <div class="flex items-center gap-4 text-xs text-text-secondary mb-4 px-1">
        <span class="inline-flex items-center gap-1.5">
          <span class="w-3 h-3 rounded-sm bg-bg-accent" />
          проведено
        </span>
        <span class="inline-flex items-center gap-1.5">
          <span class="w-3 h-3 rounded-sm bg-hours-medium" />
          осталось
        </span>
      </div>

      <div v-if="groups.length === 0" class="bg-bg-surface rounded-xl p-6 text-center text-text-secondary">
        <p>Предметы появятся, когда откроете дни с расписанием.</p>
        <RouterLink to="/" class="inline-block mt-3 text-text-accent text-sm">К календарю</RouterLink>
      </div>

      <div v-else class="space-y-4">
        <motion.section
          v-for="([group, items], groupIndex) in groups"
          :key="group"
          class="bg-bg-surface rounded-xl shadow-sm p-4"
          :initial="{ opacity: 0, y: 16 }"
          :animate="{ opacity: 1, y: 0 }"
          :transition="{ delay: groupIndex * 0.04 }"
        >
          <h3 class="font-semibold mb-3">{{ group || 'Без группы' }}</h3>
          <div class="space-y-3">
            <div v-for="item in items" :key="`${item.group}-${item.subject}`">
              <div class="flex items-baseline justify-between gap-2 mb-1">
                <p class="text-sm font-medium">{{ item.subject || 'Без предмета' }}</p>
                <p class="text-xs text-text-secondary shrink-0">
                  <template v-if="item.plannedHours > 0">
                    {{ item.conducted }} / {{ item.plannedHours }}
                  </template>
                  <template v-else>
                    {{ item.conducted }} · план не задан
                  </template>
                </p>
              </div>
              <div class="h-3 rounded-full bg-bg-surface-secondary overflow-hidden flex">
                <div
                  class="h-full bg-bg-accent transition-all duration-300"
                  :style="{ width: conductedWidth(item) }"
                />
                <div
                  class="h-full bg-hours-medium transition-all duration-300"
                  :style="{ width: remainingWidth(item) }"
                />
              </div>
              <p v-if="item.plannedHours > 0 && item.conducted > item.plannedHours" class="text-[11px] text-hours-low mt-1">
                План превышен на {{ item.conducted - item.plannedHours }}
              </p>
            </div>
          </div>
        </motion.section>
      </div>
    </div>
  </div>
</template>
