# API Спецификация

Проект предоставляет REST API для управления CTF-соревнованием. API построен на Go с использованием chi router и автоматически генерирует документацию через OpenAPI/SwaggerUI.

## Базовый URL

- Локальная разработка: `http://localhost:8080`
- Продакшен: `https://api.ctfleague.ru`

## Документация API

Интерактивная документация доступна по адресу:

- SwaggerUI: `https://api.ctfleague.ru/swagger/index.html`
- OpenAPI Schema: `https://api.ctfleague.ru/swagger/doc.json`

## Формат данных

Все запросы и ответы используют формат JSON с кодировкой UTF-8.

## Аутентификация

Большинство эндпоинтов требуют аутентификации через JWT токены. Для получения токена необходимо:

1. Зарегистрироваться через `/api/v1/auth/register`
2. Войти через `/api/v1/auth/login` и получить пару токенов (access и refresh)
3. Использовать `access_token` в заголовке `Authorization: Bearer <token>`

## Эндпоинты

### Health Check

#### GET /health

Проверка работоспособности сервиса.

**Запрос:**

```
GET /health
```

**Ответ:**

```
OK
```

**Статусы:**

- `200 OK` - сервис работает

### Аутентификация

#### POST /api/v1/auth/register

Регистрация нового пользователя.

**Запрос:**

```
POST /api/v1/auth/register
Content-Type: application/json
```

**Тело запроса:**

```json
{
  "username": "player1",
  "email": "player1@example.com",
  "password": "SecurePassword123!"
}
```

**Параметры:**

- `username` (string, required) - имя пользователя. Должно быть уникальным
- `email` (string, required) - электронная почта. Должна быть уникальной и валидной
- `password` (string, required) - пароль. Должен соответствовать требованиям сложности

**Ответ (успех):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "player1",
  "email": "player1@example.com",
  "created_at": "2024-01-15T10:30:00Z"
}
```

**Статусы:**

- `201 Created` - пользователь успешно зарегистрирован
- `400 Bad Request` - некорректные данные запроса
- `409 Conflict` - пользователь с таким username или email уже существует

#### POST /api/v1/auth/login

Вход в систему и получение JWT токенов.

**Запрос:**

```
POST /api/v1/auth/login
Content-Type: application/json
```

**Тело запроса:**

```json
{
  "email": "player1@example.com",
  "password": "SecurePassword123!"
}
```

**Параметры:**

- `email` (string, required) - электронная почта
- `password` (string, required) - пароль

**Ответ (успех):**

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "access_expires_at": 1705321800,
  "refresh_expires_at": 1705590000
}
```

**Статусы:**

- `200 OK` - успешный вход
- `400 Bad Request` - некорректные данные запроса
- `401 Unauthorized` - неверный email или пароль

#### GET /api/v1/auth/me

Получение информации о текущем аутентифицированном пользователе.

**Запрос:**

```
GET /api/v1/auth/me
Authorization: Bearer <access_token>
```

**Ответ (успех):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "player1",
  "email": "player1@example.com",
  "team_id": "660e8400-e29b-41d4-a716-446655440001",
  "created_at": "2024-01-15T10:30:00Z"
}
```

**Статусы:**

- `200 OK` - информация успешно получена
- `401 Unauthorized` - токен отсутствует или невалиден

### Пользователи

#### GET /api/v1/users/

Получение публичного профиля пользователя.

**Запрос:**

```
GET /api/v1/users/{id}
```

**Параметры пути:**

- `id` (UUID, required) - идентификатор пользователя

**Ответ (успех):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "player1",
  "team_id": "660e8400-e29b-41d4-a716-446655440001",
  "created_at": "2024-01-15T10:30:00Z",
  "solves": [
    {
      "id": "770e8400-e29b-41d4-a716-446655440002",
      "challenge_id": "880e8400-e29b-41d4-a716-446655440003",
      "solved_at": "2024-01-15T11:00:00Z"
    }
  ]
}
```

**Статусы:**

- `200 OK` - профиль найден
- `404 Not Found` - пользователь не найден

### Задачи

#### GET /api/v1/challenges

Получение списка всех задач с указанием статуса решения для команды пользователя.

**Запрос:**

```
GET /api/v1/challenges
Authorization: Bearer <access_token>
```

**Ответ (успех):**

