'use client';
import { useEffect, useRef } from 'react';
import type { LogEntry } from '@/lib/state';

const LEVEL_CLASS: Record<string, string> = {
  ok: 'text-emerald-400',
  err: 'text-red-400',
  warn: 'text-amber-400',
  info: 'text-slate-400',
};

function formatTime(ts: number): string {
  return new Date(ts).toLocaleTimeString('zh-CN', { hour12: false });
}

export function LogPanel({ logs }: { logs: LogEntry[] }) {
  const bottomRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;
    // Auto-scroll only if already near bottom
    const isNearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 80;
    if (isNearBottom) {
      bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
    }
  }, [logs.length]);

  return (
    <div
      ref={containerRef}
      className="h-44 overflow-y-auto rounded-lg p-3 font-mono text-xs leading-relaxed"
      style={{ background: '#0d1117', border: '1px solid var(--border)' }}
    >
      {logs.length === 0 && (
        <span className="text-slate-600">等待日志...</span>
      )}
      {logs.map((log, i) => (
        <div key={i} className="flex gap-2">
          <span className="text-slate-600 shrink-0">{formatTime(log.ts)}</span>
          <span className={LEVEL_CLASS[log.level] || 'text-slate-400'}>{log.text}</span>
        </div>
      ))}
      <div ref={bottomRef} />
    </div>
  );
}
