package config

import (
	"strings"
	"testing"
)

func TestConfigRoundTripAndKillSwitchDefault(t *testing.T) {
	c := Default()
	if !c.KillSwitch {
		t.Fatal("unsafe default")
	}
	c.BridgeAddress = "obfs4 127.0.0.1:9001 cert=abc iat-mode=0"
	got, e := Parse(strings.NewReader(Serialize(c)))
	if e != nil || got != c {
		t.Fatal(got, e)
	}
}
func TestConfigRejectsUnknownPrivacyKey(t *testing.T) {
	if _, e := Parse(strings.NewReader("[Shadow21Config]\nkill_swich=false\n")); e == nil {
		t.Fatal("typo accepted")
	}
}
