-- Таблица пользователей
CREATE TABLE IF NOT EXISTS users (
  id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
  telegram_id BIGINT UNIQUE NOT NULL,
  username TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Функция для автоматического обновления updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ language 'plpgsql';

-- Уроки из расписания. Строка сохраняется при успешной загрузке PDF,
-- conducted остаётся true даже если файл расписания позже удалят.
-- manual = TRUE означает, что пользователь сам поставил отметку —
-- авто-расчёт по звонкам такие строки не трогает.
CREATE TABLE IF NOT EXISTS schedule_lessons (
  id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  date DATE NOT NULL,
  period INT NOT NULL CHECK (period BETWEEN 1 AND 14),
  group_name TEXT NOT NULL,
  subject TEXT NOT NULL,
  room TEXT NOT NULL DEFAULT '',
  conducted BOOLEAN NOT NULL DEFAULT FALSE,
  manual BOOLEAN NOT NULL DEFAULT FALSE,
  conducted_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE (user_id, date, period)
);

-- Миграция для существующих баз, где таблица уже создана без manual.
ALTER TABLE schedule_lessons ADD COLUMN IF NOT EXISTS manual BOOLEAN NOT NULL DEFAULT FALSE;

-- Старая таблица ручного учета прихода/ухода больше не используется.
-- Раскомментируйте, чтобы удалить её из базы:
-- DROP TABLE IF EXISTS work_entries;

CREATE INDEX IF NOT EXISTS idx_schedule_lessons_user_date
  ON schedule_lessons (user_id, date);

CREATE INDEX IF NOT EXISTS idx_schedule_lessons_conducted
  ON schedule_lessons (user_id, group_name, subject, conducted);

-- План часов по предмету и группе. Новые пары появляются из расписания.
CREATE TABLE IF NOT EXISTS subject_plans (
  id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  group_name TEXT NOT NULL,
  subject TEXT NOT NULL,
  planned_hours INT NOT NULL DEFAULT 0 CHECK (planned_hours >= 0),
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE (user_id, group_name, subject)
);

CREATE INDEX IF NOT EXISTS idx_subject_plans_user ON subject_plans (user_id);

ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE schedule_lessons ENABLE ROW LEVEL SECURITY;
ALTER TABLE subject_plans ENABLE ROW LEVEL SECURITY;

-- Политики для аутентифицированных пользователей
CREATE POLICY "Users can view own data" ON users
  FOR SELECT USING (auth.uid()::text = id::text);

CREATE POLICY "Users can insert own data" ON users
  FOR INSERT WITH CHECK (auth.uid()::text = id::text);

CREATE POLICY "Users can update own data" ON users
  FOR UPDATE USING (auth.uid()::text = id::text);

CREATE POLICY "Users can view own schedule lessons" ON schedule_lessons
  FOR SELECT USING (user_id = auth.uid());

CREATE POLICY "Users can insert own schedule lessons" ON schedule_lessons
  FOR INSERT WITH CHECK (user_id = auth.uid());

CREATE POLICY "Users can update own schedule lessons" ON schedule_lessons
  FOR UPDATE USING (user_id = auth.uid());

CREATE POLICY "Users can delete own schedule lessons" ON schedule_lessons
  FOR DELETE USING (user_id = auth.uid());

CREATE POLICY "Users can view own subject plans" ON subject_plans
  FOR SELECT USING (user_id = auth.uid());

CREATE POLICY "Users can insert own subject plans" ON subject_plans
  FOR INSERT WITH CHECK (user_id = auth.uid());

CREATE POLICY "Users can update own subject plans" ON subject_plans
  FOR UPDATE USING (user_id = auth.uid());

CREATE POLICY "Users can delete own subject plans" ON subject_plans
  FOR DELETE USING (user_id = auth.uid());

DROP TRIGGER IF EXISTS update_schedule_lessons_updated_at ON schedule_lessons;
CREATE TRIGGER update_schedule_lessons_updated_at
  BEFORE UPDATE ON schedule_lessons
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_subject_plans_updated_at ON subject_plans;
CREATE TRIGGER update_subject_plans_updated_at
  BEFORE UPDATE ON subject_plans
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();
