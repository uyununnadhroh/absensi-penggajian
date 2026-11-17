# 🚀 Deployment Guide

Panduan lengkap deploy aplikasi Absensi & Penggajian ke production.

## 🏗️ Architecture

```
Frontend (Next.js) → Vercel
Backend (Go)       → Railway.app
Database           → Supabase (PostgreSQL)
```

## 📋 Prerequisites

1. GitHub account
2. Vercel account (login with GitHub)
3. Railway account (login with GitHub)  
4. Supabase account (login with GitHub)

## 🗄️ Step 1: Setup Database (Supabase)

### 1.1 Create Project

1. Go to https://supabase.com
2. Click "New Project"
3. Fill in:
   - **Name**: absensi-penggajian
   - **Database Password**: [create strong password, SAVE IT!]
   - **Region**: Southeast Asia (Singapore)
4. Click "Create new project"
5. Wait ~2 minutes

### 1.2 Run Schema

1. Go to SQL Editor
2. Click "New Query"
3. Copy entire content from `backend/db/schema_postgres.sql`
4. Paste and click "Run"
5. Verify: Table Editor should show 7 tables

### 1.3 Get Connection String

1. Go to Settings → Database
2. Copy "Connection String" (Session mode)
3. Replace `[YOUR-PASSWORD]` with your database password
4. Save this connection string!

Example:
```
postgresql://postgres.xxx:PASSWORD@aws-0-ap-southeast-1.pooler.supabase.com:5432/postgres
```

## 🚂 Step 2: Deploy Backend (Railway)

### 2.1 Create Project

1. Go to https://railway.app
2. Click "New Project"
3. Select "Deploy from GitHub repo"
4. Select: `absensi-penggajian`
5. Railway will auto-detect Go

### 2.2 Configure Environment

1. Click project → Variables
2. Add these variables:

```
PORT=8080

DSN=postgresql://postgres.xxx:PASSWORD@xxx.supabase.com:5432/postgres

JWT_SECRET=your-random-32-character-secret-key-here

TIMEZONE=Asia/Jakarta

CORS_ORIGINS=https://your-app.vercel.app
```

**Generate JWT_SECRET:**
```bash
openssl rand -base64 32
```

### 2.3 Configure Build

1. Go to Settings
2. In "Deploy" section:
   - **Root Directory**: `/`
   - **Build Command**: `cd backend && go build -o ../server ./cmd/server`
   - **Start Command**: `./server`

3. Click "Deploy"
4. Wait ~3 minutes
5. Copy Railway URL: `https://your-app.up.railway.app`

### 2.4 Test Backend

```bash
curl https://your-app.up.railway.app/api/health
```

Expected response:
```json
{"status":"ok"}
```

## ▲ Step 3: Deploy Frontend (Vercel)

### 3.1 Import Project

1. Go to https://vercel.com
2. Click "Add New..." → "Project"
3. Import repository: `absensi-penggajian`

### 3.2 Configure Project

```
Framework Preset: Next.js
Root Directory: frontend
Build Command: npm run build
Output Directory: .next
Install Command: npm install
```

### 3.3 Environment Variables

Add this variable:

```
NEXT_PUBLIC_API_URL=https://your-app.up.railway.app
```

(Use your actual Railway URL)

### 3.4 Deploy

1. Click "Deploy"
2. Wait ~2 minutes
3. Copy Vercel URL: `https://your-app.vercel.app`

### 3.5 Update CORS

1. Go back to Railway
2. Update environment variable:
   ```
   CORS_ORIGINS=https://your-app.vercel.app
   ```
3. Railway will auto-redeploy

## 🔐 Step 4: Create Admin Account

### Option A: Via Railway CLI

```bash
# Install Railway CLI
npm install -g @railway/cli

# Login
railway login

# Link to project
railway link

# Create admin
railway run go run ./backend/cmd/createadmin \
  -email admin@perusahaan.com \
  -nama "Admin HRD" \
  -password "admin12345"
```

### Option B: Via Supabase SQL

1. Generate bcrypt hash for password "admin12345":
   ```
   $2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
   ```

2. Run in Supabase SQL Editor:
   ```sql
   INSERT INTO users (email, password_hash, role, nama) VALUES
     ('admin@perusahaan.com', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'admin', 'Admin HRD');
   ```

## ✅ Step 5: Test Production

1. Open: `https://your-app.vercel.app`
2. Login:
   - Email: `admin@perusahaan.com`
   - Password: `admin12345`
3. Create employee with hourly wage
4. Test check-in/out
5. View payslip

## 🔄 Auto-Deploy

Both Railway and Vercel are connected to GitHub:

```bash
git add .
git commit -m "feat: new feature"
git push origin main
```

Both platforms will auto-deploy! 🎉

## 💰 Costs (Free Tier)

- **Supabase**: 500MB database, 50K API calls/month
- **Railway**: $5 credit/month (enough for this app)
- **Vercel**: 100GB bandwidth, unlimited builds

**Total**: $0/month for normal usage

## 📊 Monitoring

- **Railway Logs**: Dashboard → Deployments → View Logs
- **Vercel Logs**: Dashboard → Deployments → View Function Logs
- **Supabase Logs**: Dashboard → Logs

## 🐛 Troubleshooting

### Database Connection Failed

1. Check connection string format
2. Ensure `?sslmode=require` at the end
3. Verify password is correct

### CORS Error

1. Ensure Vercel URL in Railway `CORS_ORIGINS`
2. No trailing slash: `https://app.vercel.app` ✓
3. Redeploy Railway after env change

### Build Failed

1. Check logs for detailed error
2. Verify all dependencies in go.mod
3. Test build locally first

## 📚 Resources

- [Supabase Docs](https://supabase.com/docs)
- [Railway Docs](https://docs.railway.app)
- [Vercel Docs](https://vercel.com/docs)

## 🎉 Done!

Your app is now:
- ✅ Live in production
- ✅ Auto-deploy from GitHub
- ✅ SSL/HTTPS enabled
- ✅ Free hosting
- ✅ Scalable
- ✅ Auto-backup database
