/* ============================================================
   SNIP — URL Shortener App
   ============================================================ */

(() => {
  'use strict';

  // ---- Configuration ----
  // During local development, call the same origin.
  // When hosted on Cloudflare Pages, call your deployed Go backend API URL (e.g., 'https://my-backend.onrender.com').
  const API_BASE = window.location.hostname.includes('localhost') || 
                   window.location.hostname.includes('127.0.0.1') || 
                   window.location.hostname === '' || 
                   window.location.protocol === 'file:'
    ? 'http://localhost:8080'
    : 'https://url-shortner-gou3.onrender.com';
  const STORAGE_KEY = 'snip_recent_urls';

  // Auth State
  let token = localStorage.getItem('snip_auth_token') || '';
  let username = localStorage.getItem('snip_username') || '';
  let isLoginMode = true;

  // ---- DOM References ----
  const $ = (sel) => document.querySelector(sel);
  const $$ = (sel) => document.querySelectorAll(sel);

  const dom = {
    // Auth elements
    authCard:         $('#auth-card'),
    authForm:         $('#auth-form'),
    authUsername:     $('#auth-username'),
    authPassword:     $('#auth-password'),
    authSubmitBtn:    $('#auth-submit-button'),
    authToggleLink:   $('#auth-toggle-link'),
    authTitle:        $('#auth-title'),
    authSubtitle:     $('#auth-subtitle'),
    userNav:          $('#user-nav'),
    navUsername:      $('#nav-username'),
    logoutBtn:        $('#logout-button'),

    // Shorten form
    shortenCard:    $('#shorten-card'),
    form:           $('#shorten-form'),
    urlInput:       $('#url-input'),
    pasteBtn:       $('#paste-button'),
    shortenBtn:     $('#shorten-button'),
    btnText:        $('#shorten-button .btn__text'),
    btnLoader:      $('#shorten-button .btn__loader'),
    optionsToggle:  $('#options-toggle'),
    optionsPanel:   $('#options-panel'),
    aliasInput:     $('#custom-alias'),
    expirySelect:   $('#expiry-select'),

    // Result card
    resultCard:     $('#result-card'),
    resultShortUrl: $('#result-short-url'),
    copyResultBtn:  $('#copy-result-button'),
    resultClicks:   $('#result-clicks span'),
    resultExpiry:   $('#result-expiry span'),

    // Recent URLs
    recentSection:  $('#recent-section'),
    recentSkeleton: $('#recent-skeleton'),
    recentEmpty:    $('#recent-empty'),
    recentList:     $('#recent-urls-list'),
    recentCount:    $('#recent-count'),

    // Analytics modal
    modalOverlay:   $('#analytics-modal'),
    modalContent:   $('#analytics-modal-content'),
    modalCloseBtn:  $('#modal-close-button'),
    modalTotalNum:  $('#modal-total-clicks .modal__stat-number'),
    modalChart:     $('#modal-clicks-chart'),
    modalDevices:   $('#modal-devices'),
    modalBrowsers:  $('#modal-browsers'),
    modalReferrers: $('#modal-referrers'),

    // Toast
    toastContainer: $('#toast-container'),
  };

  // ---- Utility Helpers ----

  /**
   * Simple URL validator
   */
  function isValidUrl(str) {
    try {
      const url = new URL(str);
      return ['http:', 'https:'].includes(url.protocol);
    } catch {
      return false;
    }
  }

  /**
   * Truncate a string to maxLen, adding ellipsis
   */
  function truncate(str, maxLen = 50) {
    if (!str) return '';
    return str.length > maxLen ? str.slice(0, maxLen) + '…' : str;
  }

  /**
   * Format a number with commas: 1234567 → "1,234,567"
   */
  function formatNumber(n) {
    if (n == null) return '0';
    return Number(n).toLocaleString('en-US');
  }

  /**
   * Relative time: "2 hours ago", "just now", etc.
   */
  function timeAgo(dateStr) {
    if (!dateStr) return '';
    const now = Date.now();
    const then = new Date(dateStr).getTime();
    const diffSec = Math.floor((now - then) / 1000);

    if (diffSec < 10)    return 'just now';
    if (diffSec < 60)    return `${diffSec}s ago`;
    const mins = Math.floor(diffSec / 60);
    if (mins < 60)       return `${mins}m ago`;
    const hours = Math.floor(mins / 60);
    if (hours < 24)      return `${hours}h ago`;
    const days = Math.floor(hours / 24);
    if (days < 30)       return `${days}d ago`;
    const months = Math.floor(days / 30);
    if (months < 12)     return `${months}mo ago`;
    const years = Math.floor(months / 12);
    return `${years}y ago`;
  }

  /**
   * Format expiry for display
   */
  function friendlyExpiry(expiresAt) {
    if (!expiresAt) return 'Never expires';
    const d = new Date(expiresAt);
    const now = Date.now();
    if (d.getTime() <= now) return 'Expired';
    const diff = d.getTime() - now;
    const mins = Math.floor(diff / 60000);
    if (mins < 60)   return `Expires in ${mins}m`;
    const hrs = Math.floor(mins / 60);
    if (hrs < 24)    return `Expires in ${hrs}h`;
    const days = Math.floor(hrs / 24);
    return `Expires in ${days}d`;
  }

  // ---- Toast Notification System ----

  function showToast(message, type = 'info', durationMs = 4000) {
    const toast = document.createElement('div');
    toast.className = `toast toast--${type}`;
    toast.textContent = message;
    dom.toastContainer.appendChild(toast);

    setTimeout(() => {
      toast.classList.add('removing');
      toast.addEventListener('animationend', () => toast.remove());
    }, durationMs);
  }

  // ---- Clipboard ----

  async function copyToClipboard(text, feedbackBtn) {
    try {
      await navigator.clipboard.writeText(text);

      if (feedbackBtn) {
        const iconCopy  = feedbackBtn.querySelector('.icon-copy');
        const iconCheck = feedbackBtn.querySelector('.icon-check');
        const label     = feedbackBtn.querySelector('.btn__label');

        if (iconCopy)  iconCopy.hidden = true;
        if (iconCheck) iconCheck.hidden = false;
        if (label)     label.textContent = 'Copied!';
        feedbackBtn.classList.add('copied');

        setTimeout(() => {
          if (iconCopy)  iconCopy.hidden = false;
          if (iconCheck) iconCheck.hidden = true;
          if (label)     label.textContent = 'Copy';
          feedbackBtn.classList.remove('copied');
        }, 2000);
      }

      showToast('Copied to clipboard!', 'success', 2500);
    } catch {
      // Fallback for older browsers
      const ta = document.createElement('textarea');
      ta.value = text;
      ta.style.cssText = 'position:fixed;left:-9999px';
      document.body.appendChild(ta);
      ta.select();
      document.execCommand('copy');
      ta.remove();
      showToast('Copied to clipboard!', 'success', 2500);
    }
  }

  // ---- LocalStorage Helpers ----

  function saveUrlsToStorage(urls) {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(urls));
    } catch { /* quota exceeded — silently ignore */ }
  }

  function loadUrlsFromStorage() {
    try {
      return JSON.parse(localStorage.getItem(STORAGE_KEY)) || [];
    } catch {
      return [];
    }
  }

  // ---- API Layer ----

  async function apiPost(endpoint, body) {
    const headers = { 'Content-Type': 'application/json' };
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    const res = await fetch(`${API_BASE}${endpoint}`, {
      method: 'POST',
      headers,
      body: JSON.stringify(body),
    });

    if (res.status === 401) {
      handleTokenExpiry();
      throw new Error('Session expired. Please log in again.');
    }

    if (!res.ok) {
      const errData = await res.json().catch(() => ({}));
      throw new Error(errData.error || errData.message || `Request failed (${res.status})`);
    }
    return res.json();
  }

  async function apiGet(endpoint) {
    const headers = {};
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    const res = await fetch(`${API_BASE}${endpoint}`, { headers });

    if (res.status === 401) {
      handleTokenExpiry();
      throw new Error('Session expired. Please log in again.');
    }

    if (!res.ok) {
      const errData = await res.json().catch(() => ({}));
      throw new Error(errData.error || errData.message || `Request failed (${res.status})`);
    }
    return res.json();
  }

  async function apiDelete(endpoint) {
    const headers = {};
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    const res = await fetch(`${API_BASE}${endpoint}`, { method: 'DELETE', headers });

    if (res.status === 401) {
      handleTokenExpiry();
      throw new Error('Session expired. Please log in again.');
    }

    if (!res.ok) {
      const errData = await res.json().catch(() => ({}));
      throw new Error(errData.error || errData.message || `Request failed (${res.status})`);
    }
    return res.json().catch(() => ({}));
  }

  function handleTokenExpiry() {
    token = '';
    username = '';
    localStorage.removeItem('snip_auth_token');
    localStorage.removeItem('snip_username');
    updateViewState();
  }

  // ---- Authentication Logic ----

  function toggleAuthMode() {
    isLoginMode = !isLoginMode;
    if (isLoginMode) {
      dom.authTitle.textContent = 'Welcome Back';
      dom.authSubtitle.textContent = 'Login to access your personal links';
      dom.authSubmitBtn.querySelector('.btn__text').textContent = 'Log In';
      dom.authToggleLink.textContent = 'Need an account? Sign up';
    } else {
      dom.authTitle.textContent = 'Create Account';
      dom.authSubtitle.textContent = 'Sign up to start shortening links';
      dom.authSubmitBtn.querySelector('.btn__text').textContent = 'Sign Up';
      dom.authToggleLink.textContent = 'Already have an account? Log in';
    }
    dom.authForm.reset();
  }

  async function handleAuthSubmit(e) {
    e.preventDefault();
    const userVal = dom.authUsername.value.trim();
    const passVal = dom.authPassword.value;

    if (!userVal || !passVal) {
      showToast('Please enter both username and password!', 'error');
      return;
    }

    const endpoint = isLoginMode ? '/api/login' : '/api/register';
    
    // Toggle loader
    dom.authSubmitBtn.disabled = true;
    dom.authSubmitBtn.querySelector('.btn__text').hidden = true;
    dom.authSubmitBtn.querySelector('.btn__loader').hidden = false;

    try {
      const data = await apiPost(endpoint, { username: userVal, password: passVal });
      
      token = data.token;
      username = data.username;
      localStorage.setItem('snip_auth_token', token);
      localStorage.setItem('snip_username', username);

      showToast(isLoginMode ? 'Welcome back!' : 'Account created successfully!', 'success');
      dom.authForm.reset();
      updateViewState();
    } catch (err) {
      showToast(err.message || 'Authentication failed. Please check credentials.', 'error');
    } finally {
      dom.authSubmitBtn.disabled = false;
      dom.authSubmitBtn.querySelector('.btn__text').hidden = false;
      dom.authSubmitBtn.querySelector('.btn__loader').hidden = true;
    }
  }

  function handleLogout() {
    token = '';
    username = '';
    localStorage.removeItem('snip_auth_token');
    localStorage.removeItem('snip_username');
    showToast('Logged out successfully!', 'info');
    updateViewState();
  }

  function updateViewState() {
    if (token) {
      // Authenticated state
      dom.authCard.hidden = true;
      dom.shortenCard.hidden = false;
      dom.recentSection.hidden = false;
      dom.userNav.hidden = false;
      dom.navUsername.textContent = username;
      
      loadRecentUrls();
    } else {
      // Unauthenticated state
      dom.authCard.hidden = false;
      dom.shortenCard.hidden = true;
      dom.resultCard.hidden = true;
      dom.recentSection.hidden = true;
      dom.userNav.hidden = true;
      dom.navUsername.textContent = '';
      dom.recentList.innerHTML = '';
    }
  }

  // ---- Core Features ----

  /**
   * Shorten a URL
   */
  async function shortenUrl() {
    const rawUrl = dom.urlInput.value.trim();

    if (!rawUrl) {
      showToast('Please enter a URL first!', 'error');
      dom.urlInput.focus();
      return;
    }

    if (!isValidUrl(rawUrl)) {
      showToast('That doesn\'t look like a valid URL. Include http:// or https://', 'error');
      dom.urlInput.focus();
      return;
    }

    // Show loading
    setLoading(true);

    try {
      const body = { url: rawUrl };

      const alias = dom.aliasInput.value.trim();
      if (alias) body.custom_alias = alias;

      const expiry = dom.expirySelect.value;
      if (expiry) body.expires_in = expiry;

      const data = await apiPost('/api/shorten', body);

      // Show result card
      showResultCard(data);

      // Clear inputs
      dom.urlInput.value = '';
      dom.aliasInput.value = '';

      // Refresh recent list
      loadRecentUrls();

      showToast('Link created successfully!', 'success');
    } catch (err) {
      showToast(err.message || 'Something went wrong. Please try again.', 'error');
    } finally {
      setLoading(false);
    }
  }

  function setLoading(isLoading) {
    dom.btnText.hidden   = isLoading;
    dom.btnLoader.hidden = !isLoading;
    dom.shortenBtn.disabled = isLoading;
  }

  function showResultCard(data) {
    const shortUrl = data.short_url || `${window.location.origin}/${data.short_code || data.code || ''}`;

    dom.resultShortUrl.href        = shortUrl;
    dom.resultShortUrl.textContent = shortUrl.replace(/^https?:\/\//, '');

    dom.resultClicks.textContent   = `${formatNumber(data.clicks || 0)} clicks`;
    dom.resultExpiry.textContent   = friendlyExpiry(data.expires_at);

    dom.resultCard.hidden = false;
    // Re-trigger animation
    dom.resultCard.style.animation = 'none';
    dom.resultCard.offsetHeight; // reflow
    dom.resultCard.style.animation = '';

    // Wire copy button
    dom.copyResultBtn.onclick = () => copyToClipboard(shortUrl, dom.copyResultBtn);
  }

  /**
   * Load recent URLs from API (with localStorage fallback)
   */
  async function loadRecentUrls() {
    if (!token) return;

    dom.recentSkeleton.hidden = false;
    dom.recentEmpty.hidden    = true;
    dom.recentList.innerHTML  = '';

    try {
      const data = await apiGet('/api/urls');
      const urls = Array.isArray(data) ? data : (data.urls || []);

      saveUrlsToStorage(urls);
      renderUrlList(urls);
    } catch {
      // Fallback to localStorage
      const cached = loadUrlsFromStorage();
      if (cached.length) {
        renderUrlList(cached);
      } else {
        dom.recentSkeleton.hidden = true;
        dom.recentEmpty.hidden    = false;
      }
    }
  }

  function renderUrlList(urls) {
    dom.recentSkeleton.hidden = true;

    if (!urls.length) {
      dom.recentEmpty.hidden = false;
      dom.recentCount.textContent = '';
      return;
    }

    dom.recentEmpty.hidden = true;
    dom.recentCount.textContent = `${urls.length} link${urls.length !== 1 ? 's' : ''}`;

    dom.recentList.innerHTML = urls.map((u, i) => {
      const code     = u.short_code || u.code || '';
      const shortUrl = u.short_url || `${window.location.origin}/${code}`;
      const display  = shortUrl.replace(/^https?:\/\//, '');
      const original = truncate(u.original_url || u.url || '', 50);
      const clicks   = formatNumber(u.click_count || u.clicks || 0);
      const created  = timeAgo(u.created_at);

      return `
        <article class="url-card" style="animation-delay: ${i * 0.06}s" data-code="${code}">
          <div class="url-card__info">
            <a href="${shortUrl}" class="url-card__short" target="_blank" rel="noopener noreferrer">${display}</a>
            <p class="url-card__original" title="${u.original_url || u.url || ''}">${original}</p>
            <div class="url-card__meta">
              <span class="url-card__meta-tag">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
                ${clicks} clicks
              </span>
              <span class="url-card__meta-tag">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
                ${created}
              </span>
            </div>
          </div>
          <div class="url-card__actions">
            <button class="url-card__btn url-card__btn--stats" title="View analytics" aria-label="View analytics for ${code}" onclick="window.__snip.showAnalytics('${code}')">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="20" x2="18" y2="10"/><line x1="12" y1="20" x2="12" y2="4"/><line x1="6" y1="20" x2="6" y2="14"/></svg>
            </button>
            <button class="url-card__btn url-card__btn--copy" title="Copy short URL" aria-label="Copy short URL" onclick="window.__snip.copyCardUrl('${shortUrl}', this)">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
            </button>
            <button class="url-card__btn url-card__btn--delete" title="Delete link" aria-label="Delete link ${code}" onclick="window.__snip.deleteUrl('${code}')">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
            </button>
          </div>
        </article>`;
    }).join('');
  }

  /**
   * Delete a URL
   */
  async function deleteUrl(code) {
    if (!confirm('Delete this link? This action cannot be undone.')) return;

    try {
      await apiDelete(`/api/urls/${code}`);
      showToast('Link deleted!', 'success');
      loadRecentUrls();
    } catch (err) {
      showToast(err.message || 'Could not delete link.', 'error');
    }
  }

  /**
   * Show analytics modal
   */
  async function showAnalytics(code) {
    dom.modalOverlay.hidden = false;
    document.body.style.overflow = 'hidden';

    // Reset content
    dom.modalTotalNum.textContent  = '…';
    dom.modalChart.innerHTML       = '<p class="bar-chart__empty">Loading…</p>';
    dom.modalDevices.innerHTML     = '';
    dom.modalBrowsers.innerHTML    = '';
    dom.modalReferrers.innerHTML   = '<li class="referrer-list__empty">Loading…</li>';

    try {
      const data = await apiGet(`/api/urls/${code}/stats`);

      // Total clicks
      dom.modalTotalNum.textContent = formatNumber(data.total_clicks ?? data.clicks ?? 0);

      // Clicks over time chart
      renderClicksChart(data.clicks_per_day || data.clicks_over_time || []);

      // Device breakdown
      renderBreakdown(dom.modalDevices, data.device_breakdown || data.devices || {});

      // Browser breakdown
      renderBreakdown(dom.modalBrowsers, data.browser_breakdown || data.browsers || {});

      // Referrers
      renderReferrers(data.top_referers || data.referrers || []);

    } catch (err) {
      dom.modalTotalNum.textContent = '–';
      dom.modalChart.innerHTML = '<p class="bar-chart__empty">Could not load analytics</p>';
      showToast(err.message || 'Failed to load analytics.', 'error');
    }
  }

  function renderClicksChart(dataPoints) {
    if (!dataPoints || !dataPoints.length) {
      dom.modalChart.innerHTML = '<p class="bar-chart__empty">No click data yet</p>';
      return;
    }

    const max = Math.max(...dataPoints.map(d => d.count || d.clicks || 0), 1);

    dom.modalChart.innerHTML = dataPoints.map(d => {
      const val = d.count || d.clicks || 0;
      const pct = Math.max((val / max) * 100, 4);
      const label = d.date || d.day || '';
      // Show short date label
      const shortLabel = label.length > 5 ? label.slice(5) : label; // "MM-DD"

      return `
        <div class="bar-chart__bar-wrap">
          <span class="bar-chart__value">${val}</span>
          <div class="bar-chart__bar" style="height: ${pct}%"></div>
          <span class="bar-chart__label">${shortLabel}</span>
        </div>`;
    }).join('');
  }

  function renderBreakdown(container, dataObj) {
    const entries = Array.isArray(dataObj)
      ? dataObj.map(e => ({ name: e.device || e.browser || e.name || 'Unknown', count: e.count }))
      : Object.entries(dataObj).map(([name, count]) => ({ name, count }));

    if (!entries.length) {
      container.innerHTML = '<p style="color:var(--text-secondary);font-size:0.88rem;">No data yet</p>';
      return;
    }

    const total = entries.reduce((sum, e) => sum + (e.count || 0), 0) || 1;

    container.innerHTML = entries.map(e => {
      const pct = Math.round(((e.count || 0) / total) * 100);
      return `
        <div class="breakdown-item">
          <div class="breakdown-item__header">
            <span class="breakdown-item__name">${e.name}</span>
            <span class="breakdown-item__pct">${pct}%</span>
          </div>
          <div class="breakdown-item__bar-bg">
            <div class="breakdown-item__bar-fill" style="width: ${pct}%"></div>
          </div>
        </div>`;
    }).join('');
  }

  function renderReferrers(referrers) {
    const list = Array.isArray(referrers)
      ? referrers.map(r => ({ domain: r.referer || r.domain || r.name || 'Direct', count: r.count }))
      : Object.entries(referrers).map(([domain, count]) => ({ domain, count }));

    if (!list.length) {
      dom.modalReferrers.innerHTML = '<li class="referrer-list__empty">No referrer data yet</li>';
      return;
    }

    dom.modalReferrers.innerHTML = list.map(r => `
      <li class="referrer-list__item">
        <span class="referrer-list__domain">${r.domain || 'Direct'}</span>
        <span class="referrer-list__count">${formatNumber(r.count)}</span>
      </li>`).join('');
  }

  function closeModal() {
    dom.modalOverlay.hidden = true;
    document.body.style.overflow = '';
  }

  // ---- Event Bindings ----

  // Auth submits
  dom.authForm.addEventListener('submit', handleAuthSubmit);
  dom.authToggleLink.addEventListener('click', (e) => {
    e.preventDefault();
    toggleAuthMode();
  });
  dom.logoutBtn.addEventListener('click', handleLogout);

  // Form submit
  dom.form.addEventListener('submit', (e) => {
    e.preventDefault();
    shortenUrl();
  });

  // Paste button
  dom.pasteBtn.addEventListener('click', async () => {
    try {
      const text = await navigator.clipboard.readText();
      dom.urlInput.value = text;
      dom.urlInput.focus();
      showToast('Pasted from clipboard!', 'info', 2000);
    } catch {
      showToast('Could not read clipboard. Try Ctrl+V instead.', 'error');
    }
  });

  // Options toggle
  dom.optionsToggle.addEventListener('click', () => {
    const isOpen = dom.optionsPanel.hidden;
    dom.optionsPanel.hidden = !isOpen;
    dom.optionsToggle.setAttribute('aria-expanded', isOpen);
  });

  // Modal close
  dom.modalCloseBtn.addEventListener('click', closeModal);

  // Click outside modal content to close
  dom.modalOverlay.addEventListener('click', (e) => {
    if (e.target === dom.modalOverlay) closeModal();
  });

  // ---- Keyboard Shortcuts ----

  document.addEventListener('keydown', (e) => {
    // Escape closes modal
    if (e.key === 'Escape' && !dom.modalOverlay.hidden) {
      closeModal();
      return;
    }

    // Ctrl/Cmd + V while input is not focused → focus input and paste
    if ((e.ctrlKey || e.metaKey) && e.key === 'v') {
      if (document.activeElement !== dom.urlInput && document.activeElement !== dom.aliasInput && token) {
        e.preventDefault();
        dom.urlInput.focus();
        navigator.clipboard.readText()
          .then(text => {
            dom.urlInput.value = text;
            showToast('Pasted from clipboard!', 'info', 2000);
          })
          .catch(() => {});
      }
    }
  });

  // ---- Expose globals for inline onclick handlers ----
  window.__snip = {
    showAnalytics,
    deleteUrl,
    copyCardUrl: (url, btn) => copyToClipboard(url, null).then(() => {
      btn.style.color = 'var(--accent-secondary)';
      setTimeout(() => btn.style.color = '', 1500);
    }),
  };

  // ---- Init ----
  updateViewState();

})();
