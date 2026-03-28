#!/bin/bash
# Demo script for AgentLedger promotional video
# This simulates realistic output for the recording

# Colors
CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
BOLD='\033[1m'
DIM='\033[2m'
NC='\033[0m'

case "$1" in
  install)
    echo -e "${GREEN}==> Downloading agentledger v0.1.0${NC}"
    sleep 0.3
    echo -e "${GREEN}==> Installing agentledger to /opt/homebrew/bin/agentledger${NC}"
    sleep 0.3
    echo -e "🍺 agentledger was successfully installed!"
    ;;

  start)
    echo -e "${DIM}time=2026-03-27T10:00:01Z${NC} level=${CYAN}INFO${NC} msg=\"storage backend: sqlite\""
    sleep 0.15
    echo -e "${DIM}time=2026-03-27T10:00:01Z${NC} level=${CYAN}INFO${NC} msg=\"budget enforcement enabled\" daily_limit=\$50.00 monthly_limit=\$500.00"
    sleep 0.15
    echo -e "${DIM}time=2026-03-27T10:00:01Z${NC} level=${CYAN}INFO${NC} msg=\"agent session tracking enabled\" loop_threshold=20"
    sleep 0.15
    echo -e "${DIM}time=2026-03-27T10:00:01Z${NC} level=${CYAN}INFO${NC} msg=\"rate limiting enabled\" default_rpm=60"
    sleep 0.15
    echo -e "${DIM}time=2026-03-27T10:00:01Z${NC} level=${CYAN}INFO${NC} msg=\"dashboard enabled\" url=http://localhost:8787"
    sleep 0.15
    echo -e "${DIM}time=2026-03-27T10:00:01Z${NC} level=${CYAN}INFO${NC} msg=\"registered provider route\" prefix=/groq"
    sleep 0.1
    echo -e "${DIM}time=2026-03-27T10:00:01Z${NC} level=${CYAN}INFO${NC} msg=\"registered provider route\" prefix=/mistral"
    sleep 0.1
    echo -e "${DIM}time=2026-03-27T10:00:01Z${NC} level=${CYAN}INFO${NC} msg=\"registered provider route\" prefix=/deepseek"
    sleep 0.15
    echo -e "${DIM}time=2026-03-27T10:00:01Z${NC} level=${CYAN}INFO${NC} msg=${BOLD}\"starting HTTP server\"${NC} listen=:8787"
    sleep 0.3

    # Simulate incoming requests
    echo ""
    echo -e "${DIM}time=2026-03-27T10:00:05Z${NC} level=${CYAN}INFO${NC} msg=\"proxied request\" agent=code-reviewer session=sess_a1b2c3 model=gpt-4o tokens=2450 cost=\$0.0245"
    sleep 0.4
    echo -e "${DIM}time=2026-03-27T10:00:08Z${NC} level=${CYAN}INFO${NC} msg=\"proxied request\" agent=code-reviewer session=sess_a1b2c3 model=gpt-4o tokens=3100 cost=\$0.0310"
    sleep 0.4
    echo -e "${DIM}time=2026-03-27T10:00:12Z${NC} level=${CYAN}INFO${NC} msg=\"proxied request\" agent=research-bot session=sess_d4e5f6 model=claude-sonnet-4-20250514 tokens=4200 cost=\$0.0378"
    sleep 0.4
    echo -e "${DIM}time=2026-03-27T10:00:15Z${NC} level=${CYAN}INFO${NC} msg=\"proxied request\" agent=code-reviewer session=sess_a1b2c3 model=gpt-4o tokens=1800 cost=\$0.0180"
    sleep 0.4
    echo -e "${DIM}time=2026-03-27T10:00:18Z${NC} level=${CYAN}INFO${NC} msg=\"proxied request\" agent=deploy-agent session=sess_g7h8i9 model=gpt-4o-mini tokens=950 cost=\$0.0008"
    sleep 0.3

    # Loop detection
    echo ""
    echo -e "${DIM}time=2026-03-27T10:00:22Z${NC} level=${YELLOW}WARN${NC} msg=${BOLD}\"loop detected\"${NC} agent=research-bot session=sess_d4e5f6 repeated_calls=20 path=/v1/chat/completions"
    sleep 0.3
    echo -e "${DIM}time=2026-03-27T10:00:22Z${NC} level=${YELLOW}WARN${NC} msg=\"budget soft limit reached\" agent=research-bot spend=\$40.12 limit=\$50.00 pct=80%"
    sleep 0.3

    # More requests
    echo -e "${DIM}time=2026-03-27T10:00:25Z${NC} level=${CYAN}INFO${NC} msg=\"proxied request\" agent=deploy-agent session=sess_g7h8i9 model=gpt-4o-mini tokens=1200 cost=\$0.0010"
    sleep 0.3

    # Budget hard limit
    echo -e "${DIM}time=2026-03-27T10:00:30Z${NC} level=${RED}ERROR${NC} msg=${BOLD}\"request blocked: daily budget exceeded\"${NC} agent=research-bot spend=\$50.02 limit=\$50.00"
    sleep 0.2
    echo -e "${DIM}time=2026-03-27T10:00:30Z${NC} level=${CYAN}INFO${NC} msg=\"alert sent\" channel=slack agent=research-bot reason=\"daily budget exceeded\""
    ;;

  costs)
    echo ""
    echo -e "${BOLD}PROVIDER      MODEL                 REQUESTS    INPUT TOKENS    OUTPUT TOKENS    COST (USD)${NC}"
    echo    "--------      -----                 --------    ------------    -------------    ----------"
    echo    "openai        gpt-4o                      47          142,500           38,200       \$1.2150"
    echo    "openai        gpt-4o-mini                 83           95,000           28,500       \$0.0713"
    echo    "anthropic     claude-sonnet-4              22           68,000           18,400       \$0.5100"
    echo    "anthropic     claude-3-haiku              156           45,000           12,800       \$0.0225"
    echo    "groq          llama-3.3-70b                31           82,000           24,000       \$0.0085"
    echo    "--------      -----                 --------    ------------    -------------    ----------"
    echo -e "${BOLD}TOTAL                                339          432,500          121,900       \$1.8273${NC}"
    echo ""
    ;;

  costs-agent)
    echo ""
    echo -e "${BOLD}AGENT              REQUESTS    INPUT TOKENS    OUTPUT TOKENS    COST (USD)${NC}"
    echo    "-----              --------    ------------    -------------    ----------"
    echo    "code-reviewer           156          245,000           68,000       \$0.8450"
    echo    "research-bot             89          120,000           32,000       \$0.6200"
    echo    "deploy-agent             52           42,500           12,400       \$0.1823"
    echo -e "pr-summarizer            42           25,000            9,500       \$0.1800"
    echo    "--------           --------    ------------    -------------    ----------"
    echo -e "${BOLD}TOTAL                   339          432,500          121,900       \$1.8273${NC}"
    echo ""
    ;;
esac
