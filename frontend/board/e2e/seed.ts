/**
 * E2E seed script - called before Playwright tests.
 * Sets up competition + challenges + full fixture data via the admin REST API.
 *
 * Usage:  VITE_BACKEND_PORT=8090 bun e2e/seed.ts
 */

import * as fs from 'fs'
import * as path from 'path'
import { fileURLToPath } from 'url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

function loadEnvFile(filePath: string): Record<string, string> {
  try {
    const env: Record<string, string> = {}
    for (const line of fs.readFileSync(filePath, 'utf-8').split('\n')) {
      const trimmed = line.trim()
      if (!trimmed || trimmed.startsWith('#')) continue
      const eqIdx = trimmed.indexOf('=')
      if (eqIdx === -1) continue
      const key = trimmed.slice(0, eqIdx).trim()
      const val = trimmed.slice(eqIdx + 1).trim().replace(/^['"]|['"]$/g, '')
      if (key) env[key] = val
    }
    return env
  } catch {
    return {}
  }
}

// Fallback: read admin credentials from project-root .env.local when not set via E2E_* vars.
// This allows running seed against local-compose (which seeds from ADMIN_EMAIL/ADMIN_PASSWORD
// in .env.local) without having to pass credentials explicitly each time.
const localEnv = loadEnvFile(path.resolve(__dirname, '../../../.env.local'))

const BASE = `http://localhost:${process.env['VITE_BACKEND_PORT'] ?? '8090'}`
const ADMIN_EMAIL = process.env['E2E_EMAIL'] ?? localEnv['ADMIN_EMAIL'] ?? 'admin@example.com'
const ADMIN_PASSWORD = process.env['E2E_PASSWORD'] ?? localEnv['ADMIN_PASSWORD'] ?? 'password'

// ---------------------------------------------------------------------------
// HTTP helpers
// ---------------------------------------------------------------------------

async function apiFetch(path: string, token?: string, init?: RequestInit): Promise<unknown> {
  const res = await fetch(`${BASE}/api/v1${path}`, {
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...init?.headers,
    },
    ...init,
  })
  if (!res.ok) {
    const body = await res.text().catch(() => '')
    throw new Error(`${init?.method ?? 'GET'} ${path} -> ${res.status}: ${body}`)
  }
  const text = await res.text()
  return text ? JSON.parse(text) : null
}

async function apiFetchOptional(
  path: string,
  token?: string,
  init?: RequestInit,
): Promise<{ ok: boolean; status: number; data: unknown }> {
  const res = await fetch(`${BASE}/api/v1${path}`, {
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...init?.headers,
    },
    ...init,
  })
  const text = await res.text().catch(() => '')
  return { ok: res.ok, status: res.status, data: text ? JSON.parse(text) : null }
}

// ---------------------------------------------------------------------------
// Auth
// ---------------------------------------------------------------------------

async function login(email: string, password: string): Promise<string> {
  const data = (await apiFetch('/auth/login', undefined, {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  })) as { access_token: string }
  return data.access_token
}

// ---------------------------------------------------------------------------
// Competition
// ---------------------------------------------------------------------------

async function ensureCompetitionActive(token: string): Promise<void> {
  const comp = (await apiFetch('/admin/competition', token)) as { status?: string }
  if (comp.status === 'active') {
    console.log('competition already active')
    return
  }

  const now = new Date()
  await apiFetch('/admin/competition', token, {
    method: 'PUT',
    body: JSON.stringify({
      name: 'E2E CTF',
      start_time: new Date(now.getTime() - 2 * 60 * 60 * 1000).toISOString(),
      end_time: new Date(now.getTime() + 24 * 60 * 60 * 1000).toISOString(),
      is_public: true,
    }),
  })
  console.log('competition activated')
}

// ---------------------------------------------------------------------------
// Challenges
// ---------------------------------------------------------------------------

interface ChallengeSpec {
  title: string
  category: string
  description: string
  flag: string
  points: number
  state: 'visible' | 'hidden'
}

const CHALLENGES: ChallengeSpec[] = [
  {
    title: 'Hello World',
    category: 'Web',
    description: 'Find the flag on the homepage.',
    flag: 'CTF{hello_world}',
    points: 100,
    state: 'visible',
  },
  {
    title: 'Base64 Basics',
    category: 'Crypto',
    description: 'Decode this: `Q1RGe2Jhc2U2NF9pc19ub3RfZW5jcnlwdGlvbn0=`',
    flag: 'CTF{base64_is_not_encryption}',
    points: 150,
    state: 'visible',
  },
  {
    title: 'Hidden in Plain Sight',
    category: 'Misc',
    description: 'Sometimes the flag is right there.',
    flag: 'CTF{look_closer}',
    points: 200,
    state: 'visible',
  },
]

