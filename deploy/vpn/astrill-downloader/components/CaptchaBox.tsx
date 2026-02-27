'use client';
import { useState, useEffect, useRef } from 'react';

interface Props {
  captchaB64: string | null;
  ocrText: string;
  onSubmit: (solution: string) => void;
  onSkip: () => void;
}

export function CaptchaBox({ captchaB64, ocrText, onSubmit, onSkip }: Props) {
  const [value, setValue] = useState('');
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (captchaB64) {
      setValue(ocrText || '');
      inputRef.current?.focus();
    }
  }, [captchaB64, ocrText]);

  function submit() {
    if (!value.trim()) return;
    onSubmit(value.trim());
    setValue('');
  }

  if (!captchaB64) return null;

  return (
    <div className="rounded-xl p-4 mb-4" style={{ background: 'var(--surface)', border: '1px solid #f59e0b50' }}>
      <h3 className="text-sm font-semibold mb-3" style={{ color: '#f59e0b' }}>✍️ 请输入验证码</h3>
      <div className="flex items-center gap-3 flex-wrap">
        <img
          src={`data:image/jpeg;base64,${captchaB64}`}
          alt="CAPTCHA"
          className="h-12 rounded-lg bg-white object-contain"
          style={{ minWidth: '80px' }}
        />
        <input
          ref={inputRef}
          value={value}
          onChange={e => setValue(e.target.value)}
          onKeyDown={e => e.key === 'Enter' && submit()}
          placeholder="输入验证码..."
          className="flex-1 min-w-28 px-3 py-2 rounded-lg font-mono text-lg tracking-widest outline-none transition-colors"
          style={{
            background: '#0d1117',
            border: '2px solid var(--border)',
            color: '#e2e8f0',
          }}
          onFocus={e => (e.target.style.borderColor = '#3b82f6')}
          onBlur={e => (e.target.style.borderColor = 'var(--border)')}
        />
        <button
          onClick={submit}
          disabled={!value.trim()}
          className="px-4 py-2 rounded-lg font-medium text-sm text-white transition-opacity hover:opacity-80"
          style={{ background: '#10b981', cursor: value.trim() ? 'pointer' : 'not-allowed', opacity: value.trim() ? 1 : 0.4 }}
        >
          ✓ 提交
        </button>
        <button
          onClick={onSkip}
          className="px-4 py-2 rounded-lg text-sm transition-opacity hover:opacity-80"
          style={{ background: 'var(--surface2)', border: '1px solid var(--border)', color: '#94a3b8', cursor: 'pointer' }}
        >
          跳过
        </button>
      </div>
    </div>
  );
}
