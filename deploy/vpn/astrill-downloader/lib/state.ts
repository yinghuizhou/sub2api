export interface LogEntry {
  ts: number;
  level: 'info' | 'ok' | 'err' | 'warn';
  text: string;
}

export interface ServerOption {
  value: string;
  label: string;
  country: string;
}

export interface Job {
  server: ServerOption;
  captchaB64: string | null;
  ocrText: string;
  status: 'pending' | 'captcha' | 'recognizing' | 'waiting' | 'downloading' | 'done' | 'failed' | 'skipped' | 'cached';
  file: string | null;
  retries: number;
}

export interface AppState {
  phase: 'login' | 'select' | 'downloading' | 'done' | 'error';
  servers: ServerOption[];
  certId: string | null;
  csrfToken: string | null;
  currentIndex: number;
  captchaB64: string | null;
  ocrText: string;
  downloaded: Array<{ label: string; file: string }>;
  failed: string[];
  message: string;
  autoMode: boolean;
  modeVal: string;
  jobs: Job[];
  logs: LogEntry[];
}

const initialState: AppState = {
  phase: 'login',
  servers: [],
  certId: null,
  csrfToken: null,
  currentIndex: -1,
  captchaB64: null,
  ocrText: '',
  downloaded: [],
  failed: [],
  message: '',
  autoMode: false,
  modeVal: 'tcp',
  jobs: [],
  logs: [],
};

// Persist state across HMR reloads using globalThis
const g = globalThis as Record<string, unknown>;
if (!g.__astrill_store) {
  g.__astrill_store = {
    state: { ...initialState },
    sseClients: new Map<string, (data: string) => void>(),
  };
}

interface Store {
  state: AppState;
  sseClients: Map<string, (data: string) => void>;
}

const store = g.__astrill_store as Store;

export function getState(): AppState {
  return store.state;
}

export function setState(updater: Partial<AppState> | ((s: AppState) => void)) {
  if (typeof updater === 'function') {
    updater(store.state);
  } else {
    Object.assign(store.state, updater);
  }
}

export function addLog(level: LogEntry['level'], text: string) {
  const logs = store.state.logs;
  logs.push({ ts: Date.now(), level, text });
  if (logs.length > 200) logs.splice(0, logs.length - 200);
}

export function setMsg(text: string, level: LogEntry['level'] = 'info') {
  store.state.message = text;
  addLog(level, text);
  broadcast();
  console.log(text);
}

// Strip sensitive fields before broadcasting to SSE clients.
// captchaB64 is only kept for the 'waiting' job (user needs to solve it manually).
export function sanitizeForBroadcast(state: AppState): object {
  const { csrfToken: _csrf, certId: _cert, captchaB64: _topCap, ...safe } = state;
  const sanitizedJobs = safe.jobs.map(job => ({
    ...job,
    captchaB64: job.status === 'waiting' ? job.captchaB64 : null,
  }));
  return { ...safe, jobs: sanitizedJobs };
}

export function broadcast() {
  const data = JSON.stringify(sanitizeForBroadcast(store.state));
  for (const send of store.sseClients.values()) {
    try { send(data); } catch { /* client disconnected */ }
  }
}

// Atomic batch lock — JS single-threaded event loop makes check-and-set race-free.
// Returns true if the lock was acquired (phase set to 'downloading').
export function tryAcquireBatch(): boolean {
  if (store.state.phase === 'downloading') return false;
  store.state.phase = 'downloading';
  broadcast();
  return true;
}

// Release the batch lock. Only resets phase if still 'downloading' to avoid
// overriding a 'done' phase already set by startBatch completion.
export function releaseBatch(failed = false): void {
  if (store.state.phase === 'downloading') {
    store.state.phase = failed ? 'error' : 'done';
    broadcast();
  }
}

export function getSseClientCount(): number {
  return store.sseClients.size;
}

export function addSseClient(id: string, send: (data: string) => void) {
  store.sseClients.set(id, send);
}

export function removeSseClient(id: string) {
  store.sseClients.delete(id);
}