```json
[
  {
    "id": "880e8400-e29b-41d4-a716-446655440003",
    "title": "Задача 1",
    "description": "Описание задачи",
    "category": "Web",
    "points": 100,
    "is_hidden": false,
    "solved": true
  },
  {
    "id": "990e8400-e29b-41d4-a716-446655440004",
    "title": "Задача 2",
    "description": "Описание задачи 2",
    "category": "Crypto",
    "points": 200,
    "is_hidden": false,
    "solved": false
  }
]
```

**Статусы:**

- `200 OK` - список успешно получен
- `401 Unauthorized` - токен отсутствует или невалиден

#### GET /api/v1/challenges/{id}/first-blood

Получение информации о том, кто первым решил задачу.

**Запрос:**

```
GET /api/v1/challenges/{id}/first-blood
```

**Параметры пути:**

- `id` (UUID, required) - идентификатор задачи

**Ответ (успех):**

```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "player1",
  "team_id": "660e8400-e29b-41d4-a716-446655440001",
  "team_name": "Команда А",
  "solved_at": "2024-01-15T10:45:00Z"
}
```

**Статусы:**

- `200 OK` - информация успешно получена
- `404 Not Found` - задачу еще никто не решил или задача не найдена

#### POST /api/v1/challenges/{id}/submit

Отправка флага на проверку.

**Запрос:**

```
POST /api/v1/challenges/{id}/submit
Authorization: Bearer <access_token>
Content-Type: application/json
```

**Параметры пути:**

- `id` (UUID, required) - идентификатор задачи

**Тело запроса:**

```json
{
  "flag": "CTF{flag_here}"
}
```

**Параметры:**

- `flag` (string, required) - флаг для проверки

**Ответ (успех):**

```json
{
  "message": "flag accepted"
}
```

**Ответ (неверный флаг):**

```json
{
  "error": "invalid flag"
}
```

**Статусы:**

- `200 OK` - флаг принят
- `400 Bad Request` - неверный флаг или некорректные данные
- `401 Unauthorized` - токен отсутствует или невалиден
- `403 Forbidden` - пользователь не состоит в команде
- `404 Not Found` - задача не найдена
- `409 Conflict` - задача уже решена командой
- `429 Too Many Requests` - превышен лимит попыток (5 попыток в минуту)

**Rate Limiting:**
Данный эндпоинт ограничен 5 запросами в минуту с одного IP-адреса.

#### POST /api/v1/admin/challenges

Создание новой задачи (только для администраторов).

**Запрос:**

```
POST /api/v1/admin/challenges
Authorization: Bearer <access_token>
Content-Type: application/json
```

**Тело запроса:**

```json
{
  "title": "Новая задача",
  "description": "Описание задачи",
  "category": "Web",
  "points": 100,
  "flag": "CTF{flag_here}",
  "is_hidden": false
}
```

**Параметры:**

- `title` (string, required) - название задачи
- `description` (string, required) - описание задачи
- `category` (string, required) - категория задачи
- `points` (integer, required) - количество очков
- `flag` (string, required) - правильный флаг
- `is_hidden` (boolean, optional) - скрыта ли задача

**Ответ (успех):**

```json
{
  "id": "880e8400-e29b-41d4-a716-446655440003",
  "title": "Новая задача",
  "description": "Описание задачи",
  "category": "Web",
  "points": 100,
  "is_hidden": false,
  "solved": false
}
```

**Статусы:**

- `201 Created` - задача успешно создана
- `400 Bad Request` - некорректные данные запроса
- `401 Unauthorized` - токен отсутствует или невалиден
- `403 Forbidden` - недостаточно прав доступа

#### PUT /api/v1/admin/challenges/

Обновление задачи (только для администраторов).

**Запрос:**

```
PUT /api/v1/admin/challenges/{id}
Authorization: Bearer <access_token>
Content-Type: application/json
```

**Параметры пути:**

- `id` (UUID, required) - идентификатор задачи

**Тело запроса:**

```json
{
  "title": "Обновленное название",
  "description": "Обновленное описание",
  "category": "Web",
  "points": 150,
  "flag": "CTF{new_flag}",
  "is_hidden": false
}
```

**Ответ (успех):**

```json
{
  "id": "880e8400-e29b-41d4-a716-446655440003",
  "title": "Обновленное название",
  "description": "Обновленное описание",
  "category": "Web",
  "points": 150,
  "is_hidden": false,
  "solved": false
}
```

**Статусы:**

