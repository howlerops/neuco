# Neuco — Deployment Guide (Fly.io + Neon + Vercel)

**Architecture**: Go API + Worker on Fly.io, PostgreSQL on Neon, Frontend on Vercel
**Estimated cost**: $0-5/month (free tiers + minimal usage)

---

## Prerequisites

- Fly CLI installed (`flyctl`)
- Neon account (neon.tech)
- Vercel account (vercel.com)
- GitHub OAuth App created (github.com/settings/developers)
- Anthropic API key
- OpenAI API key

---

## Step 1: Create Neon Database

1. Go to [console.neon.tech](https://console.neon.tech)
2. Create a new project named **neuco**
3. Region: **US East (Ohio)** (closest to Fly.io's `iad` region)
4. Postgres version: **16**
5. Once created, copy the connection string. It looks like:
   ```
   postgresql://neondb_owner:password@ep-xxx.us-east-2.aws.neon.tech/neondb?sslmode=require
   ```

### Enable pgvector

In the Neon SQL Editor, run:
```sql
CREATE EXTENSION IF NOT EXISTS vector;
```

### Run Migrations

From your local machine:
```bash
# Install golang-migrate if you don't have it
brew install golang-migrate  # macOS

# Set the Neon connection string
export DATABASE_URL="postgresql://neondb_owner:password@ep-xxx.us-east-2.aws.neon.tech/neondb?sslmode=require"

# Run all migrations
migrate -path migrations -database "$DATABASE_URL" up
```

Verify:
```bash
psql "$DATABASE_URL" -c "\dt"
# Should show: users, organizations, org_members, projects, signals,
# feature_candidates, specs, generations, pipeline_runs, pipeline_tasks,
# copilot_notes, integrations, audit_log, feature_flags, river_job, etc.
```

---

## Step 2: Deploy API to Fly.io

```bash
cd /path/to/neuco

# Create the API app
fly apps create neuco-api --org personal

# Set secrets (these are encrypted, never in config files)
fly secrets set \
  DATABASE_URL="postgresql://neondb_owner:password@ep-xxx.us-east-2.aws.neon.tech/neondb?sslmode=require" \
  JWT_SECRET="$(openssl rand -base64 64)" \
  INTERNAL_API_TOKEN="$(openssl rand -base64 32)" \
  ANTHROPIC_API_KEY="sk-ant-..." \
  OPENAI_API_KEY="sk-..." \
  GITHUB_CLIENT_ID="your-github-oauth-client-id" \
  GITHUB_CLIENT_SECRET="your-github-oauth-secret" \
  FRONTEND_URL="https://neuco-web.vercel.app" \
  RESEND_API_KEY="re_..." \
  --app neuco-api

# Deploy (--build-target is required for multi-stage Dockerfile)
fly deploy --config fly.api.toml --app neuco-api --build-target server
```

Note your API URL:
```
https://neuco-api.fly.dev
```

### Verify API
```bash
curl https://neuco-api.fly.dev/operator/health
```

---

## Step 3: Deploy Worker to Fly.io

```bash
# Create the worker app
fly apps create neuco-worker --org personal

# Set secrets (same DB + API keys, no HTTP-specific ones)
fly secrets set \
  DATABASE_URL="postgresql://neondb_owner:password@ep-xxx.us-east-2.aws.neon.tech/neondb?sslmode=require" \
  ANTHROPIC_API_KEY="sk-ant-..." \
  OPENAI_API_KEY="sk-..." \
  RESEND_API_KEY="re_..." \
  --app neuco-worker

# Deploy (--build-target is required for multi-stage Dockerfile)
fly deploy --config fly.worker.toml --app neuco-worker --build-target worker
```

### Managing the Worker (On-Demand Demos)

The worker costs money while running (shared CPU). For on-demand demos:

```bash
# Start worker before a demo
fly scale count 1 --app neuco-worker

# Stop worker when done
fly scale count 0 --app neuco-worker

# Check status
fly status --app neuco-worker
```

---

## Step 4: Deploy Frontend to Vercel

```bash
cd neuco-web

# Install Vercel CLI if needed
npm install -g vercel

# Deploy
vercel

# Set environment variables in Vercel dashboard:
# VITE_API_BASE_URL = https://neuco-api.fly.dev
# VITE_GITHUB_CLIENT_ID = your-github-oauth-client-id
```

Or via CLI:
```bash
vercel env add VITE_API_BASE_URL production
# Enter: https://neuco-api.fly.dev

vercel env add VITE_GITHUB_CLIENT_ID production
# Enter: your-github-oauth-client-id

# Redeploy to pick up env vars
vercel --prod
```

Your frontend URL: `https://neuco-web.vercel.app` (or custom domain)

---

## Step 5: Configure GitHub OAuth

1. Go to [github.com/settings/developers](https://github.com/settings/developers)
2. Edit your OAuth App
3. Set **Authorization callback URL** to:
   ```
   https://neuco-web.vercel.app/auth/callback
   ```
4. Set **Homepage URL** to:
   ```
   https://neuco-web.vercel.app
   ```

---

## Step 6: Verify Everything Works

```bash
# 1. Check API health
curl -s https://neuco-api.fly.dev/operator/health

# 2. Open the frontend
open https://neuco-web.vercel.app

# 3. Sign in with GitHub

# 4. Create a project, upload a CSV with some test signals

# 5. Check worker is processing
fly logs --app neuco-worker
```

---

## Ongoing Deploys

### Deploy API changes
```bash
fly deploy --config fly.api.toml --app neuco-api --build-target server
```

### Deploy Worker changes
```bash
fly deploy --config fly.worker.toml --app neuco-worker --build-target worker
```

### Deploy Frontend changes
```bash
cd neuco-web && vercel --prod
```

### Run new migrations
```bash
export DATABASE_URL="your-neon-connection-string"
migrate -path migrations -database "$DATABASE_URL" up
```

---

## Cost Breakdown

| Service | Tier | Cost |
|---------|------|------|
| Fly.io API | Free (shared-cpu-1x, 256MB) | $0/mo* |
| Fly.io Worker | Free when stopped, ~$2/mo if running | $0-2/mo |
| Neon Postgres | Free (0.5GB, 191 compute hours) | $0/mo |
| Vercel Frontend | Free (Hobby) | $0/mo |
| **Total** | | **$0-2/mo** |

*Fly.io free tier includes 3 shared-cpu-1x VMs with 256MB RAM each. The API and worker each use one.

---

## Scaling Up (When Needed)

| Trigger | Action |
|---------|--------|
| Need more DB storage | Neon Launch plan ($19/mo, 10GB) |
| Need worker always-on | Upgrade to Fly dedicated CPU ($7/mo) |
| Need more API capacity | `fly scale count 2 --app neuco-api` |
| Need custom domain | Add in Fly dashboard + Vercel dashboard |
| Need production DB | Neon Scale ($69/mo) or switch to Supabase/RDS |

---

## Useful Commands

```bash
# View API logs
fly logs --app neuco-api

# View worker logs
fly logs --app neuco-worker

# SSH into API container
fly ssh console --app neuco-api

# Check Neon DB
psql "$DATABASE_URL"

# Scale worker for demo
fly scale count 1 --app neuco-worker

# Stop worker after demo
fly scale count 0 --app neuco-worker
```
