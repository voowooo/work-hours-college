export interface Bell {
  period: number
  start: string
  end: string
  startMinutes: number
  endMinutes: number
}

export type LessonPhase = 'upcoming' | 'ongoing' | 'done' | 'unknown'

const WEEKDAY: Array<[string, string]> = [
  ['08:00', '08:45'],
  ['08:55', '09:40'],
  ['09:50', '10:35'],
  ['10:45', '11:30'],
  ['11:40', '12:25'],
  ['12:45', '13:30'],
  ['13:40', '14:25'],
  ['14:35', '15:20'],
  ['15:30', '16:15'],
  ['16:35', '17:20'],
  ['17:30', '18:15'],
  ['18:25', '19:10'],
  ['19:20', '20:05'],
  ['20:15', '21:00'],
]

const SATURDAY: Array<[string, string]> = [
  ['08:00', '08:45'],
  ['08:55', '09:40'],
  ['09:50', '10:35'],
  ['10:45', '11:30'],
  ['11:40', '12:25'],
  ['12:35', '13:20'],
  ['13:30', '14:15'],
  ['14:25', '15:10'],
]

function toBell(period: number, start: string, end: string): Bell {
  return {
    period,
    start,
    end,
    startMinutes: toMinutes(start),
    endMinutes: toMinutes(end),
  }
}

function toMinutes(value: string): number {
  const [hours, minutes] = value.split(':').map(Number)
  return (hours ?? 0) * 60 + (minutes ?? 0)
}

const WEEKDAY_BELLS = WEEKDAY.map(([start, end], index) => toBell(index + 1, start, end))
const SATURDAY_BELLS = SATURDAY.map(([start, end], index) => toBell(index + 1, start, end))

const WEEKDAYS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'] as const

function minskParts(now: Date): { date: string; minutes: number } {
  const parts = new Intl.DateTimeFormat('en-GB', {
    timeZone: 'Europe/Minsk',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hourCycle: 'h23',
  }).formatToParts(now)
  const read = (type: Intl.DateTimeFormatPartTypes) => parts.find(part => part.type === type)?.value ?? ''
  return {
    date: `${read('year')}-${read('month')}-${read('day')}`,
    minutes: Number(read('hour')) * 60 + Number(read('minute')),
  }
}

function weekdayIndex(date: string): number {
  const [year, month, day] = date.split('-').map(Number)
  const instant = new Date(Date.UTC(year ?? 1970, (month ?? 1) - 1, day ?? 1, 9, 0, 0))
  const name = new Intl.DateTimeFormat('en-US', {
    timeZone: 'Europe/Minsk',
    weekday: 'short',
  }).format(instant)
  return WEEKDAYS.indexOf(name as (typeof WEEKDAYS)[number])
}

export function lessonBell(date: string, period: number): Bell | null {
  const day = weekdayIndex(date)
  const table = day === 6 ? SATURDAY_BELLS : day === 0 ? [] : WEEKDAY_BELLS
  return table.find(bell => bell.period === period) ?? null
}

export function lessonPhase(date: string, period: number, now = new Date()): LessonPhase {
  const today = minskParts(now)
  if (date < today.date) return 'done'
  if (date > today.date) return 'upcoming'

  const bell = lessonBell(date, period)
  if (!bell) return 'unknown'
  if (today.minutes >= bell.endMinutes) return 'done'
  if (today.minutes >= bell.startMinutes) return 'ongoing'
  return 'upcoming'
}

export function isLessonConducted(date: string, period: number, now = new Date()): boolean {
  return lessonPhase(date, period, now) === 'done'
}

// Сегодняшняя дата по Минску в формате YYYY-MM-DD.
export function minskDateString(now = new Date()): string {
  return minskParts(now).date
}
