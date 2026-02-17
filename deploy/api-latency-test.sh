#!/usr/bin/env bash
# =============================================================================
# Sub2API Latency Test Script
# =============================================================================
# Usage:
#   ./api-latency-test.sh <domain> [options]
#
# Examples:
#   ./api-latency-test.sh api.example.com                    # Basic test, 50 rounds
#   ./api-latency-test.sh api.example.com -n 100             # 100 rounds
#   ./api-latency-test.sh api.example.com -n 0               # Infinite loop (Ctrl+C to stop)
#   ./api-latency-test.sh api.example.com -i 3               # 3 second interval
#   ./api-latency-test.sh api.example.com -k YOUR_API_KEY    # Test AI gateway endpoint
#   ./api-latency-test.sh api.example.com --report-only      # Only show final report from log
# =============================================================================

set -euo pipefail

# ─── Defaults ────────────────────────────────────────────────────────────────
DOMAIN=""
ROUNDS=50
INTERVAL=2
API_KEY=""
REPORT_ONLY=false
LOG_DIR="./latency-logs"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# ─── Colors ──────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# ─── Parse Arguments ────────────────────────────────────────────────────────
usage() {
    echo "Usage: $0 <domain> [options]"
    echo ""
    echo "Options:"
    echo "  -n <num>       Number of test rounds (default: 50, 0 = infinite)"
    echo "  -i <seconds>   Interval between tests (default: 2)"
    echo "  -k <api_key>   API Key for /v1/messages endpoint test"
    echo "  --report-only  Only generate report from existing log"
    echo "  -h, --help     Show this help"
    exit 1
}

[[ $# -lt 1 ]] && usage
DOMAIN="$1"; shift

while [[ $# -gt 0 ]]; do
    case "$1" in
        -n) ROUNDS="$2"; shift 2 ;;
        -i) INTERVAL="$2"; shift 2 ;;
        -k) API_KEY="$2"; shift 2 ;;
        --report-only) REPORT_ONLY=true; shift ;;
        -h|--help) usage ;;
        *) echo "Unknown option: $1"; usage ;;
    esac
done

# Strip protocol prefix if provided
DOMAIN="${DOMAIN#https://}"
DOMAIN="${DOMAIN#http://}"
DOMAIN="${DOMAIN%/}"

BASE_URL="https://${DOMAIN}"
LOG_FILE="${LOG_DIR}/${DOMAIN//\//_}_${TIMESTAMP}.csv"

mkdir -p "$LOG_DIR"

