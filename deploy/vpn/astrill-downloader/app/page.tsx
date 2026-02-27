'use client';
import { useEffect, useState } from 'react';
import type { AppState, ServerOption } from '@/lib/state';
import { ServerList } from '@/components/ServerList';
import { JobList } from '@/components/JobList';
import { CaptchaBox } from '@/components/CaptchaBox';
import { AIConfigCard } from '@/components/AIConfigCard';
import { LogPanel } from '@/components/LogPanel';

const INITIAL_STATE: AppState = {
  phase: 'login', servers: [], certId: null, csrfToken: null,
  currentIndex: -1, captchaB64: null, ocrText: '', downloaded: [],
  failed: [], message: '', autoMode: false, modeVal: 'tcp', jobs: [], logs: [],
};

export default function HomePage() {
  const [state, setState] = useState<AppState>(INITIAL_STATE);
  const [mode, setMode] = useState('TCP');
  const [auto, setAuto] = useState(true);

  useEffect(() => {
    const sse = new EventSource('/api/events');
    sse.onmessage = e => {
      try { setState(JSON.parse(e.data) as AppState); } catch { /* ignore */ }
    };
    sse.onerror = () => {
      // Reconnect handled by browser automatically
    };
    return () => sse.close();
  }, []);

  async function handleStart(selected: ServerOption[]) {
    await fetch('/api/start', {
      method: 'POST',
      body: JSON.stringify({ servers: selected, mode, auto }),
    });
  }

  async function handleStop() {
    await fetch('/api/stop', { method: 'POST', body: '{}' });
  }

  async function handleCaptcha(solution: string) {
    await fetch('/api/captcha', { method: 'POST', body: JSON.stringify({ solution }) });
  }

  async function handleSkip() {
    await fetch('/api/skip', { method: 'POST', body: '{}' });
  }

  const waitingJob = state.jobs.find(j => j.status === 'waiting');
  const { phase } = state;

  return (
    <div className="min-h-screen py-6 px-4">
      <div className="max-w-2xl mx-auto">
        {/* Header */}
        <div className="text-center mb-6">
          <h1 className="text-xl font-bold mb-1"
            style={{ background: 'linear-gradient(135deg, #60a5fa, #34d399)', WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent' }}>
            Astrill OpenVPN 批量下载器
          </h1>
          <p className="text-xs" style={{ color: '#475569' }}>自动遍历服务器，批量下载 .ovpn 配置文件</p>
        </div>

        {/* Status bar */}
        <div
          className="px-4 py-3 rounded-lg mb-4 text-sm"
          style={{
            background: 'var(--surface)',
            border: `1px solid ${phase === 'done' ? '#10b981' : phase === 'error' ? '#ef4444' : 'var(--border)'}`,
            borderLeft: `3px solid ${phase === 'done' ? '#10b981' : phase === 'error' ? '#ef4444' : '#3b82f6'}`,
            color: phase === 'done' ? '#10b981' : phase === 'error' ? '#ef4444' : '#94a3b8',
          }}
        >
          {state.message || '⏳ 等待中...'}
        </div>

        {/* Login waiting */}
        {phase === 'login' && (
          <div className="rounded-xl p-8 text-center mb-4" style={{ background: 'var(--surface)', border: '1px solid var(--border)' }}>
            <div className="text-4xl mb-3">🔐</div>
            <p className="text-sm" style={{ color: '#94a3b8' }}>请在打开的浏览器窗口中登录 Astrill，登录后将自动继续...</p>
            <div className="mt-4 flex justify-center">
              <div className="w-6 h-6 rounded-full border-2 border-blue-500 border-t-transparent animate-spin" />
            </div>
          </div>
        )}

        {/* AI Config (show during select and downloading) */}
        {(phase === 'select' || phase === 'downloading') && <AIConfigCard />}

        {/* Server selection */}
        {phase === 'select' && (
          <ServerList
            servers={state.servers}
            mode={mode}
            onModeChange={setMode}
            auto={auto}
            onAutoChange={setAuto}
            onStart={handleStart}
          />
        )}

        {/* Captcha box (when waiting for manual input) */}
        {waitingJob && (
          <CaptchaBox
            captchaB64={waitingJob.captchaB64}
            ocrText={waitingJob.ocrText}
            onSubmit={handleCaptcha}
            onSkip={handleSkip}
          />
        )}

        {/* Job list (during download and after done) */}
        {(phase === 'downloading' || phase === 'done') && state.jobs.length > 0 && (
          <JobList
            jobs={state.jobs}
            currentIndex={state.currentIndex}
            phase={phase}
            onStop={handleStop}
          />
        )}

        {/* Done summary */}
        {phase === 'done' && (
          <div className="rounded-xl p-5 mb-4" style={{ background: 'var(--surface)', border: '1px solid #10b98150' }}>
            <h2 className="text-base font-semibold mb-3">🎉 下载完成</h2>
            <div className="flex gap-6 text-sm mb-3">
              <span>✅ 成功: <strong className="text-emerald-400">{state.downloaded.length}</strong> 个</span>
              <span>❌ 失败: <strong className="text-red-400">{state.failed.length}</strong> 个</span>
            </div>
            <div className="font-mono text-xs space-y-1" style={{ color: '#64748b' }}>
              {state.downloaded.map((d, i) => (
                <div key={i} className="text-emerald-500">✅ {d.label} → {d.file}</div>
              ))}
              {state.failed.map((f, i) => (
                <div key={i} className="text-red-400">❌ {f}</div>
              ))}
            </div>
          </div>
        )}

        {/* Log panel — always visible */}
        <div className="rounded-xl p-5" style={{ background: 'var(--surface)', border: '1px solid var(--border)' }}>
          <h2 className="text-sm font-semibold mb-2 flex items-center gap-2" style={{ color: '#64748b' }}>
            <span>📋</span> 实时日志
          </h2>
          <LogPanel logs={state.logs} />
        </div>
      </div>
    </div>
  );
}
