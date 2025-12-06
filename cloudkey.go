package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/jnovack/go-version"

	"github.com/tabalt/pidfile"

	"github.com/coreos/pkg/flagutil"
	// "github.com/jnovack/cloudkey/display"
	"cloudkey/display"
	_ "github.com/jnovack/cloudkey/fonts"
)

var tags = map[string]string{
	"SYSLOG_IDENTIFIER": "cloudkey",
}

var opts display.CmdLineOpts

func main() {
	display.New(opts)
}

func init() {
	flag.Float64Var(&opts.Delay, "delay", 7500, "delay in milliseconds between screens")
	flag.BoolVar(&opts.Reset, "reset", false, "reset/clear the screen")
	flag.BoolVar(&opts.Demo, "demo", false, "use fake data for display only")
	flag.StringVar(&opts.Pidfile, "pidfile", "/var/run/zeromon.pid", "pidfile")
	flag.StringVar(&opts.UDMBaseURL, "udm-baseurl", "https://192.168.1.1:443", "UDM Pro base URL")
	flag.StringVar(&opts.UDMUsername, "udm-username", "", "UDM Pro username")
	flag.StringVar(&opts.UDMPassword, "udm-password", "", "UDM Pro password")
	flag.StringVar(&opts.UDMSite, "udm-site", "default", "UDM Pro site ID")
	flag.StringVar(&opts.UDMVersion, "udm-version", "8.0.28", "UDM Pro controller version")
	flag.BoolVar(&opts.Version, "version", false, "print version and exit")
	flagutil.SetFlagsFromEnv(flag.CommandLine, "CLOUDKEY")
	flag.Parse()

	if opts.Version {
		// already printed version
		os.Exit(0)
	}

	pid, err := pidfile.Create(opts.Pidfile)
	if err != nil {
		fmt.Printf("Error creating PID file: %s\n", err)
		os.Exit(1)
	}

	// Setup Service
	// https://fabianlee.org/2017/05/21/golang-running-a-go-binary-as-a-systemd-service-on-ubuntu-16-04/
	fmt.Println("Starting cloudkey service")
	// setup signal catching
	sigs := make(chan os.Signal, 1)
	// catch all signals since not explicitly listing
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	// method invoked upon seeing signal
	go func() {
		s := <-sigs
		display.Shutdown()
		fmt.Printf("Received signal '%s', shutting down\n", s)
		fmt.Println("Stopping cloudkey service")
		_ = pid.Clear()
		os.Exit(1)
	}()
}
