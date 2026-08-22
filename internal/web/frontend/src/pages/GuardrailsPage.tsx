import { useState, useEffect, useCallback, type FormEvent } from 'react'
import { PlusIcon, Trash2Icon } from 'lucide-react'

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
import {
  listGuardrails,
  createGuardrail,
  deleteGuardrail,
  listGuardrailEvents,
  deleteGuardrailEvent,
  type Guardrail,
  type GuardrailEvent,
} from '@/api/client'

function ModeBadge({ mode }: { mode: string }) {
  return <Badge variant={mode === 'reject' ? 'default' : 'outline'}>{mode}</Badge>
}

export default function GuardrailsPage() {
  const [guardrails, setGuardrails] = useState<Guardrail[]>([])
  const [events, setEvents] = useState<GuardrailEvent[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [flash, setFlash] = useState('')
  const [modalOpen, setModalOpen] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<Guardrail | null>(null)

  const [pattern, setPattern] = useState('')
  const [mode, setMode] = useState('reject')
  const [replaceBy, setReplaceBy] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const [reloadTick, setReloadTick] = useState(0)
  const reload = useCallback(() => setReloadTick((t) => t + 1), [])

  useEffect(() => {
    let stale = false
    ;(async () => {
      try {
        const [g, e] = await Promise.all([
          listGuardrails(),
          listGuardrailEvents({ limit: '50' }),
        ])
        if (stale) return
        setGuardrails(g || [])
        setEvents(e.records || [])
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
    if (!pattern || !mode) {
      setError('Pattern and Mode are required.')
      return
    }
    setSubmitting(true)
    setError('')
    try {
      await createGuardrail({
        pattern,
        mode,
        replace_by: mode === 'replace' ? replaceBy : undefined,
      })
      setFlash('Guardrail created.')
      setPattern('')
      setMode('reject')
      setReplaceBy('')
      setModalOpen(false)
      reload()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to create guardrail')
    } finally {
      setSubmitting(false)
    }
  }

  async function handleDeleteGuardrail(id: string) {
    try {
      await deleteGuardrail(id)
      setFlash('Guardrail deleted.')
      reload()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to delete guardrail')
    }
  }

  async function handleDeleteEvent(id: string) {
    try {
      await deleteGuardrailEvent(id)
      setFlash('Event deleted.')
      reload()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to delete event')
    }
  }

  return (
    <div>
      <PageHeader
        index="04"
        title="Guardrails"
        description="Patterns screened out of every request before it reaches a provider."
        actions={
          <Button onClick={() => setModalOpen(true)}>
            <PlusIcon /> Add rule
          </Button>
        }
      />

      {flash && <DismissibleAlert variant="success" className="mb-4" onClose={() => setFlash('')}>{flash}</DismissibleAlert>}
      {error && <DismissibleAlert variant="destructive" className="mb-4" onClose={() => setError('')}>{error}</DismissibleAlert>}

      {/* Guardrail rules */}
      <p className="kicker mb-2 text-muted-foreground">Rules</p>
      <Card className="mb-8 gap-0 py-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>ID</TableHead>
              <TableHead>Pattern</TableHead>
              <TableHead>Mode</TableHead>
              <TableHead>Replacement</TableHead>
              <TableHead>Created</TableHead>
              <TableHead className="w-12" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              <TableRow>
                <TableCell colSpan={6} className="py-6 text-center font-mono text-xs text-muted-foreground">
                  loading…
                </TableCell>
              </TableRow>
            ) : guardrails.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} className="py-6 text-center font-mono text-xs text-muted-foreground">
                  no guardrail rules
                </TableCell>
              </TableRow>
            ) : (
              guardrails.map((g) => (
                <TableRow key={g.id}>
                  <TableCell className="font-mono text-xs text-muted-foreground">{g.id.slice(0, 8)}</TableCell>
                  <TableCell className="font-mono text-xs">{g.pattern}</TableCell>
                  <TableCell><ModeBadge mode={g.mode} /></TableCell>
                  <TableCell className="font-mono text-xs">{g.replace_by || '—'}</TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">{formatDateTime(g.created_at)}</TableCell>
                  <TableCell className="py-1 text-right">
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      aria-label="Delete guardrail rule"
                      className="text-muted-foreground hover:text-destructive"
                      onClick={() => setDeleteTarget(g)}
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

      {/* Guardrail events */}
      <p className="kicker mb-2 text-muted-foreground">Recent events</p>
      <Card className="gap-0 py-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>ID</TableHead>
              <TableHead>Pattern</TableHead>
              <TableHead>Mode</TableHead>
              <TableHead>API key</TableHead>
              <TableHead>Input text</TableHead>
              <TableHead>Time</TableHead>
              <TableHead className="w-12" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {events.length === 0 ? (
              <TableRow>
                <TableCell colSpan={7} className="py-6 text-center font-mono text-xs text-muted-foreground">
                  no events recorded
                </TableCell>
              </TableRow>
            ) : (
              events.map((ev) => (
                <TableRow key={ev.id}>
                  <TableCell className="font-mono text-xs text-muted-foreground">{ev.id.slice(0, 8)}</TableCell>
                  <TableCell className="font-mono text-xs">{ev.pattern}</TableCell>
                  <TableCell><ModeBadge mode={ev.mode} /></TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">{ev.api_key_id.slice(0, 8)}</TableCell>
                  <TableCell className="max-w-xs truncate whitespace-nowrap text-xs" title={ev.input_text}>
                    {ev.input_text}
                  </TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">{formatDateTime(ev.created_at)}</TableCell>
                  <TableCell className="py-1 text-right">
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      aria-label="Delete event"
                      className="text-muted-foreground hover:text-destructive"
                      onClick={() => handleDeleteEvent(ev.id)}
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

      {/* Create rule */}
      <Dialog open={modalOpen} onOpenChange={setModalOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Add guardrail rule</DialogTitle>
            <DialogDescription>
              Requests matching the pattern are rejected, or have the match replaced.
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={handleCreate} className="grid gap-4">
            <div className="grid gap-2">
              <Label htmlFor="rule-pattern">Pattern (regex)</Label>
              <Input
                id="rule-pattern"
                className="font-mono text-xs"
                value={pattern}
                onChange={(e) => setPattern(e.target.value)}
                placeholder="e.g. \bpassword\b"
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="rule-mode">Mode</Label>
              <Select value={mode} onValueChange={(v) => setMode(v || 'reject')}>
                <SelectTrigger id="rule-mode" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="reject">Reject</SelectItem>
                  <SelectItem value="replace">Replace</SelectItem>
                </SelectContent>
              </Select>
            </div>
            {mode === 'replace' && (
              <div className="grid gap-2">
                <Label htmlFor="rule-replace">Replacement text</Label>
                <Input
                  id="rule-replace"
                  value={replaceBy}
                  onChange={(e) => setReplaceBy(e.target.value)}
                  placeholder="e.g. [REDACTED]"
                />
              </div>
            )}
            <DialogFooter>
              <Button type="submit" disabled={submitting}>
                {submitting ? 'Creating…' : 'Create rule'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => { if (!open) setDeleteTarget(null) }}
        title="Delete guardrail rule"
        description={`Delete the rule matching “${deleteTarget?.pattern}”?`}
        confirmLabel="Delete"
        onConfirm={() => { if (deleteTarget) handleDeleteGuardrail(deleteTarget.id) }}
      />
    </div>
  )
}
