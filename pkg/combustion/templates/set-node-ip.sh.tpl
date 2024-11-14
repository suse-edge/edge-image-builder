#!/bin/bash

get_default_ipv4_address() {
    local default_route=$(ip -4 route show default | sort -k7n | head -n1)
    if [ -z "$default_route" ]; then
        echo "No default IPv4 route found" >&2
        return 1
    fi

    local interface=$(echo "$default_route" | awk '{print $5}')

    local ip_address=$(ip -4 addr show dev "$interface" | grep -w inet | head -n1 | awk '{print $2}' | cut -d'/' -f1)

    if [ -z "$ip_address" ]; then
        echo "No IPv4 address found for interface $interface" >&2
        return 1
    fi

    echo "$ip_address"
}

get_default_ipv6_address() {
    local default_route=$(ip -6 route show default | sort -k7n | head -n1)
    if [ -z "$default_route" ]; then
        echo "No default IPv6 route found" >&2
        return 1
    fi

    local interface=$(echo "$default_route" | awk '{print $5}')

    local ip_address=$(ip -6 addr show dev "$interface" | grep -w inet6 | grep -v fe80 | grep 'scope global' | head -n1 | awk '{print $2}' | cut -d'/' -f1)

    if [ -z "$ip_address" ]; then
        echo "No IPv6 address found for interface $interface" >&2
        return 1
    fi

    echo "$ip_address"
}

MAX_ATTEMPTS=5
DELAY=5
IPv4_ADDRESS=""
IPv6_ADDRESS=""
IPv4={{ .IPv4Enabled }}
IPv6={{ .IPv6Enabled }}
prioritizeIPv6={{ .PrioritizeIPv6 }}

{{- if .RKE2 }}
CONFIG_FILE="/etc/rancher/rke2/config.yaml"
{{- else }}
CONFIG_FILE="/etc/rancher/k3s/config.yaml"
{{- end }}


try_get_ipv4() {
    local attempt=1
    while [ $attempt -le $MAX_ATTEMPTS ]; do
        echo "Attempt $attempt to get IPv4 address..."
        IPv4_ADDRESS=$(get_default_ipv4_address)
        if [ -n "$IPv4_ADDRESS" ]; then
            echo "Successfully got IPv4 address: $IPv4_ADDRESS"
            return 0
        fi

        if [ $attempt -lt $MAX_ATTEMPTS ]; then
            echo "Waiting $DELAY seconds before next attempt..."
            sleep $DELAY
        fi
        ((attempt++))
    done
    echo "Failed to get IPv4 address after $MAX_ATTEMPTS attempts"
    return 1
}

try_get_ipv6() {
    local attempt=1
    while [ $attempt -le $MAX_ATTEMPTS ]; do
        echo "Attempt $attempt to get IPv6 address..."
        IPv6_ADDRESS=$(get_default_ipv6_address)
        if [ -n "$IPv6_ADDRESS" ]; then
            echo "Successfully got IPv6 address: $IPv6_ADDRESS"
            return 0
        fi

        if [ $attempt -lt $MAX_ATTEMPTS ]; then
            echo "Waiting $DELAY seconds before next attempt..."
            sleep $DELAY
        fi
        ((attempt++))
    done
    echo "Failed to get IPv6 address after $MAX_ATTEMPTS attempts"
    return 1
}

# Main execution
if [ "${IPv4}" = "true" ]; then
    try_get_ipv4
fi

if [ "${IPv6}" = "true" ]; then
    try_get_ipv6
fi

update_config() {
    if [ "${IPv4}" = "true" ] && [ "${IPv6}" = "true" ]; then
        if [ "$prioritizeIPv6" = "false" ]; then
            echo "node-ip: ${IPv4_ADDRESS},${IPv6_ADDRESS}" >> "$CONFIG_FILE"
            echo "Added IPv4,IPv6 addresses to config (IPv4 prioritized)"
        else
            echo "node-ip: ${IPv6_ADDRESS},${IPv4_ADDRESS}" >> "$CONFIG_FILE"
            echo "Added IPv6,IPv4 addresses to config (IPv6 prioritized)"
#            if [ "$IS_SERVER" = "true" ]; then
#              echo "kube-apiserver-arg:" >> "$CONFIG_FILE"
#              echo "  - \"advertise-address=${IPv6_ADDRESS}\"" >> "$CONFIG_FILE"
#            fi
        fi
    elif [ "${IPv4}" = "true" ]; then
        echo "node-ip: ${IPv4_ADDRESS}" >> "$CONFIG_FILE"
        echo "Added IPv4 address to config"
    elif [ "${IPv6}" = "true" ]; then
        echo "node-ip: ${IPv6_ADDRESS}" >> "$CONFIG_FILE"
        echo "Added IPv6 address to config"
#        if [ "$IS_SERVER" = "true" ]; then
#          echo "kube-apiserver-arg:" >> "$CONFIG_FILE"
#          echo "  - \"advertise-address=${IPv6_ADDRESS}\"" >> "$CONFIG_FILE"
#        fi
    fi
}

echo "----------------------------------------"
echo "Results:"
if [ "${IPv4}" = "true" ]; then
    echo "IPv4 Address: ${IPv4_ADDRESS:-Not found}"
fi
if [ "${IPv6}" = "true" ]; then
    echo "IPv6 Address: ${IPv6_ADDRESS:-Not found}"
fi

if { [ "${IPv4}" = "true" ] && [ -n "$IPv4_ADDRESS" ]; } || \
   { [ "${IPv6}" = "true" ] && [ -n "$IPv6_ADDRESS" ]; }; then
    update_config
else
    echo "Error: No valid IP addresses found to update config"
    exit 0
fi