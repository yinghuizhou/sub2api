#!/usr/bin/env node
/**
 * Astrill OpenVPN Config Batch Downloader v3
 *
 * Based on actual Astrill web panel DOM structure (inspected Feb 2026).
 *
 * Flow:
 *   1. Opens browser → you login manually
 *   2. Navigates to OpenVPN certificates page
 *   3. For each target server: opens download modal, selects server, waits for CAPTCHA
 *   4. You solve CAPTCHA and click download → script captures the file
 *   5. Repeats for next server
 *
 * Usage:
 *   npx playwright install chromium   # first time only
 *   node download-astrill-ovpn.mjs [--servers la,ny,denver] [--mode tcp|udp]
 */
import { chromium } from 'playwright';
import { mkdirSync, writeFileSync } from 'fs';
import { join, resolve } from 'path';

const OUTPUT_DIR = resolve(import.meta.dirname || '.', 'ovpn-configs');
const ASTRILL_LOGIN = 'https://www.astrill.com/member-zone/log-in';
const ASTRILL_CERTS = 'https://www.astrill.com/member-zone/tools/openvpn-certificates';

// Recommended 10G servers for Sub2API (value = Astrill server ID)
const SERVERS = {
  'la-10g':      { value: '1199', label: '*USA - Los Angeles 10G' },
  'la-10gb':     { value: '1413', label: '*USA - Los Angeles 10GB' },
  'ny-10g':      { value: '1197', label: '*USA - New York 10G-1' },
  'denver-10g':  { value: '1416', label: '*USA - Denver 10G' },
  'phoenix-10g': { value: '1414', label: '*USA - Phoenix 10G' },
  'ashburn-10g': { value: '1207', label: '*USA - Ashburn 10G' },
  'chicago-10gb':{ value: '1417', label: '*USA - Chicago 10GB' },
  'manchester':  { value: '1410', label: '*UK - Manchester 10G' },
  'nl-10g':      { value: '1407', label: '*Netherlands 10G' },
  'frankfurt':   { value: '1418', label: '*Germany - Frankfurt 10G' },
};

// Aliases for CLI convenience
const ALIASES = {
  la: 'la-10g', ny: 'ny-10g', denver: 'denver-10g',
  phoenix: 'phoenix-10g', ashburn: 'ashburn-10g',
  chicago: 'chicago-10gb', uk: 'manchester', nl: 'nl-10g', de: 'frankfurt',
};

function parseArgs() {
  const args = process.argv.slice(2);
  let serverKeys = ['la-10g', 'ny-10g', 'denver-10g']; // default: 3 diverse US servers
  let mode = 'tcp'; // tcp = reliable, better for proxy use

  for (let i = 0; i < args.length; i++) {
    if (args[i] === '--servers' && args[i + 1]) {
      serverKeys = args[++i].split(',').map(k => ALIASES[k.trim()] || k.trim());
    } else if (args[i] === '--mode' && args[i + 1]) {
      mode = args[++i];
    } else if (args[i] === '--list') {
      console.log('\nAvailable servers:');
      for (const [key, srv] of Object.entries(SERVERS)) {
        const aliases = Object.entries(ALIASES).filter(([, v]) => v === key).map(([a]) => a);
        const aliasStr = aliases.length ? ` (alias: ${aliases.join(', ')})` : '';
        console.log(`  ${key.padEnd(16)} ${srv.label}${aliasStr}`);
      }
      process.exit(0);
    } else if (args[i] === '--help') {
      console.log(`
Usage: node download-astrill-ovpn.mjs [options]

Options:
  --servers la,ny,denver   Comma-separated server keys (default: la-10g,ny-10g,denver-10g)
  --mode tcp|udp           Connection mode (default: tcp)
  --list                   List all available servers
  --help                   Show this help

Examples:
  node download-astrill-ovpn.mjs                          # Download 3 default servers
  node download-astrill-ovpn.mjs --servers la,ashburn,nl   # Specific servers
  node download-astrill-ovpn.mjs --mode udp                # UDP mode
`);
      process.exit(0);
    }
  }

  // Validate server keys
  const targets = [];
  for (const key of serverKeys) {
    if (!SERVERS[key]) {
      console.error(`Unknown server: ${key}. Use --list to see available servers.`);
      process.exit(1);
    }
    targets.push({ key, ...SERVERS[key] });
  }

  return { targets, mode };
}