- `200 OK` - задача успешно обновлена
- `400 Bad Request` - некорректные данные запроса
- `401 Unauthorized` - токен отсутствует или невалиден
- `403 Forbidden` - недостаточно прав доступа
- `404 Not Found` - задача не найдена

#### DELETE /api/v1/admin/challenges/

Удаление задачи (только для администраторов).

**Запрос:**

```
DELETE /api/v1/admin/challenges/{id}
Authorization: Bearer <access_token>
```

**Параметры пути:**

- `id` (UUID, required) - идентификатор задачи

**Статусы:**

- `204 No Content` - задача успешно удалена
- `401 Unauthorized` - токен отсутствует или невалиден
- `403 Forbidden` - недостаточно прав доступа
- `404 Not Found` - задача не найдена

### Команды

#### POST /api/v1/teams

Создание новой команды. Пользователь становится капитаном команды.

**Запрос:**

```
POST /api/v1/teams
Authorization: Bearer <access_token>
Content-Type: application/json
```

**Тело запроса:**

```json
{
  "name": "Команда А"
}
```

**Параметры:**

- `name` (string, required) - название команды. Должно быть уникальным

**Ответ (успех):**

```json
{
  "id": "660e8400-e29b-41d4-a716-446655440001",
  "name": "Команда А",
  "invite_token": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "captain_id": "550e8400-e29b-41d4-a716-446655440000",
  "created_at": "2024-01-15T10:30:00Z"
}
```

**Статусы:**

- `201 Created` - команда успешно создана
- `400 Bad Request` - некорректные данные запроса
- `401 Unauthorized` - токен отсутствует или невалиден
- `409 Conflict` - команда с таким названием уже существует или пользователь уже состоит в команде

#### POST /api/v1/teams/join

Вступление в команду по токену приглашения.

**Запрос:**

```
POST /api/v1/teams/join
Authorization: Bearer <access_token>
Content-Type: application/json
```

**Тело запроса:**

```json
{
  "invite_token": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
}
```

**Параметры:**

- `invite_token` (string, required) - токен приглашения команды

**Ответ (успех):**

```json
{
  "id": "660e8400-e29b-41d4-a716-446655440001",
  "name": "Команда А",
  "invite_token": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "captain_id": "550e8400-e29b-41d4-a716-446655440000",
  "created_at": "2024-01-15T10:30:00Z"
}
```

**Статусы:**

- `200 OK` - успешно вступил в команду
- `400 Bad Request` - некорректные данные запроса
- `401 Unauthorized` - токен отсутствует или невалиден
- `404 Not Found` - команда с таким токеном не найдена
- `409 Conflict` - пользователь уже состоит в команде

#### GET /api/v1/teams/my

Получение информации о команде текущего пользователя со списком участников.

**Запрос:**

```
GET /api/v1/teams/my
Authorization: Bearer <access_token>
```

**Ответ (успех):**

```json
{
  "id": "660e8400-e29b-41d4-a716-446655440001",
  "name": "Команда А",
  "invite_token": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "captain_id": "550e8400-e29b-41d4-a716-446655440000",
  "created_at": "2024-01-15T10:30:00Z",
  "members": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "username": "player1",
      "team_id": "660e8400-e29b-41d4-a716-446655440001",
      "role": "captain"
    },
    {
      "id": "550e8400-e29b-41d4-a716-446655440005",
      "username": "player2",
      "team_id": "660e8400-e29b-41d4-a716-446655440001",
      "role": "member"
    }
  ]
}
```

**Статусы:**

- `200 OK` - информация успешно получена
- `401 Unauthorized` - токен отсутствует или невалиден
- `404 Not Found` - пользователь не состоит в команде

#### GET /api/v1/teams/

Получение информации о команде по идентификатору.

**Запрос:**

```
GET /api/v1/teams/{id}
Authorization: Bearer <access_token>
```

**Параметры пути:**

- `id` (UUID, required) - идентификатор команды

**Ответ (успех):**

```json
{
  "id": "660e8400-e29b-41d4-a716-446655440001",
  "name": "Команда А",
  "invite_token": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "captain_id": "550e8400-e29b-41d4-a716-446655440000",
  "created_at": "2024-01-15T10:30:00Z"
}
```

**Статусы:**

- `200 OK` - команда найдена
- `401 Unauthorized` - токен отсутствует или невалиден
- `404 Not Found` - команда не найдена

### Турнирная таблица

