import { useState, useEffect } from 'react'
import { ChevronLeftIcon, ChevronRightIcon } from 'lucide-react'

import PageHeader from '@/components/PageHeader'
import DismissibleAlert from '@/components/DismissibleAlert'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
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
import { formatCompact, formatDateTime } from '@/lib/format'
import { queryUsage, listKeys, listProviders, type UsageRecord, type APIKey, type Provider } from '@/api/client'

// Radix selects cannot carry an empty item value; ALL stands in for "no filter".
const ALL = 'all'

function StatTile({ label, value }: { label: string; value: number }) {
  return (
    <Card className="gap-1.5 py-4">
      <CardContent className="px-4">
        <p className="kicker text-muted-foreground">{label}</p>
        <p className="mt-1.5 font-mono text-2xl font-medium" title={value.toLocaleString()}>
          {formatCompact(value)}
        </p>
      </CardContent>
    </Card>
  )
}

export default function UsagePage() {
  const [records, setRecords] = useState<UsageRecord[]>([])
  const [total, setTotal] = useState(0)
  const [keys, setKeys] = useState<APIKey[]>([])
  const [providers, setProviders] = useState<Provider[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  // Filters
  const [filterKeyId, setFilterKeyId] = useState(ALL)
  const [filterProviderId, setFilterProviderId] = useState(ALL)
  const [filterStart, setFilterStart] = useState('')
  const [filterEnd, setFilterEnd] = useState('')
  const [page, setPage] = useState(1)
  const perPage = 50

  useEffect(() => {
    let stale = false
    ;(async () => {
      try {
        const params: Record<string, string> = {
          limit: String(perPage),
          offset: String((page - 1) * perPage),
        }
        if (filterKeyId !== ALL) params.api_key_id = filterKeyId
        if (filterProviderId !== ALL) params.provider_id = filterProviderId
        if (filterStart) params.start = new Date(filterStart).toISOString()
        if (filterEnd) params.end = new Date(filterEnd).toISOString()

        const [result, k, p] = await Promise.all([
          queryUsage(params),
          listKeys(),
          listProviders(),
        ])
        if (stale) return
        setRecords(result.records || [])
        setTotal(result.total)
        setKeys(k || [])
        setProviders(p || [])
      } catch (err: unknown) {
        if (!stale) setError(err instanceof Error ? err.message : 'Failed to load usage')
      } finally {
        if (!stale) setLoading(false)
      }
    })()
    return () => { stale = true }
  }, [page, filterKeyId, filterProviderId, filterStart, filterEnd])

  const totalPages = Math.max(1, Math.ceil(total / perPage))

  // Compute page-level stats
  const promptTotal = records.reduce((s, r) => s + r.prompt_tokens, 0)
  const completionTotal = records.reduce((s, r) => s + r.completion_tokens, 0)
  const allTotal = records.reduce((s, r) => s + r.total_tokens, 0)

  function keyName(id: string): string {
    return keys.find((k) => k.id === id)?.name || id.slice(0, 8)
  }

  function providerName(id: string): string {
    return providers.find((p) => p.id === id)?.name || id.slice(0, 8)
  }

  function clearFilters() {
    setFilterKeyId(ALL)
    setFilterProviderId(ALL)
    setFilterStart('')
    setFilterEnd('')
    setPage(1)
  }

  return (
    <div>
      <PageHeader
        index="03"
        title="Usage"
        description="Token usage per request, as recorded by the proxy."
      />

      {error && <DismissibleAlert variant="destructive" className="mb-4" onClose={() => setError('')}>{error}</DismissibleAlert>}

      {/* Filters */}
      <Card className="mb-4 py-4">
        <CardContent className="flex flex-wrap items-end gap-4 px-4">
          <div className="grid gap-2">
            <Label htmlFor="filter-key">API key</Label>
            <Select value={filterKeyId} onValueChange={(v) => { setFilterKeyId(v); setPage(1) }}>
              <SelectTrigger id="filter-key" size="sm" className="min-w-40">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={ALL}>All keys</SelectItem>
                {keys.map((k) => (
                  <SelectItem key={k.id} value={k.id}>{k.name}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="grid gap-2">
            <Label htmlFor="filter-provider">Provider</Label>
            <Select value={filterProviderId} onValueChange={(v) => { setFilterProviderId(v); setPage(1) }}>
              <SelectTrigger id="filter-provider" size="sm" className="min-w-40">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={ALL}>All providers</SelectItem>
                {providers.map((p) => (
                  <SelectItem key={p.id} value={p.id}>{p.name}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="grid gap-2">
            <Label htmlFor="filter-start">Start</Label>
            <Input
              id="filter-start"
              type="datetime-local"
              className="h-8 font-mono text-xs"
              value={filterStart}
              onChange={(e) => { setFilterStart(e.target.value); setPage(1) }}
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="filter-end">End</Label>
            <Input
              id="filter-end"
              type="datetime-local"
              className="h-8 font-mono text-xs"
              value={filterEnd}
              onChange={(e) => { setFilterEnd(e.target.value); setPage(1) }}
            />
          </div>
          <Button size="sm" variant="ghost" onClick={clearFilters}>Clear</Button>
        </CardContent>
      </Card>

      {/* Stats for the current page of records */}
      <div className="mb-4 grid grid-cols-2 gap-4 lg:grid-cols-4">
        <StatTile label="Total records" value={total} />
        <StatTile label="Prompt tokens" value={promptTotal} />
        <StatTile label="Completion tokens" value={completionTotal} />
        <StatTile label="Total tokens" value={allTotal} />
      </div>

      {/* Records */}
      <Card className="gap-0 py-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>ID</TableHead>
              <TableHead>Key</TableHead>
              <TableHead>Provider</TableHead>
              <TableHead>Model</TableHead>
              <TableHead className="text-right">Prompt</TableHead>
              <TableHead className="text-right">Completion</TableHead>
              <TableHead className="text-right">Total</TableHead>
              <TableHead>Time</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              <TableRow>
                <TableCell colSpan={8} className="py-6 text-center font-mono text-xs text-muted-foreground">
                  loading…
                </TableCell>
              </TableRow>
            ) : records.length === 0 ? (
              <TableRow>
                <TableCell colSpan={8} className="py-6 text-center font-mono text-xs text-muted-foreground">
                  no usage records
                </TableCell>
              </TableRow>
            ) : (
              records.map((r) => (
                <TableRow key={r.id}>
                  <TableCell className="font-mono text-xs text-muted-foreground">{r.id.slice(0, 8)}</TableCell>
                  <TableCell>{keyName(r.api_key_id)}</TableCell>
                  <TableCell>{providerName(r.provider_id)}</TableCell>
                  <TableCell className="font-mono text-xs">{r.model}</TableCell>
                  <TableCell className="text-right font-mono text-xs tabular-nums">{r.prompt_tokens.toLocaleString()}</TableCell>
                  <TableCell className="text-right font-mono text-xs tabular-nums">{r.completion_tokens.toLocaleString()}</TableCell>
                  <TableCell className="text-right font-mono text-xs font-medium tabular-nums">{r.total_tokens.toLocaleString()}</TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">{formatDateTime(r.created_at)}</TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </Card>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="mt-4 flex items-center justify-center gap-3">
          <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage(page - 1)}>
            <ChevronLeftIcon /> Previous
          </Button>
          <span className="font-mono text-xs text-muted-foreground">
            p. {page}/{totalPages}
          </span>
          <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => setPage(page + 1)}>
            Next <ChevronRightIcon />
          </Button>
        </div>
      )}
    </div>
  )
}