# ─── Report Generator ───────────────────────────────────────────────────────
generate_report() {
    local csv_file="$1"

    if [[ ! -f "$csv_file" ]] || [[ $(wc -l < "$csv_file") -lt 2 ]]; then
        echo -e "${RED}No data to report.${NC}"
        return
    fi

    echo ""
    echo -e "${BOLD}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${BOLD}  API Latency Test Report${NC}"
    echo -e "${BOLD}  Domain: ${CYAN}${DOMAIN}${NC}"
    echo -e "${BOLD}  Time:   $(date '+%Y-%m-%d %H:%M:%S')${NC}"
    echo -e "${BOLD}═══════════════════════════════════════════════════════════════${NC}"
    echo ""

    # Process each endpoint from CSV
    local endpoints
    endpoints=$(tail -n +2 "$csv_file" | cut -d',' -f2 | sort -u)

    for endpoint in $endpoints; do
        local data
        data=$(tail -n +2 "$csv_file" | awk -F',' -v ep="$endpoint" '$2 == ep')

        local total success fail
        total=$(echo "$data" | wc -l | tr -d ' ')
        success=$(echo "$data" | awk -F',' '$7 >= 200 && $7 < 400' | wc -l | tr -d ' ')
        fail=$((total - success))
        local success_rate
        if [[ $total -gt 0 ]]; then
            success_rate=$(awk "BEGIN {printf \"%.1f\", ($success/$total)*100}")
        else
            success_rate="0.0"
        fi

        # Extract TTFB values (column 5) for successful requests
        local ttfb_values
        ttfb_values=$(echo "$data" | awk -F',' '$7 >= 200 && $7 < 400 {print $5}')

        if [[ -z "$ttfb_values" ]]; then
            echo -e "${BOLD}[ ${endpoint} ]${NC} — ${RED}All requests failed${NC}"
            echo ""
            continue
        fi

        # Calculate stats using awk
        local stats
        stats=$(echo "$ttfb_values" | awk '
        BEGIN { min=999999; max=0; sum=0; n=0 }
        {
            val = $1 * 1000;  # Convert seconds to ms
            if (val < min) min = val;
            if (val > max) max = val;
            sum += val;
            n++;
            values[n] = val;
        }
        END {
            if (n == 0) { print "0 0 0 0 0 0"; exit }
            avg = sum / n;

            # Sort for percentiles
            for (i = 1; i <= n; i++)
                for (j = i+1; j <= n; j++)
                    if (values[i] > values[j]) {
                        t = values[i]; values[i] = values[j]; values[j] = t;
                    }

            p50_idx = int(n * 0.5);  if (p50_idx < 1) p50_idx = 1;
            p90_idx = int(n * 0.9);  if (p90_idx < 1) p90_idx = 1;
            p99_idx = int(n * 0.99); if (p99_idx < 1) p99_idx = 1;

            printf "%.0f %.0f %.0f %.0f %.0f %.0f", min, avg, values[p50_idx], values[p90_idx], values[p99_idx], max;
        }')

        local min avg p50 p90 p99 max_val
        read -r min avg p50 p90 p99 max_val <<< "$stats"

        # Color code based on avg latency
        local color
        if (( avg < 100 )); then color="$GREEN";
        elif (( avg < 200 )); then color="$YELLOW";
        else color="$RED"; fi

        echo -e "${BOLD}[ ${endpoint} ]${NC}"
        echo -e "  Requests: ${total}  |  Success: ${GREEN}${success}${NC}  |  Failed: ${RED}${fail}${NC}  |  Rate: ${success_rate}%"
        echo ""
        echo -e "  ${BOLD}Latency (TTFB):${NC}"
        echo -e "  ┌──────────┬──────────┬──────────┬──────────┬──────────┬──────────┐"
        echo -e "  │  ${BOLD}Min${NC}     │  ${BOLD}Avg${NC}     │  ${BOLD}P50${NC}     │  ${BOLD}P90${NC}     │  ${BOLD}P99${NC}     │  ${BOLD}Max${NC}     │"
        echo -e "  ├──────────┼──────────┼──────────┼──────────┼──────────┼──────────┤"
        printf "  │ ${color}%5dms${NC}  │ ${color}%5dms${NC}  │ ${color}%5dms${NC}  │ ${color}%5dms${NC}  │ ${color}%5dms${NC}  │ ${color}%5dms${NC}  │\n" \
            "$min" "$avg" "$p50" "$p90" "$p99" "$max_val"
        echo -e "  └──────────┴──────────┴──────────┴──────────┴──────────┴──────────┘"

        # Latency distribution bar chart
        local under50 under100 under200 under500 over500
        under50=$(echo "$ttfb_values" | awk '{v=$1*1000} v<50' | wc -l | tr -d ' ')
        under100=$(echo "$ttfb_values" | awk '{v=$1*1000} v>=50 && v<100' | wc -l | tr -d ' ')
        under200=$(echo "$ttfb_values" | awk '{v=$1*1000} v>=100 && v<200' | wc -l | tr -d ' ')
        under500=$(echo "$ttfb_values" | awk '{v=$1*1000} v>=200 && v<500' | wc -l | tr -d ' ')
        over500=$(echo "$ttfb_values" | awk '{v=$1*1000} v>=500' | wc -l | tr -d ' ')

        local s_total=$success
        echo ""
        echo -e "  ${BOLD}Distribution:${NC}"
        for label_val in "  <50ms:$under50" " 50-100ms:$under100" "100-200ms:$under200" "200-500ms:$under500" "  >500ms:$over500"; do
            local label="${label_val%%:*}"
            local count="${label_val##*:}"
            local pct=0
            [[ $s_total -gt 0 ]] && pct=$((count * 40 / s_total))
            local bar=""
            for ((b=0; b<pct; b++)); do bar+="█"; done
            printf "  %9s │ %-40s %d\n" "$label" "$bar" "$count"
        done
        echo ""

        # Cloudflare detection
        local cf_check
        cf_check=$(echo "$data" | head -1 | cut -d',' -f8)
        if [[ "$cf_check" == "yes" ]]; then
            local cf_location
            cf_location=$(echo "$data" | head -1 | cut -d',' -f9)
            echo -e "  Cloudflare: ${GREEN}✓ Active${NC} (Edge: ${CYAN}${cf_location}${NC})"
        else
            echo -e "  Cloudflare: ${RED}✗ Not detected${NC}"
        fi
        echo ""
    done

    # Timeline (last 20 data points per endpoint)
    echo -e "${BOLD}─── Timeline (recent TTFB) ────────────────────────────────────${NC}"
    for endpoint in $endpoints; do
        echo -e "  ${BOLD}${endpoint}:${NC}"
        tail -n +2 "$csv_file" | awk -F',' -v ep="$endpoint" '
        $2 == ep {
            ttfb = $5 * 1000;
            time = $1;
            # Extract HH:MM:SS from timestamp
            split(time, a, " ");
            t = a[2];
            if (ttfb < 100) color = "🟢";
            else if (ttfb < 200) color = "🟡";
            else if (ttfb < 500) color = "🟠";
            else color = "🔴";
            printf "  %s %s %4.0fms  ", t, color, ttfb;
            bars = int(ttfb / 10);
            if (bars > 60) bars = 60;
            for (i = 0; i < bars; i++) printf "▇";
            printf "\n";
        }' | tail -20
        echo ""
    done

    echo -e "${BOLD}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "  Log file: ${CYAN}${csv_file}${NC}"
    echo -e "${BOLD}═══════════════════════════════════════════════════════════════${NC}"
}

# ─── Report Only Mode ────────────────────────────────────────────────────────
if $REPORT_ONLY; then
    latest_log=$(ls -t "${LOG_DIR}"/*.csv 2>/dev/null | head -1)
    if [[ -z "$latest_log" ]]; then
        echo -e "${RED}No log files found in ${LOG_DIR}/${NC}"
        exit 1
    fi
    echo -e "Using log: ${CYAN}${latest_log}${NC}"
    generate_report "$latest_log"
    exit 0
fi

# ─── Test Functions ──────────────────────────────────────────────────────────
test_endpoint() {
    local endpoint="$1"
    local method="${2:-GET}"
    local data="${3:-}"
    local headers="${4:-}"

    local curl_cmd="curl -s -o /dev/null -w '%{time_namelookup},%{time_connect},%{time_appconnect},%{time_starttransfer},%{time_total},%{http_code}' -X ${method}"

    # Add headers
    if [[ -n "$headers" ]]; then
        while IFS= read -r h; do
            curl_cmd+=" -H '${h}'"
        done <<< "$headers"
    fi

    # Add data
    if [[ -n "$data" ]]; then
        curl_cmd+=" -d '${data}'"
    fi

    curl_cmd+=" --max-time 30 '${BASE_URL}${endpoint}'"

    local result
    result=$(eval "$curl_cmd" 2>/dev/null || echo "0,0,0,0,0,000")

    local dns connect tls ttfb total http_code
    IFS=',' read -r dns connect tls ttfb total http_code <<< "$result"

    # Check for Cloudflare headers
    local cf_ray cf_detected="no" cf_location=""
    cf_ray=$(curl -sI --max-time 10 "${BASE_URL}${endpoint}" 2>/dev/null | grep -i "cf-ray:" | head -1 || true)
    if [[ -n "$cf_ray" ]]; then
        cf_detected="yes"
        cf_location=$(echo "$cf_ray" | sed 's/.*-//' | tr -d '[:space:]')
    fi

    # Write CSV row
    echo "$(date '+%Y-%m-%d %H:%M:%S'),${endpoint},${dns},${connect},${ttfb},${total},${http_code},${cf_detected},${cf_location}" >> "$LOG_FILE"

    # Live output
    local ttfb_ms
    ttfb_ms=$(awk "BEGIN {printf \"%.0f\", ${ttfb} * 1000}")
    local color
    if (( ttfb_ms < 100 )); then color="$GREEN";
    elif (( ttfb_ms < 200 )); then color="$YELLOW";
    elif (( ttfb_ms < 500 )); then color="$YELLOW";
    else color="$RED"; fi

    local status_color="$GREEN"
    [[ "$http_code" -ge 400 ]] && status_color="$RED"
    [[ "$http_code" == "000" ]] && status_color="$RED" && http_code="TIMEOUT"

    printf "  %-25s  ${status_color}%s${NC}  DNS:${BLUE}%5.0fms${NC}  TCP:${BLUE}%5.0fms${NC}  TLS:${BLUE}%5.0fms${NC}  TTFB:${color}%5.0fms${NC}  Total:${BLUE}%5.0fms${NC}" \
        "$endpoint" "$http_code" \
        "$(awk "BEGIN{printf \"%.0f\",$dns*1000}")" \
        "$(awk "BEGIN{printf \"%.0f\",($connect-$dns)*1000}")" \
        "$(awk "BEGIN{printf \"%.0f\",($tls-$connect)*1000}")" \
        "$ttfb_ms" \
        "$(awk "BEGIN{printf \"%.0f\",$total*1000}")"

    if [[ "$cf_detected" == "yes" ]]; then
        printf "  ${CYAN}CF:${cf_location}${NC}"
    fi
    echo ""
}

# ─── Main ────────────────────────────────────────────────────────────────────

# CSV header
echo "timestamp,endpoint,dns_s,connect_s,ttfb_s,total_s,http_code,cloudflare,cf_location" > "$LOG_FILE"

echo -e "${BOLD}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${BOLD}  Sub2API Latency Test${NC}"
echo -e "  Target:   ${CYAN}${BASE_URL}${NC}"
echo -e "  Rounds:   ${ROUNDS} (0 = infinite)"
echo -e "  Interval: ${INTERVAL}s"
echo -e "  Log:      ${CYAN}${LOG_FILE}${NC}"
if [[ -n "$API_KEY" ]]; then
    echo -e "  API Key:  ${GREEN}configured (will test /v1/messages)${NC}"
fi
echo -e "${BOLD}═══════════════════════════════════════════════════════════════${NC}"
echo ""

# Trap Ctrl+C to show report before exit
trap 'echo -e "\n\n${YELLOW}Interrupted. Generating report...${NC}"; generate_report "$LOG_FILE"; exit 0' INT

round=0
while true; do
    round=$((round + 1))

    if [[ "$ROUNDS" -gt 0 ]] && [[ $round -gt "$ROUNDS" ]]; then
        break
    fi

    echo -e "${BOLD}── Round ${round}$([ "$ROUNDS" -gt 0 ] && echo "/${ROUNDS}") ── $(date '+%H:%M:%S') ──${NC}"

    # Test health endpoint
    test_endpoint "/health"

    # Test frontend (root page)
    test_endpoint "/"

    # Test API endpoint (if API key provided)
    if [[ -n "$API_KEY" ]]; then
        test_endpoint "/v1/messages" "POST" \
            '{"model":"claude-sonnet-4-20250514","max_tokens":1,"messages":[{"role":"user","content":"hi"}]}' \
            "x-api-key: ${API_KEY}
anthropic-version: 2023-06-01
content-type: application/json"
    fi

    echo ""
    sleep "$INTERVAL"
done

# Final report
generate_report "$LOG_FILE"