async function seedChallenges(
  token: string,
): Promise<Record<string, string>> {
  const existing = (await apiFetch('/challenges', token)) as Array<{ id?: string; title: string }>
  const byTitle = new Map(existing.map((c) => [c.title, c.id ?? '']))

  for (const ch of CHALLENGES) {
    if (byTitle.has(ch.title)) {
      console.log(`challenge already exists: ${ch.title}`)
      continue
    }
    const created = (await apiFetch('/admin/challenges', token, {
      method: 'POST',
      body: JSON.stringify(ch),
    })) as { id?: string }
    byTitle.set(ch.title, created.id ?? '')
    console.log(`created challenge: ${ch.title}`)
  }

  return Object.fromEntries(byTitle)
}

// ---------------------------------------------------------------------------
// Users
// ---------------------------------------------------------------------------

interface UserSpec {
  username: string
  email: string
  password: string
}

const EXTRA_USERS: UserSpec[] = [
  { username: 'user1', email: 'user1@example.com', password: 'Password1' },
  { username: 'user2', email: 'user2@example.com', password: 'Password1' },
  { username: 'banned', email: 'banned@example.com', password: 'Password1' },
]

async function seedUsers(token: string): Promise<Record<string, string>> {
  const idByEmail: Record<string, string> = {}

  for (const u of EXTRA_USERS) {
    // Search by username to check if it exists
    const list = (await apiFetch(
      `/admin/users?q=${u.username}&field=username&per_page=5`,
      token,
    )) as { data?: Array<{ id?: string; email?: string }> }
    const found = (list.data ?? []).find((x) => x.email === u.email)

    if (found?.id) {
      idByEmail[u.email] = found.id
      console.log(`user already exists: ${u.username}`)
      continue
    }

    const created = (await apiFetch('/admin/users', token, {
      method: 'POST',
      body: JSON.stringify(u),
    })) as { id?: string }
    idByEmail[u.email] = created.id ?? ''
    console.log(`created user: ${u.username}`)
  }

  return idByEmail
}

async function ensureBanned(
  adminToken: string,
  userId: string,
  isBanned: boolean,
): Promise<void> {
  if (isBanned) {
    const result = await apiFetchOptional(`/admin/users/${userId}/ban`, adminToken, {
      method: 'POST',
      body: JSON.stringify({ reason: 'E2E banned fixture' }),
    })
    if (result.ok) console.log('banned user: banned@example.com')
    else if (result.status === 409 || result.status === 400)
      console.log('user already banned: banned@example.com')
    else throw new Error(`ban failed: ${JSON.stringify(result)}`)
  }
}

// ---------------------------------------------------------------------------
// Team Alpha
// ---------------------------------------------------------------------------

async function seedTeamAlpha(
  user1Token: string,
  adminToken: string,
  user1Id: string,
  authDir: string,
): Promise<string> {
  // Check if user1 is already on a team
  const myTeam = await apiFetchOptional('/teams/my', user1Token)

  if (myTeam.ok) {
    const team = myTeam.data as { id?: string; invite_token?: string }
    console.log(`team already exists for user1: ${team.id}`)
    if (team.invite_token) {
      fs.writeFileSync(path.join(authDir, 'team-alpha-token.txt'), team.invite_token)
    }
    return team.id ?? ''
  }

  // Create Team Alpha as user1
  const created = (await apiFetch('/teams', user1Token, {
    method: 'POST',
    body: JSON.stringify({ name: 'Team Alpha' }),
  })) as { id?: string; invite_token?: string }

  const teamId = created.id ?? ''
  if (created.invite_token) {
    fs.writeFileSync(path.join(authDir, 'team-alpha-token.txt'), created.invite_token)
  }
  console.log(`created team: Team Alpha (${teamId})`)
  return teamId
}

// ---------------------------------------------------------------------------
// Hints
// ---------------------------------------------------------------------------

async function seedHint(
  token: string,
  challengeId: string,
  challengeTitle: string,
): Promise<void> {
  // Public hints endpoint: content hidden for non-unlocked, but count is visible
  const hints = (await apiFetch(`/challenges/${challengeId}/hints`, token)) as Array<{
    id?: string
  }>
  if (hints.length > 0) {
    console.log(`hint already exists for: ${challengeTitle}`)
    return
  }
  await apiFetch(`/admin/challenges/${challengeId}/hints`, token, {
    method: 'POST',
    body: JSON.stringify({ content: 'Look at the page source.', cost: 10 }),
  })
  console.log(`created hint for: ${challengeTitle}`)
}

