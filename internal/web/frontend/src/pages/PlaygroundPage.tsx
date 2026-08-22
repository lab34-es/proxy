import { useState, useRef, useEffect, useCallback } from 'react'
import { SendIcon, XIcon } from 'lucide-react'

import PageHeader from '@/components/PageHeader'
import DismissibleAlert from '@/components/DismissibleAlert'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { cn } from '@/lib/utils'

interface Message {
  role: 'user' | 'assistant' | 'system'
  content: string
}

export default function PlaygroundPage() {
  const [apiKey, setApiKey] = useState('')
  const [model, setModel] = useState('')
  const [systemPrompt, setSystemPrompt] = useState('')
  const [input, setInput] = useState('')
  const [conversation, setConversation] = useState<Message[]>([])
  const [streaming, setStreaming] = useState(false)
  const [error, setError] = useState('')
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const abortRef = useRef<AbortController | null>(null)

  const scrollToBottom = useCallback(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [])

  useEffect(() => { scrollToBottom() }, [conversation, scrollToBottom])

  async function sendMessage() {
    const text = input.trim()
    if (!apiKey.trim()) {
      setError('Please enter an API key.')
      return
    }
    if (!model.trim()) {
      setError('Please enter a model name.')
      return
    }
    if (!text) return

    setError('')
    const userMsg: Message = { role: 'user', content: text }
    const newConvo = [...conversation, userMsg]
    setConversation(newConvo)
    setInput('')

    const messages: Message[] = []
    if (systemPrompt.trim()) {
      messages.push({ role: 'system', content: systemPrompt.trim() })
    }
    messages.push(...newConvo)

    setStreaming(true)
    const assistantMsg: Message = { role: 'assistant', content: '' }
    setConversation([...newConvo, assistantMsg])

    const controller = new AbortController()
    abortRef.current = controller

    try {
      const resp = await fetch('/v1/chat/completions', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${apiKey.trim()}`,
        },
        body: JSON.stringify({
          model: model.trim(),
          messages,
          stream: true,
        }),
        signal: controller.signal,
      })

      if (!resp.ok) {
        const errText = await resp.text()
        setConversation((prev) => {
          const updated = [...prev]
          updated[updated.length - 1] = { role: 'assistant', content: `Error: ${errText}` }
          return updated
        })
        setStreaming(false)
        return
      }

      const reader = resp.body!.getReader()
      const decoder = new TextDecoder()
      let assistantText = ''
      let buffer = ''

      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''

        for (const line of lines) {
          if (!line.startsWith('data: ')) continue
          const data = line.slice(6).trim()
          if (data === '[DONE]') continue

          try {
            const parsed = JSON.parse(data)
            const delta = parsed.choices?.[0]?.delta?.content
            if (delta) {
              assistantText += delta
              setConversation((prev) => {
                const updated = [...prev]
                updated[updated.length - 1] = { role: 'assistant', content: assistantText }
                return updated
              })
            }
          } catch {
            // skip malformed chunks
          }
        }
      }
    } catch (err: unknown) {
      if (err instanceof Error && err.name !== 'AbortError') {
        setConversation((prev) => {
          const updated = [...prev]
          updated[updated.length - 1] = { role: 'assistant', content: `Error: ${err.message}` }
          return updated
        })
      }
    } finally {
      setStreaming(false)
      abortRef.current = null
    }
  }

  function clearConversation() {
    if (abortRef.current) abortRef.current.abort()
    setConversation([])
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey && !streaming) {
      e.preventDefault()
      sendMessage()
    }
  }

  return (
    <div className="flex h-[calc(100svh-3rem)] flex-col lg:h-[calc(100svh-4rem)]">
      <PageHeader
        index="05"
        title="Playground"
        description="Send a request through the proxy with one of its API keys."
        actions={
          <Button variant="outline" size="sm" onClick={clearConversation}>
            <XIcon /> Clear
          </Button>
        }
      />

      {error && <DismissibleAlert variant="destructive" className="mb-4" onClose={() => setError('')}>{error}</DismissibleAlert>}

      {/* Request configuration */}
      <Card className="mb-4 py-4">
        <CardContent className="grid gap-4 px-4">
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="grid gap-2">
              <Label htmlFor="pg-key">API key</Label>
              <Input
                id="pg-key"
                type="password"
                className="h-8 font-mono text-xs"
                value={apiKey}
                onChange={(e) => setApiKey(e.target.value)}
                placeholder="llmp-…"
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="pg-model">Model</Label>
              <Input
                id="pg-model"
                className="h-8 font-mono text-xs"
                value={model}
                onChange={(e) => setModel(e.target.value)}
                placeholder="e.g. gpt-4"
              />
            </div>
          </div>
          <div className="grid gap-2">
            <Label htmlFor="pg-system">System prompt</Label>
            <Textarea
              id="pg-system"
              className="min-h-8 text-xs"
              rows={1}
              value={systemPrompt}
              onChange={(e) => setSystemPrompt(e.target.value)}
              placeholder="Optional system prompt…"
            />
          </div>
        </CardContent>
      </Card>

      {/* Conversation log */}
      <div className="mb-4 flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto border bg-card p-4">
        {conversation.length === 0 ? (
          <p className="mt-8 text-center font-mono text-xs text-muted-foreground">
            send a message to start a conversation
          </p>
        ) : (
          conversation.map((msg, i) => (
            <div
              key={i}
              className={cn('flex max-w-[85%] flex-col gap-1', msg.role === 'user' ? 'self-end items-end' : 'self-start')}
            >
              <span className="kicker text-muted-foreground">
                {msg.role === 'user' ? 'you' : 'assistant'}
              </span>
              <div
                className={cn(
                  'border px-3 py-2 text-sm whitespace-pre-wrap break-words',
                  msg.role === 'user'
                    ? 'border-carbon bg-carbon text-bone dark:border-bone/90 dark:bg-bone/90 dark:text-ink'
                    : 'bg-background',
                )}
              >
                {msg.content || (streaming && i === conversation.length - 1 ? '▍' : '')}
              </div>
            </div>
          ))
        )}
        <div ref={messagesEndRef} />
      </div>

      {/* Input */}
      <div className="flex gap-2">
        <Input
          className="flex-1"
          placeholder="Type a message…"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={handleKeyDown}
          disabled={streaming}
        />
        <Button onClick={sendMessage} disabled={streaming}>
          <SendIcon /> {streaming ? 'Streaming…' : 'Send'}
        </Button>
      </div>
    </div>
  )
}
