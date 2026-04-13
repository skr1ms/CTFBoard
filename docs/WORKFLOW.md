# AstroCTFb - Application Workflow

Полный флоу взаимодействий пользователей с платформой: от регистрации до решения задач и просмотра скорборда.

---

## Общая карта маршрутов

```mermaid
graph TD
    Client([Client])

    subgraph PUBLIC["Public (no auth)"]
        REG[POST /auth/register]
        LOGIN[POST /auth/login]
        VERIFY[POST /auth/verify-email]
        STATUS[GET /competition/status]
    end

    subgraph AUTH_ONLY["Auth only (no team/ban check)"]
        RESEND[POST /auth/resend-verification]
    end

    subgraph BASIC_AUTH["Auth + verified"]
        ME_GET[GET /auth/me]
        ME_UPD[PATCH /auth/me]
        USERS[GET /users]
        TEAMS_LIST[GET /teams]
        TEAM_CREATE[POST /teams]
        TEAM_JOIN[POST /teams/join]
        TEAM_LEAVE[POST /teams/leave]
        TEAM_SOLO[POST /teams/solo]
        CAPTAIN[POST /teams/me/captain]
        DISBAND[DELETE /teams/me]
    end

    subgraph PROTECTED["Auth + verified + team + not banned"]
        CHALLENGES[GET /challenges]
        CHALLENGE_GET[GET /challenges/:id]
        SUBMIT[POST /challenges/:id/submit]
        HINT[POST /challenges/:id/hints/:hintId/unlock]
        SCOREBOARD[GET /scoreboard]
        SB_GRAPH[GET /scoreboard/graph]
        STATS[GET /statistics/*]
        FB[GET /challenges/:id/first-blood]
        NOTIF[GET /notifications]
        WS[WS /ws]
    end

    subgraph ADMIN["Admin only"]
        ADMIN_COMP[/admin/competition]
        ADMIN_CHALL[/admin/challenges]
        ADMIN_USERS[/admin/users]
        ADMIN_TEAMS[/admin/teams]
        ADMIN_BACKUP[/admin/backup]
    end

    Client --> PUBLIC
    Client --> AUTH_ONLY
    Client --> BASIC_AUTH
    Client --> PROTECTED
    Client --> ADMIN
```

---

## Middleware Stack (порядок применения)

```mermaid
graph LR
    REQ([Request]) --> AUTH

    AUTH["Auth()\nJWT / API Token"] --> INJECT
    INJECT["InjectUser()\nload *domain.User"] --> BAN
    BAN["RequireNotBanned()\nuser.IsBanned"] --> VER
    VER["RequireVerified()\nuser.IsEmailVerified"] --> TEAM
    TEAM["RequireTeam()\nuser.TeamID != nil"] --> TBAN
    TBAN["RequireTeamNotBanned()"] --> VIS
    VIS["ChallengeVisibility()\ncomp state"] --> RATE
    RATE["RateLimit()\nRedis sliding window"] --> HANDLER

    HANDLER([Handler])

    style AUTH fill:#f9a825
    style BAN fill:#e53935
    style RATE fill:#1e88e5
```

Для публичных роутов - нет middleware.
Для auth-only роутов - только `Auth` + `InjectUser`.
Для защищённых - полный стек выше.

---

## Регистрация и вход

```mermaid
sequenceDiagram
    participant C as Client
    participant H as Handler
    participant UC as UserUseCase
    participant DB as PostgreSQL

    Note over C,DB: Регистрация
    C->>H: POST /auth/register {username, email, password}
    H->>UC: Register(username, email, password)
    UC->>DB: BEGIN TRANSACTION
    UC->>DB: AdvisoryLock(email) + AdvisoryLock(username)
    UC->>DB: SettingsRepo.Get() - check RegistrationOpen
    UC->>DB: UserRepo.GetByEmail() - uniqueness
    UC->>DB: UserRepo.GetByUsername() - uniqueness
    Note over UC: bcrypt(password) [via semaphore]
    UC->>DB: UserRepo.Create(user)
    alt CompMode == SoloOnly
        UC->>DB: TeamRepo.Create(soloTeam)
        UC->>DB: UserRepo.UpdateTeamID(user, soloTeam)
    end
    UC->>DB: COMMIT
    H-->>C: 201 Created

    Note over C,DB: Вход
    C->>H: POST /auth/login {email, password}
    H->>UC: Login(email, password)
    UC->>DB: FailedLoginTracker.IsLocked(email) - check lockout
    UC->>DB: UserRepo.GetByEmail(email)
    Note over UC: bcrypt.Compare(password, hash) [via semaphore]
    alt password mismatch
        UC->>DB: RecordFailedLogin(email)
        H-->>C: 401 Unauthorized
    else password ok
        UC->>DB: ClearFailedLogin(email)
        UC->>UC: JWTService.GenerateTokenPair(userID, role)
        H-->>C: 200 {accessToken, refreshToken}
    end
```

