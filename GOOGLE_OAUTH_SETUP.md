# Google OAuth 2.0 Setup Guide - Fix redirect_uri_mismatch Error

## Problem
You're getting the error:
```
Error 400: redirect_uri_mismatch
You can't sign in because this app sent an invalid request.
```

This occurs when the redirect URI sent by your app **does NOT match** what's configured in Google Cloud Console.

---

## Solution: Step-by-Step Setup

### Step 1: Create a Google Cloud Project

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Sign in with your Google account
3. Click the project dropdown at the top
4. Click "NEW PROJECT"
5. Enter project name: `URL Shortener` (or any name)
6. Click "CREATE"

---

### Step 2: Enable Google+ API

1. In Cloud Console, go to **APIs & Services** > **Library**
2. Search for `Google+ API`
3. Click on it and click **ENABLE**

---

### Step 3: Create OAuth 2.0 Credentials

1. Go to **APIs & Services** > **Credentials**
2. Click **CREATE CREDENTIALS** button (top left)
3. Select **OAuth client ID**
4. If prompted to configure the OAuth consent screen:
   - Click **CONFIGURE CONSENT SCREEN**
   - Choose **External** user type
   - Click **CREATE**
   - Fill in:
     - **App name**: `Snip URL Shortener`
     - **User support email**: Your email
     - **Developer contact**: Your email
   - Click **SAVE AND CONTINUE**
   - Skip optional scopes, click **SAVE AND CONTINUE**
   - Review and click **BACK TO DASHBOARD**

5. Now create OAuth credentials:
   - Go back to **Credentials** page
   - Click **CREATE CREDENTIALS** > **OAuth client ID**
   - Choose **Web application**
   - Name: `URL Shortener App`
   - Under **Authorized redirect URIs**, add these URLs:
     ```
     http://localhost:8080/api/auth/google/callback
     ```
   - If deploying to production, also add:
     ```
     https://your-domain.com/api/auth/google/callback
     ```
   - Click **CREATE**

6. A modal appears with your credentials:
   - Copy the **Client ID**
   - Copy the **Client Secret**

---

### Step 4: Update Your `.env` File

Update the `.env` file in your project root:

```env
# Google OAuth 2.0 Configuration
GOOGLE_CLIENT_ID=YOUR_COPIED_CLIENT_ID_HERE
GOOGLE_CLIENT_SECRET=YOUR_COPIED_CLIENT_SECRET_HERE
GOOGLE_REDIRECT_URL=http://localhost:8080/api/auth/google/callback
```

**Important:** Keep the `GOOGLE_REDIRECT_URL` exactly as shown above for local development.

---

### Step 5: Run the Application Locally

1. **Start PostgreSQL** (using Docker):
   ```bash
   docker-compose up postgres redis
   ```

2. **Build and run the app**:
   ```bash
   go run ./cmd/api/main.go
   ```

3. **Open in browser**:
   ```
   http://localhost:8080
   ```

---

### Step 6: Test Google Login

1. Click **"Sign in with Google"** button on the app
2. You should be redirected to Google's login page
3. Sign in with your Google account
4. You'll be asked to grant permissions - click **ALLOW**
5. You should be redirected back to the app and logged in ✅

---

## Troubleshooting

### Still getting redirect_uri_mismatch?

**Issue 1: Wrong URL Format**
- The redirect URL is case-sensitive and must match exactly
- ❌ Don't use: `http://localhost:8080/api/auth/google/Callback`
- ✅ Use: `http://localhost:8080/api/auth/google/callback`

**Issue 2: Accessing app from wrong URL**
- If you open `127.0.0.1:8080`, the frontend API_BASE will use `http://127.0.0.1:8080`
- This won't match `http://localhost:8080` in Google Cloud
- Solution: Always use `http://localhost:8080` in browser

**Issue 3: Credentials expired or incorrect**
- Double-check you copied the Client ID and Secret correctly
- Special characters like `_`, `-`, etc. matter!
- Re-download the credentials if unsure

**Issue 4: Port mismatch**
- Make sure the app is running on port 8080
- Check: `SERVER_PORT=8080` in your `.env`
- Check in console: `Starting Server on port 8080...`

---

## For Production Deployment

When deploying to a real server (e.g., Render, Heroku, AWS):

1. Add your production domain to Google Cloud:
   - Go to **Credentials** > **OAuth 2.0 Client IDs** > Your web app
   - Edit and add:
     ```
     https://your-domain.com/api/auth/google/callback
     ```

2. Update your `.env`:
   ```env
   BASE_URL=https://your-domain.com
   GOOGLE_REDIRECT_URL=https://your-domain.com/api/auth/google/callback
   ```

3. Deploy and test again

---

## Architecture Overview

The Google OAuth flow works as follows:

```
1. User clicks "Sign in with Google"
   ↓
2. Frontend redirects to: /api/auth/google/login
   ↓
3. Backend constructs Google Auth URL with redirect_uri and redirects user
   ↓
4. User sees Google login screen
   ↓
5. User grants permissions
   ↓
6. Google redirects to: YOUR_REDIRECT_URI with authorization code
   ↓
7. Backend exchanges code for JWT token
   ↓
8. Backend redirects to frontend (logged in ✅)
```

The redirect_uri in step 3 **MUST** match what's registered in Google Cloud Console, otherwise you get the error.

---

## Need Help?

If still having issues:

1. Check your `.env` file has non-empty `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET`
2. Verify the `GOOGLE_REDIRECT_URL` matches exactly in Google Cloud Console
3. Check browser console (F12 > Network) to see the redirect URLs being used
4. Make sure you're accessing the app from `http://localhost:8080`, not `127.0.0.1`

