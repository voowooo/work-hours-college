import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { isLessonConducted, lessonPhase, minskDateString, type LessonPhase } from '../features/bells'

function apiOrigin(): string {
  const configured = (import.meta.env.VITE_API_URL || 'http://localhost:8080/query').replace(/\/query\/?$/, '')
  if (typeof window === 'undefined') return configured
  const here = window.location.hostname
  const apiIsLocal = configured.includes('localhost') || configured.includes('127.0.0.1')
  const pageIsLocal = here === 'localhost' || here === '127.0.0.1'
  if (apiIsLocal && !pageIsLocal) return ''
  return configured
}

// Ключи старой версии, которая хранила отметки и план в localStorage.
// Оставлены только для однократного переноса данных на сервер.
const CATALOG_KEY = 'schedule_catalog_v1'
const MARKS_KEY = 'schedule_marks_v1'

export interface Lesson {
  period: number
  group: string
  subject: string
  room: string
  conducted?: boolean
  // Урок добавлен вручную (нет в PDF) — его можно удалить.
  added?: boolean
}

export interface DaySchedule {
  date: string
  found: boolean
  lessons: Lesson[]
  status: 'idle' | 'loading' | 'ready' | 'missing' | 'error'
  error: string | null
}

export interface CatalogItem {
  group: string
  subject: string
  plannedHours: number
  conducted: number
}

export interface LessonMark {
  date: string
  period: number
  group: string
  subject: string
  room: string
  conducted: boolean
}

export interface SubjectStat {
  group: string
  subject: string
  plannedHours: number
  conducted: number
  remaining: number
}

export interface NotificationStatus {
  enabled: boolean
  mutedUntil: string
}

function loadJSON<T>(key: string, fallback: T): T {
  try {
    const raw = localStorage.getItem(key)
    return raw ? (JSON.parse(raw) as T) : fallback
  } catch {
    return fallback
  }
}

function authHeaders(): Record<string, string> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  const userId = localStorage.getItem('telegram_user_id')
  if (userId) headers['X-Telegram-User-ID'] = userId
  const username = localStorage.getItem('telegram_username')
  if (username) headers['X-Telegram-Username'] = username
  return headers
}

function hasAccount(): boolean {
  return Boolean(localStorage.getItem('telegram_user_id'))
}

function clampHours(value: number): number {
  return Number.isFinite(value) ? Math.max(0, Math.round(value)) : 0
}

