#!/usr/bin/env python3
"""Astrill OpenVPN Config Scraper

Uses Playwright to automate downloading OpenVPN configuration files
from Astrill VPN web panel, and optionally registers them with Sub2API.

Usage:
    python scraper.py                        # Full scrape + download
    python scraper.py --dry-run              # List servers only
    python scraper.py --register-only        # Register existing manifest
    python scraper.py --config path/to/config.json
"""

import argparse
import asyncio
import json
import logging
import os
import re
import sys
from datetime import datetime, timezone
from pathlib import Path

import httpx
from playwright.async_api import async_playwright, Page, Browser

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
logger = logging.getLogger("astrill-scraper")

# ---------------------------------------------------------------------------
# Config
# ---------------------------------------------------------------------------

def load_config(path: str) -> dict:
    """Load configuration from a JSON file."""
    config_path = Path(path)
    if not config_path.exists():
        logger.error("Config file not found: %s", path)
        logger.info("Copy config.example.json to config.json and fill in credentials.")
        sys.exit(1)

    with open(config_path, "r") as f:
        config = json.load(f)

    required = ["astrill_username", "astrill_password"]
    for key in required:
        if not config.get(key):
            logger.error("Missing required config key: %s", key)
            sys.exit(1)

    config.setdefault("output_dir", "/etc/openvpn/configs")
    config.setdefault("sub2api_url", "http://127.0.0.1:8080")
    config.setdefault("sub2api_api_key", "")
    config.setdefault("auto_register", False)
    config.setdefault("socks_port_start", 10801)
    return config


# ---------------------------------------------------------------------------
# Login
# ---------------------------------------------------------------------------

ASTRILL_LOGIN_URL = "https://members.astrill.com/log-in"


async def login(page: Page, username: str, password: str) -> None:
    """Login to Astrill web management panel."""
    logger.info("Navigating to Astrill login page...")
    await page.goto(ASTRILL_LOGIN_URL, wait_until="networkidle", timeout=30000)

    # TODO: Update selectors after inspecting Astrill web panel
    email_input = page.locator('input[name="email"], input[type="email"], #email')
    password_input = page.locator('input[name="password"], input[type="password"], #password')
    submit_btn = page.locator('button[type="submit"], input[type="submit"], .login-btn')

    await email_input.fill(username)
    await password_input.fill(password)
    await submit_btn.click()

    # Wait for navigation after login
    try:
        await page.wait_for_url("**/dashboard**", timeout=15000)
        logger.info("Login successful.")
    except Exception:
        # Take screenshot on failure for debugging
        screenshot_path = "/tmp/astrill-login-failure.png"
        await page.screenshot(path=screenshot_path, full_page=True)
        logger.error("Login failed. Screenshot saved to %s", screenshot_path)
        raise RuntimeError("Astrill login failed - check credentials and selectors")


# ---------------------------------------------------------------------------
# Server listing
# ---------------------------------------------------------------------------

# TODO: Update this URL after inspecting Astrill web panel
ASTRILL_VPN_CONFIG_URL = "https://members.astrill.com/tools/openvpn-configuration"


async def list_servers(page: Page) -> list[dict]:
    """Navigate to OpenVPN config page and enumerate available servers."""
    logger.info("Navigating to OpenVPN configuration page...")
    await page.goto(ASTRILL_VPN_CONFIG_URL, wait_until="networkidle", timeout=30000)

    servers = []

    # TODO: Update selectors after inspecting Astrill web panel
    # The actual DOM structure is unknown. Below is a best-guess approach
    # assuming a list/table of server entries.
    server_elements = await page.locator(
        '.server-list .server-item, '
        'table.servers tbody tr, '
        '.vpn-server-row'
    ).all()

    if not server_elements:
        # Fallback: try to find a select/dropdown of servers
        # TODO: Update selector after inspecting Astrill web panel
        select_options = await page.locator(
            'select.server-select option, '
            'select#server option, '
            '.server-dropdown option'
        ).all()
        for opt in select_options:
            value = await opt.get_attribute("value")
            text = (await opt.text_content() or "").strip()
            if value and text:
                parsed = _parse_server_name(text)
                parsed["value"] = value
                servers.append(parsed)
    else:
        for el in server_elements:
            text = (await el.text_content() or "").strip()
            if not text:
                continue
            parsed = _parse_server_name(text)

            # TODO: Update selectors for dedicated flag
            dedicated_el = el.locator('.dedicated, .premium, [data-dedicated]')
            parsed["is_dedicated"] = await dedicated_el.count() > 0

            # Try to get a download link or identifier
            # TODO: Update selector after inspecting Astrill web panel
            link = el.locator('a[href*="download"], button.download')
            if await link.count() > 0:
                parsed["download_href"] = await link.first.get_attribute("href")

            servers.append(parsed)

    logger.info("Found %d servers.", len(servers))
    return servers


