<script setup lang="ts">
import { ref, computed } from 'vue'
import { motion, AnimatePresence } from 'motion-v'

const STORAGE_KEY = 'alt_viewed'

const images = import.meta.glob('../assets/alt/*.{jpg,jpeg,png,webp,gif}', { eager: true, import: 'default' })
const imageList = Object.values(images).filter(Boolean) as string[]
const imageKeys = Object.keys(images)

const currentImage = ref<string | null>(null)
const currentKey = ref<string | null>(null)
const showEmpty = ref(false)
const loading = ref(false)
const imageLoaded = ref(false)
const showGallery = ref(false)
const lightboxIndex = ref<number | null>(null)

let touchStartX = 0
let touchStartY = 0

function getViewed(): Set<string> {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    return raw ? new Set(JSON.parse(raw)) : new Set()
  } catch {
    return new Set()
  }
}

function saveViewed(viewed: Set<string>) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify([...viewed]))
}

const viewed = ref<Set<string>>(getViewed())
const viewedCount = computed(() => viewed.value.size)
const totalCount = computed(() => imageList.length)

const viewedImages = computed(() =>
  imageKeys
    .map((key, i) => ({ key, src: imageList[i] }))
    .filter(item => item.key && viewed.value.has(item.key))
    .map(item => item.src!)
)

function getRandomImage() {
  if (imageList.length === 0) {
    showEmpty.value = true
    setTimeout(() => { showEmpty.value = false }, 2000)
    return
  }

  const index = Math.floor(Math.random() * imageList.length)
  const newKey = imageKeys[index] ?? null
  const newSrc = imageList[index] ?? null

  if (newKey) {
    const v = getViewed()
    v.add(newKey)
    saveViewed(v)
    viewed.value = v
  }

  loading.value = true
  imageLoaded.value = false
  currentKey.value = newKey
  currentImage.value = null
  requestAnimationFrame(() => {
    currentImage.value = newSrc
  })
}

function onImageLoad() {
  imageLoaded.value = true
  loading.value = false
}

function onImageError() {
  loading.value = false
  imageLoaded.value = false
}

function openLightbox(index: number) {
  lightboxIndex.value = index
}

function closeLightbox() {
  lightboxIndex.value = null
}

function lightboxNext() {
  if (lightboxIndex.value === null) return
  lightboxIndex.value = (lightboxIndex.value + 1) % viewedImages.value.length
}

function lightboxPrev() {
  if (lightboxIndex.value === null) return
  lightboxIndex.value = (lightboxIndex.value - 1 + viewedImages.value.length) % viewedImages.value.length
}

function onLightboxTouchStart(e: TouchEvent) {
  touchStartX = e.touches[0]?.clientX ?? 0
  touchStartY = e.touches[0]?.clientY ?? 0
}

function onLightboxTouchEnd(e: TouchEvent) {
  const dx = (e.changedTouches[0]?.clientX ?? 0) - touchStartX
  const dy = (e.changedTouches[0]?.clientY ?? 0) - touchStartY
  if (Math.abs(dx) > Math.abs(dy) && Math.abs(dx) > 50) {
    dx < 0 ? lightboxNext() : lightboxPrev()
  }
}
</script>

