import {
  createContext,
  useContext,
  useEffect,
  useState,
  useSyncExternalStore,
  type ReactNode,
} from 'react'

type Theme = 'light' | 'dark' | 'system'

interface ThemeContextType {
  theme: Theme
  resolvedTheme: 'light' | 'dark'
  setTheme: (theme: Theme) => void
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined)

const STORAGE_KEY = 'llmp_theme'

const query = '(prefers-color-scheme: dark)'

function subscribeSystem(callback: () => void) {
  const mq = window.matchMedia(query)
  mq.addEventListener('change', callback)
  return () => mq.removeEventListener('change', callback)
}

function systemPrefersDark(): boolean {
  return window.matchMedia(query).matches
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setThemeState] = useState<Theme>(() => {
    const stored = localStorage.getItem(STORAGE_KEY)
    return stored === 'light' || stored === 'dark' ? stored : 'system'
  })
  const systemDark = useSyncExternalStore(subscribeSystem, systemPrefersDark)

  const resolvedTheme: 'light' | 'dark' =
    theme === 'system' ? (systemDark ? 'dark' : 'light') : theme

  useEffect(() => {
    document.documentElement.classList.toggle('dark', resolvedTheme === 'dark')
  }, [resolvedTheme])

  const setTheme = (next: Theme) => {
    if (next === 'system') {
      localStorage.removeItem(STORAGE_KEY)
    } else {
      localStorage.setItem(STORAGE_KEY, next)
    }
    setThemeState(next)
  }

  return (
    <ThemeContext.Provider value={{ theme, resolvedTheme, setTheme }}>
      {children}
    </ThemeContext.Provider>
  )
}

export function useTheme(): ThemeContextType {
  const ctx = useContext(ThemeContext)
  if (!ctx) throw new Error('useTheme must be used within ThemeProvider')
  return ctx
}
