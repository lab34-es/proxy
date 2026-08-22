import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'

import Lockup from '@/components/brand/Lockup'
import Monogram from '@/components/brand/Monogram'
import DismissibleAlert from '@/components/DismissibleAlert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useAuth } from '@/context/AuthContext'

export default function LoginPage() {
  const [token, setToken] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const { login } = useAuth()
  const navigate = useNavigate()

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (!token.trim()) {
      setError('Admin token is required.')
      return
    }

    setLoading(true)
    setError('')

    try {
      // Validate token by making a test API call.
      const resp = await fetch('/admin/providers', {
        headers: { Authorization: `Bearer ${token.trim()}` },
      })
      if (resp.status === 401) {
        setError('Invalid admin token.')
        setLoading(false)
        return
      }
      if (!resp.ok) {
        setError('Failed to verify token.')
        setLoading(false)
        return
      }
      login(token.trim())
      navigate('/', { replace: true })
    } catch {
      setError('Failed to connect to the server.')
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-svh flex-col items-center justify-center gap-8 p-6">
      <div className="flex flex-col items-center gap-5">
        <Monogram size={64} />
        <Lockup className="text-3xl" />
      </div>

      <Card className="w-full max-w-sm">
        <CardHeader>
          <CardTitle>Console</CardTitle>
          <CardDescription>Enter your admin token to open the dashboard.</CardDescription>
        </CardHeader>
        <CardContent>
          {error && (
            <DismissibleAlert variant="destructive" className="mb-4" onClose={() => setError('')}>
              {error}
            </DismissibleAlert>
          )}
          <form onSubmit={handleSubmit} className="grid gap-4">
            <div className="grid gap-2">
              <Label htmlFor="admin-token">Admin token</Label>
              <Input
                id="admin-token"
                type="password"
                placeholder="••••••••"
                value={token}
                onChange={(e) => setToken(e.target.value)}
                autoFocus
              />
            </div>
            <Button type="submit" className="w-full" disabled={loading}>
              {loading ? 'Verifying…' : 'Sign in'}
            </Button>
          </form>
        </CardContent>
      </Card>

      <p className="kicker text-muted-foreground">AI tooling &amp; training for IT operations</p>
    </div>
  )
}
