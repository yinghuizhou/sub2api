'use client';
import { useState, useMemo } from 'react';
import type { ServerOption } from '@/lib/state';

function getRegionInfo(label: string): { flag: string; tagClass: string } {
  if (/US|United States|Chicago|Los Angeles|New York|Dallas|Miami|Seattle|San Fran|Denver|Phoenix|Atlanta|Washington/i.test(label))
    return { flag: '🇺🇸', tagClass: 'text-blue-400 bg-blue-400/10' };
  if (/UK|London|Europe|Germany|France|Netherlands|Amsterdam/i.test(label))
    return { flag: '🇪🇺', tagClass: 'text-purple-400 bg-purple-400/10' };
  if (/Japan|Korea|Singapore|Hong Kong|Tokyo|Asia/i.test(label))
    return { flag: '🌏', tagClass: 'text-amber-400 bg-amber-400/10' };
  return { flag: '🌐', tagClass: 'text-slate-400 bg-slate-400/10' };
}

interface Props {
  servers: ServerOption[];
  mode: string;
  onModeChange: (m: string) => void;
  auto: boolean;
  onAutoChange: (v: boolean) => void;
  onStart: (selected: ServerOption[]) => void;
}

export function ServerList({ servers, mode, onModeChange, auto, onAutoChange, onStart }: Props) {
  const [filter, setFilter] = useState('');
  const [selected, setSelected] = useState<Set<string>>(new Set());

  const filtered = useMemo(() =>
    servers.filter(s => !filter || s.label.toLowerCase().includes(filter.toLowerCase())),
    [servers, filter]
  );

  function toggle(value: string) {
    setSelected(prev => {
      const next = new Set(prev);
      next.has(value) ? next.delete(value) : next.add(value);
      return next;
    });
  }

  function selectAll(region?: string) {
    setSelected(new Set(
      servers
        .filter(s => !region || (region === 'US' && getRegionInfo(s.label).flag === '🇺🇸'))
        .map(s => s.value)
    ));
  }

  const selectedCount = selected.size;

  return (
    <div className="rounded-xl p-5 mb-4" style={{ background: 'var(--surface)', border: '1px solid var(--border)' }}>
      <h2 className="text-base font-semibold mb-3 flex items-center gap-2">
        <span>📋</span> 选择要下载的服务器
      </h2>

      {/* Controls */}
      <div className="flex gap-2 mb-3 flex-wrap items-center">
        <select
          value={mode}
          onChange={e => onModeChange(e.target.value)}
          className="px-3 py-1.5 text-sm rounded-md font-mono"
          style={{ background: 'var(--surface2)', border: '1px solid var(--border)', color: '#e2e8f0' }}
        >
          <option value="TCP">TCP (推荐)</option>
          <option value="UDP">UDP (快速)</option>
        </select>
        <input
          value={filter}
          onChange={e => setFilter(e.target.value)}
          placeholder="搜索服务器..."
          className="flex-1 min-w-32 px-3 py-1.5 text-sm rounded-md outline-none"
          style={{ background: 'var(--surface2)', border: '1px solid var(--border)', color: '#e2e8f0' }}
        />
        <button onClick={() => selectAll('US')} className="px-3 py-1.5 text-xs rounded-md cursor-pointer hover:opacity-80 transition-opacity"
          style={{ background: 'var(--surface2)', border: '1px solid var(--border)', color: '#94a3b8' }}>全选美国</button>
        <button onClick={() => selectAll()} className="px-3 py-1.5 text-xs rounded-md cursor-pointer hover:opacity-80 transition-opacity"
          style={{ background: 'var(--surface2)', border: '1px solid var(--border)', color: '#94a3b8' }}>全选</button>
        <button onClick={() => setSelected(new Set())} className="px-3 py-1.5 text-xs rounded-md cursor-pointer hover:opacity-80 transition-opacity"
          style={{ background: 'var(--surface2)', border: '1px solid var(--border)', color: '#94a3b8' }}>取消</button>
      </div>

      {/* Server list */}
      <div className="rounded-lg overflow-y-auto" style={{ maxHeight: '300px', border: '1px solid var(--border)' }}>
        {filtered.map(s => {
          const { flag, tagClass } = getRegionInfo(s.label);
          return (
            <label key={s.value} className="flex items-center gap-2 px-3 py-2 cursor-pointer text-sm transition-colors hover:bg-white/5"
              style={{ borderBottom: '1px solid var(--border)' }}>
              <input
                type="checkbox"
                checked={selected.has(s.value)}
                onChange={() => toggle(s.value)}
                className="accent-blue-500 w-3.5 h-3.5"
              />
              <span className="w-5">{flag}</span>
              <span className="flex-1 text-slate-200">{s.label}</span>
              <span className={`text-xs px-1.5 py-0.5 rounded font-mono ${tagClass}`}>{s.country || s.value}</span>
            </label>
          );
        })}
      </div>

      <div className="text-xs mt-2 mb-3" style={{ color: '#64748b' }}>
        已选择 {selectedCount} / {servers.length} 个服务器
      </div>

      {/* Auto mode */}
      <label className="flex items-center gap-2 text-sm mb-4 cursor-pointer" style={{ color: '#94a3b8' }}>
        <input type="checkbox" checked={auto} onChange={e => onAutoChange(e.target.checked)}
          className="accent-emerald-500 w-4 h-4" />
        🤖 AI 自动识别验证码（识别后自动提交，可手动修正）
      </label>

      <div className="flex justify-end">
        <button
          disabled={selectedCount === 0}
          onClick={() => onStart(servers.filter(s => selected.has(s.value)))}
          className="px-6 py-2.5 rounded-lg font-medium text-sm transition-all"
          style={{
            background: selectedCount === 0 ? '#1e293b' : '#3b82f6',
            color: selectedCount === 0 ? '#475569' : '#fff',
            cursor: selectedCount === 0 ? 'not-allowed' : 'pointer',
          }}
        >
          🚀 开始批量下载
        </button>
      </div>
    </div>
  );
}
