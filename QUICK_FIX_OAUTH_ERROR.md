# Quick Fix: Google OAuth redirect_uri_mismatch Error

## ⚠️ You're Seeing This Error:
```
Error 400: redirect_uri_mismatch
You can't sign in because this app sent an invalid request.
```

---

## ✅ IMMEDIATE FIX (2 minutes)

### Step 1: Check Your .env File
Open `.env` in your project and verify:

```env
GOOGLE_CLIENT_ID=YOUR_VALUE_HERE
GOOGLE_CLIENT_SECRET=YOUR_VALUE_HERE
GOOGLE_REDIRECT_URL=http://localhost:8080/api/auth/google/callback
```

**All three must be set!** If any are empty, the flow will fail.

---

### Step 2: Update Google Cloud Console

1. Go to: https://console.cloud.google.com/
2. Sign in with your Google account
3. Find your project
4. Go to **APIs & Services** > **Credentials**
5. Click on your **OAuth 2.0 Client IDs** (Web application)
6. Under **Authorized redirect URIs**, make sure this is listed:
   ```
   http://localhost:8080/api/auth/google/callback
   ```
7. If NOT there, click **Edit** and add it
8. Click **Save**

---

### Step 3: Verify You Have Credentials
Look at your OAuth credentials and copy:
- **Client ID** → Paste into `GOOGLE_CLIENT_ID=` in .env
- **Client Secret** → Paste into `GOOGLE_CLIENT_SECRET=` in .env

**Example .env after update:**
```env
GOOGLE_CLIENT_ID=YOUR_GOOGLE_CLIENT_ID.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=YOUR_GOOGLE_CLIENT_SECRET
GOOGLE_REDIRECT_URL=http://localhost:8080/api/auth/google/callback
```

---

### Step 4: Restart the App
```bash
# Stop the running app (Ctrl+C)
# Then restart:
go run ./cmd/api/main.go
```

---

### Step 5: Test Again
1. Go to http://localhost:8080
2. Click **"Sign in with Google"**
3. ✅ You should now see Google's login page!

---

## 🔍 Still Not Working?

### Issue 1: Redirect URL Mismatch
**Problem:** The redirect URL doesn't match exactly

**Checklist:**
- ❌ `http://127.0.0.1:8080/api/auth/google/callback` (wrong host)
- ❌ `http://localhost:8080/api/auth/google/Callback` (capital C, wrong!)
- ❌ `http://localhost:8080/api/auth/google/callback/` (extra slash)
- ✅ `http://localhost:8080/api/auth/google/callback` (CORRECT!)

**Fix:** Make sure Google Cloud Console has this EXACT URL registered.

---

### Issue 2: Wrong Port
**Problem:** App running on different port than registered

**Check:**
1. In browser address bar: `http://localhost:8080` ← what port?
2. In `.env`: `SERVER_PORT=8080` ← matches?
3. Google Cloud: registered URI uses `8080`? ← matches?

If app shows `Server on port 8080` but you access `localhost:3000`, that's the problem!

**Fix:** Access `http://localhost:8080` exactly

---

### Issue 3: Expired Credentials
**Problem:** Client Secret or ID is old

**Fix:**
1. Go to Google Cloud Console > Credentials
2. Delete the old OAuth client
3. Create a NEW one
4. Copy the NEW Client ID and Secret to `.env`

---

### Issue 4: .env Not Being Loaded
**Problem:** App not reading .env file

**Check:** In terminal when app starts, look for:
```
Connecting to PostgreSQL...
```

If you see an error or unusual database host, the .env isn't loading.

**Fix:**
```bash
# Make sure you run from project root:
cd "c:\Users\bvssa\OneDrive\Desktop\URL SHORTNER"
go run ./cmd/api/main.go
```

---

### Issue 5: CORS or Frontend Issue
**Problem:** Frontend can't communicate with backend

**Check in Browser Console (F12 > Console):**
Look for errors like:
- `CORS error`
- `Cannot reach http://localhost:8080`
- `TypeError`

**Fix:** Make sure app is running and accessible at `http://localhost:8080`

---

## 📋 Complete Checklist

Before testing Google login, verify ALL of these:

- [ ] App is running: `go run ./cmd/api/main.go`
- [ ] You see: `Starting Server on port 8080...`
- [ ] App is accessible: Open browser to `http://localhost:8080`
- [ ] You see the Snip homepage with "Sign in with Google" button
- [ ] `.env` file has `GOOGLE_CLIENT_ID` (not empty)
- [ ] `.env` file has `GOOGLE_CLIENT_SECRET` (not empty)
- [ ] `.env` file has `GOOGLE_REDIRECT_URL=http://localhost:8080/api/auth/google/callback`
- [ ] Google Cloud Console has OAuth credentials created
- [ ] Google Cloud Console has `http://localhost:8080/api/auth/google/callback` registered as redirect URI
- [ ] You're accessing app from `http://localhost:8080` (not `127.0.0.1` or different port)

If ALL boxes are checked ✓, click "Sign in with Google" and it should work!

---

## 🚀 Still Stuck?

If it's still not working:

1. **Check the browser network tab (F12 > Network)**
   - Click "Sign in with Google"
   - Look at the request to `http://localhost:8080/api/auth/google/login`
   - Check the Response - it should contain a Google OAuth URL
   - If you see an error, screenshot it

2. **Check the app console logs**
   - Look for error messages when you click the button
   - Screenshot and share them

3. **Verify Google Cloud Console again**
   - Screenshot your OAuth 2.0 credentials page
   - Screenshot the "Authorized redirect URIs" section
   - Make sure the exact URL is there

---

## ✅ Success Indicators

When Google OAuth is properly configured:

1. You click "Sign in with Google"
2. You're redirected to Google's login page (accounts.google.com)
3. You sign in with your Google account
4. You see a permission dialog
5. You click "Allow"
6. You're redirected BACK to the app
7. You're now logged in! ✅

