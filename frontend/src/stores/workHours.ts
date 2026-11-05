import { defineStore } from 'pinia'
import { ref } from 'vue'

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/query'

export interface WorkEntry {
  id: string
  userId: string
  date: string
  arrivedAt: string | null
  leftAt: string | null
  hoursWorked: number
}

export interface MonthStats {
  month: string
  totalHours: number
  entries: WorkEntry[]
}

export const useWorkHoursStore = defineStore('workHours', () => {
  const entries = ref<Map<string, WorkEntry>>(new Map())
  const currentMonthStats = ref<MonthStats | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  const getHeaders = () => {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json'
    }
    const userId = localStorage.getItem('telegram_user_id')
    if (userId) {
      headers['X-Telegram-User-ID'] = userId
    }
    const username = localStorage.getItem('telegram_username')
    if (username) {
      headers['X-Telegram-Username'] = username
    }
    return headers
  }

  const graphqlFetch = async (query: string, variables?: Record<string, any>) => {
    console.log('GraphQL request:', { query, variables, headers: getHeaders() })
    const response = await fetch(API_URL, {
      method: 'POST',
      headers: getHeaders(),
      body: JSON.stringify({ query, variables })
    })
    const data = await response.json()
    console.log('GraphQL response:', data)
    if (data.errors) {
      throw new Error(data.errors[0].message)
    }
    return data.data
  }

  const fetchEntry = async (date: string): Promise<WorkEntry | null> => {
    try {
      const data = await graphqlFetch(`
        query GetEntry($date: String!) {
          workEntry(date: $date) {
            id
            userId
            date
            arrivedAt
            leftAt
            hoursWorked
          }
        }
      `, { date })
      const entry = data.workEntry
      if (entry) {
        entries.value.set(date, entry)
      }
      return entry
    } catch (err) {
      console.error('Error fetching entry:', err)
      return null
    }
  }

  const fetchEntries = async (startDate: string, endDate: string): Promise<WorkEntry[]> => {
    loading.value = true
    error.value = null
    try {
      const data = await graphqlFetch(`
        query GetEntries($startDate: String!, $endDate: String!) {
          workEntries(startDate: $startDate, endDate: $endDate) {
            id
            userId
            date
            arrivedAt
            leftAt
            hoursWorked
          }
        }
      `, { startDate, endDate })
      const result = data.workEntries || []
      result.forEach((entry: WorkEntry) => {
        entries.value.set(entry.date, entry)
      })
      return result
    } catch (err: any) {
      error.value = err.message
      return []
    } finally {
      loading.value = false
    }
  }

  const upsertEntry = async (
    date: string,
    arrivedAt: string | null,
    leftAt: string | null
  ): Promise<WorkEntry | null> => {
    loading.value = true
    error.value = null
    try {
      const data = await graphqlFetch(`
        mutation UpsertEntry($input: UpsertWorkEntryInput!) {
          upsertWorkEntry(input: $input) {
            id
            userId
            date
            arrivedAt
            leftAt
            hoursWorked
          }
        }
      `, { input: { date, arrivedAt, leftAt } })
      const entry = data.upsertWorkEntry
      entries.value.set(date, entry)
      return entry
    } catch (err: any) {
      error.value = err.message
      return null
    } finally {
      loading.value = false
    }
  }

  const deleteEntry = async (date: string): Promise<boolean> => {
    try {
      await graphqlFetch(`
        mutation DeleteEntry($date: String!) {
          deleteWorkEntry(date: $date)
        }
      `, { date })
      entries.value.delete(date)
      return true
    } catch (err) {
      console.error('Error deleting entry:', err)
      return false
    }
  }

  const fetchMonthStats = async (year: number, month: number): Promise<MonthStats | null> => {
    loading.value = true
    error.value = null
    try {
      const data = await graphqlFetch(`
        query GetMonthStats($year: Int!, $month: Int!) {
          monthStats(year: $year, month: $month) {
            month
            totalHours
            entries {
              id
              userId
              date
              arrivedAt
              leftAt
              hoursWorked
            }
          }
        }
      `, { year, month })
      const stats = data.monthStats
      currentMonthStats.value = stats
      stats.entries.forEach((entry: WorkEntry) => {
        entries.value.set(entry.date, entry)
      })
      return stats
    } catch (err: any) {
      error.value = err.message
      return null
    } finally {
      loading.value = false
    }
  }

  const getEntryByDate = (date: string): WorkEntry | undefined => {
    return entries.value.get(date)
  }

  const getHoursForDate = (date: string): number => {
    const entry = entries.value.get(date)
    return entry?.hoursWorked || 0
  }

  return {
    entries,
    currentMonthStats,
    loading,
    error,
    fetchEntry,
    fetchEntries,
    upsertEntry,
    deleteEntry,
    fetchMonthStats,
    getEntryByDate,
    getHoursForDate
  }
})
