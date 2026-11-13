/*
 * JSONAir Admin — page behaviour.
 * Embedded into the binary and served from /static/app.js.
 *
 * Every feature here is progressive: pages render and submit without it.
 * No inline scripts or handlers are used, so a strict CSP can be added later.
 */
(function () {
  'use strict';

  const $ = (sel, root) => (root || document).querySelector(sel);
  const $$ = (sel, root) => Array.from((root || document).querySelectorAll(sel));

  const debounce = (fn, ms) => {
    let t;
    return function () {
      clearTimeout(t);
      t = setTimeout(fn, ms);
    };
  };

  const fmtDuration = (secs) => {
    if (secs < 60) return secs + 's';
    const m = Math.floor(secs / 60);
    if (m < 60) return m + ' min';
    const h = Math.floor(m / 60);
    return h + 'h ' + (m % 60) + 'm';
  };

  // Set when a form is legitimately submitting, so the unsaved-changes
  // prompt does not fire on our own navigation.
  let allowLeave = false;

  /* ---------- CSRF token ---------- */

  // The token is tied to the session cookie. When the session changes (for
  // example the user signs in again in another tab), /session hands out the
  // new token and setToken() updates the page with it.
  const csrfMeta = $('meta[name="csrf-token"]');
  const getToken = () => (csrfMeta ? csrfMeta.content : '');
  const setToken = (t) => {
    if (!t) return;
    if (csrfMeta) csrfMeta.content = t;
    $$('input[name="csrf_token"]').forEach((i) => {
      i.value = t;
    });
  };

  /* ---------- Flash messages ---------- */

  $$('[data-dismiss]').forEach((btn) => {
    btn.addEventListener('click', () => {
      const el = btn.closest('[data-flash]');
      if (el) el.remove();
    });
  });

  /* ---------- Copy to clipboard ---------- */

  const copyText = (text) => {
    if (navigator.clipboard && window.isSecureContext) {
      return navigator.clipboard.writeText(text);
    }
    // Plain-HTTP fallback (navigator.clipboard needs a secure context).
    return new Promise((resolve, reject) => {
      const ta = document.createElement('textarea');
      ta.value = text;
      ta.setAttribute('readonly', '');
      ta.style.position = 'fixed';
      ta.style.opacity = '0';
      document.body.appendChild(ta);
      ta.select();
      try {
        document.execCommand('copy') ? resolve() : reject(new Error('copy failed'));
      } catch (e) {
        reject(e);
      } finally {
        ta.remove();
      }
    });
  };

  $$('[data-copy]').forEach((btn) => {
    let timer;
    const show = (label) => {
      btn.setAttribute('data-copied', label);
      clearTimeout(timer);
      timer = setTimeout(() => btn.removeAttribute('data-copied'), 1500);
    };
    btn.addEventListener('click', () => {
      copyText(btn.getAttribute('data-copy')).then(
        () => show('Copied'),
        () => show('Copy failed')
      );
    });
  });

  /* ---------- Delete confirmation dialog ---------- */

  const dialog = $('#delete-dialog');
  let pendingDelete = null;

  $$('form[data-confirm-delete]').forEach((form) => {
    form.addEventListener('submit', (e) => {
      e.preventDefault();
      const label = form.dataset.label || 'this configuration';
      const key = form.dataset.key || '';

      if (!dialog || typeof dialog.showModal !== 'function') {
        if (window.confirm('Delete ' + label + '? This cannot be undone.')) {
          allowLeave = true;
          form.submit();
        }
        return;
      }

      pendingDelete = form;
      $('[data-dialog-label]', dialog).textContent = label;
      const keyWrap = $('[data-dialog-key-wrap]', dialog);
      keyWrap.hidden = !key;
      $('[data-dialog-key]', dialog).textContent = key;
      dialog.showModal();
    });
  });

  if (dialog) {
    $('[data-dialog-cancel]', dialog).addEventListener('click', () => dialog.close());
    $('[data-dialog-confirm]', dialog).addEventListener('click', () => {
      if (!pendingDelete) return;
      const form = pendingDelete;
      const btn = $('[data-dialog-confirm]', dialog);
      btn.disabled = true;
      submitWithFreshSession(form).then((sent) => {
        if (!sent) {
          btn.disabled = false;
          dialog.close();
        }
      });
    });
    // Click on the backdrop closes the dialog.
    dialog.addEventListener('click', (e) => {
      if (e.target === dialog) dialog.close();
    });
    dialog.addEventListener('close', () => {
      pendingDelete = null;
    });
  }

  /* ---------- Session countdown & expiry ---------- */

  const sessEl = $('[data-session-remaining]');
  const banner = $('#session-banner');
  let sessionDeadline = null;

  const showBanner = (expired, secsLeft) => {
    if (!banner) return;
    banner.textContent = '';
    banner.classList.toggle('is-expired', expired);
    const msg = document.createElement('span');
    msg.textContent = expired
      ? 'Your session has expired. Saving now would lose your changes.'
      : 'Your session ends in ' + fmtDuration(secsLeft) + '. Save your work before then.';
    const link = document.createElement('a');
    link.href = '/login';
    link.target = '_blank';
    link.rel = 'noopener';
    link.textContent = expired ? 'Sign in again in a new tab, then save here' : 'Sign in again in a new tab';
    banner.append(msg, link);
    banner.hidden = false;
  };

  const tick = () => {
    if (sessionDeadline === null) return;
    const left = Math.round((sessionDeadline - Date.now()) / 1000);
    if (left <= 0) {
      sessEl.textContent = ' · session expired';
      showBanner(true, 0);
      return;
    }
    sessEl.textContent = ' · session ends in ' + fmtDuration(left);
    if (left <= 300) {
      showBanner(false, left);
    } else if (banner) {
      banner.hidden = true;
    }
  };

  // Asks the server whether we are still signed in (for example after the
  // user signed in again in another tab), and refreshes the countdown and the
  // CSRF token. Resolves to 'ok', 'expired' or 'error' (network trouble).
  let lastCheck = 0;
  const checkSession = () => {
    lastCheck = Date.now();
    return fetch('/session', {
      credentials: 'same-origin',
      cache: 'no-store',
      headers: { Accept: 'application/json' },
    })
      .then((r) => {
        if (r.redirected || r.status === 401) return 'expired';
        if (!r.ok) return 'error';
        return r.json().then((s) => {
          setToken(s.csrf);
          if (sessionDeadline !== null) sessionDeadline = Date.now() + s.remaining * 1000;
          tick();
          return 'ok';
        });
      })
      .catch(() => 'error');
  };

  // Submits a form once the session (and so its CSRF token) is confirmed
  // current. On an expired session nothing is sent, so typed edits survive.
  // Resolves to true when the form was submitted.
  const submitWithFreshSession = (form) =>
    checkSession().then((state) => {
      if (state === 'expired') {
        if (sessionDeadline !== null) sessionDeadline = Date.now();
        showBanner(true, 0);
        return false;
      }
      // 'ok', or 'error': let the normal submit report any problem.
      allowLeave = true;
      form.submit();
      return true;
    });

  if (sessEl) {
    const rem = parseInt(sessEl.dataset.sessionRemaining, 10);
    if (!isNaN(rem)) {
      sessionDeadline = Date.now() + rem * 1000;
      tick();
      setInterval(tick, 1000);
      // Coming back to this tab: they may have signed in again elsewhere.
      window.addEventListener('focus', () => {
        if (Date.now() - lastCheck > 30000) checkSession();
      });
    }
  }

  /* ---------- Unsaved changes, Cmd/Ctrl+S ---------- */

  const tracked = $('form[data-track-changes]');

  if (tracked) {
    const snapshot = () =>
      Array.from(new FormData(tracked).entries())
        .map((kv) => kv[0] + '=' + kv[1])
        .join('\u0000');

    const initial = snapshot();
    const indicator = $('[data-dirty-indicator]');
    // A form re-rendered after a failed save holds unsaved values already.
    let dirty = tracked.hasAttribute('data-start-dirty');

    const showDirty = () => {
      if (indicator) indicator.hidden = !dirty;
    };
    const check = debounce(() => {
      dirty = tracked.hasAttribute('data-start-dirty') || snapshot() !== initial;
      showDirty();
    }, 150);
    showDirty();

    tracked.addEventListener('input', check);
    tracked.addEventListener('change', check);

    // Confirm the session first: an expired session or a stale token would
    // otherwise throw away everything typed into the form.
    let submitting = false;
    tracked.addEventListener('submit', (e) => {
      e.preventDefault();
      if (submitting) return;
      submitting = true;
      submitWithFreshSession(tracked).then((sent) => {
        if (!sent) submitting = false;
      });
    });

    window.addEventListener('beforeunload', (e) => {
      if (dirty && !allowLeave) {
        e.preventDefault();
        e.returnValue = '';
      }
    });

    document.addEventListener('keydown', (e) => {
      if ((e.metaKey || e.ctrlKey) && !e.altKey && !e.shiftKey && e.key.toLowerCase() === 's') {
        e.preventDefault();
        if (typeof tracked.requestSubmit === 'function') tracked.requestSubmit();
        else submitWithFreshSession(tracked);
      }
    });
  }

  /* ---------- Rename warning (edit page) ---------- */

  const renameWarn = $('#rename-warning');
  if (renameWarn) {
    const watched = $$('[data-original]');
    const typeField = $('#f-type');
    const nameField = $('#f-name');
    const textEl = $('[data-rename-text]', renameWarn);

    const code = (s) => {
      const c = document.createElement('code');
      c.textContent = s;
      return c;
    };

    const update = () => {
      const changed = watched.filter((f) => f.value !== f.dataset.original);
      watched.forEach((f) => f.classList.toggle('is-changed', changed.indexOf(f) !== -1));
      if (!changed.length) {
        renameWarn.hidden = true;
        return;
      }
      const labels = changed.map((f) => f.dataset.label);
      const list =
        labels.length === 1
          ? labels[0]
          : labels.slice(0, -1).join(', ') + ' and ' + labels[labels.length - 1];
      textEl.textContent = '';
      textEl.append(
        'You changed the ' + list + '. Agents polling for ',
        code(typeField.dataset.original + ' / ' + nameField.dataset.original),
        ' with the current key will stop receiving this configuration. Update them first.'
      );
      renameWarn.hidden = false;
    };

    watched.forEach((f) => {
      f.addEventListener('input', update);
      f.addEventListener('change', update);
    });
    update();
  }

  /* ---------- Config editor ---------- */

  const editor = $('[data-editor]');
  if (editor) {
    const ta = $('textarea', editor);
    const gutter = $('[data-gutter]', editor);
    const statLines = $('[data-stat-lines]', editor);
    const statBytes = $('[data-stat-bytes]', editor);
    const status = $('[data-verify-status]', editor);
    const segButtons = $$('[data-verify]', editor);
    const encoder = window.TextEncoder ? new TextEncoder() : null;
    let lastLineCount = 0;

    const countLines = (s) => {
      let n = 1;
      for (let i = s.indexOf('\n'); i !== -1; i = s.indexOf('\n', i + 1)) n++;
      return n;
    };

    const fmtBytes = (b) => {
      if (b < 1024) return b.toLocaleString() + (b === 1 ? ' byte' : ' bytes');
      if (b < 1048576) return (b / 1024).toFixed(1) + ' KB';
      return (b / 1048576).toFixed(1) + ' MB';
    };

    const refreshGutter = () => {
      const lines = countLines(ta.value);
      if (lines !== lastLineCount) {
        const out = new Array(lines);
        for (let i = 0; i < lines; i++) out[i] = i + 1;
        gutter.textContent = out.join('\n');
        lastLineCount = lines;
      }
      gutter.scrollTop = ta.scrollTop;
      statLines.textContent = lines.toLocaleString() + (lines === 1 ? ' line' : ' lines');
    };

    const refreshBytes = debounce(() => {
      const v = ta.value;
      statBytes.textContent = fmtBytes(encoder ? encoder.encode(v).length : v.length);
    }, 200);

    const setStatus = (kind, text) => {
      status.className = 'verify-status' + (kind ? ' is-' + kind : '');
      status.textContent = text;
    };

    ta.addEventListener('input', () => {
      refreshGutter();
      refreshBytes();
      // A previous verify result no longer describes the text.
      if (status.textContent) setStatus('', '');
    });
    ta.addEventListener('scroll', () => {
      gutter.scrollTop = ta.scrollTop;
    });
    refreshGutter();
    refreshBytes();

    // Replace a range while keeping the browser's undo history when possible.
    const replaceRange = (start, end, text) => {
      ta.focus();
      ta.setSelectionRange(start, end);
      let ok = false;
      try {
        ok = document.execCommand('insertText', false, text);
      } catch (e) {
        ok = false;
      }
      if (!ok) {
        ta.setRangeText(text, start, end, 'end');
        ta.dispatchEvent(new Event('input', { bubbles: true }));
      }
    };

    const INDENT = '  ';

    const indent = (outdent) => {
      const v = ta.value;
      const s = ta.selectionStart;
      const en = ta.selectionEnd;

      if (!outdent && s === en) {
        replaceRange(s, en, INDENT);
        return;
      }

      const blockStart = v.lastIndexOf('\n', s - 1) + 1;
      let lastPos = en;
      if (en > s && v[en - 1] === '\n') lastPos = en - 1;
      let blockEnd = v.indexOf('\n', lastPos);
      if (blockEnd === -1) blockEnd = v.length;

      const block = v.slice(blockStart, blockEnd);
      let firstDelta = 0;
      const lines = block.split('\n').map((line, i) => {
        if (!outdent) {
          if (i === 0) firstDelta = INDENT.length;
          return INDENT + line;
        }
        const m = line.match(/^( {1,2}|\t)/);
        const cut = m ? m[0].length : 0;
        if (i === 0) firstDelta = -cut;
        return line.slice(cut);
      });
      const next = lines.join('\n');
      if (next === block) return;

      replaceRange(blockStart, blockEnd, next);
      const newStart = Math.max(blockStart, s + firstDelta);
      const newEnd = Math.max(newStart, en + (next.length - block.length));
      ta.setSelectionRange(newStart, s === en ? newStart : newEnd);
    };

    // Tab indents. Press Esc first to let Tab move focus out (keyboard users).
    let escaped = false;
    ta.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') {
        escaped = true;
        return;
      }
      if (e.key !== 'Tab' || e.ctrlKey || e.metaKey || e.altKey) {
        escaped = false;
        return;
      }
      if (escaped) {
        escaped = false;
        return;
      }
      e.preventDefault();
      indent(e.shiftKey);
    });

    segButtons.forEach((btn) => {
      btn.addEventListener('click', () => {
        const fmt = btn.dataset.verify;
        const name = fmt.toUpperCase();
        segButtons.forEach((b) => b.setAttribute('aria-pressed', b === btn ? 'true' : 'false'));
        setStatus('pending', 'Checking ' + name + '…');

        const send = () => {
          const body = new FormData();
          body.append('format', fmt);
          body.append('data', ta.value);
          return fetch('/validate', {
            method: 'POST',
            body: body,
            credentials: 'same-origin',
            headers: { Accept: 'application/json', 'X-CSRF-Token': getToken() },
          });
        };

        send()
          .then((r) => {
            // Stale token (session changed): refresh it and retry once.
            if (r.status !== 403) return r;
            return checkSession().then((state) => (state === 'ok' ? send() : r));
          })
          .then((r) => {
            if (r.status === 401 || r.redirected) {
              setStatus('err', 'Your session has expired. Sign in again in a new tab, then retry.');
              return null;
            }
            return r.json();
          })
          .then((res) => {
            if (!res) return;
            if (res.valid) setStatus('ok', 'Valid ' + name);
            else setStatus('err', name + ': ' + res.error);
          })
          .catch((err) => setStatus('err', 'Request failed: ' + err.message));
      });
    });
  }

  /* ---------- Login ---------- */

  const login = $('form[data-login]');
  if (login) {
    login.addEventListener('submit', () => {
      const btn = $('button[type="submit"]', login);
      // Defer so the button's value is still submitted with the form.
      setTimeout(() => {
        btn.disabled = true;
        btn.textContent = 'Signing in…';
      }, 0);
    });
  }
})();