#### GET /api/v1/scoreboard

Получение текущего состояния турнирной таблицы.

**Запрос:**

```
GET /api/v1/scoreboard
```

**Ответ (успех):**

```json
[
  {
    "team_id": "660e8400-e29b-41d4-a716-446655440001",
    "team_name": "Команда А",
    "points": 300
  },
  {
    "team_id": "770e8400-e29b-41d4-a716-446655440006",
    "team_name": "Команда Б",
    "points": 200
  }
]
```

**Статусы:**

- `200 OK` - таблица успешно получена

**Примечание:**
Таблица отсортирована по убыванию очков. При равенстве очков выше располагается команда, решившая последнюю задачу раньше.

#### GET /api/v1/events

Получение потока событий турнирной таблицы через Server-Sent Events (SSE).

**Запрос:**

```
GET /api/v1/events
```

**Ответ:**
Поток данных в формате SSE. Каждые 5 секунд отправляется обновленное состояние турнирной таблицы:

```
data: [{"team_id":"...","team_name":"...","points":300}]

data: [{"team_id":"...","team_name":"...","points":300}]
...
```

**Статусы:**

- `200 OK` - поток установлен

**Примечание:**
Соединение остается открытым до закрытия клиентом. Данные обновляются каждые 5 секунд.

## Схемы данных

### RegisterRequest

**Поля:**

- `username` (string, required) - имя пользователя
- `email` (string, required) - электронная почта
- `password` (string, required) - пароль

### LoginRequest

**Поля:**

- `email` (string, required) - электронная почта
- `password` (string, required) - пароль

### RegisterResponse

**Поля:**

- `id` (UUID, required) - идентификатор пользователя
- `username` (string, required) - имя пользователя
- `email` (string, required) - электронная почта
- `created_at` (datetime, required) - время создания аккаунта

### TokenPair

**Поля:**

- `access_token` (string, required) - JWT токен доступа
- `refresh_token` (string, required) - JWT токен обновления
- `access_expires_at` (integer, required) - время истечения access токена (Unix timestamp)
- `refresh_expires_at` (integer, required) - время истечения refresh токена (Unix timestamp)

### MeResponse

**Поля:**

- `id` (UUID, required) - идентификатор пользователя
- `username` (string, required) - имя пользователя
- `email` (string, required) - электронная почта
- `team_id` (UUID, nullable) - идентификатор команды
- `created_at` (datetime, required) - время создания аккаунта

### UserProfileResponse

**Поля:**

- `id` (UUID, required) - идентификатор пользователя
- `username` (string, required) - имя пользователя
- `team_id` (UUID, nullable) - идентификатор команды
- `created_at` (datetime, required) - время создания аккаунта
- `solves` (array of SolveResponse, required) - список решенных задач

### SolveResponse

**Поля:**

- `id` (UUID, required) - идентификатор решения
- `challenge_id` (UUID, required) - идентификатор задачи
- `solved_at` (datetime, required) - время решения

### ChallengeResponse

**Поля:**

- `id` (UUID, required) - идентификатор задачи
- `title` (string, required) - название задачи
- `description` (string, required) - описание задачи
- `category` (string, required) - категория задачи
- `points` (integer, required) - количество очков
- `is_hidden` (boolean, required) - скрыта ли задача
- `solved` (boolean, required) - решена ли задача командой

### CreateChallengeRequest

**Поля:**

- `title` (string, required) - название задачи
- `description` (string, required) - описание задачи
- `category` (string, required) - категория задачи
- `points` (integer, required) - количество очков
- `flag` (string, required) - правильный флаг
- `is_hidden` (boolean, optional) - скрыта ли задача

### UpdateChallengeRequest

**Поля:**

- `title` (string, required) - название задачи
- `description` (string, required) - описание задачи
- `category` (string, required) - категория задачи
- `points` (integer, required) - количество очков
- `flag` (string, optional) - правильный флаг
- `is_hidden` (boolean, optional) - скрыта ли задача

### SubmitFlagRequest

**Поля:**

- `flag` (string, required) - флаг для проверки

### TeamResponse

**Поля:**

- `id` (UUID, required) - идентификатор команды
- `name` (string, required) - название команды
- `invite_token` (string, required) - токен приглашения
- `captain_id` (UUID, required) - идентификатор капитана
- `created_at` (datetime, required) - время создания команды

### TeamWithMembersResponse

