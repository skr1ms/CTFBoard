---
name: Testing analysis report
overview: "Аналитический отчёт по тестированию проекта: что хорошо, что плохо, что стоит добавить/убрать."
todos: []
isProject: false
---

# Анализ тестирования проекта

## Общая картина

- **164 тестовых файла**, ~1412 тест-функций, 4 fuzz-теста
- 3 уровня: unit, integration (testcontainers + Postgres), e2e (полный HTTP-стек + Redis + S3)
- Отдельные load-тесты (Vegeta) с тегом `load`
- CI: `make test-race` на push/PR; load по расписанию

## Что хорошо

### Архитектура тестирования

- Чёткое разделение unit / integration / e2e / load -- каждый уровень в своей директории
- Testcontainers для Postgres, Redis, SeaweedFS -- тесты не зависят от локального окружения
- Race-тесты для конкурентных операций: `hint_race_test`, `solve_race_test`, `team_race_test`, `submission_race_test`
- CI запускает `go test -race` -- гонки ловятся автоматически

### Покрытие usecase-слоя

- Все usecase-пакеты покрыты тестами (после текущей миграции -- без хелперов)
- Error paths тестируются почти везде (repo error, not found, forbidden)
- Fuzz-тесты для AES-шифрования и валидации флагов

### pkg/ покрытие

- **Все 15 пакетов** в `pkg/` имеют тесты
- Особенно хорошо: `crypto`, `cache`, `jwt`, `validator`, `scoring`, `websocket`, `httputil`

### Инфраструктура

- Makefile с гранулярными целями: `test-unit`, `test-integration`, `test-e2e`, `test-usecase-{domain}`, `test-coverage-unit`
- Нет orphan mocks -- все 76 моков используются
- `t.Parallel()` используется повсеместно

## Что плохо

### 1. Middleware: не все error paths покрыты

В middleware-тестах пропущены:

- `ChallengeVisibility`: ошибка `compRepo.Get` (500-путь)
- `ScoreboardVisibility`: ошибка `settingsGetter.Get` (500-путь)
- `RateLimit`: ошибка key extraction
- `CompetitionActive`/`CompetitionEnded`: нет тестов на эти middleware

### 2. Нет benchmark-тестов

0 бенчмарков при наличии performance-критичных компонентов:

- `pkg/cache` (TTL-кеш, bounded cache)
- `pkg/scoring` (динамический скоринг)
- `competition/batcher.go` (batch-запись submissions)
- `pkg/websocket/hub.go` (broadcast)

### 3. Отсутствие coverage в CI

- Нет Codecov/Coveralls интеграции
- Coverage считается только локально через `make test-coverage-unit`
- Нет порога "не мерджить PR если coverage упал"

## Что можно удалить

### 1. Удалённый хелпер backup_helper_test.go

В `git status` файл помечен как `deleted` -- просто зафиксировать удаление, оно уже сделано.

### 2. Неиспользуемые импорты

После миграции хелперов стоит проверить что в новых test-файлах нет неиспользуемых импортов (компилятор поймает, но стоит проверить).

## Что стоит добавить (приоритезировано)

### Приоритет 1: Benchmark-тесты для hot paths

- `BenchmarkTTLCache_GetOrLoad`, `BenchmarkBoundedCache_Concurrent`
- `BenchmarkDynamicScoring`
- `BenchmarkSubmissionBatcher_Enqueue`
- `BenchmarkWebsocketHub_Broadcast`

### Приоритет 2: Недостающие error paths в middleware

- 4-5 тестов на пропущенные ветки (ChallengeVisibility Get error, ScoreboardVisibility Get error, и т.д.)

### Приоритет 3: Coverage gate в CI

- Добавить `go tool cover` + threshold check в CI pipeline
- Или интегрировать Codecov для визуализации

