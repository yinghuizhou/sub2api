'use client';
import { useState, useEffect } from 'react';

interface ConfigState {
  apiKey: string;
  apiUrl: string;
  model: string;
  hasKey: boolean;
}

export function AIConfigCard() {
  const [cfg, setCfg] = useState<ConfigState>({ apiKey: '', apiUrl: '', model: '', hasKey: false });
  const [apiUrlInput, setApiUrlInput] = useState('');
  const [apiKeyInput, setApiKeyInput] = useState('');
  const [modelInput, setModelInput] = useState('');
  const [showKey, setShowKey] = useState(false);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    fetch('/api/config').then(r => r.json()).then((data: ConfigState) => {
      setCfg(data);
      setApiUrlInput(data.apiUrl);
      setModelInput(data.model);
    }).catch(() => { /* ignore */ });
  }, []);

  async function save() {
    setSaving(true);
    const body: Record<string, string> = {};
    if (apiUrlInput) body.apiUrl = apiUrlInput;
    if (apiKeyInput) body.apiKey = apiKeyInput;
    if (modelInput) body.model = modelInput;
    await fetch('/api/config', { method: 'POST', body: JSON.stringify(body) });
    setApiKeyInput('');
    // Refresh
    const data = await fetch('/api/config').then(r => r.json()) as ConfigState;
    setCfg(data);
    setSaving(false);
  }

  return (
    <div className="rounded-xl p-5 mb-4" style={{ background: 'var(--surface)', border: '1px solid var(--border)' }}>
      <h2 className="text-base font-semibold mb-3 flex items-center gap-2">
        <span>🔑</span> AI 验证码识别配置
      </h2>

      <div className="flex flex-col gap-2.5">
        <Row label="接口地址">
          <input value={apiUrlInput} onChange={e => setApiUrlInput(e.target.value)}
            placeholder={cfg.apiUrl || 'http://localhost:8080'}
            style={{ background: 'var(--surface2)', border: '1px solid var(--border)', color: '#e2e8f0' }}
            className="flex-1 px-3 py-1.5 rounded-md text-sm font-mono outline-none" />
        </Row>

        <Row label="API Key">
          <input
            type={showKey ? 'text' : 'password'}
            value={apiKeyInput} onChange={e => setApiKeyInput(e.target.value)}
            placeholder={cfg.hasKey ? cfg.apiKey : '输入 API Key'}
            style={{ background: 'var(--surface2)', border: '1px solid var(--border)', color: '#e2e8f0' }}
            className="flex-1 px-3 py-1.5 rounded-md text-sm font-mono outline-none"
          />
          <button onClick={() => setShowKey(!showKey)}
            className="px-2.5 py-1.5 rounded-md text-sm cursor-pointer hover:opacity-80 transition-opacity"
            style={{ background: 'var(--surface2)', border: '1px solid var(--border)', color: '#94a3b8' }}>
            {showKey ? '🙈' : '👁'}
          </button>
        </Row>

        <Row label="模型">
          <input value={modelInput} onChange={e => setModelInput(e.target.value)}
            placeholder={cfg.model || 'claude-haiku-4-5-20251001 (默认)'}
            style={{ background: 'var(--surface2)', border: '1px solid var(--border)', color: '#e2e8f0' }}
            className="flex-1 px-3 py-1.5 rounded-md text-sm font-mono outline-none" />
          <button onClick={save} disabled={saving}
            className="px-4 py-1.5 rounded-md text-sm text-white font-medium cursor-pointer hover:opacity-80 transition-opacity"
            style={{ background: '#10b981' }}>
            {saving ? '保存中...' : '保存'}
          </button>
        </Row>
      </div>

      <div className="mt-2 text-xs" style={{ color: cfg.hasKey ? '#10b981' : '#f59e0b' }}>
        {cfg.hasKey
          ? `✅ 已配置 — 接口: ${cfg.apiUrl || '未设置'}`
          : '⚠️ 未配置 — 请填入 sub2api 地址和 API Key'}
      </div>
    </div>
  );
}

function Row({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-center gap-2">
      <label className="text-sm shrink-0 w-16 text-right" style={{ color: '#94a3b8' }}>{label}</label>
      {children}
    </div>
  );
}
