# Work Hours Counter

Приложение для учета рабочих часов в Telegram Mini App.

## Структура проекта

```
workHoursCounter/
├── frontend/          # Vue 3 + Vite + Tailwind CSS
│   ├── src/
│   │   ├── components/
│   │   │   └── TimeInput.vue
│   │   ├── composables/
│   │   │   └── useTelegram.ts
│   │   ├── stores/
│   │   │   └── workHours.ts
│   │   └── views/
│   │       ├── CalendarView.vue
│   │       └── StatsView.vue
│   └── ...
└── backend/           # Go + PostgreSQL
    ├── cmd/server/
    │   └── main.go
    └── ...
```

## Установка

### Supabase (PostgreSQL)

1. Создайте проект в [Supabase](https://supabase.com)
2. Выполните SQL из `supabase-schema.sql`
3. Скопируйте connection string

### Backend

```bash
cd backend

# Установите зависимости
go mod tidy

# Настройте переменные окружения
export DB_HOST=your-supabase-host
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=your-password
export DB_NAME=workhours

# Запустите сервер
go run cmd/server/main.go
```

### Frontend

```bash
cd frontend

# Установите зависимости
npm install

# Создайте .env файл
cp .env.example .env

# Запустите в режиме разработки
npm run dev
```

## Supabase Schema

```sql
-- Таблица пользователей
CREATE TABLE users (
  id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
  telegram_id BIGINT UNIQUE NOT NULL,
  username TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Функция для расчета часов (IMMUTABLE)
-- Поддерживает ночные смены: если left_time < arrived, считаем что ушел на следующий день
CREATE OR REPLACE FUNCTION calculate_hours_worked(arrived TIME, left_time TIME)
RETURNS DECIMAL(5,2)
LANGUAGE plpgsql
IMMUTABLE
AS $$
DECLARE
  diff INTERVAL;
  hours DECIMAL(5,2);
BEGIN
  IF arrived IS NULL OR left_time IS NULL THEN
    RETURN 0;
  END IF;
  
  diff := left_time - arrived;
  
  -- Если разница отрицательная, значит ушел на следующий день
  IF diff < INTERVAL '0' THEN
    diff := diff + INTERVAL '24 hours';
  END IF;
  
  hours := EXTRACT(EPOCH FROM diff) / 3600;
  RETURN ROUND(hours, 2);
END;
$$;

-- Таблица рабочих записей
CREATE TABLE work_entries (
  id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  date DATE NOT NULL,
  arrived_at TIME,
  left_at TIME,
  hours_worked DECIMAL(5,2) GENERATED ALWAYS AS (
    calculate_hours_worked(arrived_at, left_at)
  ) STORED,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE(user_id, date)
);

-- Индексы
CREATE INDEX idx_work_entries_user_date ON work_entries(user_id, date);

-- RLS политики
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE work_entries ENABLE ROW LEVEL SECURITY;

CREATE POLICY "Users can view own data" ON users
  FOR SELECT USING (auth.uid()::text = id::text);

CREATE POLICY "Users can view own entries" ON work_entries
  FOR ALL USING (user_id = auth.uid());
```

## API (GraphQL)

Endpoint: `POST /query`

### Queries

```graphql
# Получить текущего пользователя
query {
  me {
    id
    telegramId
    username
  }
}

# Получить запись по дате
query {
  workEntry(date: "2024-01-15") {
    id
    date
    arrivedAt
    leftAt
    hoursWorked
  }
}

# Получить записи за период
query {
  workEntries(startDate: "2024-01-01", endDate: "2024-01-31") {
    id
    date
    arrivedAt
    leftAt
    hoursWorked
  }
}

# Статистика за месяц
query {
  monthStats(year: 2024, month: 1) {
    month
    totalHours
    entries {
      id
      date
      arrivedAt
      leftAt
      hoursWorked
    }
  }
}
```

### Mutations

```graphql
# Создать/обновить запись
mutation {
  upsertWorkEntry(input: {
    date: "2024-01-15"
    arrivedAt: "09:00"
    leftAt: "18:00"
  }) {
    id
    date
    arrivedAt
    leftAt
    hoursWorked
  }
}

# Удалить запись
mutation {
  deleteWorkEntry(date: "2024-01-15")
}
```

## Функциональность

### Календарь (главный экран)
- Отображение календаря на текущий месяц
- Свайп влево/вправо для переключения месяцев
- Выбор даты для установки времени прихода/ухода
- Автоматический расчет отработанных часов

### Статистика
- Календарь по неделям
- Часы под каждым днем
- Итого за неделю и месяц
- Средние часы в день
