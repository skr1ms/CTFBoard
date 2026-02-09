# Техническое задание на разработку платформы CTFBoard

**Contributing:** Проект открыт для всех, кто хочет помочь с разработкой бэкенда или фронтенда. Как участвовать: [CONTRIBUTING.md](.github/CONTRIBUTING.md).

## 1. Введение

Целью данного проекта является разработка программного комплекса для проведения соревнований формата Capture The Flag (CTF). Система представляет собой клиент-серверное веб-приложение, обеспечивающее регистрацию участников, выдачу заданий, автоматическую проверку решений (флагов) и отображение турнирной таблицы в режиме реального времени.

## 2. Используемые технологии и инструменты

### 2.1 Серверная часть (Backend)

- **Язык программирования:** Go (версия 1.22 и выше).
- **HTTP-маршрутизация:** Библиотека `chi` (github.com/go-chi/chi/v5) — выбрана для обеспечения строгой совместимости со стандартной библиотекой `net/http` и реализации модульной архитектуры.
- **База данных:** PostgreSQL 17.
- **Драйвер БД:** `pgx/v5` (обеспечение высокой производительности операций ввода-вывода).
- **Аутентификация:** JWT (JSON Web Tokens) согласно стандарту RFC 7519.

### 2.2 Клиентская часть (Frontend)

- **Фреймворк:** React / Vue.js.
- **Взаимодействие с сервером:** RESTful API.
- **Событийная модель:** Server-Sent Events (SSE) для обновления данных без перезагрузки страницы.

## 3. Функциональные требования

### 3.1 Ролевая модель

Система должна поддерживать следующие роли пользователей с разграничением прав доступа:

1. **Гость (Unauthenticated User):**
    - Просмотр главной страницы и информации о соревновании.
    - Регистрация новой учетной записи (Команды).
    - Аутентификация (Вход в систему).
    - Просмотр публичной турнирной таблицы (Scoreboard).
2. **Участник (User/Team):**
    - Просмотр списка доступных задач (Challenges).
    - Отправка решения (флага) на проверку.
    - Просмотр статистики собственного профиля.
3. **Администратор (Admin):**
    - Создание, редактирование и удаление задач.
    - Управление видимостью задач (публикация/скрытие).
    - Мониторинг журнала действий пользователей.

### 3.2 Основные модули системы

#### 3.2.1 Модуль соревнований

- Задачи должны быть разделены по категориям (Web, Crypto, Pwn, Reverse, Forensics, Misc).
- Каждая задача имеет фиксированную или динамическую стоимость в очках.
- Система должна предотвращать повторную отправку верно решенного флага одной командой.

#### 3.2.2 Модуль турнирной таблицы (Scoreboard)

- Отображение рейтинга команд в порядке убывания набранных очков.
- При равенстве очков выше располагается команда, решившая последнюю задачу раньше по времени.
- Обновление данных на клиенте должно происходить автоматически (push-уведомления через SSE) при изменении состояния (сдача флага любым участником).

#### 3.2.3 Модуль безопасности

- **Rate Limiting:** Реализация механизма ограничения частоты запросов для эндпоинта проверки флагов (не более 5 попыток в минуту). При превышении лимита — временная блокировка возможности ввода.
- **Валидация данных:** Строгая проверка входящих данных на соответствие типам и форматам.

## 4. Описание API интерфейса (REST)

