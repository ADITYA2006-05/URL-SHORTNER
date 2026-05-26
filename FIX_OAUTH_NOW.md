# 🔧 IMMEDIATE ACTION CHECKLIST - Fix Google OAuth Error

## Your Error
```
Error 400: redirect_uri_mismatch
You can't sign in because this app sent an invalid request.
```

## Why You're Getting This
The app is trying to use: `http://localhost:8080/api/auth/google/callback`
But Google Cloud has registered: **SOMETHING DIFFERENT** (or nothing at all)

---

## ✅ FIX IT RIGHT NOW (5 minutes)

### Step 1️⃣ - Open Google Cloud Console
```
Go to: https://console.cloud.google.com/
Sign in with your Google account
```

### Step 2️⃣ - Find Your OAuth Credentials
1. Click on **APIs & Services** (left sidebar)
2. Click on **Credentials**
3. Look for "OAuth 2.0 Client IDs" section
4. Find the one called "Web application"
5. Click **Edit** (pencil icon)

### Step 3️⃣ - Add the Callback URL
Under "Authorized redirect URIs", add this EXACT URL:
```
http://localhost:8080/api/auth/google/callback
```

Click **Save**

### Step 4️⃣ - Copy Your Credentials
On the credentials page, you should see:
- **Client ID** (long string ending in .apps.googleusercontent.com)
- **Client Secret** (long random string starting with GOCSPX-...)

Copy both!

### Step 5️⃣ - Update Your .env File
Open `.env` in your project and update:

```env
GOOGLE_CLIENT_ID=PASTE_YOUR_CLIENT_ID_HERE
GOOGLE_CLIENT_SECRET=PASTE_YOUR_CLIENT_SECRET_HERE
GOOGLE_REDIRECT_URL=http://localhost:8080/api/auth/google/callback
```

**Make sure:**
- GOOGLE_REDIRECT_URL is EXACTLY as shown above (case-sensitive!)
- No extra spaces or characters
- All three values are filled in (not empty)

### Step 6️⃣ - Restart Your App
```bash
# Stop the running app (Ctrl+C)

# Restart it:
go run ./cmd/api/main.go
```

### Step 7️⃣ - Test It! 🎉
1. Go to: http://localhost:8080
2. Click: **"Sign in with Google"**
3. You should see Google login page ✅

---

## ❌ If It Still Doesn't Work

### Problem 1: I Don't Have Google Credentials Yet
**Solution:** Follow this guide: [GOOGLE_OAUTH_SETUP.md](./GOOGLE_OAUTH_SETUP.md)

### Problem 2: Copy-Paste Still Giving Error
**Checklist:**
- [ ] Did you copy credentials from the SAME project?
- [ ] Did you save the redirect URI before copying?
- [ ] Did you paste into the RIGHT lines in .env?
- [ ] Did you restart the app after updating .env?
- [ ] Are you accessing http://localhost:8080 (NOT 127.0.0.1)?

### Problem 3: Can't Find Google Cloud Console
1. Try: https://console.cloud.google.com/apis/credentials
2. You might need to create a project first
3. If so, follow: [GOOGLE_OAUTH_SETUP.md](./GOOGLE_OAUTH_SETUP.md) Step 1-2

### Problem 4: Multiple Projects in Google Cloud
**Common Issue:** You created credentials in project A but looking in project B

**Fix:**
1. Look at your `.env` GOOGLE_CLIENT_ID
2. In Google Cloud, switch projects (dropdown at top)
3. Look for a project that has that Client ID
4. That's the correct project!

---

## 🎓 What's Actually Happening

When you click "Sign in with Google":

```
1. Your browser → http://localhost:8080/api/auth/google/login
   ↓
2. Backend says: "Go to Google, but when done, come back to:
   http://localhost:8080/api/auth/google/callback"
   ↓
3. Your browser → Google login page
   ↓
4. You sign in and grant permission
   ↓
5. Google redirects → http://localhost:8080/api/auth/google/callback
   ↓
6. Backend exchanges the code for a token
   ↓
7. You're logged in! ✅

BUT if Google Cloud says "I don't recognize that callback URL"
→ ERROR: redirect_uri_mismatch ❌
```

The fix ensures Google recognizes the callback URL.

---

## 📋 Final Verification

Before clicking "Sign in with Google", verify:

1. App is running
   ```bash
   Terminal shows: "Starting Server on port 8080..."
   ```

2. Browser shows the app
   ```
   Go to: http://localhost:8080
   You should see: "Snip — URL Shortener" page
   You should see: "Sign in with Google" button
   ```

3. .env is correct
   ```
   GOOGLE_CLIENT_ID is NOT empty
   GOOGLE_CLIENT_SECRET is NOT empty
   GOOGLE_REDIRECT_URL=http://localhost:8080/api/auth/google/callback
   ```

4. Google Cloud has the redirect URI
   ```
   console.cloud.google.com
   → APIs & Services
   → Credentials
   → OAuth Client
   → "Authorized redirect URIs" contains:
      http://localhost:8080/api/auth/google/callback
   ```

All 4 checked? ✅ Click the button!

---

## 🆘 Still Stuck?

1. Read: [QUICK_FIX_OAUTH_ERROR.md](./QUICK_FIX_OAUTH_ERROR.md) (Detailed troubleshooting)
2. Read: [GOOGLE_OAUTH_SETUP.md](./GOOGLE_OAUTH_SETUP.md) (Full setup guide)
3. Check browser console (F12 > Console) for JavaScript errors
4. Check app console for Go backend errors

---

## ✅ Success Sign

When it works, you'll see:
1. Click "Sign in with Google"
2. Redirected to accounts.google.com (Google login)
3. You sign in with your Google account
4. A permission screen appears
5. Click "Allow" or "Continue"
6. Redirected back to http://localhost:8080
7. Page shows you're logged in ✅

That's it! Google OAuth is working! 🎉

