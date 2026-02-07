#!/bin/bash
# Clawdbot post-installation setup script

echo ""
echo "Setting up Clawdbot environment..."
echo ""

# Create init file for first login
cat > /home/clawdbot/.clawdbot-init << 'INIT'
echo ""
echo "  Clawdbot Setup Instructions"
echo ""
echo "  1. Configure Clawdbot: nano ~/.clawdbot/config.yml"
echo "  2. Login to provider (WhatsApp/Telegram/Signal): clawdbot login"
echo "  3. Test gateway: clawdbot gateway"
echo ""
echo "  Docs: https://docs.clawd.bot"
echo ""
rm -f ~/.clawdbot-init
INIT

# Add to bashrc for one-time display
grep -q 'clawdbot-init' /home/clawdbot/.bashrc 2>/dev/null || \
  echo '[ -f ~/.clawdbot-init ] && source ~/.clawdbot-init' >> /home/clawdbot/.bashrc

echo "Switching to clawdbot user..."
sudo -u clawdbot -i bash
