import { Outlet, NavLink, useNavigate } from 'react-router-dom'
import {
  ServerIcon,
  KeyRoundIcon,
  ChartNoAxesColumnIcon,
  ShieldIcon,
  TerminalIcon,
  LogOutIcon,
  MoonIcon,
  SunIcon,
} from 'lucide-react'

import Lockup from '@/components/brand/Lockup'
import Monogram from '@/components/brand/Monogram'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { useTheme } from '@/lib/theme'
import { useAuth } from '@/context/AuthContext'
import { cn } from '@/lib/utils'

const NAV_ITEMS = [
  { path: 'providers', index: '01', label: 'Providers', icon: ServerIcon },
  { path: 'keys', index: '02', label: 'API keys', icon: KeyRoundIcon },
  { path: 'usage', index: '03', label: 'Usage', icon: ChartNoAxesColumnIcon },
  { path: 'guardrails', index: '04', label: 'Guardrails', icon: ShieldIcon },
  { path: 'playground', index: '05', label: 'Playground', icon: TerminalIcon },
]

function ThemeToggle() {
  const { resolvedTheme, setTheme } = useTheme()
  return (
    <Button
      variant="ghost"
      size="icon-sm"
      aria-label={resolvedTheme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
      onClick={() => setTheme(resolvedTheme === 'dark' ? 'light' : 'dark')}
    >
      {resolvedTheme === 'dark' ? <SunIcon /> : <MoonIcon />}
    </Button>
  )
}

export default function Layout() {
  const navigate = useNavigate()
  const { logout } = useAuth()

  return (
    <div className="flex min-h-svh">
      <aside className="flex w-60 shrink-0 flex-col border-r">
        <div className="flex h-14 items-center justify-between gap-2 px-4">
          <Lockup className="text-lg" />
          <ThemeToggle />
        </div>
        <Separator />

        <nav className="flex-1 py-3">
          <p className="kicker px-4 pb-2 text-muted-foreground/80">Console</p>
          <ul className="grid gap-px">
            {NAV_ITEMS.map((item) => (
              <li key={item.path}>
                <NavLink
                  to={item.path}
                  className={({ isActive }) =>
                    cn(
                      'flex items-center gap-3 border-l-2 border-transparent px-4 py-2 text-sm text-muted-foreground transition-colors',
                      'hover:bg-accent hover:text-accent-foreground',
                      isActive && 'border-brass bg-accent text-foreground',
                    )
                  }
                >
                  <span className="font-mono text-xs text-muted-foreground/70">{item.index}</span>
                  <item.icon className="size-4" aria-hidden />
                  <span>{item.label}</span>
                </NavLink>
              </li>
            ))}
          </ul>
        </nav>

        <Separator />
        <div className="py-3">
          <button
            type="button"
            onClick={() => {
              logout()
              navigate('/login')
            }}
            className="flex w-full items-center gap-3 border-l-2 border-transparent px-4 py-2 text-sm text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
          >
            <span className="font-mono text-xs text-muted-foreground/70">·</span>
            <LogOutIcon className="size-4" aria-hidden />
            <span>Log out</span>
          </button>
        </div>

        {/* The monogram signs the footer — it certifies; it does not shout. */}
        <div className="flex items-center gap-2 border-t px-4 py-3 font-mono text-xs text-muted-foreground">
          <span className="flex-1 truncate">lab34 — llm-proxy</span>
          <Monogram size={20} variant="outline" />
        </div>
      </aside>

      <main className="min-w-0 flex-1 overflow-auto">
        <div className="mx-auto max-w-6xl p-6 lg:p-8">
          <Outlet />
        </div>
      </main>
    </div>
  )
}
