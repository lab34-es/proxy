import { useState, useEffect, useCallback, type FormEvent } from 'react'
import { CopyIcon, PlusIcon, Trash2Icon } from 'lucide-react'

import PageHeader from '@/components/PageHeader'
import DismissibleAlert from '@/components/DismissibleAlert'
import ConfirmDialog from '@/components/ConfirmDialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { formatDateTime } from '@/lib/format'
import { listKeys, createKey, revokeKey, listProviders, type APIKey, type Provider } from '@/api/client'

export default function KeysPage() {
  const [keys, setKeys] = useState<APIKey[]>([])
  const [providers, setProviders] = useState<Provider[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [flash, setFlash] = useState('')
  const [newKey, setNewKey] = useState('')
  const [modalOpen, setModalOpen] = useState(false)
  const [revokeTarget, setRevokeTarget] = useState<APIKey | null>(null)

  const [name, setName] = useState('')
  const [providerId, setProviderId] = useState('')
  const [rpm, setRpm] = useState('60')
  const [submitting, setSubmitting] = useState(false)

  const [reloadTick, setReloadTick] = useState(0)
  const reload = useCallback(() => setReloadTick((t) => t + 1), [])

  useEffect(() => {
    let stale = false
    ;(async () => {
      try {
        const [k, p] = await Promise.all([listKeys(), listProviders()])
        if (stale) return
        setKeys(k || [])
        setProviders(p || [])
      } catch (err: unknown) {
        if (!stale) setError(err instanceof Error ? err.message : 'Failed to load data')
      } finally {
        if (!stale) setLoading(false)
      }
    })()
    return () => { stale = true }
  }, [reloadTick])

  async function handleCreate(e: FormEvent) {
    e.preventDefault()
    if (!name || !providerId) {
      setError('Name and Provider are required.')
      return
    }
    setSubmitting(true)
    setError('')
    try {
      const result = await createKey({
        name,
        provider_id: providerId,
        rate_limit_rpm: parseInt(rpm) || 60,
      })
      setNewKey(result.key)
      setName('')
      setProviderId('')
      setRpm('60')
      setModalOpen(false)
      reload()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to create key')
    } finally {
      setSubmitting(false)
    }
  }

  async function handleRevoke(id: string) {
    try {
      await revokeKey(id)
      setFlash('Key revoked.')
      reload()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to revoke key')
    }
  }

  function providerName(id: string): string {
    return providers.find((p) => p.id === id)?.name || id.slice(0, 8)
  }

  return (
    <div>
      <PageHeader
        index="02"
        title="API keys"
        description="Keys clients use to call the proxy. Each key maps to one provider."
        actions={
          <Button onClick={() => setModalOpen(true)}>
            <PlusIcon /> Create key
          </Button>
        }
      />

      {newKey && (
        <DismissibleAlert
          title="new api key — copy it now, it will not be shown again"
          className="mb-4"
          onClose={() => setNewKey('')}
          endAction={
            <Button
              variant="ghost"
              size="icon-sm"
              aria-label="Copy key to clipboard"
              onClick={() => {
                navigator.clipboard.writeText(newKey)
                setFlash('Key copied to clipboard.')
              }}
            >
              <CopyIcon />
            </Button>
          }
        >
          <code className="font-mono text-xs break-all">{newKey}</code>
        </DismissibleAlert>
      )}

      {flash && <DismissibleAlert variant="success" className="mb-4" onClose={() => setFlash('')}>{flash}</DismissibleAlert>}
      {error && <DismissibleAlert variant="destructive" className="mb-4" onClose={() => setError('')}>{error}</DismissibleAlert>}

      <Card className="gap-0 py-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>ID</TableHead>
              <TableHead>Name</TableHead>
              <TableHead>Provider</TableHead>
              <TableHead className="text-right">RPM</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Created</TableHead>
              <TableHead className="w-12" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              <TableRow>
                <TableCell colSpan={7} className="py-6 text-center font-mono text-xs text-muted-foreground">
                  loading…
                </TableCell>
              </TableRow>
            ) : keys.length === 0 ? (
              <TableRow>
                <TableCell colSpan={7} className="py-6 text-center font-mono text-xs text-muted-foreground">
                  no api keys created
                </TableCell>
              </TableRow>
            ) : (
              keys.map((k) => (
                <TableRow key={k.id}>
                  <TableCell className="font-mono text-xs text-muted-foreground">{k.id.slice(0, 8)}</TableCell>
                  <TableCell>{k.name}</TableCell>
                  <TableCell>{providerName(k.provider_id)}</TableCell>
                  <TableCell className="text-right font-mono text-xs tabular-nums">{k.rate_limit_rpm}</TableCell>
                  <TableCell>
                    {k.revoked_at ? (
                      <Badge variant="muted">revoked</Badge>
                    ) : (
                      <Badge variant="outline">
                        <span className="size-1.5 bg-brass" aria-hidden />
                        active
                      </Badge>
                    )}
                  </TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">{formatDateTime(k.created_at)}</TableCell>
                  <TableCell className="py-1 text-right">
                    {!k.revoked_at && (
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        aria-label={`Revoke key ${k.name}`}
                        className="text-muted-foreground hover:text-destructive"
                        onClick={() => setRevokeTarget(k)}
                      >
                        <Trash2Icon />
                      </Button>
                    )}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </Card>

      <Dialog open={modalOpen} onOpenChange={setModalOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create API key</DialogTitle>
            <DialogDescription>The key value is shown once, on creation.</DialogDescription>
          </DialogHeader>
          <form onSubmit={handleCreate} className="grid gap-4">
            <div className="grid gap-2">
              <Label htmlFor="key-name">Name</Label>
              <Input id="key-name" value={name} onChange={(e) => setName(e.target.value)} placeholder="e.g. my-app-key" />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="key-provider">Provider</Label>
              <Select value={providerId} onValueChange={setProviderId}>
                <SelectTrigger id="key-provider" className="w-full">
                  <SelectValue placeholder="Select a provider" />
                </SelectTrigger>
                <SelectContent>
                  {providers.map((p) => (
                    <SelectItem key={p.id} value={p.id}>{p.name}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="grid gap-2">
              <Label htmlFor="key-rpm">Rate limit (RPM)</Label>
              <Input id="key-rpm" type="number" value={rpm} onChange={(e) => setRpm(e.target.value)} />
            </div>
            <DialogFooter>
              <Button type="submit" disabled={submitting}>
                {submitting ? 'Creating…' : 'Create key'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={revokeTarget !== null}
        onOpenChange={(open) => { if (!open) setRevokeTarget(null) }}
        title="Revoke API key"
        description={`Revoke “${revokeTarget?.name}”? It will no longer be usable.`}
        confirmLabel="Revoke"
        onConfirm={() => { if (revokeTarget) handleRevoke(revokeTarget.id) }}
      />
    </div>
  )
}
