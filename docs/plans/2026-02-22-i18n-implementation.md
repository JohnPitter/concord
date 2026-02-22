# i18n Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add internationalization (i18n) supporting English, Portuguese, Spanish, Chinese, and Japanese across the entire Concord frontend.

**Architecture:** Custom lightweight i18n using Svelte 5 runes — a reactive `locale` store, typed JSON translation files per language, and a `t(key)` function. No external library needed since this is a Wails desktop app (no SSR). Language preference persisted in localStorage, auto-detected from browser on first use. The Go backend error messages stay in English (they are developer-facing API errors, not shown to users directly).

**Tech Stack:** Svelte 5 runes ($state, $derived), TypeScript, JSON locale files

**Languages:** `pt` (Portuguese - default), `en` (English), `es` (Spanish), `zh` (Chinese Simplified), `ja` (Japanese)

---

## Task 1: Create i18n Core Infrastructure

**Files:**
- Create: `frontend/src/lib/i18n/index.ts`
- Create: `frontend/src/lib/i18n/types.ts`
- Create: `frontend/src/lib/i18n/locales/pt.json`

**What to build:**

`types.ts` — Flat key-value type for all translation keys. Keys use dot notation by component group:
```
auth.welcome, auth.subtitle, auth.enterCode, auth.openGithub, ...
mode.welcome, mode.howConnect, mode.p2p, mode.server, ...
nav.directMessages, nav.addServer, ...
chat.searchPlaceholder, chat.search, chat.clear, chat.noResults, ...
server.create, server.join, server.delete, server.invite, ...
voice.connected, voice.disconnect, voice.mute, voice.unmute, ...
settings.title, settings.account, settings.audio, settings.appearance, ...
common.cancel, common.close, common.ok, common.loading, common.copy, common.copied, ...
```

`index.ts` — Core i18n module:
```typescript
import { writable, derived } from 'svelte/store'; // svelte/store still works in Svelte 5

export type Locale = 'pt' | 'en' | 'es' | 'zh' | 'ja';

const STORAGE_KEY = 'concord-locale';
const DEFAULT_LOCALE: Locale = 'pt';

// Detect initial locale from localStorage or browser
function detectLocale(): Locale {
  const stored = localStorage.getItem(STORAGE_KEY);
  if (stored && isValidLocale(stored)) return stored as Locale;
  const browser = navigator.language.split('-')[0];
  if (isValidLocale(browser)) return browser as Locale;
  return DEFAULT_LOCALE;
}

function isValidLocale(l: string): l is Locale {
  return ['pt', 'en', 'es', 'zh', 'ja'].includes(l);
}

// Locale store
export const locale = writable<Locale>(detectLocale());

// Cache loaded translations
const cache = new Map<Locale, Record<string, string>>();

// Current translations (reactive)
export const translations = derived(locale, ($locale, set) => {
  loadLocale($locale).then(set);
});

async function loadLocale(l: Locale): Promise<Record<string, string>> {
  if (cache.has(l)) return cache.get(l)!;
  const mod = await import(`./locales/${l}.json`);
  const data = mod.default as Record<string, string>;
  cache.set(l, data);
  return data;
}

// Set locale + persist
export function setLocale(l: Locale) {
  localStorage.setItem(STORAGE_KEY, l);
  locale.set(l);
}

// Translation function — use with $translations
// Usage: t($translations, 'auth.welcome')
// With params: t($translations, 'chat.results', { count: '3', query: 'hello' })
export function t(
  trans: Record<string, string>,
  key: string,
  params?: Record<string, string>
): string {
  let val = trans[key] ?? key;
  if (params) {
    for (const [k, v] of Object.entries(params)) {
      val = val.replaceAll(`{${k}}`, v);
    }
  }
  return val;
}

// Available locales for settings UI
export const LOCALES: { code: Locale; name: string; flag: string }[] = [
  { code: 'pt', name: 'Português', flag: '🇧🇷' },
  { code: 'en', name: 'English', flag: '🇺🇸' },
  { code: 'es', name: 'Español', flag: '🇪🇸' },
  { code: 'zh', name: '中文', flag: '🇨🇳' },
  { code: 'ja', name: '日本語', flag: '🇯🇵' },
];
```

`pt.json` — Portuguese translations (the current language used in most components). Extract ALL hardcoded strings from every component. This is the base/reference locale. Use flat dot-notation keys.

**Commit:** `feat(i18n): core infrastructure — locale store, t() function, pt.json base`

---

## Task 2: Create Translation Files (en, es, zh, ja)

**Files:**
- Create: `frontend/src/lib/i18n/locales/en.json`
- Create: `frontend/src/lib/i18n/locales/es.json`
- Create: `frontend/src/lib/i18n/locales/zh.json`
- Create: `frontend/src/lib/i18n/locales/ja.json`

**What to build:**

Translate ALL keys from `pt.json` into each language. Every file must have the exact same keys as `pt.json`. Use accurate, natural translations (not Google Translate quality).