---

## OAuth вход (GitHub / Google)

```mermaid
sequenceDiagram
    participant C as Client
    participant H as Handler
    participant UC as UserUseCase
    participant EXT as GitHub/Google API
    participant DB as PostgreSQL

    C->>H: GET /auth/oauth/{provider}
    H-->>C: 302 Redirect -> provider OAuth URL (state cookie set)

    C->>EXT: User authorizes
    EXT-->>C: Redirect -> /auth/oauth/{provider}/callback?code=...&state=...

    C->>H: GET /auth/oauth/{provider}/callback
    H->>UC: OAuthCallback(provider, code, state)
    UC->>EXT: ExchangeCode(code) -> accessToken
    UC->>EXT: GetUserInfo(accessToken) -> {email, providerID}
    UC->>DB: BEGIN TRANSACTION
    UC->>DB: OAuthRepo.GetByProviderID() - find existing link
    alt linked account exists
        UC->>DB: UserRepo.GetByID(linkedUserID)
    else new account
        UC->>DB: UserRepo.GetByEmail(email)
        alt email exists -> link account
            UC->>DB: OAuthRepo.Create(userID, providerID)
        else new user
            UC->>DB: UserRepo.Create(user, isVerified=true)
            UC->>DB: OAuthRepo.Create(userID, providerID)
        end
    end
    UC->>DB: COMMIT
    UC-->>H: GenerateTokenPair(userID, role)
    H-->>C: 200 {accessToken, refreshToken}
```

---

## Управление командой

```mermaid
sequenceDiagram
    participant C as Client
    participant H as Handler
    participant UC as TeamUseCase
    participant DB as PostgreSQL
    participant Cache as Redis

    Note over C,Cache: Создание команды
    C->>H: POST /teams {name}
    H->>UC: Create(name, captainID)
    UC->>DB: BEGIN TRANSACTION
    UC->>DB: UserRepo.Lock(captainID)
    UC->>DB: Check: user не в команде, компетиция разрешает команды
    UC->>DB: TeamRepo.GetByName() - uniqueness
    UC->>DB: TeamRepo.Create(team)
    UC->>DB: UserRepo.UpdateTeamID(userID, teamID)
    UC->>DB: COMMIT
    UC->>Cache: InvalidateUserCache(userID)
    H-->>C: 201 Created

    Note over C,Cache: Вступление по инвайту
    C->>H: POST /teams/join {inviteToken}
    H->>UC: Join(inviteToken, userID, confirmReset)
    UC->>DB: Guard.RequireTeamSwitch() - check competition state
    UC->>DB: BEGIN TRANSACTION
    UC->>DB: UserRepo.Lock(userID)
    UC->>DB: TeamRepo.GetByInviteToken(inviteToken)
    UC->>DB: Check: invite не истёк, команда не полная, не забанена
    UC->>DB: Lock(oldTeam), Lock(newTeam)
    alt user был в другой команде
        UC->>DB: SoloTeam cleanup (если авто-соло)
        UC->>DB: AuditLog(left old team)
    end
    UC->>DB: UserRepo.UpdateTeamID(userID, newTeamID)
    UC->>DB: AuditLog(joined new team)
    UC->>DB: COMMIT
    UC->>Cache: InvalidateUserCache + InvalidateScoreboard
    H-->>C: 200 OK

    Note over C,Cache: Выход из команды
    C->>H: POST /teams/leave
    H->>UC: Leave(userID)
    UC->>DB: Guard.RequireTeamSwitch()
    UC->>DB: BEGIN TRANSACTION
    UC->>DB: UserRepo.Lock(userID) -> TeamRepo.Lock(teamID)
    UC->>DB: Validate: не соло-режим, не последний, не капитан, минимальный размер
    UC->>DB: UserRepo.UpdateTeamID(userID, nil)
    UC->>DB: AuditLog(left)
    UC->>DB: COMMIT
    UC->>Cache: InvalidateUserCache + InvalidateScoreboard
    H-->>C: 200 OK
```

