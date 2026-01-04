# cloudkey

**cloudkey** is a replacement for `/usr/bin/ck-ui` on your Ubiquiti Cloud Key
Generation 2 device.

![screenshot](https://raw.githubusercontent.com/jnovack/cloudkey/master/doc/screenshot.gif)

*Note: Delay is slowed down to show fading between screens.*

## Features

### Display Screens

The 160x60 LCD cycles through up to 6 information screens:

| Screen | Content |
|--------|---------|
| CPU | Current CPU usage percentage |
| RAM | Used/Total memory in GB + percentage |
| Swap | Used/Total swap in GB + percentage |
| Network | Hostname, LAN IP, WAN IP |
| Speedtest | Download/Upload speeds from UDM Pro |
| Kubernetes | Node count, cluster health, pod/container count (optional) |

### LED Status Indicators

Supports all Cloud Key Gen2 LEDs including rack mount accessories:

**Main Unit LEDs** (`blue`, `white`):
- White during boot
- Blue when running normally

**Rack Mount LEDs** (`rack:blue`, `rack:white`, `ulogo_ctrl`):

The rack LEDs indicate system health at a glance:

| LED State | Meaning |
|-----------|---------|
| Solid Blue | Healthy - CPU and RAM below 80% |
| Solid White | Warning - CPU or RAM between 80-95% |
| Blinking White | Critical - CPU or RAM above 95%, or UDM connection error |

The Ubiquiti logo LED (`ulogo_ctrl`) stays on while the service is running.

### UDM Pro Integration

Fetches speedtest results from your UDM Pro via the UniFi API. Configure credentials via environment variables (see Configuration section).

### Kubernetes Integration

Displays cluster status including node health, pod counts, and container counts. The screen shows:
- **Row 1**: Ready/Total nodes (e.g., `8/8 nodes`)
- **Row 2**: Cluster health status (`Healthy` or `Degraded`)
- **Row 3**: Running pods with container count (e.g., `195 pods (312)`)

When the cluster becomes unreachable:
- If data was previously fetched, shows last known values with an asterisk (`*`)
- If never connected, shows `K8s offline`

Enable via `CLOUDKEY_K8S_ENABLED=true` in your configuration.

## Installation

### Quick Start

1. `ssh ubnt@UniFi-CloudKeyG2`
2. `mv /usr/bin/ck-ui /usr/bin/ck-ui.original`
3. `curl -Lo /usr/local/ck-ui LINK_FROM_RELEASES_PAGE`

### Developers

1. Have a working Go environment.
2. `GOOS=linux GOARCH=arm64 go build cloudkey.go`
3. SCP the file over to your Cloud Key.

At this point, you can choose to backup and overwrite the `/usr/bin/ck-ui`
file or create a new systemd service, depending on your linux experience.

#### Using the `systemd` Service

Disable the old service first.

1. `systemctl disable ck-ui`
2. `systemctl stop ck-ui`

Install this one.

1. scp `cloudkey.service` to the `/lib/systemd/system/` directory.
2. `touch /etc/cloudkey.env`
3. `systemctl enable cloudkey`
4. `systemctl start cloudkey`

## Configuration

Set environment variables in `/etc/cloudkey.env`:

```bash
CLOUDKEY_DELAY=7500              # Screen carousel delay in milliseconds

# UDM Pro Integration
CLOUDKEY_UDM_BASEURL=https://192.168.1.1:443
CLOUDKEY_UDM_USERNAME=admin
CLOUDKEY_UDM_PASSWORD=yourpassword
CLOUDKEY_UDM_SITE=default
CLOUDKEY_UDM_VERSION=8.0.28

# Kubernetes Integration (optional)
CLOUDKEY_K8S_ENABLED=true
CLOUDKEY_K8S_KUBECONFIG=/path/to/.kube/config
```

## Makefile Commands

```bash
make buildnew    # Build for ARM64 (Cloud Key)
make install     # Install binary and systemd service
make update      # Backup current binary, then deploy new version
make rollback    # Restore previous binary from backup
make status      # Check service status
make logs        # Follow service logs
make stop        # Stop service
```

The `update` target automatically backs up the running binary to `/usr/local/bin/cloudkey.backup` before deploying, allowing safe rollback if issues occur.

## Why?

I am an edge case.  I do not use my Cloud Key device for Unifi.  I think it is
a great sexy little hardware device, but to manage a network off of what is
essentially a POE SDCard, you are insane.

Issues with stability are [very well documented](https://help.ubnt.com/hc/en-us/articles/360000128688-UniFi-Troubleshooting-Offline-Cloud-Key-and-Other-Stability-Issues#4).
Using mongodb on an sdcard (limited write cycles) without *automatically*
repairing has lead me to have to recover 4 times in 2 years even with the
secondary USB power from the UPS. That is NOT remotely production stable.
Run Unifi on a server, not a "raspberry pi".

With that said, I am sure you are asking yourself *"Why do you have it all?"*
The Ubiquiti Cloud Key Gen2 is a POE, ARMv7, Single-Board-Computer with
on-board battery backup and a 160x60 framebuffer display built-in.  It is
sexy, for under $200. It looks like an iDevice.

Sure, you can buy a $35 Raspberry Pi, add a case, with a touchscreen, with
a power-supply, and blah blah, but I'll pay for quality and craftsmanship so
it does not look like another Frankenstein project around my house.

I can ship it to my parents, tell them to plug one cable into the new-fangled
doo-hickey and tell them to call their ISP when it has a sad face on it
(feature not developed yet).