**Commit:** `feat(i18n): add en, es, zh, ja translation files`

---

## Task 3: Migrate Auth Components (Login, ModeSelector, P2PProfile)

**Files:**
- Modify: `frontend/src/lib/components/auth/Login.svelte`
- Modify: `frontend/src/lib/components/auth/ModeSelector.svelte`
- Modify: `frontend/src/lib/components/auth/P2PProfile.svelte`

**What to do:**

In each component:
1. Add import: `import { translations, t } from '$lib/i18n';`
2. Subscribe to translations: `const trans = $derived($translations);` (inside `<script>`)
3. Replace every hardcoded string with `t(trans, 'key.name')`
4. For strings with dynamic values, use params: `t(trans, 'key', { name: value })`

Example — Login.svelte change:
```svelte
<!-- Before -->
<h1>Welcome to Concord</h1>

<!-- After -->
<h1>{t(trans, 'auth.welcome')}</h1>
```

**Commit:** `feat(i18n): migrate auth components — Login, ModeSelector, P2PProfile`

---

## Task 4: Migrate Layout Components (ServerSidebar, ChannelSidebar, MainContent, DMSidebar)

**Files:**
- Modify: `frontend/src/lib/components/layout/ServerSidebar.svelte`
- Modify: `frontend/src/lib/components/layout/ChannelSidebar.svelte`
- Modify: `frontend/src/lib/components/layout/MainContent.svelte`
- Modify: `frontend/src/lib/components/layout/DMSidebar.svelte`

**What to do:** Same pattern as Task 3. Import i18n, replace all hardcoded strings.

**Commit:** `feat(i18n): migrate layout components — sidebars, main content`

---

## Task 5: Migrate Layout Components Part 2 (FriendsList, ActiveNow, NoServers, MemberSidebar)

**Files:**
- Modify: `frontend/src/lib/components/layout/FriendsList.svelte`
- Modify: `frontend/src/lib/components/layout/ActiveNow.svelte`
- Modify: `frontend/src/lib/components/layout/NoServers.svelte`
- Modify: `frontend/src/lib/components/layout/MemberSidebar.svelte`

**What to do:** Same pattern. FriendsList has the most strings (~30+), be thorough.

**Commit:** `feat(i18n): migrate remaining layout components — friends, active, noservers`

---

## Task 6: Migrate Chat Components (MessageInput, MessageList, MessageBubble, FileAttachment)

**Files:**
- Modify: `frontend/src/lib/components/chat/MessageInput.svelte`
- Modify: `frontend/src/lib/components/chat/MessageList.svelte`
- Modify: `frontend/src/lib/components/chat/MessageBubble.svelte`
- Modify: `frontend/src/lib/components/chat/FileAttachment.svelte`

**What to do:** Same pattern. Watch for emoji category labels in MessageInput.

**Commit:** `feat(i18n): migrate chat components — input, list, bubble, attachment`

---

## Task 7: Migrate Server + Voice + Settings + P2P Components

**Files:**
- Modify: `frontend/src/lib/components/server/CreateServer.svelte`
- Modify: `frontend/src/lib/components/server/JoinServer.svelte`
- Modify: `frontend/src/lib/components/server/ServerInfoModal.svelte`
- Modify: `frontend/src/lib/components/voice/VoiceControls.svelte`
- Modify: `frontend/src/lib/components/voice/TranslationToggle.svelte`
- Modify: `frontend/src/lib/components/settings/SettingsPanel.svelte`
- Modify: `frontend/src/lib/components/p2p/P2PApp.svelte`
- Modify: `frontend/src/lib/components/p2p/P2PChatArea.svelte`
- Modify: `frontend/src/lib/components/p2p/P2PPeerSidebar.svelte`

**What to do:** Same pattern. SettingsPanel should add a language selector using `LOCALES` and `setLocale()`.

**Commit:** `feat(i18n): migrate server, voice, settings, p2p components`

---

## Task 8: Migrate App.svelte + Final Verification

**Files:**
- Modify: `frontend/src/App.svelte`

**What to do:**
1. Import and wire up i18n in the root component
2. Replace hardcoded strings in App.svelte
3. Ensure the `$translations` store resolves before first render (loading state)
4. Run `npx svelte-check --threshold error` to verify no TypeScript errors
5. Verify Wails dev compiles

**Commit:** `feat(i18n): migrate App.svelte + full i18n complete for 5 languages`

---

## Parallelization Strategy (for Teams)

Tasks can be parallelized as follows:
- **Task 1** (core) — must complete first, all others depend on it
- **Task 2** (translations) — must complete second (provides all locale files)
- **Tasks 3-7** (component migrations) — can ALL run in parallel after Task 2
- **Task 8** (App.svelte + verification) — runs last

Optimal team: 1 lead + 3 workers
- Lead: Task 1 → Task 2 → Task 8
- Worker A: Task 3 (auth) + Task 6 (chat)
- Worker B: Task 4 (layout pt1) + Task 7 (server/voice/settings/p2p)
- Worker C: Task 5 (layout pt2)