// ---------------------------------------------------------------------------
// Requirements
// ---------------------------------------------------------------------------

async function seedRequirement(
  token: string,
  challengeId: string,
  requiresId: string,
  challengeTitle: string,
): Promise<void> {
  await apiFetch(`/admin/challenges/${challengeId}/requirements`, token, {
    method: 'PUT',
    body: JSON.stringify({ requirement_ids: [requiresId] }),
  })
  console.log(`set requirement for: ${challengeTitle}`)
}

// ---------------------------------------------------------------------------
// Tags
// ---------------------------------------------------------------------------

async function seedTag(token: string, challengeId: string, challengeTitle: string): Promise<void> {
  const tags = (await apiFetch('/tags', token)) as Array<{ id?: string; name?: string }>
  let webEasyId = tags.find((t) => t.name === 'web-easy')?.id

  if (!webEasyId) {
    const created = (await apiFetch('/admin/tags', token, {
      method: 'POST',
      body: JSON.stringify({ name: 'web-easy', color: '#3b82f6' }),
    })) as { id?: string }
    webEasyId = created.id
    console.log('created tag: web-easy')
  } else {
    console.log('tag already exists: web-easy')
  }

  // Attach tag to challenge via UpdateChallenge (tag_ids)
  const ch = (await apiFetch(`/challenges/${challengeId}`, token)) as {
    title?: string
    category?: string
    description?: string
    points?: number
    state?: string
    tags?: Array<{ id?: string }>
  }
  const existingTagIds: string[] = (ch.tags ?? []).map((t) => t.id ?? '').filter(Boolean)
  if (existingTagIds.includes(webEasyId ?? '')) {
    console.log(`tag already attached to: ${challengeTitle}`)
    return
  }
  await apiFetch(`/admin/challenges/${challengeId}`, token, {
    method: 'PUT',
    body: JSON.stringify({
      title: ch.title ?? challengeTitle,
      category: ch.category ?? 'Web',
      description: ch.description ?? '',
      points: ch.points ?? 100,
      state: ch.state ?? 'visible',
      tag_ids: [...existingTagIds, webEasyId],
    }),
  })
  console.log(`attached web-easy tag to: ${challengeTitle}`)
}

// ---------------------------------------------------------------------------
// Brackets
// ---------------------------------------------------------------------------

async function seedBrackets(token: string): Promise<void> {
  const brackets = (await apiFetch('/brackets', token)) as Array<{ name?: string }>
  const names = new Set(brackets.map((b) => b.name))

  if (!names.has('Rookie')) {
    await apiFetch('/admin/brackets', token, {
      method: 'POST',
      body: JSON.stringify({ name: 'Rookie', is_default: true }),
    })
    console.log('created bracket: Rookie')
  } else {
    console.log('bracket already exists: Rookie')
  }

  if (!names.has('Pro')) {
    await apiFetch('/admin/brackets', token, {
      method: 'POST',
      body: JSON.stringify({ name: 'Pro', is_default: false }),
    })
    console.log('created bracket: Pro')
  } else {
    console.log('bracket already exists: Pro')
  }
}

// ---------------------------------------------------------------------------
// Custom fields
// ---------------------------------------------------------------------------

async function seedFields(token: string): Promise<void> {
  const fields = (await apiFetch('/fields?entity_type=user', token)) as Array<{ name?: string }>
  if (fields.some((f) => f.name === 'country')) {
    console.log('field already exists: country')
    return
  }
  await apiFetch('/admin/fields', token, {
    method: 'POST',
    body: JSON.stringify({ name: 'country', field_type: 'text', entity_type: 'user', required: false, order_index: 0 }),
  })
  console.log('created field: country')
}

// ---------------------------------------------------------------------------
// Pages
// ---------------------------------------------------------------------------

async function seedPages(token: string): Promise<void> {
  const pages = (await apiFetch('/admin/pages', token)) as Array<{ slug?: string }>
  const slugs = new Set(pages.map((p) => p.slug))

  if (!slugs.has('rules')) {
    await apiFetch('/admin/pages', token, {
      method: 'POST',
      body: JSON.stringify({
        title: 'Rules',
        slug: 'rules',
        content: '# Rules\n\nPlay fair.',
        is_draft: false,
        order_index: 0,
      }),
    })
    console.log('created page: rules')
  } else {
    console.log('page already exists: rules')
  }

  if (!slugs.has('draft-me')) {
    await apiFetch('/admin/pages', token, {
      method: 'POST',
      body: JSON.stringify({
        title: 'Draft Me',
        slug: 'draft-me',
        content: 'This is a draft.',
        is_draft: true,
        order_index: 99,
      }),
    })
    console.log('created page: draft-me')
  } else {
    console.log('page already exists: draft-me')
  }
}

