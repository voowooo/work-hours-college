export function formatDateLocal(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

export function getMonday(date: Date): Date {
  const d = new Date(date)
  const day = d.getDay()
  const diff = d.getDate() - day + (day === 0 ? -6 : 1)
  d.setDate(diff)
  d.setHours(0, 0, 0, 0)
  return d
}

export function addDays(date: Date, days: number): Date {
  const d = new Date(date)
  d.setDate(d.getDate() + days)
  return d
}

export function getWeekDays(date: Date): Date[] {
  const monday = getMonday(date)
  const days: Date[] = []
  for (let i = 0; i < 7; i++) {
    days.push(addDays(monday, i))
  }
  return days
}

export function isSameDay(date1: Date, date2: Date): boolean {
  return formatDateLocal(date1) === formatDateLocal(date2)
}

export function isToday(date: Date): boolean {
  return isSameDay(date, new Date())
}

export function getDayName(date: Date): string {
  return date.toLocaleDateString('ru', { weekday: 'short' })
}

export function getMonthName(date: Date): string {
  return date.toLocaleString('ru', { month: 'long' })
}

export function formatDateFull(dateStr: string): string {
  return new Date(dateStr + 'T12:00:00').toLocaleDateString('ru', {
    weekday: 'long',
    day: 'numeric',
    month: 'long'
  })
}

export function formatDateRange(days: Date[]): string {
  if (days.length < 7) return ''
  const start = days[0]?.toLocaleDateString('ru', { day: 'numeric', month: 'short' })
  const end = days[6]?.toLocaleDateString('ru', { day: 'numeric', month: 'short' })
  return `${start} - ${end}`
}

export function getDaysInMonth(year: number, month: number): number {
  return new Date(year, month + 1, 0).getDate()
}

export function getFirstDayOfWeek(year: number, month: number): number {
  return new Date(year, month, 1).getDay()
}

export function formatHours(decimalHours: number): string {
  if (decimalHours === 0) return '0ч'
  const h = Math.floor(decimalHours)
  const m = Math.round((decimalHours - h) * 60)
  if (m === 0) return `${h}ч`
  if (h === 0) return `${m}мин`
  return `${h}ч ${m}мин`
}

export function getMonthDays(year: number, month: number): Array<{ day: number; date: string }> {
  const result: Array<{ day: number; date: string }> = []
  const firstDay = getFirstDayOfWeek(year, month)
  const adjustedFirstDay = firstDay === 0 ? 6 : firstDay - 1
  const daysInMonth = getDaysInMonth(year, month)

  for (let i = 0; i < adjustedFirstDay; i++) {
    result.push({ day: 0, date: '' })
  }

  for (let i = 1; i <= daysInMonth; i++) {
    const date = new Date(year, month, i)
    result.push({ day: i, date: formatDateLocal(date) })
  }

  return result
}
