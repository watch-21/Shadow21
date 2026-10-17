package platform

import "testing"

func TestDistributionAdapters(t *testing.T) {
	for id, family := range map[string]string{"kali": "debian", "parrot": "debian", "ubuntu": "debian", "debian": "debian", "fedora": "fedora", "arch": "arch", "opensuse-tumbleweed": "suse"} {
		d := Parse("ID=" + id + "\nPRETTY_NAME=\"Test Linux\"\n")
		if d.Family != family || d.PackageManager == "" || d.TorUser == "" || d.TorUnit == "" {
			t.Fatal(d)
		}
	}
	if Parse("ID=unknown").Family != "" {
		t.Fatal("unknown supported")
	}
}
