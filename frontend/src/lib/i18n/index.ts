import { writable, derived, type Readable } from 'svelte/store'

export type Locale = 'pt' | 'en' | 'es' | 'zh' | 'ja'

const STORAGE_KEY = 'concord-locale'
const DEFAULT_LOCALE: Locale = 'pt'
const VALID_LOCALES: Locale[] = ['pt', 'en', 'es', 'zh', 'ja']

function isValidLocale(l: string): l is Locale {
  return VALID_LOCALES.includes(l as Locale)
}

function detectLocale(): Locale {
  const stored = localStorage.getItem(STORAGE_KEY)
  if (stored && isValidLocale(stored)) return stored
  const browser = navigator.language.split('-')[0]
  if (isValidLocale(browser)) return browser
  return DEFAULT_LOCALE
}

export const locale = writable<Locale>(detectLocale())

const cache = new Map<Locale, Record<string, string>>()

async function loadLocale(l: Locale): Promise<Record<string, string>> {
  if (cache.has(l)) return cache.get(l)!
  const mod = await import(`./locales/${l}.json`)
  const data = mod.default as Record<string, string>
  cache.set(l, data)
  return data
}

export const translations: Readable<Record<string, string>> = derived(
  locale,
  ($locale: Locale, set: (value: Record<string, string>) => void) => {
    loadLocale($locale).then(set)
  },
  {} as Record<string, string>,
)

export function setLocale(l: Locale) {
  localStorage.setItem(STORAGE_KEY, l)
  locale.set(l)
}

export function t(
  trans: Record<string, string>,
  key: string,
  params?: Record<string, string>,
): string {
  let val = trans[key] ?? key
  if (params) {
    for (const [k, v] of Object.entries(params)) {
      val = val.split(`{${k}}`).join(v)
    }
  }
  return val
}

export const LOCALES: { code: Locale; name: string; flag: string }[] = [
  { code: 'pt', name: 'Portugues', flag: '\u{1F1E7}\u{1F1F7}' },
  { code: 'en', name: 'English', flag: '\u{1F1FA}\u{1F1F8}' },
  { code: 'es', name: 'Espanol', flag: '\u{1F1EA}\u{1F1F8}' },
  { code: 'zh', name: '\u4E2D\u6587', flag: '\u{1F1E8}\u{1F1F3}' },
  { code: 'ja', name: '\u65E5\u672C\u8A9E', flag: '\u{1F1EF}\u{1F1F5}' },
]