# Region mapping for common Astrill server name patterns
REGION_MAP = {
    "US": "us", "Canada": "ca", "UK": "eu-west", "Germany": "eu-central",
    "Netherlands": "eu-west", "France": "eu-west", "Japan": "ap-northeast",
    "Singapore": "ap-southeast", "Hong Kong": "ap-east", "Australia": "ap-southeast",
    "South Korea": "ap-northeast", "Taiwan": "ap-east", "India": "ap-south",
    "Brazil": "sa-east", "Mexico": "na-south",
}

# Country code mapping
COUNTRY_CODE_MAP = {
    "US": "US", "United States": "US", "Canada": "CA", "UK": "GB",
    "United Kingdom": "GB", "Germany": "DE", "Netherlands": "NL",
    "France": "FR", "Japan": "JP", "Singapore": "SG", "Hong Kong": "HK",
    "Australia": "AU", "South Korea": "KR", "Taiwan": "TW", "India": "IN",
    "Brazil": "BR", "Mexico": "MX", "Sweden": "SE", "Switzerland": "CH",
    "Italy": "IT", "Spain": "ES", "Ireland": "IE", "Norway": "NO",
}


def _parse_server_name(raw_name: str) -> dict:
    """Parse a server name like 'US - Chicago #1' into structured data."""
    # Common patterns: "US - Chicago #1", "Germany - Frankfurt", "Japan - Tokyo #2"
    match = re.match(r"^(.+?)\s*[-–]\s*(.+?)(?:\s*#(\d+))?$", raw_name.strip())
    if match:
        country_part = match.group(1).strip()
        city = match.group(2).strip()
        number = match.group(3)
    else:
        country_part = raw_name.strip()
        city = ""
        number = None

    country_code = COUNTRY_CODE_MAP.get(country_part, country_part[:2].upper())
    region = REGION_MAP.get(country_part, "unknown")

    return {
        "server_name": raw_name.strip(),
        "region": region,
        "country": country_code,
        "city": city,
        "is_dedicated": False,
        "number": number,
    }


def _to_instance_name(server: dict) -> str:
    """Generate a slug instance name from server data."""
    parts = []
    parts.append(server["country"].lower())
    if server["city"]:
        city_slug = re.sub(r"[^a-z0-9]+", "-", server["city"].lower()).strip("-")
        parts.append(city_slug)
    if server.get("number"):
        parts.append(server["number"])
    return "-".join(parts)


# ---------------------------------------------------------------------------
# Download configs
# ---------------------------------------------------------------------------


async def download_config(page: Page, server: dict, output_dir: str) -> str | None:
    """Download a single .ovpn config file for a server.

    Returns the path to the saved file, or None on failure.
    """
    instance_name = _to_instance_name(server)
    out_path = os.path.join(output_dir, f"{instance_name}.ovpn")

    try:
        if server.get("download_href"):
            # Direct download link available
            # TODO: Update selector after inspecting Astrill web panel
            async with page.expect_download(timeout=15000) as download_info:
                await page.goto(server["download_href"])
            download = await download_info.value
            await download.save_as(out_path)
        elif server.get("value"):
            # Select from dropdown and trigger download
            # TODO: Update selectors after inspecting Astrill web panel
            await page.locator(
                'select.server-select, select#server, .server-dropdown'
            ).select_option(value=server["value"])
            await page.wait_for_timeout(500)

            async with page.expect_download(timeout=15000) as download_info:
                await page.locator(
                    'button.download-btn, a.download-config, #download-ovpn'
                ).click()
            download = await download_info.value
            await download.save_as(out_path)
        else:
            logger.warning("No download mechanism for server: %s", server["server_name"])
            return None

        logger.info("Downloaded: %s -> %s", server["server_name"], out_path)
        return out_path

    except Exception as e:
        logger.error("Failed to download config for %s: %s", server["server_name"], e)
        return None


async def batch_download(page: Page, servers: list[dict], output_dir: str) -> list[str]:
    """Download all .ovpn configs. Returns list of successfully saved paths."""
    os.makedirs(output_dir, exist_ok=True)
    saved = []
    for i, server in enumerate(servers, 1):
        logger.info("Downloading [%d/%d]: %s", i, len(servers), server["server_name"])
        path = await download_config(page, server, output_dir)
        if path:
            saved.append(path)
        # Polite delay between downloads
        await page.wait_for_timeout(1000)
    logger.info("Downloaded %d/%d configs.", len(saved), len(servers))
    return saved


# ---------------------------------------------------------------------------
# Manifest generation
# ---------------------------------------------------------------------------


def generate_manifest(
    servers: list[dict], output_dir: str, socks_port_start: int
) -> list[dict]:
    """Generate JSON manifest compatible with vpn-manager.sh deploy-batch format."""
    manifest = []
    port = socks_port_start

    for server in servers:
        instance_name = _to_instance_name(server)
        ovpn_file = os.path.join(output_dir, f"{instance_name}.ovpn")

        entry = {
            "server_name": server["server_name"],
            "instance_name": instance_name,
            "region": server["region"],
            "country": server["country"],
            "city": server["city"],
            "ovpn_file": ovpn_file,
            "socks_port": port,
            "is_dedicated": server.get("is_dedicated", False),
        }
        manifest.append(entry)
        port += 1

    logger.info("Generated manifest with %d entries (ports %d-%d).",
                len(manifest), socks_port_start, port - 1)
    return manifest


