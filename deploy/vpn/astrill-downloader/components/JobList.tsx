'use client';
import { useEffect, useRef } from 'react';
import type { Job } from '@/lib/state';

const STATUS_ICON: Record<string, string> = {
  pending: '⏳', captcha: '🔄', recognizing: '🤖', waiting: '✍️',
  downloading: '📥', done: '✅', failed: '❌', skipped: '⏭️', cached: '💾',
};

interface Props {
  jobs: Job[];
  currentIndex: number;
  phase: string;
  onStop: () => void;
}

export function JobList({ jobs, currentIndex, phase, onStop }: Props) {
  const activeRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    activeRef.current?.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
  }, [currentIndex]);

  const done = jobs.filter(j => j.status === 'done' || j.status === 'cached').length;
  const total = jobs.length;
  const pct = total > 0 ? (done / total) * 100 : 0;

  return (
    <div className="rounded-xl p-5 mb-4" style={{ background: 'var(--surface)', border: '1px solid var(--border)' }}>
      <div className="flex items-center justify-between mb-3">
        <h2 className="text-base font-semibold flex items-center gap-2">
          <span>📥</span> 批量下载
          <span className="text-sm font-normal" style={{ color: '#64748b' }}>({done}/{total} 完成)</span>
        </h2>
        {phase === 'downloading' && (
          <button onClick={onStop}
            className="px-3 py-1 text-xs rounded-md transition-opacity hover:opacity-80"
            style={{ background: 'var(--surface2)', border: '1px solid var(--border)', color: '#94a3b8', cursor: 'pointer' }}>
            ⏹ 停止
          </button>
        )}
      </div>

      {/* Progress bar */}
      <div className="h-1.5 rounded-full mb-3 overflow-hidden" style={{ background: 'var(--surface2)' }}>
        <div className="h-full rounded-full transition-all duration-500"
          style={{ width: `${pct}%`, background: 'linear-gradient(90deg, #3b82f6, #10b981)' }} />
      </div>

      {/* Job list */}
      <div className="rounded-lg overflow-y-auto" style={{ maxHeight: '400px', border: '1px solid var(--border)' }}>
        {jobs.map((job, i) => {
          const isActive = i === currentIndex && phase === 'downloading';
          const isDone = ['done', 'failed', 'skipped', 'cached'].includes(job.status);
          return (
            <div
              key={i}
              ref={isActive ? activeRef : undefined}
              className={`grid gap-2 items-center px-3 py-1.5 text-xs transition-colors ${isDone ? 'opacity-60' : ''}`}
              style={{
                gridTemplateColumns: '28px 1fr 80px 70px 24px',
                borderBottom: '1px solid var(--border)',
                background: isActive ? 'var(--surface2)' : 'transparent',
                borderLeft: isActive ? '2px solid #3b82f6' : '2px solid transparent',
              }}
            >
              <span className="font-mono text-center" style={{ color: '#475569', fontSize: '0.65rem' }}>{i + 1}</span>
              <span className="overflow-hidden text-ellipsis whitespace-nowrap" style={{ color: '#cbd5e1' }} title={job.server.label}>
                {job.server.label}
                {job.retries > 0 && (
                  <span className="ml-1 px-1 py-0.5 rounded text-black font-bold" style={{ background: '#f59e0b', fontSize: '0.6rem' }}>
                    ×{job.retries}
                  </span>
                )}
              </span>
              {job.captchaB64
                ? <img src={`data:image/jpeg;base64,${job.captchaB64}`} className="h-6 rounded object-contain bg-white" alt="captcha" />
                : <span style={{ color: '#475569' }}>—</span>
              }
              {job.file
                ? <span className="overflow-hidden text-ellipsis font-mono" style={{ color: '#10b981', fontSize: '0.65rem' }}>{job.file}</span>
                : job.ocrText
                  ? <span className="font-mono font-semibold tracking-widest overflow-hidden text-ellipsis" style={{ color: '#a78bfa' }}>{job.ocrText}</span>
                  : <span style={{ color: '#475569' }}>—</span>
              }
              <span className="text-center">{STATUS_ICON[job.status] || '⏳'}</span>
            </div>
          );
        })}
      </div>
    </div>
  );
}
