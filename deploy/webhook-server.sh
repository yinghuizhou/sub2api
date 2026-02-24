#!/bin/bash
# =============================================================================
# Sub2API Webhook 接收服务
# 用途：在服务器上监听 Gitee Webhook，自动触发部署
# 端口：9000
# 实现：基于 Python3 http.server，支持并发、安全 token 校验、flock 部署锁
# =============================================================================

set -euo pipefail

WEBHOOK_PORT="${WEBHOOK_PORT:-9000}"
WEBHOOK_SECRET="${WEBHOOK_SECRET:?ERROR: WEBHOOK_SECRET environment variable is required}"
DEPLOY_SCRIPT="${DEPLOY_SCRIPT:-/opt/sub2api/deploy.sh}"
LOG_FILE="${LOG_FILE:-/var/log/sub2api-webhook.log}"

# Ensure log file exists
touch "$LOG_FILE"
chmod 644 "$LOG_FILE"

echo "[$(date +'%Y-%m-%d %H:%M:%S')] Starting Sub2API Webhook Server on port $WEBHOOK_PORT" | tee -a "$LOG_FILE"

exec python3 - "$WEBHOOK_PORT" "$WEBHOOK_SECRET" "$DEPLOY_SCRIPT" "$LOG_FILE" << 'PYTHON_EOF'
import sys
import os
import hmac
import json
import re
import subprocess
import threading
import fcntl
import logging
from http.server import HTTPServer, BaseHTTPRequestHandler
from datetime import datetime

# Read config from command-line args
WEBHOOK_PORT = int(sys.argv[1])
WEBHOOK_SECRET = sys.argv[2]
DEPLOY_SCRIPT = sys.argv[3]
LOG_FILE = sys.argv[4]
DEPLOY_LOCK = "/var/lock/sub2api-webhook-deploy.lock"

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format="[%(asctime)s] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
    handlers=[
        logging.FileHandler(LOG_FILE),
        logging.StreamHandler(sys.stdout),
    ],
)
logger = logging.getLogger("webhook")


def sanitize(text: str) -> str:
    """Strip control characters from text to prevent log injection."""
    return re.sub(r"[\x00-\x08\x0b\x0c\x0e-\x1f\x7f]", "", str(text))


def run_deploy(body: str) -> None:
    """Execute the deploy script under flock to prevent concurrent deploys."""
    try:
        fd = os.open(DEPLOY_LOCK, os.O_CREAT | os.O_RDWR, 0o644)
        try:
            fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except OSError:
            logger.warning("Another deploy is already running, skipping")
            os.close(fd)
            return

        logger.info("Starting deploy...")
        result = subprocess.run(
            ["bash", DEPLOY_SCRIPT],
            capture_output=True,
            text=True,
            timeout=600,  # 10 minute timeout
        )

        if result.returncode == 0:
            logger.info("Deploy succeeded")
        else:
            logger.error("Deploy failed (exit code %d)", result.returncode)
            if result.stderr:
                logger.error("stderr: %s", sanitize(result.stderr[-2000:]))

        if result.stdout:
            logger.info("stdout: %s", sanitize(result.stdout[-2000:]))

        fcntl.flock(fd, fcntl.LOCK_UN)
        os.close(fd)

    except subprocess.TimeoutExpired:
        logger.error("Deploy timed out after 600 seconds")
    except Exception as e:
        logger.error("Deploy error: %s", sanitize(str(e)))


class WebhookHandler(BaseHTTPRequestHandler):
    """HTTP handler for Gitee webhook requests."""

    # Suppress default access log
    def log_message(self, format, *args):
        logger.info("HTTP %s", sanitize(format % args))

    def do_GET(self):
        """Health check endpoint."""
        if self.path == "/health":
            self._respond(200, {"status": "ok"})
        else:
            self._respond(404, {"error": "Not Found"})

    def do_POST(self):
        """Handle webhook POST requests."""
        if self.path != "/webhook":
            self._respond(404, {"error": "Not Found"})
            return

        # Read request body
        content_length = int(self.headers.get("Content-Length", 0))
        if content_length > 1_048_576:  # 1MB limit
            self._respond(413, {"error": "Payload too large"})
            return

        body = self.rfile.read(content_length).decode("utf-8", errors="replace")

        # Verify token using constant-time comparison
        token = self.headers.get("X-Gitee-Token", "")
        if not hmac.compare_digest(token, WEBHOOK_SECRET):
            logger.warning("Token verification failed from %s", self.client_address[0])
            self._respond(401, {"error": "Unauthorized"})
            return

        logger.info("Received valid webhook request from %s", self.client_address[0])

        # Parse payload and check branch
        try:
            payload = json.loads(body)
            ref = sanitize(payload.get("ref", "unknown"))
            commit = sanitize(str(payload.get("after", payload.get("commit", "unknown")))[:12])
            logger.info("ref=%s commit=%s", ref, commit)

            # Only deploy on main branch push
            if ref not in ("refs/heads/main", "refs/heads/master"):
                logger.info("Skipping deploy for non-main branch: %s", ref)
                self._respond(200, {"status": "skipped", "reason": "not main branch"})
                return
        except (json.JSONDecodeError, AttributeError):
            logger.info("Body (non-JSON): %s", sanitize(body[:200]))

        # Run deploy asynchronously
        thread = threading.Thread(target=run_deploy, args=(body,), daemon=True)
        thread.start()

        # Respond immediately
        self._respond(200, {"status": "deploying"})

    def _respond(self, status_code: int, data: dict) -> None:
        """Send JSON response."""
        body = json.dumps(data).encode("utf-8")
        self.send_response(status_code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)


class ThreadedHTTPServer(HTTPServer):
    """Handle requests in separate threads for concurrency."""
    allow_reuse_address = True

    def process_request(self, request, client_address):
        thread = threading.Thread(target=self._handle, args=(request, client_address))
        thread.daemon = True
        thread.start()

    def _handle(self, request, client_address):
        try:
            self.finish_request(request, client_address)
        except Exception:
            self.handle_error(request, client_address)
        finally:
            self.shutdown_request(request)


def main():
    server = ThreadedHTTPServer(("0.0.0.0", WEBHOOK_PORT), WebhookHandler)
    logger.info("Webhook server listening on 0.0.0.0:%d", WEBHOOK_PORT)
    logger.info("Deploy script: %s", DEPLOY_SCRIPT)

    try:
        server.serve_forever()
    except KeyboardInterrupt:
        logger.info("Shutting down webhook server")
        server.shutdown()


if __name__ == "__main__":
    main()

PYTHON_EOF