# ---------------------------------------------------------------------------
# Sub2API registration
# ---------------------------------------------------------------------------


async def register_proxies(
    manifest: list[dict], api_url: str, api_key: str
) -> None:
    """Register proxies with Sub2API Admin API."""
    if not api_key:
        logger.error("Cannot register proxies: sub2api_api_key is not set.")
        return

    endpoint = f"{api_url.rstrip('/')}/api/v1/admin/proxies/batch"
    headers = {
        "x-api-key": api_key,
        "Content-Type": "application/json",
    }

    payload = []
    for entry in manifest:
        payload.append({
            "name": entry["instance_name"],
            "protocol": "socks5",
            "host": "127.0.0.1",
            "port": entry["socks_port"],
            "region": entry["region"],
            "country": entry["country"],
            "city": entry["city"],
            "group_name": f"astrill-{entry['region']}",
            "is_dedicated": entry["is_dedicated"],
            "ovpn_config_path": entry["ovpn_file"],
        })

    logger.info("Registering %d proxies with Sub2API at %s ...", len(payload), endpoint)

    async with httpx.AsyncClient(timeout=30.0) as client:
        try:
            resp = await client.post(endpoint, headers=headers, json=payload)
            if resp.status_code in (200, 201):
                logger.info("Successfully registered %d proxies.", len(payload))
            else:
                logger.error("Registration failed: HTTP %d - %s",
                             resp.status_code, resp.text[:500])
        except httpx.HTTPError as e:
            logger.error("HTTP error during registration: %s", e)


# ---------------------------------------------------------------------------
# CLI & main
# ---------------------------------------------------------------------------


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Astrill OpenVPN Config Scraper for Sub2API"
    )
    parser.add_argument(
        "--config", default="config.json",
        help="Path to config JSON file (default: config.json)",
    )
    parser.add_argument(
        "--dry-run", action="store_true",
        help="List available servers without downloading configs",
    )
    parser.add_argument(
        "--register-only", action="store_true",
        help="Register existing manifest.json without scraping",
    )
    parser.add_argument(
        "--headed", action="store_true",
        help="Run browser in headed mode for debugging",
    )
    return parser.parse_args()


async def main() -> None:
    args = parse_args()
    config = load_config(args.config)

    # --register-only: skip scraping, just register from manifest
    if args.register_only:
        manifest_path = os.path.join(config["output_dir"], "manifest.json")
        if not os.path.exists(manifest_path):
            logger.error("Manifest not found: %s", manifest_path)
            sys.exit(1)
        with open(manifest_path, "r") as f:
            manifest = json.load(f)
        logger.info("Loaded %d entries from existing manifest.", len(manifest))
        await register_proxies(manifest, config["sub2api_url"], config["sub2api_api_key"])
        return

    async with async_playwright() as p:
        browser = await p.chromium.launch(headless=not args.headed)
        context = await browser.new_context(
            user_agent=(
                "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "
                "AppleWebKit/537.36 (KHTML, like Gecko) "
                "Chrome/120.0.0.0 Safari/537.36"
            )
        )
        page = await context.new_page()

        try:
            await login(page, config["astrill_username"], config["astrill_password"])
            servers = await list_servers(page)

            if not servers:
                logger.warning("No servers found. Check selectors and login state.")
                screenshot_path = "/tmp/astrill-no-servers.png"
                await page.screenshot(path=screenshot_path, full_page=True)
                logger.info("Screenshot saved to %s", screenshot_path)
                return

            # Print server list
            for i, s in enumerate(servers, 1):
                ded = " [DEDICATED]" if s.get("is_dedicated") else ""
                print(f"  {i:3d}. {s['server_name']}{ded}")

            if args.dry_run:
                logger.info("Dry run complete. %d servers found.", len(servers))
                return

            # Download configs
            await batch_download(page, servers, config["output_dir"])

            # Generate and save manifest
            manifest = generate_manifest(
                servers, config["output_dir"], config["socks_port_start"]
            )
            manifest_path = os.path.join(config["output_dir"], "manifest.json")
            os.makedirs(config["output_dir"], exist_ok=True)
            with open(manifest_path, "w") as f:
                json.dump(manifest, f, indent=2)
            logger.info("Manifest saved to %s", manifest_path)

            # Optional: register with Sub2API
            if config["auto_register"]:
                await register_proxies(
                    manifest, config["sub2api_url"], config["sub2api_api_key"]
                )

        except Exception:
            logger.exception("Scraper encountered an error.")
            screenshot_path = "/tmp/astrill-error.png"
            try:
                await page.screenshot(path=screenshot_path, full_page=True)
                logger.info("Error screenshot saved to %s", screenshot_path)
            except Exception:
                pass
            raise
        finally:
            await browser.close()


if __name__ == "__main__":
    asyncio.run(main())
