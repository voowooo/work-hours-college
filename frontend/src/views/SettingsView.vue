<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { motion } from 'motion-v'
import { useScheduleStore, type CatalogItem } from '../stores/schedule'

const store = useScheduleStore()
const saveError = ref('')

const autoSubjectAddingGuideStorageKey = 'autoSubjectAddingGuideDismissed'
const showAutoSubjectAddingGuide = ref(
  typeof localStorage === 'undefined' || localStorage.getItem(autoSubjectAddingGuideStorageKey) !== 'true'
)

function dismissAutoSubjectAddingGuide() {
  showAutoSubjectAddingGuide.value = false
  localStorage.setItem(autoSubjectAddingGuideStorageKey, 'true')
}

const groups = computed(() => {
  const map = new Map<string, CatalogItem[]>()
  const items = [...store.catalog].sort(
    (a, b) => a.group.localeCompare(b.group, 'ru') || a.subject.localeCompare(b.subject, 'ru')
  )
  for (const item of items) {
    const list = map.get(item.group) ?? []
    list.push(item)
    map.set(item.group, list)
  }
  return [...map.entries()]
})

async function updateHours(item: CatalogItem, event: Event) {
  const raw = (event.target as HTMLInputElement).value
  const parsed = raw.trim() === '' ? 0 : Number(raw)
  try {
    await store.setPlannedHours(item.group, item.subject, parsed)
    saveError.value = ''
  } catch {
    saveError.value = 'Не удалось сохранить план, попробуйте ещё раз'
  }
}
</script>

<template>
  <div class="bg-bg-primary">
    <div class="max-w-md mx-auto p-4">
      <motion.h2 class="text-xl font-semibold mb-1" :initial="{ opacity: 0, y: -12 }" :animate="{ opacity: 1, y: 0 }">
        Настройки
      </motion.h2>
      <p class="text-sm text-text-secondary mb-4">
        Сколько часов всего нужно провести по каждому предмету в каждой группе.
      </p>

      <div v-if="showAutoSubjectAddingGuide" class="mb-2">
        <div class="text-xs text-text-secondary bg-bg-surface-secondary p-2 rounded-lg">
          <p>
            Новые предметы и группы добавляются из расписания автоматически.
          </p>
          <button class="mt-1 text-sm text-text-accent" @click="dismissAutoSubjectAddingGuide">
            Понятно
          </button>
        </div>
      </div>

      <p v-if="saveError" class="text-sm text-hours-low mb-3">{{ saveError }}</p>

      <div v-if="groups.length === 0" class="bg-bg-surface rounded-xl p-6 text-center text-text-secondary">
        <p>Пока пусто. Откройте день в календаре — предметы появятся сами.</p>
        <RouterLink to="/" class="inline-block mt-3 text-text-accent text-sm">К календарю</RouterLink>
      </div>

      <div v-else class="space-y-4">
        <motion.section v-for="([group, items], groupIndex) in groups" :key="group"
          class="bg-bg-surface rounded-xl shadow-sm p-4" :initial="{ opacity: 0, y: 16 }"
          :animate="{ opacity: 1, y: 0 }" :transition="{ delay: groupIndex * 0.04 }">
          <h3 class="font-semibold mb-3">{{ group || 'Без группы' }}</h3>
          <div class="space-y-3">
            <label v-for="item in items" :key="`${item.group}-${item.subject}`"
              class="flex items-center justify-between gap-3">
              <span class="text-sm">{{ item.subject || 'Без предмета' }}</span>
              <input type="number" min="0" step="1" inputmode="numeric"
                class="w-20 rounded-lg bg-bg-surface-secondary px-3 py-2 text-right text-sm outline-none focus:ring-2 focus:ring-bg-accent/40"
                :value="item.plannedHours || ''" placeholder="0" @change="updateHours(item, $event)" />
            </label>
          </div>
        </motion.section>
      </div>
    </div>
  </div>
</template>
