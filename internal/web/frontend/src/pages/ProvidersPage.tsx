import { useState, useEffect, useCallback, type FormEvent } from 'react'
import { PlusIcon, Trash2Icon } from 'lucide-react'

import PageHeader from '@/components/PageHeader'
import DismissibleAlert from '@/components/DismissibleAlert'
import ConfirmDialog from '@/components/ConfirmDialog'
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
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { formatDateTime } from '@/lib/format'
import { listProviders, createProvider, deleteProvider, type Provider } from '@/api/client'

export default function ProvidersPage() {
  const [providers, setProviders] = useState<Provider[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [flash, setFlash] = useState('')
  const [modalOpen, setModalOpen] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<Provider | null>(null)

  const [name, setName] = useState('')
  const [baseUrl, setBaseUrl] = useState('')
  const [apiKey, setApiKey] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const [reloadTick, setReloadTick] = useState(0)
  const reload = useCallback(() => setReloadTick((t) => t + 1), [])

  useEffect(() => {
    let stale = false
    ;(async () => {
      try {
        const data = await listProviders()
        if (stale) return
        setProviders(data || [])
      } catch (err: unknown) {
        if (!stale) setError(err instanceof Error ? err.message : 'Failed to load providers')
      } finally {
        if (!stale) setLoading(false)
      }
    })()
    return () => { stale = true }
  }, [reloadTick])

  async function handleCreate(e: FormEvent) {
    e.preventDefault()
    if (!name || !baseUrl || !apiKey) {
      setError('All fields are required.')
      return
    }
    setSubmitting(true)
    setError('')
    try {
      await createProvider({ name, base_url: baseUrl, api_key: apiKey })
      setFlash('Provider created successfully.')
      setName('')
      setBaseUrl('')
      setApiKey('')
      setModalOpen(false)
      reload()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to create provider')
    } finally {
      setSubmitting(false)
    }
  }

  async function handleDelete(id: string) {
    try {
      await deleteProvider(id)
      setFlash('Provider deleted.')
      reload()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to delete provider')
    }
  }

  return (
    <div>
      <PageHeader
        index="01"
        title="Providers"
        description="The upstream LLM backends that requests are forwarded to."
        actions={
          <Button onClick={() => setModalOpen(true)}>
            <PlusIcon /> Add provider
          </Button>
        }
      />

      {flash && <DismissibleAlert variant="success" className="mb-4" onClose={() => setFlash('')}>{flash}</DismissibleAlert>}
      {error && <DismissibleAlert variant="destructive" className="mb-4" onClose={() => setError('')}>{error}</DismissibleAlert>}

      <Card className="gap-0 py-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>ID</TableHead>
              <TableHead>Name</TableHead>
              <TableHead>Base URL</TableHead>
              <TableHead>Created</TableHead>
              <TableHead className="w-12" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              <TableRow>
                <TableCell colSpan={5} className="py-6 text-center font-mono text-xs text-muted-foreground">
                  loading…
                </TableCell>
              </TableRow>
            ) : providers.length === 0 ? (
              <TableRow>
                <TableCell colSpan={5} className="py-6 text-center font-mono text-xs text-muted-foreground">
                  no providers configured
                </TableCell>
              </TableRow>
            ) : (
              providers.map((p) => (
                <TableRow key={p.id}>
                  <TableCell className="font-mono text-xs text-muted-foreground">{p.id.slice(0, 8)}</TableCell>
                  <TableCell>{p.name}</TableCell>
                  <TableCell className="font-mono text-xs">{p.base_url}</TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">{formatDateTime(p.created_at)}</TableCell>
                  <TableCell className="py-1 text-right">
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      aria-label={`Delete provider ${p.name}`}
                      className="text-muted-foreground hover:text-destructive"
                      onClick={() => setDeleteTarget(p)}
                    >
                      <Trash2Icon />
                    </Button>
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
            <DialogTitle>Add provider</DialogTitle>
            <DialogDescription>Register an upstream backend and the key used to reach it.</DialogDescription>
          </DialogHeader>
          <form onSubmit={handleCreate} className="grid gap-4">
            <div className="grid gap-2">
              <Label htmlFor="provider-name">Name</Label>
              <Input id="provider-name" value={name} onChange={(e) => setName(e.target.value)} placeholder="e.g. OpenAI" />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="provider-url">Base URL</Label>
              <Input
                id="provider-url"
                className="font-mono text-xs"
                value={baseUrl}
                onChange={(e) => setBaseUrl(e.target.value)}
                placeholder="https://api.openai.com"
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="provider-key">API key</Label>
              <Input
                id="provider-key"
                type="password"
                value={apiKey}
                onChange={(e) => setApiKey(e.target.value)}
                placeholder="sk-…"
              />
            </div>
            <DialogFooter>
              <Button type="submit" disabled={submitting}>
                {submitting ? 'Creating…' : 'Create provider'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => { if (!open) setDeleteTarget(null) }}
        title="Delete provider"
        description={`Delete “${deleteTarget?.name}”? All associated API keys will also be deleted.`}
        confirmLabel="Delete"
        onConfirm={() => { if (deleteTarget) handleDelete(deleteTarget.id) }}
      />
    </div>
  )
}
