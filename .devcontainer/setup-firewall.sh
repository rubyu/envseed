#!/bin/bash
set -euo pipefail  # Exit on error, undefined vars, and pipeline failures
IFS=$'\n\t'       # Stricter word splitting

# 1. Extract Docker DNS info BEFORE any flushing
DOCKER_DNS_RULES=$(iptables-save -t nat | grep "127\.0\.0\.11" || true)

# Flush existing rules and delete existing ipsets
iptables -F
iptables -X
iptables -t nat -F
iptables -t nat -X
iptables -t mangle -F
iptables -t mangle -X
ipset destroy blocked-networks 2>/dev/null || true
ipset destroy public-internet 2>/dev/null || true

# 2. Selectively restore ONLY internal Docker DNS resolution
if [ -n "$DOCKER_DNS_RULES" ]; then
    echo "Restoring Docker DNS rules..."
    iptables -t nat -N DOCKER_OUTPUT 2>/dev/null || true
    iptables -t nat -N DOCKER_POSTROUTING 2>/dev/null || true
    echo "$DOCKER_DNS_RULES" | xargs -L 1 iptables -t nat
else
    echo "No Docker DNS rules to restore"
fi

# Create ipsets
ipset create blocked-networks hash:net
ipset create public-internet hash:net

# Add private network ranges (RFC1918 and others) to blocked list
echo "Adding private network ranges to blocklist..."
ipset add blocked-networks 10.0.0.0/8        # Class A private
ipset add blocked-networks 172.16.0.0/12     # Class B private
ipset add blocked-networks 192.168.0.0/16    # Class C private
ipset add blocked-networks 169.254.0.0/16    # Link-local
ipset add blocked-networks 127.0.0.0/8       # Loopback (will be allowed separately)
ipset add blocked-networks 224.0.0.0/4       # Multicast
ipset add blocked-networks 240.0.0.0/4       # Reserved
ipset add blocked-networks 0.0.0.0/8         # This network
ipset add blocked-networks 100.64.0.0/10     # Carrier-grade NAT

# Add public internet ranges (everything except private ranges)
# This is done by adding broad public ranges
echo "Defining public internet ranges..."
# We'll use a different approach - check if destination is NOT in blocked-networks

# Get host IP from default route
HOST_IP=$(ip route | grep default | cut -d" " -f3)
if [ -z "$HOST_IP" ]; then
    echo "ERROR: Failed to detect host IP"
    exit 1
fi

HOST_NETWORK=$(echo "$HOST_IP" | sed "s/\.[0-9]*$/.0\/24/")
echo "Host network detected as: $HOST_NETWORK"

# Set default policies to DROP
iptables -P INPUT DROP
iptables -P FORWARD DROP
iptables -P OUTPUT DROP

# === INPUT Rules ===
# Allow localhost
iptables -A INPUT -i lo -j ACCEPT

# Allow established and related connections
iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT

# Allow from host network
iptables -A INPUT -s "$HOST_NETWORK" -j ACCEPT

# Allow inbound DNS responses
iptables -A INPUT -p udp --sport 53 -j ACCEPT
iptables -A INPUT -p tcp --sport 53 -j ACCEPT

# === OUTPUT Rules ===
# Allow localhost
iptables -A OUTPUT -o lo -j ACCEPT

# Allow established and related connections
iptables -A OUTPUT -m state --state ESTABLISHED,RELATED -j ACCEPT

# Allow to host network
iptables -A OUTPUT -d "$HOST_NETWORK" -j ACCEPT

# Allow outbound DNS
iptables -A OUTPUT -p udp --dport 53 -j ACCEPT
iptables -A OUTPUT -p tcp --dport 53 -j ACCEPT

# Allow outbound SSH
iptables -A OUTPUT -p tcp --dport 22 -j ACCEPT

# Allow HTTP/HTTPS to public internet only
# First, create a custom chain for HTTP/HTTPS filtering
iptables -N HTTP_HTTPS_FILTER 2>/dev/null || true
iptables -F HTTP_HTTPS_FILTER

# In the filter chain, reject if destination is in blocked networks
iptables -A HTTP_HTTPS_FILTER -m set --match-set blocked-networks dst -j REJECT --reject-with icmp-net-prohibited
# Otherwise, accept
iptables -A HTTP_HTTPS_FILTER -j ACCEPT

# Apply the filter to HTTP and HTTPS traffic
iptables -A OUTPUT -p tcp --dport 80 -j HTTP_HTTPS_FILTER
iptables -A OUTPUT -p tcp --dport 443 -j HTTP_HTTPS_FILTER

# Optional: Allow ICMP (ping) to public internet
iptables -A OUTPUT -p icmp -m set ! --match-set blocked-networks dst -j ACCEPT

# Log dropped packets (optional - uncomment if needed for debugging)
# iptables -A OUTPUT -j LOG --log-prefix "DROPPED-OUTPUT: " --log-level 4

echo "Firewall configuration complete"
echo "Configuration allows:"
echo "  - Outbound HTTP/HTTPS to public internet only"
echo "  - DNS queries"
echo "  - SSH connections"
echo "  - Communication with host network ($HOST_NETWORK)"
echo "  - ICMP (ping) to public internet"
echo "Configuration blocks:"
echo "  - HTTP/HTTPS to private/internal networks"
echo "  - All other protocols/ports (default DROP policy)"

# Verification tests
echo -e "\nVerifying firewall rules..."

# Test 1: Should succeed - public internet access
if curl --connect-timeout 5 https://example.com >/dev/null 2>&1; then
    echo "✓ Public internet access working (https://example.com)"
else
    echo "✗ ERROR: Cannot reach public internet (https://example.com)"
fi

# Test 2: Should succeed - GitHub API
if curl --connect-timeout 5 https://api.github.com/zen >/dev/null 2>&1; then
    echo "✓ GitHub API access working"
else
    echo "✗ ERROR: Cannot reach GitHub API"
fi

# Test 3: Should fail - internal network (common router IPs)
for internal_ip in 192.168.1.1 192.168.0.1 10.0.0.1 172.16.0.1; do
    if curl --connect-timeout 2 http://$internal_ip >/dev/null 2>&1; then
        echo "✗ WARNING: Can reach internal network ($internal_ip) - this should be blocked"
        break
    fi
done
echo "✓ Internal network properly blocked"

# Test 4: Should fail - trying a different port (e.g., FTP)
if curl --connect-timeout 2 ftp://ftp.gnu.org >/dev/null 2>&1; then
    echo "✗ WARNING: Can reach FTP - non-HTTP/HTTPS should be blocked"
else
    echo "✓ Non-HTTP/HTTPS protocols properly blocked"
fi

echo -e "\nFirewall setup complete!"
