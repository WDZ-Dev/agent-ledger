# AgentLedger Promotional Demo
# Usage: cd agent-ledger && vhs promo/demo.tape

Output promo/demo.mp4
Output promo/demo.gif

Set Shell "bash"
Set FontSize 16
Set Width 1100
Set Height 650
Set Padding 24
Set LetterSpacing 0
Set LineHeight 1.2
Set Theme { "name": "AgentLedger", "black": "#0d1117", "red": "#f85149", "green": "#56d364", "yellow": "#e3b341", "blue": "#388bfd", "magenta": "#bc8cff", "cyan": "#39c5cf", "white": "#c9d1d9", "brightBlack": "#484f58", "brightRed": "#f85149", "brightGreen": "#56d364", "brightYellow": "#e3b341", "brightBlue": "#388bfd", "brightMagenta": "#bc8cff", "brightCyan": "#39c5cf", "brightWhite": "#f0f6fc", "background": "#0d1117", "foreground": "#c9d1d9", "selection": "#264f78", "cursor": "#388bfd" }
Set TypingSpeed 40ms
Set CursorBlink false

# Hidden setup
Hide
Type `export PATH="$PWD/promo/bin:$PATH" PS1='$ ' && clear`
Enter
Sleep 300ms
Show

# ── Scene 1: Install ──
Sleep 500ms
Type "# Install AgentLedger"
Enter
Type "brew install wdz-dev/tap/agentledger"
Sleep 500ms
Enter
Sleep 2s

# ── Scene 2: One env var — zero code changes ──
Sleep 200ms
Type "# Point your agents at the proxy (zero code changes)"
Enter
Type "export OPENAI_BASE_URL=http://localhost:8787/v1"
Sleep 400ms
Enter
Sleep 1.5s

# ── Scene 3: Start the proxy ──
Type "# Start the proxy"
Enter
Type "agentledger serve"
Sleep 400ms
Enter

# Startup + request logs + loop detection + budget block
Sleep 12s

# ── Scene 4: Stop and check costs ──
Ctrl+C
Sleep 500ms

# Clear for readability and show cost report
Type "clear"
Enter
Sleep 300ms

Type "# What did our agents spend?"
Enter
Type "agentledger costs --last 24h"
Sleep 400ms
Enter
Sleep 3.5s

# ── Scene 5: Cost by agent ──
Type "agentledger costs --last 24h --by agent"
Sleep 400ms
Enter
Sleep 4s
