import { ref } from 'vue'

const KEY = 'unigate-theme'
type Mode = 'light' | 'dark'
const stored = localStorage.getItem(KEY) as Mode | null
const systemDark = typeof window.matchMedia === 'function' ? window.matchMedia('(prefers-color-scheme: dark)') : null

export const theme = ref<Mode>(stored || (systemDark?.matches ? 'dark' : 'light'))
const explicit = ref(!!stored)

function apply() {
  document.documentElement.dataset.theme = theme.value
}

export function toggleTheme() {
  theme.value = theme.value === 'dark' ? 'light' : 'dark'
  localStorage.setItem(KEY, theme.value)
  explicit.value = true
  apply()
}

export function initTheme() {
  apply()
  systemDark?.addEventListener('change', (e) => {
    if (!explicit.value) {
      theme.value = e.matches ? 'dark' : 'light'
      apply()
    }
  })
}