---

## Решение задачи (submit флага)

```mermaid
sequenceDiagram
    participant C as Client
    participant MW as Middleware Stack
    participant H as Handler
    participant UC as SolveUseCase
    participant DB as PostgreSQL
    participant Cache as Redis
    participant WS as WebSocket Broadcaster

    C->>MW: POST /challenges/{id}/submit {flag}
    MW->>MW: Auth() - parse JWT
    MW->>MW: InjectUser() - load user
    MW->>MW: RequireNotBanned()
    MW->>MW: RequireVerified()
    MW->>MW: RequireTeam()
    MW->>MW: RequireTeamNotBanned()
    MW->>MW: ChallengeVisibility() - comp active?
    MW->>MW: RateLimit() - sliding window
    MW->>H: handler(ctx, req)

    H->>UC: Create(solve{userID, challengeID, flag})

    UC->>DB: BEGIN TRANSACTION
    UC->>DB: CompRepo.GetForUpdate() - check IsSubmissionAllowed()
    UC->>DB: UserRepo.Lock(userID)
    UC->>DB: UserRepo.GetByID() - check IsBanned
    UC->>DB: Resolve teamID from user.TeamID
    UC->>DB: TeamRepo.Lock(teamID)
    UC->>DB: TeamRepo.GetByID() - check IsBanned, mode compat

    UC->>DB: ChallengeRepo.GetByID() - check visible, not hidden
    UC->>DB: SolveRepo.Upsert(solve) - insert or update

    alt first solve of this challenge
        Note over UC,DB: isFirstBlood = true
    end

    UC->>DB: COMMIT

    UC->>Cache: InvalidateScoreboardCache(teamID)
    UC->>Cache: InvalidateChallengeListCache(teamID)

    alt freeze NOT active
        UC->>WS: NotifySolve(teamID, challengeID, isFirstBlood)
        WS-->>C: WS event: "solve" broadcast to all connected clients
    end

    H-->>C: 200 {correct: true} or 200 {correct: false}
```

---

## Скорборд и заморозка

```mermaid
sequenceDiagram
    participant C as Client
    participant UC as SolveUseCase
    participant Local as LocalCache (2s TTL)
    participant Redis as Redis (15s TTL)
    participant DB as PostgreSQL

    C->>UC: GetScoreboard(bracketID, forceLive=false)

    UC->>UC: getCompetition() -> check IsFreezeActive()

    alt freeze active AND not forceLive
        Note over UC: key = "scoreboard:frozen:{freeze_ts}:{bracketID}"
    else live
        Note over UC: key = "scoreboard:live:{bracketID}"
    end

    UC->>Local: Get(key)
    alt local cache hit (< 2s)
        Local-->>UC: cached entries
    else local cache miss
        UC->>Redis: Get(key)
        alt redis cache hit (< 15s)
            Redis-->>UC: cached entries
        else redis cache miss
            alt freeze active
                UC->>DB: SolveRepo.GetScoreboardByBracketFrozen(bracketID, freezeTime)
            else
                UC->>DB: SolveRepo.GetScoreboardByBracket(bracketID)
            end
            DB-->>UC: raw scoreboard rows
            UC->>Redis: Set(key, entries, TTL=15s)
        end
        UC->>Local: Set(key, entries, TTL=2s)
    end

    UC-->>C: sorted scoreboard entries
```

---

## Покупка подсказки