**Документация и интерактивная проверка запросов:** [Swagger UI](https://api.ctfleague.ru/swagger/index.html#/)

| Метод | Эндпоинт | Уровень доступа |
| :---- | :------- | :-------------- |
| **POST** | `/api/v1/auth/login` | Public |
| **POST** | `/api/v1/auth/register` | Public |
| **GET** | `/api/v1/auth/verify-email` | Public |
| **POST** | `/api/v1/auth/forgot-password` | Public |
| **POST** | `/api/v1/auth/reset-password` | Public |
| **GET** | `/api/v1/competition/status` | Public |
| **GET** | `/api/v1/scoreboard` | Public |
| **GET** | `/api/v1/challenges/{ID}/first-blood` | Public |
| **GET** | `/api/v1/users/{ID}` | Public |
| **GET** | `/api/v1/tags` | Public |
| **GET** | `/api/v1/fields` | Public |
| **GET** | `/api/v1/brackets` | Public |
| **GET** | `/api/v1/ratings` | Public |
| **GET** | `/api/v1/ratings/team/{ID}` | Public |
| **GET** | `/api/v1/pages` | Public |
| **GET** | `/api/v1/pages/{slug}` | Public |
| **GET** | `/api/v1/notifications` | Public |
| **GET** | `/api/v1/statistics/general` | Public |
| **GET** | `/api/v1/statistics/challenges` | Public |
| **GET** | `/api/v1/statistics/challenges/{id}` | Public |
| **GET** | `/api/v1/statistics/scoreboard` | Public |
| **GET** | `/api/v1/scoreboard/graph` | Public |
| **GET** | `/api/v1/ws` | Public |
| **GET** | `/api/v1/files/download/*` | Public |
| **POST** | `/api/v1/auth/resend-verification` | User |
| **GET** | `/api/v1/auth/me` | User |
| **GET** | `/api/v1/user/notifications` | User |
| **PATCH** | `/api/v1/user/notifications/{ID}/read` | User |
| **GET** | `/api/v1/user/tokens` | User |
| **POST** | `/api/v1/user/tokens` | User |
| **DELETE** | `/api/v1/user/tokens/{ID}` | User |
| **GET** | `/api/v1/files/{ID}/download` | User |
| **GET** | `/api/v1/teams/my` | User |
| **GET** | `/api/v1/teams/{ID}` | User |
| **POST** | `/api/v1/teams/leave` | User |
| **DELETE** | `/api/v1/teams/me` | User |
| **DELETE** | `/api/v1/teams/members/{ID}` | User |
| **POST** | `/api/v1/teams/transfer-captain` | User |
| **POST** | `/api/v1/teams` | User (verified) |
| **POST** | `/api/v1/teams/join` | User (verified) |
| **POST** | `/api/v1/teams/solo` | User (verified) |
| **GET** | `/api/v1/challenges` | User |
| **GET** | `/api/v1/challenges/{challengeID}/files` | User |
| **GET** | `/api/v1/challenges/{challengeID}/hints` | User |
| **GET** | `/api/v1/challenges/{challengeID}/comments` | User |
| **POST** | `/api/v1/challenges/{challengeID}/comments` | User |
| **DELETE** | `/api/v1/comments/{ID}` | User |
| **POST** | `/api/v1/challenges/{ID}/submit` | User |
| **POST** | `/api/v1/challenges/{challengeID}/hints/{hintID}/unlock` | User |
| **GET** | `/api/v1/admin/competition` | Admin |
| **PUT** | `/api/v1/admin/competition` | Admin |
| **GET** | `/api/v1/admin/settings` | Admin |
| **PUT** | `/api/v1/admin/settings` | Admin |
| **GET** | `/api/v1/admin/configs` | Admin |
| **GET** | `/api/v1/admin/configs/{key}` | Admin |
| **PUT** | `/api/v1/admin/configs/{key}` | Admin |
| **DELETE** | `/api/v1/admin/configs/{key}` | Admin |
| **POST** | `/api/v1/admin/challenges` | Admin |
| **PUT** | `/api/v1/admin/challenges/{ID}` | Admin |
| **DELETE** | `/api/v1/admin/challenges/{ID}` | Admin |
| **POST** | `/api/v1/admin/challenges/{challengeID}/files` | Admin |
| **POST** | `/api/v1/admin/challenges/{challengeID}/hints` | Admin |
| **PUT** | `/api/v1/admin/hints/{ID}` | Admin |
| **DELETE** | `/api/v1/admin/hints/{ID}` | Admin |
| **DELETE** | `/api/v1/admin/files/{ID}` | Admin |
| **POST** | `/api/v1/admin/awards` | Admin |
| **GET** | `/api/v1/admin/awards/team/{teamID}` | Admin |
| **POST** | `/api/v1/admin/teams/{ID}/ban` | Admin |
| **DELETE** | `/api/v1/admin/teams/{ID}/ban` | Admin |
| **PATCH** | `/api/v1/admin/teams/{ID}/hidden` | Admin |
| **PATCH** | `/api/v1/admin/teams/{ID}/bracket` | Admin |
| **POST** | `/api/v1/admin/brackets` | Admin |
| **GET** | `/api/v1/admin/brackets/{ID}` | Admin |
| **PUT** | `/api/v1/admin/brackets/{ID}` | Admin |
| **DELETE** | `/api/v1/admin/brackets/{ID}` | Admin |
| **GET** | `/api/v1/admin/ctf-events` | Admin |
| **POST** | `/api/v1/admin/ctf-events` | Admin |
| **POST** | `/api/v1/admin/ctf-events/{ID}/finalize` | Admin |
| **POST** | `/api/v1/admin/tags` | Admin |
| **PUT** | `/api/v1/admin/tags/{ID}` | Admin |
| **DELETE** | `/api/v1/admin/tags/{ID}` | Admin |
| **POST** | `/api/v1/admin/fields` | Admin |
| **PUT** | `/api/v1/admin/fields/{ID}` | Admin |
| **DELETE** | `/api/v1/admin/fields/{ID}` | Admin |
| **GET** | `/api/v1/admin/pages` | Admin |
| **POST** | `/api/v1/admin/pages` | Admin |
| **GET** | `/api/v1/admin/pages/{ID}` | Admin |
| **PUT** | `/api/v1/admin/pages/{ID}` | Admin |
| **DELETE** | `/api/v1/admin/pages/{ID}` | Admin |
| **POST** | `/api/v1/admin/notifications` | Admin |
| **POST** | `/api/v1/admin/notifications/user/{userID}` | Admin |
| **PUT** | `/api/v1/admin/notifications/{ID}` | Admin |
| **DELETE** | `/api/v1/admin/notifications/{ID}` | Admin |
| **GET** | `/api/v1/admin/submissions` | Admin |
| **GET** | `/api/v1/admin/submissions/challenge/{challengeID}` | Admin |
| **GET** | `/api/v1/admin/submissions/challenge/{challengeID}/stats` | Admin |
| **GET** | `/api/v1/admin/submissions/user/{userID}` | Admin |
| **GET** | `/api/v1/admin/submissions/team/{teamID}` | Admin |
| **GET** | `/api/v1/admin/export` | Admin |
| **GET** | `/api/v1/admin/export/zip` | Admin |
| **POST** | `/api/v1/admin/import` | Admin |
| **GET** | `/health` | Public |
| **GET** | `/metrics` | Public |
| **GET** | `/swagger/*` | Public |