<template>
  <div class="bg-bg-primary">
    <div class="max-w-md mx-auto p-4">
      <motion.div class="flex flex-col items-center gap-6" :initial="{ opacity: 0, y: 20 }"
        :animate="{ opacity: 1, y: 0 }" :transition="{ duration: 0.3 }">

        <motion.div v-if="totalCount > 0" class="text-sm text-text-secondary font-medium text-center" :initial="{ opacity: 0 }"
          :animate="{ opacity: 1 }" :transition="{ delay: 0.2 }">
          <p>
            {{ viewedCount }}/{{ totalCount }} <br>
            уникальных альтушек открыто
          </p>
        </motion.div>

        <!-- Gallery Toggle -->
        <div class="flex gap-2 w-full">
          <motion.button
            @click="showGallery = false"
            class="flex-1 py-2 rounded-xl text-sm font-medium transition-colors"
            :class="!showGallery ? 'bg-bg-accent text-text-on-accent' : 'bg-bg-surface-secondary text-text-secondary'"
            :while-tap="{ scale: 0.95 }"
          >
            Рандом
          </motion.button>
          <motion.button
            @click="showGallery = true"
            class="flex-1 py-2 rounded-xl text-sm font-medium transition-colors"
            :class="showGallery ? 'bg-bg-accent text-text-on-accent' : 'bg-bg-surface-secondary text-text-secondary'"
            :while-tap="{ scale: 0.95 }"
          >
            Галерея
          </motion.button>
        </div>

        <!-- Random Mode -->
        <template v-if="!showGallery">
          <div class="w-full min-h-[280px] flex justify-center items-center">
            <AnimatePresence mode="wait">
              <motion.div v-if="currentImage" :key="currentImage"
                class="w-full h-100 rounded-xl overflow-hidden relative"
                :initial="{ opacity: 0, scale: 0.9, y: 20 }"
                :animate="{ opacity: 1, scale: 1, y: 0 }"
                :exit="{ opacity: 0, scale: 0.9, y: -20 }"
                :transition="{ type: 'spring', stiffness: 300, damping: 25 }">

                <div v-if="loading" class="w-full aspect-[3/4] rounded-xl overflow-hidden">
                  <div class="w-full h-full skeleton-shimmer bg-bg-surface-secondary rounded-xl" />
                </div>

                <img
                  :src="currentImage"
                  alt="random"
                  class="w-full max-h-full object-cover rounded-xl transition-opacity duration-300"
                  :class="imageLoaded ? 'opacity-100' : 'opacity-0 absolute inset-0'"
                  @load="onImageLoad"
                  @error="onImageError"
                />
              </motion.div>

              <motion.div v-else-if="showEmpty" key="empty"
                class="text-center text-text-secondary py-12"
                :initial="{ opacity: 0 }"
                :animate="{ opacity: 1 }"
                :exit="{ opacity: 0 }">
                <p class="text-lg">Нет изображений</p>
                <p class="text-sm mt-1">Добавьте фото в src/assets/alt/</p>
              </motion.div>

              <motion.div v-else key="placeholder"
                class="flex flex-col items-center justify-center py-16 text-text-muted"
                :initial="{ opacity: 0 }"
                :animate="{ opacity: 1 }"
                :exit="{ opacity: 0 }">
                <svg class="w-16 h-16 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
                    d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                </svg>
              </motion.div>
            </AnimatePresence>
          </div>

          <motion.button @click="getRandomImage" class="px-8 py-3 bg-bg-accent text-text-on-accent rounded-xl shadow-md"
            :while-hover="{ scale: 1.05, y: -2 }" :while-tap="{ scale: 0.95 }"
            :transition="{ type: 'spring', stiffness: 400, damping: 17 }">
            рандомная альтушка
          </motion.button>
        </template>

        <!-- Gallery Mode -->
        <template v-else>
          <motion.div
            v-if="viewedImages.length > 0"
            class="w-full grid grid-cols-3 gap-1.5"
            :initial="{ opacity: 0 }"
            :animate="{ opacity: 1 }"
            :transition="{ duration: 0.3 }"
          >
            <motion.div
              v-for="(src, index) in viewedImages"
              :key="src"
              class="aspect-[3/4] rounded-lg overflow-hidden cursor-pointer"
              :initial="{ opacity: 0, scale: 0.8 }"
              :animate="{ opacity: 1, scale: 1 }"
              :transition="{ delay: index * 0.03, duration: 0.2 }"
              :while-tap="{ scale: 0.95 }"
              @click="openLightbox(index)"
            >
              <img
                :src="src"
                :alt="`alt ${index + 1}`"
                class="w-full h-full object-cover"
                loading="lazy"
              />
            </motion.div>
          </motion.div>

          <motion.div v-else
            class="text-center text-text-secondary py-12"
            :initial="{ opacity: 0 }"
            :animate="{ opacity: 1 }"
          >
            <p class="text-lg">Пока пусто</p>
            <p class="text-sm mt-1">Откройте альтушки через рандом</p>
          </motion.div>
        </template>
      </motion.div>
    </div>

    <!-- Lightbox -->
    <Teleport to="body">
      <AnimatePresence>
        <motion.div
          v-if="lightboxIndex !== null"
          class="fixed inset-0 z-50 flex items-center justify-center bg-black/90"
          :initial="{ opacity: 0 }"
          :animate="{ opacity: 1 }"
          :exit="{ opacity: 0 }"
          :transition="{ duration: 0.2 }"
          @click.self="closeLightbox"
          @touchstart="onLightboxTouchStart"
          @touchend="onLightboxTouchEnd"
        >
          <!-- Close -->
          <motion.button
            @click="closeLightbox"
            class="absolute top-4 right-4 z-10 w-10 h-10 flex items-center justify-center rounded-full bg-white/10 text-white"
            :while-tap="{ scale: 0.9 }"
          >
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </motion.button>

          <!-- Counter -->
          <div class="absolute top-4 left-4 z-10 text-white/70 text-sm font-medium">
            {{ lightboxIndex + 1 }} / {{ viewedImages.length }}
          </div>

          <!-- Prev -->
          <motion.button
            @click.stop="lightboxPrev"
            class="absolute left-2 z-10 w-10 h-10 flex items-center justify-center rounded-full bg-white/10 text-white"
            :while-tap="{ scale: 0.9 }"
          >
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
            </svg>
          </motion.button>

          <!-- Next -->
          <motion.button
            @click.stop="lightboxNext"
            class="absolute right-2 z-10 w-10 h-10 flex items-center justify-center rounded-full bg-white/10 text-white"
            :while-tap="{ scale: 0.9 }"
          >
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
            </svg>
          </motion.button>

          <!-- Image -->
          <AnimatePresence mode="wait">
            <motion.img
              :key="lightboxIndex"
              :src="viewedImages[lightboxIndex]"
              :alt="`alt ${lightboxIndex + 1}`"
              class="max-w-[90vw] max-h-[85vh] object-contain rounded-lg"
              :initial="{ opacity: 0, scale: 0.9 }"
              :animate="{ opacity: 1, scale: 1 }"
              :exit="{ opacity: 0, scale: 0.9 }"
              :transition="{ type: 'spring', stiffness: 300, damping: 25 }"
            />
          </AnimatePresence>
        </motion.div>
      </AnimatePresence>
    </Teleport>
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