// ---------------------------------------------------------------------------
// Notifications
// ---------------------------------------------------------------------------

async function seedNotifications(token: string, user1Id: string): Promise<void> {
  // Use public endpoint to list global notifications (returns pinned first)
  const notifs = (await apiFetch('/notifications', token)) as Array<{
    title?: string
    is_pinned?: boolean
  }>

  if (!notifs.some((n) => n.title === 'Welcome to E2E CTF')) {
    await apiFetch('/admin/notifications', token, {
      method: 'POST',
      body: JSON.stringify({
        title: 'Welcome to E2E CTF',
        content: 'Good luck to all participants!',
        type: 'info',
        is_pinned: true,
      }),
    })
    console.log('created global pinned notification')
  } else {
    console.log('global notification already exists')
  }

  // Personal notification for user1 - always create (stacks across runs, tests use >=1)
  await apiFetch(`/admin/notifications/user/${user1Id}`, token, {
    method: 'POST',
    body: JSON.stringify({
      title: 'Personal message',
      content: 'This notification is just for you.',
      type: 'info',
    }),
  })
  console.log('created personal notification for user1')
}

// ---------------------------------------------------------------------------
// Awards
// ---------------------------------------------------------------------------

async function seedAward(token: string, teamId: string): Promise<void> {
  const awards = (await apiFetch('/admin/awards', token)) as Array<{
    team_id?: string
    description?: string
  }>
  if (awards.some((a) => a.team_id === teamId && a.description === 'E2E participation award')) {
    console.log('award already exists for Team Alpha')
    return
  }
  await apiFetch('/admin/awards', token, {
    method: 'POST',
    body: JSON.stringify({ team_id: teamId, value: 50, description: 'E2E participation award' }),
  })
  console.log('created award for Team Alpha')
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

async function main(): Promise<void> {
  console.log(`seeding e2e data at ${BASE}…`)

  const authDir = path.join(__dirname, '.auth')
  fs.mkdirSync(authDir, { recursive: true })

  const adminToken = await login(ADMIN_EMAIL, ADMIN_PASSWORD)
  await ensureCompetitionActive(adminToken)

  // Raise all rate limits so auth.spec tests and concurrent seed calls don't block
  await apiFetch('/admin/settings', adminToken, {
    method: 'PUT',
    body: JSON.stringify({
      rate_limit_login_per_minute: 1000,
      rate_limit_register_per_minute: 1000,
      rate_limit_forgot_password_per_minute: 1000,
      rate_limit_reset_password_per_minute: 1000,
      rate_limit_general_ip_per_minute: 10000,
    }),
  })

  // Challenges
  const challengeIds = await seedChallenges(adminToken)
  const helloWorldId = challengeIds['Hello World'] ?? ''
  const base64Id = challengeIds['Base64 Basics'] ?? ''

  // Users
  const userIds = await seedUsers(adminToken)
  const user1Id = userIds['user1@example.com'] ?? ''
  const bannedId = userIds['banned@example.com'] ?? ''

  // Ban banned user
  if (bannedId) await ensureBanned(adminToken, bannedId, true)

  // Team Alpha (requires user1 token)
  let teamAlphaId = ''
  if (user1Id) {
    const user1Token = await login('user1@example.com', 'Password1')
    teamAlphaId = await seedTeamAlpha(user1Token, adminToken, user1Id, authDir)
  }

  // Hints, requirements, tags
  if (helloWorldId) {
    await seedHint(adminToken, helloWorldId, 'Hello World')
    await seedTag(adminToken, helloWorldId, 'Hello World')
  }
  if (base64Id && helloWorldId) {
    await seedRequirement(adminToken, base64Id, helloWorldId, 'Base64 Basics')
  }

  // Taxonomy + content
  await seedBrackets(adminToken)
  await seedFields(adminToken)
  await seedPages(adminToken)

  if (user1Id) await seedNotifications(adminToken, user1Id)
  if (teamAlphaId) await seedAward(adminToken, teamAlphaId)

  console.log('seed complete.')
}

main().catch((err: unknown) => {
  console.error('seed failed:', err)
  process.exit(1)
})
