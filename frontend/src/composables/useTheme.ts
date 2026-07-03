import { readonly, ref } from 'vue'

// Module-singleton theme state so every consumer (sidebar, landing page,
// key-usage page) shares one source of truth for the `dark` class on <html>.
// localStorage key 'theme' ('dark' | 'light') is kept for back-compat.
const isDark = ref(
  typeof document !== 'undefined' && document.documentElement.classList.contains('dark')
)

function applyDark(value: boolean) {
  isDark.value = value
  document.documentElement.classList.toggle('dark', value)
}

/**
 * Resolve the persisted (or OS-preferred) theme and apply it to <html>.
 * Called once in main.ts before mount to avoid a light-theme flash.
 */
export function initTheme() {
  let savedTheme: string | null = null
  try {
    savedTheme = localStorage.getItem('theme')
  } catch {
    // localStorage unavailable (privacy mode) — fall back to OS preference
  }
  const shouldUseDark =
    savedTheme === 'dark' ||
    (!savedTheme && typeof window !== 'undefined' && !!window.matchMedia?.('(prefers-color-scheme: dark)').matches)
  applyDark(shouldUseDark)
}

export function useTheme() {
  function setDark(value: boolean) {
    applyDark(value)
    try {
      localStorage.setItem('theme', value ? 'dark' : 'light')
    } catch {
      // persisting is best-effort
    }
  }

  function toggleTheme() {
    setDark(!isDark.value)
  }

  return {
    isDark: readonly(isDark),
    toggleTheme,
    setDark
  }
}
