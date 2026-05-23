# React / Next.js + Wave backend

Your frontend is React (Vite, Next.js, Remix, plain CRA, whatever).
You want a small, focused backend that handles auth, persistence,
and the rest of the boring stuff — without standing up an Express
or NestJS server.

## Architecture

```
┌─────────────────┐                ┌──────────────────┐
│  React frontend │  fetch /api/*  │  Wave server     │
│  (Vercel /      │ ─────────────► │  (Fly / Railway) │
│   Netlify /     │   cookies      │  server.yaml     │
│   anywhere)     │   shared       │                  │
└─────────────────┘                └──────────────────┘
```

## Step 1 — wave server.yaml

```yaml
default:
  port: 8080

env:
  JWT_SECRET: { description: "HMAC secret" }

auth:
  app:
    type: jwt
    secret: "${env:JWT_SECRET}"
    cookie_name: session
    cookie_max_age_seconds: 604800     # 1 week
    cookie_secure: true
    cookie_same_site: None             # cross-site → vercel.app ↔ fly.dev

storage:
  app:
    type: sqlite
    path: ./data.db
    tables:
      users: { columns: ["id INTEGER PRIMARY KEY AUTOINCREMENT", "email TEXT NOT NULL UNIQUE"] }
      todos:
        columns:
          - id      INTEGER PRIMARY KEY AUTOINCREMENT
          - user_id INTEGER NOT NULL
          - text    TEXT NOT NULL
          - done    INTEGER NOT NULL DEFAULT 0

routes:
  # Magic-link auth — your React app POSTs the email here
  - path: /api/auth/signin
    methods: [POST]
    type: magic-link-request
    cors_origins: ["https://my-app.vercel.app", "http://localhost:5173"]
    cors_credentials: true
    inputs: [{ name: email, source: body, type: email, required: true }]
    magic-link-request:
      for: app
      email_field: email
      callback_path: /api/auth/callback
      email_template: "Sign in: {{.link}}"

  - path: /api/auth/callback
    methods: [GET]
    type: magic-link-consume
    magic-link-consume:
      for: app
      redirect_on_success: https://my-app.vercel.app/
      redirect_on_failure: https://my-app.vercel.app/?err=expired

  - path: /api/me
    methods: [GET]
    auth: [app]
    type: storage-access
    cors_origins: ["https://my-app.vercel.app", "http://localhost:5173"]
    cors_credentials: true
    storage-access:
      source: app
      execute: "SELECT id, email FROM users WHERE id = {{getUser}} LIMIT 1"
      output_template: '{{toJSON .Data}}'

  - path: /api/todos
    methods: [GET]
    auth: [app]
    type: storage-access
    cors_origins: ["https://my-app.vercel.app", "http://localhost:5173"]
    cors_credentials: true
    storage-access:
      source: app
      execute: "SELECT * FROM todos WHERE user_id = {{getUser}} ORDER BY id DESC"
      output_template: '{{toJSON .Data}}'

  - path: /api/todos
    methods: [POST]
    auth: [app]
    type: storage-access
    cors_origins: ["https://my-app.vercel.app", "http://localhost:5173"]
    cors_credentials: true
    inputs: [{ name: text, source: body, type: string, required: true, min: 1, max: 1000 }]
    storage-access:
      source: app
      execute: "INSERT INTO todos(user_id, text) VALUES ({{getUser}}, {{text}})"
      output_template: '{"id": {{.LastInsertID}}}'
```

## Step 2 — React side

```tsx
// src/api.ts
const API = import.meta.env.VITE_API_URL || 'http://localhost:8080'

async function call(path: string, init: RequestInit = {}) {
  const r = await fetch(`${API}${path}`, {
    ...init,
    credentials: 'include',         // send the session cookie
    headers: { 'Content-Type': 'application/json', ...init.headers },
  })
  if (!r.ok) throw new Error(`${r.status} ${await r.text()}`)
  return r.status === 204 ? null : r.json()
}

export const api = {
  me:        ()                => call('/api/me'),
  signin:    (email: string)   => call('/api/auth/signin', { method: 'POST', body: JSON.stringify({ email }) }),
  listTodos: ()                => call('/api/todos'),
  addTodo:   (text: string)    => call('/api/todos',  { method: 'POST', body: JSON.stringify({ text }) }),
}
```

```tsx
// src/App.tsx
import { useEffect, useState } from 'react'
import { api } from './api'

export default function App() {
  const [me, setMe]       = useState<{email:string} | null>(null)
  const [todos, setTodos] = useState<any[]>([])
  const [email, setEmail] = useState('')

  useEffect(() => {
    api.me().then(setMe).catch(() => setMe(null))
  }, [])

  useEffect(() => {
    if (me) api.listTodos().then(setTodos)
  }, [me])

  if (!me) return (
    <form onSubmit={async (e) => {
      e.preventDefault()
      await api.signin(email)
      alert('Check your email')
    }}>
      <input value={email} onChange={(e) => setEmail(e.target.value)} type="email" required />
      <button>Sign in</button>
    </form>
  )

  return (
    <div>
      <p>Hi {me.email}</p>
      <ul>{todos.map(t => <li key={t.id}>{t.text}</li>)}</ul>
    </div>
  )
}
```

## Local development

Run both on different ports; React calls Wave directly.

```sh
# Terminal 1
JWT_SECRET=$(openssl rand -hex 32) wave serve server.yaml --port 8080

# Terminal 2
VITE_API_URL=http://localhost:8080 npm run dev    # Vite default :5173
```

The `cors_origins` list above includes both production and
localhost, so the same `server.yaml` works in both.

## Next.js variant — proxy through Next's rewrites

If you're on Next.js, you can have Next proxy `/api/*` to Wave so
the browser never sees a cross-origin request and you don't need
`cors_origins` at all:

```js
// next.config.js
module.exports = {
  async rewrites() {
    return [
      { source: '/api/:path*', destination: `${process.env.WAVE_URL}/api/:path*` },
    ]
  },
}
```

Then `WAVE_URL=https://my-api.fly.dev` in Vercel env vars.

## Production deploy

- **Frontend**: Vercel / Netlify / Cloudflare Pages, your usual flow.
- **Wave**: [Fly.io](/guide/deploy-fly) or [Docker](/guide/deploy-docker).
  ~$3/mo for a small VM with a persistent SQLite volume.
- **Custom domains**: `app.example.com` → React, `api.example.com`
  → Wave. Set `cookie_domain: .example.com` so the auth cookie is
  shared.

## Why this beats Next.js API routes + Prisma + NextAuth

- **One config file** vs. ~20 generated files
- **No node_modules** on the API side
- **Cold-start free** — Wave is a single Go binary, ~30 ms boot
- **Type-safe via JSON Schema** — your editor still gets completion
- **Same featureset** out of the box: auth, sessions, validation,
  audit log

When the auth/CRUD layer outgrows Wave (rare), you swap routes
back into Next API routes one at a time. No migration cliff.

## See also

- [Wave in your stack](/guide/wave-in-your-stack)
- [Magic-link login](/cookbook/magic-link-login)
- [OAuth (Google/GitHub)](/cookbook/oauth)
- [Token efficiency](/ai/token-efficiency)
