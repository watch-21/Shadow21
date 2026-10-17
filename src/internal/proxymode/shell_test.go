package proxymode

import "testing"

func TestConfiguredRootShell(t *testing.T) {
	for _, s := range []string{"/usr/bin/zsh", "/bin/bash", "/bin/sh", "/usr/bin/fish"} {
		if got := configuredShell("root:x:0:0:root:/root:" + s + "\n"); got != s {
			t.Fatal(got, s)
		}
	}
	for _, entry := range []string{"root:x:1000:0:root:/root:/bin/zsh", "user:x:0:0:user:/tmp:/bin/zsh", "root:x:0:0:root:/root:/tmp/evil;id", "root:x:0:0:root:/root:/usr/sbin/nologin"} {
		if got := configuredShell(entry); got != "/bin/bash" {
			t.Fatal(got)
		}
	}
}
