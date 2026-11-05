-- Таблица пользователей
CREATE TABLE IF NOT EXISTS users (
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
CREATE TABLE IF NOT EXISTS work_entries (
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

-- Индексы для оптимизации
CREATE INDEX IF NOT EXISTS idx_work_entries_user_date ON work_entries(user_id, date);

-- RLS политики (опционально, если используете Supabase Auth)
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE work_entries ENABLE ROW LEVEL SECURITY;

-- Политики для аутентифицированных пользователей
CREATE POLICY "Users can view own data" ON users
  FOR SELECT USING (auth.uid()::text = id::text);

CREATE POLICY "Users can insert own data" ON users
  FOR INSERT WITH CHECK (auth.uid()::text = id::text);

CREATE POLICY "Users can update own data" ON users
  FOR UPDATE USING (auth.uid()::text = id::text);

CREATE POLICY "Users can view own entries" ON work_entries
  FOR SELECT USING (user_id = auth.uid());

CREATE POLICY "Users can insert own entries" ON work_entries
  FOR INSERT WITH CHECK (user_id = auth.uid());

CREATE POLICY "Users can update own entries" ON work_entries
  FOR UPDATE USING (user_id = auth.uid());

CREATE POLICY "Users can delete own entries" ON work_entries
  FOR DELETE USING (user_id = auth.uid());

-- Функция для автоматического обновления updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ language 'plpgsql';

-- Триггер для автоматического обновления updated_at
CREATE TRIGGER update_work_entries_updated_at
  BEFORE UPDATE ON work_entries
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();