```mermaid
sequenceDiagram
    participant C as Client
    participant H as Handler
    participant UC as ChallengeUseCase
    participant DB as PostgreSQL

    C->>H: POST /challenges/{id}/hints/{hintId}/unlock
    H->>UC: UnlockHint(userID, challengeID, hintID)

    UC->>DB: BEGIN TRANSACTION
    UC->>DB: HintRepo.GetByIDForUpdate(hintID) - lock hint row
    UC->>DB: HintRepo.GetUnlock(teamID, hintID) - уже куплена?
    alt already unlocked
        UC->>DB: ROLLBACK
        H-->>C: 200 {hint content}
    else not unlocked yet
        UC->>DB: TeamRepo.Lock(teamID)
        UC->>DB: TeamRepo.GetBalance(teamID) - check points >= hint.Cost
        alt insufficient points
            UC->>DB: ROLLBACK
            H-->>C: 400 Insufficient points
        else
            UC->>DB: TeamRepo.DeductPoints(teamID, hint.Cost)
            UC->>DB: HintRepo.CreateUnlock(teamID, hintID)
            UC->>DB: COMMIT
            H-->>C: 200 {hint content}
        end
    end
```

---

## WebSocket соединение

```mermaid
sequenceDiagram
    participant C as Client
    participant MW as Auth Middleware
    participant WS as WS Controller
    participant HUB as Hub
    participant UC as UseCases

    C->>MW: GET /ws (Upgrade: websocket, Bearer token)
    MW->>MW: Auth() - validate JWT at handshake time
    MW-->>C: 101 Switching Protocols

    C->>WS: Connected
    WS->>HUB: Register(client)
    WS->>WS: ReadPump() goroutine - handle incoming pings
    WS->>WS: WritePump() goroutine - send outgoing events

    loop Competition events
        UC->>HUB: Broadcast(event)
        HUB->>C: WS message {type, payload}
    end

    Note over C,HUB: Events: solve, first_blood, competition_state_change, scoreboard_update

    C->>WS: Close connection
    WS->>HUB: Unregister(client)
    WS->>WS: Cancel ReadPump + WritePump
```

---

## Бан пользователя / команды

```mermaid
graph TD
    ADMIN[Admin] --> |POST /admin/users/:id/ban| BAN_USER[BanUser]
    ADMIN --> |POST /admin/teams/:id/ban| BAN_TEAM[BanTeam]

    BAN_USER --> |user.IsBanned = true| DB_USER[(users table)]
    BAN_USER --> |WasInBannedTeam = true for teammates| DB_TEAM_MEMBERS[(team members)]
    BAN_USER --> |recalculate scoreboard| SCORE[Scoreboard Recalc]

    BAN_TEAM --> |team.IsBanned = true| DB_TEAM[(teams table)]
    BAN_TEAM --> |all members: WasInBannedTeam = true| DB_TEAM_MEMBERS
    BAN_TEAM --> |recalculate scoreboard| SCORE

    SCORE --> |InvalidateCache| REDIS[(Redis)]

    MW[Auth Middleware] --> |on every request: check user.IsBanned| CHECK_BAN{banned?}
    CHECK_BAN --> |yes| REJECT[403 Forbidden]
    CHECK_BAN --> |no| PROCEED[Continue]

    style BAN_USER fill:#e53935,color:#fff
    style BAN_TEAM fill:#e53935,color:#fff
    style REJECT fill:#b71c1c,color:#fff
```

---

## Бэкап / Экспорт / Импорт

```mermaid
sequenceDiagram
    participant ADMIN as Admin
    participant H as Handler
    participant UC as BackupUseCase
    participant DB as PostgreSQL
    participant S3 as SeaweedFS (S3)

    Note over ADMIN,S3: Экспорт
    ADMIN->>H: POST /admin/backup/export
    H->>UC: Export(ctx)
    UC->>DB: BEGIN TRANSACTION (read-only snapshot)
    UC->>DB: Load: competition, challenges, hints, teams, users, solves, awards
    UC->>DB: COMMIT (snapshot consistent)
    UC->>UC: Marshal to JSON
    UC->>S3: Upload backup file
    H-->>ADMIN: 200 {backupID, url}

    Note over ADMIN,S3: Импорт
    ADMIN->>H: POST /admin/backup/import {file}
    H->>UC: Import(ctx, data)
    UC->>DB: BEGIN TRANSACTION
    UC->>DB: Validate + insert: competition, challenges, users, teams, solves
    alt validation error or FK conflict
        UC->>DB: ROLLBACK
        H-->>ADMIN: 400 Import failed (no data corruption)
    else
        UC->>DB: COMMIT
        H-->>ADMIN: 200 Import complete
    end
```

