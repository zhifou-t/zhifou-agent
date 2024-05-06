#!/bin/bash

# Define colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}Starting zhifou agent installation...${NC}"

# Verify if the script is being run as root
if [[ $EUID -ne 0 ]]; then
   echo -e "${RED}This script must be run as root${NC}" >&2
   exit 1
fi
echo -e "${GREEN}Verified root access.${NC}"

# Verify if the system supports systemd
if [[ ! -d /etc/systemd/system/ ]]; then
    echo -e "${RED}System does not support systemd${NC}" >&2
    exit 1
fi
echo -e "${GREEN}System supports systemd.${NC}"

# Check if the machine supports IPv4
if ip -4 route list default >/dev/null 2>&1; then
    echo "IPv4 is supported."
else
    echo "IPv4 is not supported. Exiting..."
    exit 1
fi

# Parse command-line options
while getopts ":s:i:c:" opt; do
  case $opt in
    s) SERVER_URL="$OPTARG" ;;
    i) INTERVAL="$OPTARG" ;;
    c) CLIENT_ID="$OPTARG" ;;
    \?) echo -e "${RED}Invalid option: -$OPTARG${NC}" >&2; exit 1 ;;
    :) echo -e "${RED}Option -$OPTARG requires an argument.${NC}" >&2; exit 1 ;;
  esac
done
echo -e "${GREEN}Parsed command-line options.${NC}"

# Verify if all required parameters are provided
if [[ -z $SERVER_URL ]] || [[ -z $INTERVAL ]] || [[ -z $CLIENT_ID ]]; then
    echo -e "${RED}Usage: $0 -s <server_url> -i <interval> -c <client_id>${NC}" >&2
    exit 1
fi
echo -e "${GREEN}All required parameters provided.${NC}"

# Get the latest release version
latest_tag=$(curl -s https://api.github.com/repos/zhifou-t/zhifou-agent/releases/latest | grep -o '"tag_name": "[^"]*' | grep -o '[^"]*$')
echo -e "${GREEN}Latest release version fetched: $latest_tag${NC}"

# Determine the file name based on the current operating system
if [[ "$(uname -s)" == "Linux" ]]; then
    if [[ "$(uname -m)" == "x86_64" ]]; then
        suffix="linux-amd64"
    elif [[ "$(uname -m)" == "aarch64" ]]; then
        suffix="linux-arm64"
    else
        echo -e "${RED}Unsupported architecture${NC}" >&2
        exit 1
    fi
else
    echo -e "${RED}Unsupported operating system${NC}" >&2
    exit 1
fi
echo -e "${GREEN}Operating system and architecture supported.${NC}"

# Determine the file name based on the suffix
file_name="zhifou-agent-${suffix}"
echo -e "${GREEN}File name determined: $file_name${NC}"

# Download the latest release file
curl -sLO https://github.com/zhifou-t/zhifou-agent/releases/download/${latest_tag}/${file_name}
echo -e "${GREEN}Latest release file downloaded.${NC}"

# Move the file to /usr/local/bin/
mv ${file_name} /usr/local/bin/zhifou-agent
chmod +x /usr/local/bin/zhifou-agent
echo -e "${GREEN}File moved to /usr/local/bin/ and made executable.${NC}"

# Create systemd service unit file
cat <<EOF | sudo tee /etc/systemd/system/zhifou-agent.service >/dev/null
[Unit]
Description=zhifou agent
After=network.target

[Service]
Type=simple
Environment="SERVER_URL=${SERVER_URL}"
Environment="INTERVAL=${INTERVAL}"
Environment="CLIENT_ID=${CLIENT_ID}"
ExecStart=/usr/local/bin/zhifou-agent
Restart=always

[Install]
WantedBy=multi-user.target
EOF
echo -e "${GREEN}Systemd service unit file created.${NC}"

# Reload systemd daemon to avoid warning message
systemctl daemon-reload >/dev/null

# Start the service
systemctl start zhifou-agent >/dev/null
echo -e "${GREEN}Service started.${NC}"

# Enable auto-start on boot
systemctl enable zhifou-agent >/dev/null
echo -e "${GREEN}Service enabled to start on boot.${NC}"

echo -e "${GREEN}zhifou agent has been successfully installed and started.${NC}"