async function main() {
  const { targets, mode } = parseArgs();

  console.log('\n=== Astrill OpenVPN Batch Downloader v3 ===\n');
  console.log(`Mode: ${mode.toUpperCase()}`);
  console.log(`Servers to download (${targets.length}):`);
  targets.forEach((t, i) => console.log(`  ${i + 1}. ${t.label}`));
  console.log();

  mkdirSync(OUTPUT_DIR, { recursive: true });

  const browser = await chromium.launch({ headless: false, slowMo: 100 });
  const context = await browser.newContext({ acceptDownloads: true });
  const page = await context.newPage();

  // Step 1: Login
  console.log('[1/4] Opening Astrill login page...');
  await page.goto(ASTRILL_LOGIN, { waitUntil: 'domcontentloaded', timeout: 30000 });
  console.log('      Please login in the browser (5 min timeout)...\n');

  try {
    await page.waitForURL(url => {
      const u = url.toString();
      return u.includes('member-zone') && !u.includes('log-in');
    }, { timeout: 300000 });
  } catch {
    console.log('Login timeout.'); await browser.close(); process.exit(1);
  }
  console.log('[1/4] Login successful!\n');

  // Step 2: Navigate to certificates page
  console.log('[2/4] Navigating to OpenVPN certificates page...');
  await page.goto(ASTRILL_CERTS, { waitUntil: 'networkidle', timeout: 30000 });
  await page.waitForTimeout(2000);

  // Find the first certificate's download button
  const certTable = page.locator('table#table-your-openvpn-certificates');
  const downloadLinks = certTable.locator('a[onclick*="openVPNCertificatesOnDownloadSingle"]');
  const certCount = await downloadLinks.count();

  if (certCount === 0) {
    console.log('No certificates found. Please create one first on the Astrill panel.');
    await browser.close();
    process.exit(1);
  }
  console.log(`[2/4] Found ${certCount} certificate(s).\n`);

  // Step 3: Download configs for each target server
  console.log('[3/4] Starting batch download...\n');
  const downloaded = [];

  for (let i = 0; i < targets.length; i++) {
    const target = targets[i];
    console.log(`--- [${i + 1}/${targets.length}] ${target.label} ---`);

    // Click the first cert's download icon to open the modal
    await downloadLinks.first().click();
    await page.waitForTimeout(1000);

    // Wait for the download modal to appear
    const modal = page.locator('#modal-download-single');
    await modal.waitFor({ state: 'visible', timeout: 5000 });

    // Select mode (tcp/udp)
    await modal.locator('select#mode').selectOption(mode);
    await page.waitForTimeout(300);

    // Select target server
    await modal.locator('select#vpn_server').selectOption(target.value);
    await page.waitForTimeout(300);

    console.log(`      Server: ${target.label} | Mode: ${mode.toUpperCase()}`);
    console.log('      >>> Please solve the CAPTCHA and click the download button <<<\n');

    // Wait for user to solve CAPTCHA and trigger download
    try {
      const download = await page.waitForEvent('download', { timeout: 120000 });
      const filename = download.suggestedFilename() || `${target.key}-${mode}.ovpn`;
      const filePath = join(OUTPUT_DIR, filename);
      await download.saveAs(filePath);
      console.log(`      Downloaded: ${filePath}\n`);
      downloaded.push({ server: target.label, file: filePath });
    } catch {
      console.log(`      Download timeout for ${target.label}, skipping...\n`);
    }

    // Close modal if still open
    const closeBtn = modal.locator('a:has-text("Close"), a:has-text("关闭")');
    if (await closeBtn.first().isVisible({ timeout: 1000 }).catch(() => false)) {
      await closeBtn.first().click();
      await page.waitForTimeout(500);
    }

    // Small delay between downloads
    if (i < targets.length - 1) {
      await page.waitForTimeout(1000);
    }
  }

  // Step 4: Summary
  console.log('\n[4/4] Summary\n');
  if (downloaded.length > 0) {
    console.log(`Successfully downloaded ${downloaded.length}/${targets.length} configs:`);
    downloaded.forEach(d => console.log(`  ${d.file}`));

    // Generate manifest for vpn-manager.sh deploy-batch
    const manifest = downloaded.map((d, idx) => ({
      ovpn_file: d.file,
      instance_name: targets[idx].key.replace(/-/g, '_'),
      socks_port: 10802 + idx, // start from 10802 (10801 is existing chicago)
      region: targets[idx].label.includes('USA') ? 'us' :
              targets[idx].label.includes('UK') ? 'eu-west' :
              targets[idx].label.includes('Netherlands') ? 'eu-west' :
              targets[idx].label.includes('Germany') ? 'eu-central' : 'unknown',
    }));
    const manifestPath = join(OUTPUT_DIR, 'manifest.json');
    writeFileSync(manifestPath, JSON.stringify(manifest, null, 2));
    console.log(`\nManifest saved: ${manifestPath}`);
    console.log('\nNext steps:');
    console.log('  1. Copy .ovpn files to server: scp -i $PEM ovpn-configs/*.ovpn root@47.76.82.51:/tmp/');
    console.log('  2. Deploy on server: sudo vpn-manager.sh deploy-batch /tmp/manifest.json');
    console.log('  3. Register in Sub2API via admin panel');
  } else {
    console.log('No configs downloaded.');
    console.log(`You can manually download .ovpn files and place them in: ${OUTPUT_DIR}`);
  }

  console.log('\nPress Ctrl+C to exit.');
  await new Promise(() => {});
}

main().catch(err => { console.error('Error:', err.message); process.exit(1); });