export const useScheduleStore = defineStore('schedule', () => {
  const days = ref<Record<string, DaySchedule>>({})
  const catalog = ref<CatalogItem[]>([])
  const notifications = ref<NotificationStatus | null>(null)
  const now = ref(new Date())
  let accountReady: Promise<void> | null = null
  let plansRequest: Promise<void> | null = null
  let backfilled = false
  let syncing = false

  function lessonIsConducted(date: string, lesson: Lesson): boolean {
    if (typeof lesson.conducted === 'boolean') return lesson.conducted
    return isLessonConducted(date, lesson.period, now.value)
  }

  // Один перенос старых локальных данных на сервер.
  async function importLocal() {
    if (!hasAccount()) return
    const lessons = loadJSON<LessonMark[]>(MARKS_KEY, [])
    const plans = loadJSON<Array<{ group: string; subject: string; plannedHours: number }>>(CATALOG_KEY, [])
    if (lessons.length === 0 && plans.length === 0) return
    const response = await fetch(`${apiOrigin()}/lessons/import`, {
      method: 'POST',
      headers: authHeaders(),
      body: JSON.stringify({ lessons, plans })
    })
    if (!response.ok) throw new Error('Не удалось перенести локальные данные')
    localStorage.removeItem(MARKS_KEY)
    localStorage.removeItem(CATALOG_KEY)
  }

  async function fetchPlans() {
    if (!hasAccount()) return
    if (plansRequest) return plansRequest
    plansRequest = (async () => {
      const response = await fetch(`${apiOrigin()}/plans`, { headers: authHeaders() })
      if (!response.ok) throw new Error('Не удалось загрузить план')
      const data = await response.json() as { plans: CatalogItem[] | null }
      catalog.value = data.plans ?? []
    })().finally(() => {
      plansRequest = null
    })
    return plansRequest
  }

  function ensureAccount(): Promise<void> {
    accountReady ??= (async () => {
      await importLocal()
      await fetchPlans()
    })().catch(err => {
      accountReady = null
      throw err
    })
    return accountReady
  }

  if (typeof window !== 'undefined') {
    // План нужен и на экранах статистики/настроек, а не только в календаре.
    void ensureAccount().catch(() => undefined)
    // Сайт может убрать старые файлы расписания — сохраняем дни месяца заранее,
    // чтобы проведенные уроки не пропали.
    void backfillMonth().catch(() => undefined)
  }

  async function fetchDay(date: string, silent = false): Promise<void> {
    const current = days.value[date]
    if (current?.status === 'loading') return
    try {
      await ensureAccount()
    } catch {
      // План подтянется при следующем запросе, расписание дня всё равно показываем.
    }

    if (!silent) {
      days.value[date] = {
        date,
        found: current?.found ?? false,
        lessons: current?.lessons ?? [],
        status: 'loading',
        error: null
      }
    }

    try {
      const response = await fetch(`${apiOrigin()}/schedule?date=${encodeURIComponent(date)}`, {
        headers: authHeaders()
      })
      if (!response.ok) {
        throw new Error('Не удалось загрузить расписание')
      }
      const data = await response.json() as { found: boolean; lessons: Lesson[] | null }
      days.value[date] = {
        date,
        found: data.found,
        lessons: data.lessons ?? [],
        status: data.found ? 'ready' : 'missing',
        error: null
      }
      // На сервере могли появиться новые пары — подтягиваем план и счетчики.
      await fetchPlans().catch(() => undefined)
    } catch (err) {
      if (silent && current) return
      days.value[date] = {
        date,
        found: false,
        lessons: current?.lessons ?? [],
        status: 'error',
        error: err instanceof Error ? err.message : 'Не удалось загрузить расписание'
      }
    }
  }

  // Разово сохраняем на сервере дни текущего месяца, чтобы уроки не пропали,
  // если файлы расписания на сайте удалят.
  async function backfillMonth(): Promise<void> {
    if (backfilled) return
    backfilled = true
    if (!hasAccount()) return
    try {
      await ensureAccount()
    } catch {
      // Без плана расписание всё равно можно сохранить.
    }

    const today = minskDateString(now.value)
    const [year, month, dayOfMonth] = today.split('-').map(Number)
    if (!year || !month || !dayOfMonth) return

    const dates: string[] = []
    for (let day = 1; day <= dayOfMonth; day++) {
      const date = `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`
      if (!days.value[date]) dates.push(date)
    }

    const chunk = 4
    for (let i = 0; i < dates.length; i += chunk) {
      await Promise.all(dates.slice(i, i + chunk).map(date => fetchDay(date, true)))
    }
  }

  async function persistDue() {
    const dates = Object.values(days.value)
      .filter(day => day.lessons.some(lesson => !lessonIsConducted(day.date, lesson) && isLessonConducted(day.date, lesson.period, now.value)))
      .map(day => day.date)
    if (dates.length === 0) return
    await Promise.all(dates.map(date => fetchDay(date, true)))
    await fetchPlans().catch(() => undefined)
  }

  if (typeof window !== 'undefined') {
    window.setInterval(() => {
      now.value = new Date()
      if (syncing) return
      syncing = true
      persistDue().finally(() => {
        syncing = false
      })
    }, 15000)
  }

  async function fetchRange(dates: string[]): Promise<void> {
    await Promise.all(dates.map(date => fetchDay(date)))
    await fetchPlans().catch(() => undefined)
  }

  function lessonsFor(date: string): Lesson[] {
    return days.value[date]?.lessons ?? []
  }

  function phase(date: string, period: number): LessonPhase {
    return lessonPhase(date, period, now.value)
  }

  function isConducted(date: string, period: number): boolean {
    const lesson = lessonsFor(date).find(item => item.period === period)
    return lesson ? lessonIsConducted(date, lesson) : isLessonConducted(date, period, now.value)
  }

  // Ручная отметка урока. Сначала обновляем экран, потом сохраняем на сервере;
  // при ошибке возвращаем прежнее значение.
  async function setConducted(date: string, lesson: Lesson, conducted: boolean): Promise<void> {
    const day = days.value[date]
    const stored = day?.lessons.find(item => item.period === lesson.period)
    const previous = stored?.conducted
    if (stored) stored.conducted = conducted

    if (!hasAccount()) return

    try {
      const response = await fetch(`${apiOrigin()}/lessons/mark`, {
        method: 'PUT',
        headers: authHeaders(),
        body: JSON.stringify({
          date,
          period: lesson.period,
          group: lesson.group,
          subject: lesson.subject,
          room: lesson.room,
          conducted
        })
      })
      if (!response.ok) throw new Error('Не удалось сохранить отметку')
      await fetchPlans().catch(() => undefined)
    } catch (err) {
      if (stored && typeof previous === 'boolean') stored.conducted = previous
      throw err
    }
  }

  async function setPlannedHours(group: string, subject: string, value: number) {
    const hours = clampHours(value)
    const index = catalog.value.findIndex(item => item.group === group && item.subject === subject)
    const previous = index === -1 ? undefined : catalog.value[index]
    if (index === -1) {
      catalog.value.push({ group, subject, plannedHours: hours, conducted: 0 })
    } else {
      catalog.value[index] = { ...previous!, plannedHours: hours }
    }

    if (!hasAccount()) return

    try {
      const response = await fetch(`${apiOrigin()}/plans`, {
        method: 'PUT',
        headers: authHeaders(),
        body: JSON.stringify({ group, subject, plannedHours: hours })
      })
      if (!response.ok) throw new Error('Не удалось сохранить план')
      await fetchPlans().catch(() => undefined)
    } catch (err) {
      if (previous) {
        catalog.value[index] = previous
      } else {
        catalog.value = catalog.value.filter(item => !(item.group === group && item.subject === subject))
      }
      throw err
    }
  }

  // Урок, добавленный вручную: его нет в PDF, но он участвует в статистике
  // и отметках по звонкам, как обычный урок.
  async function addLesson(date: string, lesson: Omit<Lesson, 'conducted' | 'added'>): Promise<void> {
    const response = await fetch(`${apiOrigin()}/lessons/add`, {
      method: 'POST',
      headers: authHeaders(),
      body: JSON.stringify({ date, ...lesson })
    })
    if (!response.ok) {
      const text = await response.text().catch(() => '')
      throw new Error(text.trim() || 'Не удалось добавить урок')
    }
    const data = await response.json() as { lesson: Lesson }
    const day = days.value[date]
    const lessons = [...(day?.lessons ?? []), data.lesson].sort((a, b) => a.period - b.period)
    days.value[date] = {
      date,
      found: true,
      lessons,
      status: 'ready',
      error: null
    }
    await fetchPlans().catch(() => undefined)
  }

  // Удаление можно только урока, добавленного вручную: уроки из PDF вернутся
  // при следующей загрузке расписания.
  async function removeLesson(date: string, period: number): Promise<void> {
    const day = days.value[date]
    const previous = day?.lessons ?? []
    if (day) day.lessons = previous.filter(item => item.period !== period)

    try {
      const response = await fetch(`${apiOrigin()}/lessons/remove`, {
        method: 'POST',
        headers: authHeaders(),
        body: JSON.stringify({ date, period })
      })
      if (!response.ok) throw new Error('Не удалось удалить урок')
      await fetchPlans().catch(() => undefined)
    } catch (err) {
      if (day) day.lessons = previous
      throw err
    }
  }

  function conductedCount(date: string): number {
    return lessonsFor(date).filter(lesson => lessonIsConducted(date, lesson)).length
  }

  // --- Уведомления бота -----------------------------------------------------

  async function fetchNotifications(): Promise<void> {
    if (!hasAccount()) {
      notifications.value = null
      return
    }
    const response = await fetch(`${apiOrigin()}/notifications`, { headers: authHeaders() })
    if (!response.ok) throw new Error('Не удалось загрузить настройки уведомлений')
    notifications.value = await response.json() as NotificationStatus
  }

  // Включение уведомлений заодно снимает режим «до завтра».
  async function setNotificationsEnabled(enabled: boolean): Promise<void> {
    const previous = notifications.value
    notifications.value = { enabled, mutedUntil: enabled ? '' : (previous?.mutedUntil ?? '') }

    if (!hasAccount()) return

    try {
      const response = await fetch(`${apiOrigin()}/notifications`, {
        method: 'PUT',
        headers: authHeaders(),
        body: JSON.stringify({ enabled })
      })
      if (!response.ok) throw new Error('Не удалось сохранить настройку')
      notifications.value = await response.json() as NotificationStatus
    } catch (err) {
      notifications.value = previous
      throw err
    }
  }

  // Пробное сообщение в Telegram с кнопкой «отключить до завтра».
  async function sendTestNotification(): Promise<void> {
    const response = await fetch(`${apiOrigin()}/notifications/test`, {
      method: 'POST',
      headers: authHeaders()
    })
    const data = await response.json() as { ok: boolean; error?: string }
    if (!response.ok || !data.ok) throw new Error(data.error || 'Не удалось отправить сообщение')
  }

  function lessonCount(date: string): number {
    return lessonsFor(date).length
  }

  const stats = computed<SubjectStat[]>(() =>
    [...catalog.value]
      .map(item => ({
        group: item.group,
        subject: item.subject,
        plannedHours: item.plannedHours,
        conducted: item.conducted,
        remaining: item.plannedHours > 0 ? Math.max(0, item.plannedHours - item.conducted) : 0
      }))
      .sort((a, b) => a.group.localeCompare(b.group, 'ru') || a.subject.localeCompare(b.subject, 'ru'))
  )

  const totalConducted = computed(() => stats.value.reduce((sum, item) => sum + item.conducted, 0))
  const totalRemaining = computed(() => stats.value.reduce((sum, item) => sum + item.remaining, 0))

  return {
    days,
    catalog,
    notifications,
    stats,
    totalConducted,
    totalRemaining,
    fetchDay,
    fetchRange,
    lessonsFor,
    phase,
    isConducted,
    setConducted,
    setPlannedHours,
    addLesson,
    removeLesson,
    fetchNotifications,
    setNotificationsEnabled,
    sendTestNotification,
    conductedCount,
    lessonCount
  }
})