**Поля:**

- `id` (UUID, required) - идентификатор команды
- `name` (string, required) - название команды
- `invite_token` (string, required) - токен приглашения
- `captain_id` (UUID, required) - идентификатор капитана
- `created_at` (datetime, required) - время создания команды
- `members` (array of UserResponse, required) - список участников

### UserResponse

**Поля:**

- `id` (UUID, required) - идентификатор пользователя
- `username` (string, required) - имя пользователя
- `team_id` (UUID, nullable) - идентификатор команды
- `role` (string, required) - роль в команде (captain/member)

### ScoreboardEntryResponse

**Поля:**

- `team_id` (UUID, required) - идентификатор команды
- `team_name` (string, required) - название команды
- `points` (integer, required) - количество очков

### ErrorResponse

**Поля:**

- `error` (string, required) - описание ошибки
- `code` (string, optional) - код ошибки

## Коды ошибок

### 400 Bad Request

Некорректный запрос. Возможные причины:

- Неверный формат данных
- Отсутствие обязательных полей
- Неверный флаг

### 401 Unauthorized

Требуется аутентификация. Возможные причины:

- Отсутствует токен в заголовке Authorization
- Токен истек или невалиден
- Неверные учетные данные при входе

### 403 Forbidden

Недостаточно прав доступа. Возможные причины:

- Пользователь не является администратором
- Пользователь не состоит в команде (для некоторых операций)

### 404 Not Found

Запрашиваемый ресурс не найден. Возможные причины:

- Пользователь не найден
- Команда не найдена
- Задача не найдена

**Пример ответа:**

```json
{
  "error": "User not found"
}
```

### 409 Conflict

Конфликт данных. Возможные причины:

- Пользователь с таким username или email уже существует
- Команда с таким названием уже существует
- Пользователь уже состоит в команде
- Задача уже решена командой

**Пример ответа:**

```json
{
  "error": "User already exists: username"
}
```

### 429 Too Many Requests

Превышен лимит запросов. Применяется только для эндпоинта `/api/v1/challenges/{id}/submit` (7 попыток в минуту).

**Пример ответа:**

```json
{
  "error": "Too many requests"
}
```

### 500 Internal Server Error

Внутренняя ошибка сервера.

**Пример ответа:**

```json
{
  "error": "Internal server error"
}
```

## Rate Limiting

Для эндпоинта `/api/v1/challenges/{id}/submit` установлено ограничение: не более 7 запросов в минуту с одного IP-адреса. При превышении лимита возвращается статус `429 Too Many Requests`.

Общий лимит для всех остальных эндпоинтов: 100 запросов в минуту с одного IP-адреса.

## CORS

API настроен для работы с CORS запросами:

- Разрешены все источники (`*`)
- Разрешенные методы: GET, POST, PUT, DELETE, OPTIONS, PATCH
- Разрешенные заголовки: Accept, Authorization, Content-Type, X-CSRF-Token

## Версионирование

API использует версионирование в URL. Текущая версия: `/api/v1/`.

Все эндпоинты находятся под префиксом `/api/v1/`, кроме `/health` и `/swagger/*`.

## Примеры использования

### Регистрация пользователя

```bash
curl -X POST https://api.ctfleague.ru/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "player1",
    "email": "player1@example.com",
    "password": "SecurePassword123!"
  }'
```

### Вход в систему

```bash
curl -X POST https://api.ctfleague.ru/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "player1@example.com",
    "password": "SecurePassword123!"
  }'
```

### Получение информации о текущем пользователе

```bash
curl https://api.ctfleague.ru/api/v1/auth/me \
  -H "Authorization: Bearer <access_token>"
```

### Получение списка задач

```bash
curl https://api.ctfleague.ru/api/v1/challenges \
  -H "Authorization: Bearer <access_token>"
```

### Отправка флага

```bash
curl -X POST https://api.ctfleague.ru/api/v1/challenges/880e8400-e29b-41d4-a716-446655440003/submit \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"flag": "CTF{flag_here}"}'
```

### Создание команды

```bash
curl -X POST https://api.ctfleague.ru/api/v1/teams \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"name": "Команда А"}'
```

### Вступление в команду

```bash
curl -X POST https://api.ctfleague.ru/api/v1/teams/join \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"invite_token": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"}'
```

### Получение турнирной таблицы

```bash
curl https://api.ctfleague.ru/api/v1/scoreboard
```
