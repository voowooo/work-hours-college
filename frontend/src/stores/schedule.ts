import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

function apiOrigin(): string {
  const configured = (import.meta.env.VITE_API_URL || 'http://localhost:8080/query').replace(/\/query\/?$/, '')
  if (typeof window === 'undefined') return configured
  const here = window.location.hostname
  const apiIsLocal = configured.includes('localhost') || configured.includes('127.0.0.1')
  const pageIsLocal = here === 'localhost' || here === '127.0.0.1'
  if (apiIsLocal && !pageIsLocal) return ''
  return configured
}

const CATALOG_KEY = 'schedule_catalog_v1'
const MARKS_KEY = 'schedule_marks_v1'

export interface Lesson {
  period: number
  group: string
  subject: string
  room: string
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

function pairKey(group: string, subject: string): string {
  return `${group.trim().toLowerCase()}|${subject.trim().toLowerCase()}`
}

function loadJSON<T>(key: string, fallback: T): T {
  try {
    const raw = localStorage.getItem(key)
    return raw ? (JSON.parse(raw) as T) : fallback
  } catch {
    return fallback
  }
}

export const useScheduleStore = defineStore('schedule', () => {
  const days = ref<Record<string, DaySchedule>>({})
  const catalog = ref<CatalogItem[]>(loadJSON(CATALOG_KEY, []))
  const marks = ref<LessonMark[]>(loadJSON(MARKS_KEY, []))

  function saveCatalog() {
    localStorage.setItem(CATALOG_KEY, JSON.stringify(catalog.value))
  }

  function saveMarks() {
    localStorage.setItem(MARKS_KEY, JSON.stringify(marks.value))
  }

  function ensureCatalog(group: string, subject: string) {
    if (!group && !subject) return
    const key = pairKey(group, subject)
    if (catalog.value.some(item => pairKey(item.group, item.subject) === key)) return
    catalog.value.push({ group, subject, plannedHours: 0 })
    saveCatalog()
  }

  function syncDay(date: string, lessons: Lesson[]) {
    const previous = marks.value.filter(mark => mark.date === date)
    const rest = marks.value.filter(mark => mark.date !== date)
    const next = lessons.map(lesson => {
      const prev = previous.find(mark => mark.period === lesson.period)
      ensureCatalog(lesson.group, lesson.subject)
      return {
        date,
        period: lesson.period,
        group: lesson.group,
        subject: lesson.subject,
        room: lesson.room,
        conducted: prev?.conducted ?? false
      }
    })
    marks.value = [...rest, ...next]
    saveMarks()
  }

  async function fetchDay(date: string): Promise<void> {
    const current = days.value[date]
    if (current?.status === 'loading') return

    days.value[date] = {
      date,
      found: current?.found ?? false,
      lessons: current?.lessons ?? [],
      status: 'loading',
      error: null
    }

    try {
      const response = await fetch(`${apiOrigin()}/schedule?date=${encodeURIComponent(date)}`)
      if (!response.ok) {
        throw new Error('Не удалось загрузить расписание')
      }
      const data = await response.json() as { found: boolean; lessons: Lesson[] | null }
      const lessons = data.lessons ?? []
      if (data.found) {
        syncDay(date, lessons)
      }
      days.value[date] = {
        date,
        found: data.found,
        lessons,
        status: data.found ? 'ready' : 'missing',
        error: null
      }
    } catch (err) {
      days.value[date] = {
        date,
        found: false,
        lessons: [],
        status: 'error',
        error: err instanceof Error ? err.message : 'Не удалось загрузить расписание'
      }
    }
  }

  async function fetchRange(dates: string[]): Promise<void> {
    await Promise.all(dates.map(date => fetchDay(date)))
  }

  function lessonsFor(date: string): Lesson[] {
    return days.value[date]?.lessons ?? []
  }

  function markFor(date: string, period: number): LessonMark | undefined {
    return marks.value.find(mark => mark.date === date && mark.period === period)
  }

  function isConducted(date: string, period: number): boolean {
    return markFor(date, period)?.conducted ?? false
  }

  function setConducted(date: string, lesson: Lesson, conducted: boolean) {
    const index = marks.value.findIndex(mark => mark.date === date && mark.period === lesson.period)
    if (index === -1) {
      marks.value.push({
        date,
        period: lesson.period,
        group: lesson.group,
        subject: lesson.subject,
        room: lesson.room,
        conducted
      })
    } else {
      const current = marks.value[index]
      if (!current) return
      marks.value[index] = { ...current, conducted }
    }
    ensureCatalog(lesson.group, lesson.subject)
    saveMarks()
  }

  function setDayConducted(date: string, conducted: boolean) {
    const lessons = lessonsFor(date)
    for (const lesson of lessons) {
      setConducted(date, lesson, conducted)
    }
  }

  function setPlannedHours(group: string, subject: string, plannedHours: number) {
    const key = pairKey(group, subject)
    const item = catalog.value.find(entry => pairKey(entry.group, entry.subject) === key)
    const hours = Number.isFinite(plannedHours) ? Math.max(0, Math.round(plannedHours)) : 0
    if (!item) {
      catalog.value.push({ group, subject, plannedHours: hours })
    } else {
      item.plannedHours = hours
    }
    saveCatalog()
  }

  function conductedCount(date: string): number {
    return marks.value.filter(mark => mark.date === date && mark.conducted).length
  }

  function lessonCount(date: string): number {
    return lessonsFor(date).length
  }

  const stats = computed<SubjectStat[]>(() => {
    const conducted = new Map<string, number>()
    for (const mark of marks.value) {
      if (!mark.conducted) continue
      const key = pairKey(mark.group, mark.subject)
      conducted.set(key, (conducted.get(key) ?? 0) + 1)
    }

    return [...catalog.value]
      .map(item => {
        const done = conducted.get(pairKey(item.group, item.subject)) ?? 0
        const remaining = item.plannedHours > 0 ? Math.max(0, item.plannedHours - done) : 0
        return {
          group: item.group,
          subject: item.subject,
          plannedHours: item.plannedHours,
          conducted: done,
          remaining
        }
      })
      .sort((a, b) => a.group.localeCompare(b.group, 'ru') || a.subject.localeCompare(b.subject, 'ru'))
  })

  const totalConducted = computed(() => stats.value.reduce((sum, item) => sum + item.conducted, 0))
  const totalRemaining = computed(() => stats.value.reduce((sum, item) => sum + item.remaining, 0))

  return {
    days,
    catalog,
    marks,
    stats,
    totalConducted,
    totalRemaining,
    fetchDay,
    fetchRange,
    lessonsFor,
    isConducted,
    setConducted,
    setDayConducted,
    setPlannedHours,
    conductedCount,
    lessonCount
  }
})