---

## Аватар пользователя

```mermaid
sequenceDiagram
    participant C as Client
    participant H as Handler
    participant UC as AvatarUseCase
    participant PROC as ImageProcessor
    participant S3 as SeaweedFS (S3)
    participant DB as PostgreSQL

    C->>H: POST /avatar (multipart/form-data, image file)
    H->>UC: UploadAvatar(userID, file)
    UC->>PROC: Process(file)
    PROC->>PROC: Validate MIME type (JPEG/PNG/WebP only)
    PROC->>PROC: Resize to max 256×256
    PROC->>PROC: Encode to WebP
    UC->>S3: Upload(avatar_webp, key=avatars/{userID}.webp)
    UC->>DB: UserRepo.UpdateAvatarURL(userID, s3_url)
    H-->>C: 200 {avatarURL}

    C->>H: DELETE /avatar
    H->>UC: DeleteAvatar(userID)
    UC->>S3: Delete(key=avatars/{userID}.webp)
    UC->>DB: UserRepo.UpdateAvatarURL(userID, nil)
    H-->>C: 204 No Content
```

---

## Динамический скоринг

```mermaid
graph LR
    SOLVE[Новый solve] --> RECALC[RecalcScores]
    RECALC --> LOCK_COMP[Lock competition row]
    LOCK_COMP --> FETCH[Fetch all solves for challenge]
    FETCH --> FORMULA["score = max + (min-max) × (N-1)² / (D-1)²"]
    FORMULA --> UPDATE[Update solve.Points for all teams]
    UPDATE --> INVALIDATE[Invalidate scoreboard cache]
    INVALIDATE --> WS_NOTIFY[WS: scoreboard_update]

    style SOLVE fill:#43a047,color:#fff
    style RECALC fill:#1e88e5,color:#fff
    style WS_NOTIFY fill:#8e24aa,color:#fff
```

**Формула:** `score = max_score + (min_score - max_score) × ((solves-1)² / (decay-1)²)`

При каждом новом solve пересчитываются очки для **всех** предыдущих решений этой задачи.

---

## Полная карта зависимостей слоёв

```mermaid
graph TD
    subgraph HTTP["HTTP Layer (controller/restapi)"]
        ROUTER[Router / Chi]
        MW_STACK[Middleware Stack]
        HANDLERS[Handlers v1]
    end

    subgraph UC["UseCase Layer"]
        UC_USER[user]
        UC_TEAM[team]
        UC_COMP[competition]
        UC_CHALL[challenge]
        UC_BACK[backup]
        UC_EMAIL[email]
        UC_NOTIF[notification]
        UC_AVATAR[avatar]
    end

    subgraph REPO["Repository Layer"]
        REPO_PG[persistent / PostgreSQL]
        REPO_WA[webapi / OAuth]
        STORAGE[storage / SeaweedFS S3]
    end

    subgraph INFRA["Infrastructure"]
        PG[(PostgreSQL 15)]
        REDIS[(Redis)]
        S3[(SeaweedFS)]
        WS_HUB[WS Hub]
    end

    ROUTER --> MW_STACK --> HANDLERS
    HANDLERS --> UC_USER & UC_TEAM & UC_COMP & UC_CHALL & UC_BACK & UC_NOTIF & UC_AVATAR

    UC_USER & UC_TEAM & UC_COMP & UC_CHALL & UC_BACK --> REPO_PG
    UC_USER --> REPO_WA
    UC_CHALL & UC_BACK & UC_AVATAR --> STORAGE
    UC_COMP --> WS_HUB

    REPO_PG --> PG
    REPO_WA --> |GitHub / Google API| EXT([External OAuth])
    STORAGE --> S3
    WS_HUB --> REDIS

    MW_STACK --> REDIS
```
