# Task Service

Сервис для управления задачами с HTTP API на Go.

## Требования

- Go `1.23+`
- Docker и Docker Compose

## Быстрый запуск через Docker Compose

```bash
docker compose up --build
```

После запуска сервис будет доступен по адресу `http://localhost:8080`.

Если `postgres` уже запускался ранее со старой схемой, пересоздай volume:

```bash
docker compose down -v
docker compose up --build
```

Причина в том, что SQL-файл из `migrations/0001_create_tasks.up.sql` монтируется в `docker-entrypoint-initdb.d` и применяется только при инициализации пустого data volume.

## Swagger

Swagger UI:

```text
http://localhost:8080/swagger/
```

OpenAPI JSON:

```text
http://localhost:8080/swagger/openapi.json
```

## API

Базовый префикс API:

```text
/api/v1
```

Основные маршруты:

- `POST /api/v1/tasks`
- `GET /api/v1/tasks`
- `GET /api/v1/tasks/{id}`
- `PUT /api/v1/tasks/{id}`
- `DELETE /api/v1/tasks/{id}`

## ЧТО ИЗМЕНИЛОСЬ В ПРИЛОЖЕНИИ 

Добавлена поддержка **периодических задач**. Теперь врач может:

### Создание периодической задачи
При создании задачи можно указать, как часто её нужно повторять:

```json
{
  "title": "Обзвон пациентов",
  "recurrence": { "type": "daily", "interval": 1 }
}
```

### Поддерживаемые типы периодичности
1. **Ежедневно** (каждый N-й день) — звонить пациентам через день
2. **Ежемесячно** (на определённые числа) — отчётность 1-го и 15-го числа
3. **На конкретные даты** — осмотры в определённые дни
4. **Чётные/нечётные дни** — задачи только на чётные или нечётные числа

### Автоматическое создание вхождений задач
- Система ночью автоматически создаёт экземпляры периодической задачи на неделю/месяц вперёд
- Врач видит все созданные задачи в списке и может отмечать их как выполненные
- Новые вхождения создаются автоматически по мере приближения дат

### Ручное добавление
Врач может явно создать одно вхождение через API:
```
POST /api/v1/tasks/{parent_id}/occurrences
{
  "title": "Отклонение от графика",
  "due_date": "2026-05-01"
}
```

---

## КАК РЕАЛИЗОВАНО

### Архитектура решения

**1. Модель данных**
- Структура `Task` расширена полями: `Recurrence`, `LastScheduledAt`, `ParentID`, `DueDate`
- `Recurrence` содержит тип и параметры периодичности (daily/monthly/specific_dates/even/odd)
- `ParentID` — связь на родительскую периодическую задачу для occurrence
- `DueDate` — дата выполнения для конкретного вхождения

**2. База данных**
```sql
-- Новые колонки в таблице tasks:
recurrence_type TEXT NULL           -- "daily", "monthly", "specific_dates", "even", "odd"
recurrence_config JSONB NULL        -- {"type":"daily","interval":1} etc.
last_scheduled_at TIMESTAMPTZ NULL  -- когда последнее вхождение было создано
parent_id BIGINT NULL               -- ссылка на родительскую задачу
due_date DATE NULL                  -- дата выполнения occurrence

-- Уникальный индекс:
CREATE UNIQUE INDEX ux_tasks_parent_due_date ON tasks(parent_id, due_date) 
  WHERE parent_id IS NOT NULL AND due_date IS NOT NULL
```

**3. Слой валидации (usecase)**
- Функция `validateRecurrence` проверяет:
  - `daily`: `interval >= 1`
  - `monthly`: `month_days` в диапазоне 1..31
  - `specific_dates`: корректный формат ISO дат "YYYY-MM-DD"
  - `even/odd`: нет доп. параметров

**4. Планировщик (cmd/scheduler)**
```bash
# Создание occurrence на N дней вперёд
go run ./cmd/scheduler --days 7

# или на остаток текущего месяца
go run ./cmd/scheduler --month
```

Логика:
- Сканирует все задачи с `recurrence != nil`
- Для каждого дня в окне проверяет `shouldScheduleForDate`:
  - `daily`: проверить интервал дней от `last_scheduled_at`
  - `monthly`: матчить номер дня месяца
  - `specific_dates`: проверить наличие в списке
  - `even/odd`: проверить чётность дня
- При совпадении создаёт occurrence с `parent_id` и `due_date`
- Обновляет `last_scheduled_at` родителя для предотвращения дублей

**5. API**
- Маршруты поддерживают объект `recurrence` в JSON
- Новый маршрут `POST /api/v1/tasks/{parent_id}/occurrences` для ручного создания

### Ключевые решения

| Решение | Причина |
|---------|---------|
| **JSONB для recurrence_config** | Гибкость: добавлять новые параметры без миграций |
| **Date-only (без времени)** | Убирает ошибки с часовыми поясами, предсказуемые индексы |
| **LastScheduledAt + уникальный индекс** | Двойная защита от дублей при перезапуске планировщика |
| **Occurrence как отдельные записи в БД** | Простая фильтрация, аудит, возможность редактирования каждого вхождения |
| **Два способа создания** | Автоматический (планировщик) + ручной (API) — покрывает все сценарии |

### Файлы, которые были изменены/добавлены

- `internal/domain/task/task.go` — добавлены структуры `Recurrence` и поля в `Task`
- `internal/repository/postgres/task_repository.go` — сохранение/чтение recurrence полей
- `internal/usecase/task/service.go` — валидация, `CreateOccurrence` метод
- `internal/transport/http/handlers/` — DTO, поддержка recurrence в JSON
- `internal/scheduler/scheduler.go` — логика создания occurrence
- `migrations/0002_add_last_scheduled.up.sql`, `0003_add_parent_due_date.up.sql` — схема БД
