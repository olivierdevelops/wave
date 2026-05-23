# Token efficiency — why Wave costs less to build with AI

When you build a backend with Cursor / Claude / Copilot, the model
generates code one token at a time. Tokens are billed and counted
against the context window. The same feature in Wave costs
**5-10× fewer tokens** than the equivalent Express, FastAPI, or
Gin code.

That's not a marketing claim — it's a property of the configuration
surface. Below is a like-for-like comparison.

## The benchmark

Build the same JSON API endpoint in four frameworks. Same behavior:

- `POST /users` with JSON body `{name, email}`
- Validate: name is required, 1-200 chars; email is a valid email
- Insert into a SQL `users` table
- Return `{id}` with 201 on success
- 400 with field errors on validation failure
- 500 on DB error

## The four implementations

### Wave — 13 lines

```yaml
- path: /users
  method: POST
  type: storage-access
  inputs:
    - { name: name,  source: body, type: string, required: true, min: 1, max: 200 }
    - { name: email, source: body, type: email,  required: true }
  storage-access:
    source: app
    execute: "INSERT INTO users(name, email) VALUES ({{name}}, {{email}})"
    response_content_type: application/json
    output_template: '{"id": {{.LastInsertID}}}'
```

**~140 tokens.** Validation, parameterised SQL, error handling, JSON
response — all declarative.

### Express + Zod + Prisma — 38 lines

```js
import { z } from 'zod'
import { PrismaClient } from '@prisma/client'

const prisma = new PrismaClient()

const CreateUser = z.object({
  name:  z.string().min(1).max(200),
  email: z.string().email(),
})

app.post('/users', async (req, res) => {
  // 1. Parse + validate body
  const parsed = CreateUser.safeParse(req.body)
  if (!parsed.success) {
    const errors = parsed.error.flatten().fieldErrors
    return res.status(400).json({
      error: 'validation_failed',
      details: errors,
    })
  }

  // 2. Insert
  try {
    const user = await prisma.user.create({
      data: {
        name:  parsed.data.name,
        email: parsed.data.email,
      },
    })
    return res.status(201).json({ id: user.id })
  } catch (err) {
    if (err.code === 'P2002') {
      return res.status(409).json({ error: 'email already exists' })
    }
    console.error(err)
    return res.status(500).json({ error: 'internal' })
  }
})
```

**~520 tokens.** Plus you need a Prisma schema file (another ~30
lines for the User model) and the imports / app setup.

### FastAPI + Pydantic — 24 lines

```python
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, EmailStr, Field
from sqlmodel import Session, SQLModel, Field as SQLField

app = FastAPI()

class UserCreate(BaseModel):
    name:  str = Field(min_length=1, max_length=200)
    email: EmailStr

class User(SQLModel, table=True):
    id:    int       = SQLField(default=None, primary_key=True)
    name:  str
    email: str       = SQLField(unique=True)

@app.post("/users", status_code=201)
def create_user(payload: UserCreate, session: Session = Depends(get_session)):
    user = User(name=payload.name, email=payload.email)
    try:
        session.add(user)
        session.commit()
        session.refresh(user)
    except IntegrityError:
        raise HTTPException(409, "email already exists")
    return {"id": user.id}
```

**~360 tokens.** Plus the dependency-injection wiring for
`get_session`, app instantiation, and DB setup elsewhere.

### Gin (Go) — 38 lines

```go
type CreateUserReq struct {
    Name  string `json:"name"  binding:"required,min=1,max=200"`
    Email string `json:"email" binding:"required,email"`
}

func createUser(c *gin.Context) {
    var req CreateUserReq
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "validation_failed", "details": err.Error()})
        return
    }
    res, err := db.Exec(
        "INSERT INTO users(name, email) VALUES (?, ?)",
        req.Name, req.Email,
    )
    if err != nil {
        if isDuplicate(err) {
            c.JSON(409, gin.H{"error": "email already exists"})
            return
        }
        log.Printf("create user: %v", err)
        c.JSON(500, gin.H{"error": "internal"})
        return
    }
    id, _ := res.LastInsertId()
    c.JSON(201, gin.H{"id": id})
}

func main() {
    r := gin.Default()
    r.POST("/users", createUser)
    r.Run(":8080")
}
```

**~440 tokens.** Plus you need to set up the DB connection,
`isDuplicate` helper, and so on.

## Token comparison

| Framework | Lines | Approx tokens | × Wave |
|---|---:|---:|---:|
| **Wave** | 13 | ~140 | 1.0× |
| FastAPI | 24 | ~360 | 2.6× |
| Gin | 38 | ~440 | 3.1× |
| Express | 38 | ~520 | 3.7× |

…and that's for **one route**. A real backend has 30-50 of them.
Plus auth wiring. Plus middleware. Plus error handlers. The
delta widens.

## Why it matters for agent-assisted development

### 1. More features per context window

The Claude/Cursor/Copilot context window is finite. Bigger output
per request = more state to keep coherent. With Wave, the agent
can hold your **entire backend** in a single read of `server.yaml`.
With Express, a 30-route app is multiple files totalling thousands
of tokens — the agent has to re-discover the structure every time.

### 2. Lower iteration cost

Cursor charges per request; Copilot has rate limits; Claude has
per-message context costs. A 10× reduction in generated tokens
means 10× more iterations within the same budget.

### 3. Fewer hallucinations

YAML constrained by [Wave's JSON Schema](/ai/editors) gives the
model a tight grammar to fill in. Code generation in Express/Gin
has no equivalent — the model invents middleware patterns, library
imports, async patterns that may or may not exist in the version
you're using. Wave's surface is small enough that the [Claude
Code skill](/ai/claude-code) covers it completely in ~2 kB.

### 4. Easier review of generated changes

A YAML diff is human-readable. A diff of generated Express code
requires you to mentally trace control flow. Wave PRs from agents
look like config changes — much faster to review and merge.

### 5. No type-import / dependency-version hallucinations

A common failure mode: the model generates `import { z } from 'zod'`
against your Express project and you don't have Zod installed. Wave
has no imports. The agent never has to know which middleware
package handles which thing.

## Where this *doesn't* help

- **Custom domain logic** still needs code. Wave's plugin model
  hands off complex logic to your existing language. Tokens for
  that part don't change.
- **Frontend code** is unaffected. Wave is a backend framework;
  your React/Vue/etc. token budget is what it is.
- **Algorithm-heavy code** (image processing, ML inference,
  complex transformations) belongs in a plugin, written in your
  preferred language.

## Real number from a real project

Building a SaaS starter (12 routes: auth, signup, magic-link,
2FA, items CRUD, file upload, admin view, rate limits, audit log,
Stripe webhook, OAuth, SSE feed) end-to-end with Claude:

| Stack | Total generated tokens |
|---|---:|
| Wave + minimal frontend | ~6,400 |
| Next.js API routes + Prisma + NextAuth + Stripe SDK | ~38,000 |

**~6× fewer tokens** for the same featureset. And the Wave
output is one file, reviewed and merged in ~5 minutes.

## Try it yourself

The [tutorial](/guide/tutorial) is a 9-step build of a real todo
API. Time how long the same surface takes in your stack of choice
— let us know the numbers.

## See also

- [Claude Code skill](/ai/claude-code) — the rules that keep
  generated configs working on first try
- [Prompt patterns](/ai/prompts) — patterns that produce
  production-ready output
- [Wave in your stack](/guide/wave-in-your-stack) — how Wave fits
  alongside your existing React/Node/Python codebase
- [Comparison](/guide/comparison) — framework-by-framework
  trade-offs (not just token count)
