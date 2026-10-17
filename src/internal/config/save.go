package config

import (
	"fmt"
	"gitlab.com/parrotsec/packages/anonsurf/internal/fileops"
	"os"
)

func Serialize(c Config) string {
	return fmt.Sprintf("[Shadow21Config]\nkill_switch=%t\nuse_sandbox=%t\nbridge_address=%q\nuse_safe_sock=%t\nauto_bridge=%t\nreject_plain_port=%t\nlist_plain_ports=%q\nhttp_proxy_address=%q\nhttp_proxy_credential=%q\nproxy_is_https=%t\n", c.KillSwitch, c.UseSandbox, c.BridgeAddress, c.UseSafeSocks, c.AutoBridge, c.RejectPlainPorts, c.ListPlainPorts, c.HTTPProxyAddress, c.HTTPProxyCredential, c.ProxyIsHTTPS)
}
func Save(c Config) error {
	original, err := os.ReadFile(SystemPath)
	if err != nil {
		return err
	}
	if err = fileops.WriteAtomic(SystemPath+".previous", original, 0600); err != nil {
		return err
	}
	return fileops.WriteAtomic(SystemPath, []byte(Serialize(c)), 0600)
}
