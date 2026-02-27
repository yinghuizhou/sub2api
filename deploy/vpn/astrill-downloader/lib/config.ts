import { readFileSync, writeFileSync, existsSync } from 'fs';
import { join, resolve } from 'path';

// Files live in deploy/vpn/ (parent of the Next.js project dir)
const VPN_DIR = resolve(process.cwd(), '..');
export const OVPN_DIR = join(VPN_DIR, 'ovpn-configs');
const CONFIG_FILE = join(VPN_DIR, '.ai-config.json');
const COOKIES_FILE = join(VPN_DIR, '.astrill-cookies.json');

export interface AiConfig {
  apiKey?: string;
  apiUrl?: string;
  model?: string;
  lastCertId?: string;
}

export interface CookieEntry {
  name: string;
  value: string;
  [key: string]: unknown;
}

export function loadConfig(): AiConfig {
  try {
    if (existsSync(CONFIG_FILE)) {
      return JSON.parse(readFileSync(CONFIG_FILE, 'utf-8')) as AiConfig;
    }
  } catch { /* ignore */ }
  return {};
}

export function saveConfig(cfg: AiConfig): void {
  try { writeFileSync(CONFIG_FILE, JSON.stringify(cfg, null, 2)); } catch { /* ignore */ }
}

export function loadCookies(): CookieEntry[] | null {
  try {
    if (existsSync(COOKIES_FILE)) {
      return JSON.parse(readFileSync(COOKIES_FILE, 'utf-8')) as CookieEntry[];
    }
  } catch { /* ignore */ }
  return null;
}

export function saveCookies(cookies: CookieEntry[]): void {
  try { writeFileSync(COOKIES_FILE, JSON.stringify(cookies, null, 2)); } catch { /* ignore */ }
}
