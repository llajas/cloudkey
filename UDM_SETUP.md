# UDM Pro Speedtest Integration (Pure Go)

This integration allows your CloudKey display to show speedtest results from your UDM Pro using a pure Go implementation - no PHP required!

## Setup Instructions

### 1. Configure UDM Pro Connection

Set the following environment variables with your UDM Pro credentials:

```bash
export CLOUDKEY_UDM_BASEURL="https://192.168.1.1"  # Your UDM Pro IP address
export CLOUDKEY_UDM_USERNAME="your_username"      # Your UDM Pro username
export CLOUDKEY_UDM_PASSWORD="your_password"      # Your UDM Pro password
export CLOUDKEY_UDM_SITE="default"                # Your site ID (usually "default")
export CLOUDKEY_UDM_VERSION="8.0.28"              # UniFi Network Controller version
```

**Alternative: Command Line Flags**
You can also use command line flags instead of environment variables:

```bash
./cloudkey -udm-baseurl="https://192.168.1.1" \
           -udm-username="your_username" \
           -udm-password="your_password" \
           -udm-site="default" \
           -udm-version="8.0.28"
```

**Security Note**: Environment variables are recommended over command line flags as they don't expose credentials in the process list.

### 2. Enable Speedtests on UDM Pro

Make sure speedtests are enabled on your UDM Pro:
1. Open UniFi Network Controller
2. Go to Settings → Internet
3. Enable "Auto Speedtest" or run manual speedtests
4. Set appropriate interval (recommended: every few hours)

### 3. Rebuild and Restart CloudKey Service

```bash
cd /path/to/cloudkey
make buildnew
sudo systemctl restart cloudkey
```

**Or use the improved Makefile targets:**
```bash
make restart        # Build and restart service in one command
make status         # Check service status
make logs          # View service logs
make test          # Run environment configuration test
make quick-test    # Run quick validation
```

### 4. Test Your Configuration

Use the provided test script to verify your environment variable setup:

```bash
./test_env_config.sh
```

This script will:
- Verify environment variables are properly set
- Test compilation with the new configuration
- Provide usage instructions

## How It Works

1. **Pure Go Client**: `src/network/unifi_client.go` implements a complete UniFi API client in Go
2. **Authentication**: Handles both UniFi OS (UDM Pro) and legacy controller authentication
3. **Session Management**: Automatic cookie handling and CSRF token management
4. **API Integration**: Directly calls the UniFi speedtest API endpoint
5. **Display Update**: The speedtest screen shows UDM Pro results instead of running local speedtests

## Features

- **Zero Dependencies**: No PHP, Composer, or external dependencies needed
- **Auto-Detection**: Automatically detects UniFi OS vs legacy controllers
- **Session Management**: Handles authentication, cookies, and CSRF tokens
- **Error Handling**: Graceful error handling with re-authentication
- **SSL Flexibility**: Configurable SSL verification for local networks
- **Time Range Support**: Can fetch specific time ranges or default to last 24 hours

## API Implementation Details

### Authentication Flow
1. **Controller Detection**: GET request to `/` to determine UniFi OS vs legacy
2. **Login**: POST to `/api/auth/login` (UniFi OS) or `/api/login` (legacy)
3. **Session Management**: Extract and store authentication cookies
4. **CSRF Handling**: Extract CSRF token from JWT (UniFi OS only)

### Speedtest API
- **Endpoint**: `/proxy/network/api/s/{site}/stat/report/archive.speedtest` (UniFi OS)
- **Method**: POST with JSON payload
- **Response**: Array of speedtest results with download/upload speeds and latency

### Data Structure
```go
type SpeedtestResult struct {
    DownloadMbps float64 `json:"download_mbps"`
    UploadMbps   float64 `json:"upload_mbps"`
    LatencyMs    float64 `json:"latency_ms"`
    Timestamp    int64   `json:"timestamp"`
}
```

## Benefits

- **More Accurate**: Tests from your UDM Pro reflect actual internet speeds
- **Faster**: No need to wait for speedtests to complete on the CloudKey
- **Less Load**: Doesn't consume CloudKey resources for speed testing
- **Pure Go**: No external dependencies or PHP runtime required
- **Secure**: Proper authentication and session management
- **Flexible**: Works with both UniFi OS and legacy controllers

## Troubleshooting

### Common Issues

1. **"config error" on display**
   - Update connection parameters in `screens.go`
   - Verify UDM Pro IP address and credentials
   - Check that user has admin privileges

2. **"check logs" error**
   - Check system logs: `journalctl -u cloudkey -f`
   - Verify UDM Pro is accessible from CloudKey
   - Check SSL certificate issues

3. **"No speedtest results found"**
   - Enable speedtests in UDM Pro settings
   - Run a manual speedtest first
   - Check if speedtests are scheduled

4. **Authentication failures**
   - Verify username and password
   - Check if UDM Pro user has sufficient permissions
   - Ensure UDM Pro firmware version is correct

### Debug Mode

Enable debug output by setting the demo flag:
```bash
./cloudkey -demo
```

This will show sample data to verify the display works correctly.

### SSL Certificate Issues

For local networks with self-signed certificates, the client automatically skips SSL verification. For production environments, you can modify the TLS configuration in `unifi_client.go`:

```go
TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
```

## Testing

### Test Environment Variable Configuration
```bash
./test_env_config.sh
```

### Test Integration
```bash
./test_udm_integration.sh
```

### Test with Demo Mode
```bash
# Set your environment variables first
export CLOUDKEY_UDM_BASEURL="https://your-udm-ip"
export CLOUDKEY_UDM_USERNAME="your_username"
export CLOUDKEY_UDM_PASSWORD="your_password"
export CLOUDKEY_UDM_SITE="default"
export CLOUDKEY_UDM_VERSION="your_firmware_version"

# Test with demo mode (no framebuffer access required)
./cloudkey -demo
```

## Security Notes

- ✅ **Environment Variables**: Now the default and most secure method
- ✅ **No Hardcoded Credentials**: Credentials are no longer stored in source code
- ✅ **HTTPS Required**: Always use HTTPS URLs for UDM Pro connections
- ✅ **Limited Permissions**: Use UDM Pro accounts with minimum required permissions
- ✅ **Network Security**: Consider network segmentation for sensitive deployments
- ⚠️ **Process Security**: Avoid using command line flags in production as they expose credentials in process lists