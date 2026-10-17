package proxymode

import (
	"fmt"
	"strings"
)

// RenderConfig changes only the requested SOCKS directives. Runtime overrides
// isolate the managed client from service data and non-SOCKS listeners.
func RenderConfig(original string) (string, error) {
	lines := []string{"# shadow21 local SOCKS configuration", "SocksPort 9050", "SocksPolicy accept *"}
	for _, line := range strings.Split(original, "\n") {
		if line == "# shadow21 local SOCKS configuration" {
			continue
		}
		f := strings.Fields(strings.SplitN(line, "#", 2)[0])
		if len(f) == 0 {
			lines = append(lines, line)
			continue
		}
		key := strings.ToLower(f[0])
		if strings.HasPrefix(key, "+") || strings.HasPrefix(key, "/") || strings.HasPrefix(key, "%") || strings.HasSuffix(strings.TrimSpace(line), "\\") {
			return "", fmt.Errorf("advanced Tor configuration requires manual migration")
		}
		switch key {
		case "socksport":
			if len(f) != 2 || (f[1] != "0" && f[1] != "9050" && f[1] != "127.0.0.1:9050") {
				return "", fmt.Errorf("existing SOCKS listener or isolation options require manual migration")
			}
			continue
		case "sockspolicy":
			if len(f) == 3 && strings.EqualFold(f[1], "accept") && f[2] == "*" {
				continue
			}
		case "orport", "dirport", "hiddenservicedir", "hiddenserviceport", "servertransportplugin", "extorport", "sockslistenaddress", "controllistenaddress":
			return "", fmt.Errorf("existing server or listener configuration is unsupported in proxy mode")
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n") + "\n", nil
}
