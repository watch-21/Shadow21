package tor

import (
	"encoding/hex"
	"fmt"
	"strings"
)

func validateSnowflakeBridge(address string) error {
	fields := strings.Fields(address)
	if len(fields) < 2 {
		return fmt.Errorf("snowflake bridge requires an address")
	}

	optionIndex := 2
	positionalFingerprint := ""
	if len(fields) > optionIndex && !strings.Contains(fields[optionIndex], "=") {
		positionalFingerprint = fields[optionIndex]
		if err := validateSnowflakeFingerprint(positionalFingerprint); err != nil {
			return err
		}
		optionIndex++
	}

	options := make(map[string]string)
	for _, field := range fields[optionIndex:] {
		key, value, ok := strings.Cut(field, "=")
		if ok {
			options[key] = value
		}
	}

	if fingerprint, ok := options["fingerprint"]; ok {
		if err := validateSnowflakeFingerprint(fingerprint); err != nil {
			return err
		}
		if positionalFingerprint != "" && !strings.EqualFold(fingerprint, positionalFingerprint) {
			return fmt.Errorf("snowflake bridge fingerprints do not match")
		}
	}
	for _, key := range []string{"url", "sqsqueue"} {
		if value, ok := options[key]; ok && !strings.HasPrefix(value, "https://") {
			return fmt.Errorf("snowflake bridge %s must use HTTPS", key)
		}
	}
	if ice, ok := options["ice"]; ok && ice == "" {
		return fmt.Errorf("snowflake bridge ICE servers cannot be empty")
	}
	return nil
}

func validateSnowflakeFingerprint(fingerprint string) error {
	if len(fingerprint) != 40 {
		return fmt.Errorf("snowflake bridge fingerprint must contain 40 hexadecimal characters")
	}
	if _, err := hex.DecodeString(fingerprint); err != nil {
		return fmt.Errorf("snowflake bridge fingerprint is invalid: %w", err)
	}
	return nil
}
