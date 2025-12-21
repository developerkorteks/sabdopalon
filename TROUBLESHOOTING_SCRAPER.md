# 🔧 Troubleshooting: Scraper Hanging Issues

## Problem: Scraper hangs at "Starting Telegram Client (gotd/td)..."

### Symptoms
```
[INFO] 🚀 Starting Telegram Client (gotd/td)...
[INFO] 📡 Connecting to Telegram servers...
[INFO] ⏳ This may take 10-30 seconds on first connection...
[INFO] 🔌 Client initialized, attempting to connect...
(hangs here - no further output)
```

### Common Causes

#### 1. **Network/Firewall Issues**
Telegram MTProto protocol might be blocked by:
- VPS firewall
- ISP blocking
- Network restrictions
- Regional censorship

**Solution:**
```bash
# Check if you can reach Telegram servers
curl -v https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getMe

# Check DNS resolution
nslookup venus.web.telegram.org

# Try with different DC (Data Center)
# Edit internal/client/client.go line 65:
DC: 4,  # Try DC 4, 5, or 1 instead of 2
```

#### 2. **Session File Issues**
Corrupted session file can cause hanging.

**Solution:**
```bash
# Remove old session and try again
rm session.json
go run cmd/scraper/main.go --phone +6287742028130
```

#### 3. **Timeout Too Short**
First connection can take 30-60 seconds depending on network.

**Solution:**
Just wait longer (up to 2 minutes on slow connections).

#### 4. **API Credentials Issue**
Wrong API ID/Hash can cause silent failures.

**Solution:**
Verify credentials in `cmd/scraper/main.go`:
```go
apiID := 22527852
apiHash := "4f595e6aac7dfe58a2cf6051360c3f14"
```

Get your own from: https://my.telegram.org/apps

### Testing Steps

#### Step 1: Test with Timeout
```bash
# Run test script
./tmp_rovodev_test_connection.sh
```

#### Step 2: Check Network
```bash
# Test Telegram API accessibility
curl -I https://api.telegram.org

# Test specific DC
nc -zv 149.154.167.50 443  # DC2
nc -zv 149.154.175.50 443  # DC4
```

#### Step 3: Enable Debug Mode
Edit `.env`:
```bash
DEBUG_MODE=true
```

Then run:
```bash
go run cmd/scraper/main.go --phone +6287742028130
```

#### Step 4: Try Different DC
Edit `internal/client/client.go`, line ~65:
```go
// Try different data centers
DC: 1,  // Netherlands
DC: 2,  // Singapore (default)
DC: 4,  // Amsterdam
DC: 5,  // Singapore
```

### Advanced Solutions

#### Use SOCKS5 Proxy
If Telegram is blocked, use a proxy:

1. Edit `internal/client/client.go`:
```go
import (
    "golang.org/x/net/proxy"
    "net"
)

// Add in telegram.Options:
Dialer: telegram.DialFunc(func(ctx context.Context, network, address string) (net.Conn, error) {
    // SOCKS5 proxy
    dialer, err := proxy.SOCKS5("tcp", "127.0.0.1:1080", nil, proxy.Direct)
    if err != nil {
        return nil, err
    }
    return dialer.Dial(network, address)
}),
```

2. Run with SSH tunnel:
```bash
# On your local machine
ssh -D 1080 -N user@your-vps

# Then run scraper
go run cmd/scraper/main.go --phone +6287742028130
```

#### Check gotd/td Logs
Add detailed logging:

```go
import (
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

// In Start() function, add:
zapLogger, _ := zap.NewDevelopment(zap.IncreaseLevel(zapcore.DebugLevel))

client := telegram.NewClient(c.appID, c.appHash, telegram.Options{
    SessionStorage: sessionStorage,
    Logger:         zapLogger,  // Add this
    // ... other options
})
```

### Quick Fixes Summary

```bash
# 1. Clean start
rm session.json
go run cmd/scraper/main.go --phone +6287742028130

# 2. With debug
DEBUG_MODE=true go run cmd/scraper/main.go --phone +6287742028130

# 3. Different DC (edit code first)
# Change DC: 2 to DC: 4 in internal/client/client.go

# 4. Test network
curl https://api.telegram.org
ping venus.web.telegram.org

# 5. Check if port 443 is open
telnet 149.154.167.50 443
```

### When to Use Bot vs Scraper

If scraper keeps hanging:

**Option 1: Use Bot Only (No Scraper)**
The bot can work without scraper if groups add the bot directly:
```bash
# Just run the bot
go run cmd/bot/main.go
```

Limitation: Bot can only read messages from groups where it's added as member.

**Option 2: Use Both**
Best setup for maximum coverage:
- Bot: Official bot for commands and summaries
- Scraper: User client to listen to all groups

### Still Not Working?

1. **Check VPS Provider**: Some providers block Telegram completely
2. **Try Different Region**: VPS in Singapore/EU generally works better
3. **Use VPN**: Connect VPS to VPN service
4. **Contact Support**: Share full error logs

### Getting Help

When asking for help, provide:
```bash
# Run this and share output
echo "=== System Info ==="
uname -a
echo ""
echo "=== Network Test ==="
curl -I https://api.telegram.org
echo ""
echo "=== Telegram DC Test ==="
nc -zv 149.154.167.50 443
echo ""
echo "=== Scraper Version ==="
go version
go list -m github.com/gotd/td
```

---

## Related Issues

- Network connectivity problems
- Regional Telegram restrictions
- Firewall/proxy configuration
- Session authentication issues

## See Also

- [QUICK_REFERENCE.md](docs/QUICK_REFERENCE.md) - Command reference
- [README.md](README.md) - Main documentation
- [gotd/td documentation](https://github.com/gotd/td) - Client library
