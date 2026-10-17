package tor

import (
	_ "embed"
	"fmt"
	"strings"

	"gitlab.com/parrotsec/packages/anonsurf/internal/config"
)

//go:embed assets/torrc.base
var defaultTorrcBase string

type TorrcOptions struct {
	UseSandbox          bool
	UseSafeSocks        bool
	RejectPlainPorts    bool
	PlaintextPorts      string
	AutoBridge          bool
	BridgeAddress       string
	BridgeAddresses     []string
	Obfs4ProxyPath      string
	SnowflakeClientPath string
	HTTPProxyAddress    string
	HTTPProxyCredential string
	ProxyIsHTTPS        bool
}

type RoutingConfig struct {
	TransPort          string
	DNSPort            string
	VirtualAddrNetwork string
	PreserveSystemDNS  bool
}

func TorrcOptionsFromConfig(cfg config.Config) TorrcOptions {
	return TorrcOptions{
		UseSandbox:          cfg.UseSandbox,
		UseSafeSocks:        cfg.UseSafeSocks,
		RejectPlainPorts:    cfg.RejectPlainPorts,
		PlaintextPorts:      cfg.ListPlainPorts,
		AutoBridge:          cfg.AutoBridge,
		BridgeAddress:       cfg.BridgeAddress,
		HTTPProxyAddress:    cfg.HTTPProxyAddress,
		HTTPProxyCredential: cfg.HTTPProxyCredential,
		ProxyIsHTTPS:        cfg.ProxyIsHTTPS,
	}
}

func RenderTorrc(options TorrcOptions) (string, error) {
	return renderTorrcWithBase(defaultTorrcBase, options)
}

func renderTorrcWithBase(base string, options TorrcOptions) (string, error) {
	base = strings.TrimRight(base, "\n")
	if strings.TrimSpace(base) == "" {
		return "", fmt.Errorf("torrc base is empty")
	}

	plaintextPorts, err := normalizePlaintextPorts(options.PlaintextPorts)
	if err != nil {
		return "", err
	}
	bridgeSettings, err := renderBridgeSettings(options)
	if err != nil {
		return "", err
	}
	proxySettings, err := renderProxySettings(options)
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	builder.WriteString(base)
	builder.WriteString("\n\n# Use sandbox\nSandbox ")
	builder.WriteString(torBool(options.UseSandbox))
	builder.WriteString("\n\n")
	builder.WriteString(bridgeSettings)
	builder.WriteString("\n\n# Use SafeSocks\nSafeSocks ")
	builder.WriteString(torBool(options.UseSafeSocks))
	builder.WriteString("\n\n")
	if options.RejectPlainPorts {
		builder.WriteString("# Set RejectPlaintextPorts\nRejectPlaintextPorts ")
	} else {
		builder.WriteString("# Set WarnPlaintextPorts\nWarnPlaintextPorts ")
	}
	builder.WriteString(plaintextPorts)
	builder.WriteString("\n")
	if proxySettings != "" {
		builder.WriteString("\n")
		builder.WriteString(proxySettings)
		builder.WriteString("\n")
	}

	return builder.String(), nil
}
