package tor

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

func ParseRoutingConfig(reader io.Reader) (RoutingConfig, error) {
	var config RoutingConfig
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		fields := strings.Fields(stripComment(scanner.Text()))
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "ClientTransportPlugin":
			for _, transport := range strings.Split(fields[1], ",") {
				if transport == "snowflake" {
					config.PreserveSystemDNS = true
				}
			}
		case "VirtualAddrNetwork":
			if config.VirtualAddrNetwork == "" {
				config.VirtualAddrNetwork = fields[1]
			}
		case "TransPort":
			if config.TransPort == "" {
				config.TransPort = portValue(fields[1])
			}
		case "DNSPort":
			if config.DNSPort == "" {
				config.DNSPort = portValue(fields[1])
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return config, err
	}
	if config.TransPort == "" {
		return config, fmt.Errorf("Tor TransPort not found")
	}
	if config.DNSPort == "" {
		return config, fmt.Errorf("Tor DNSPort not found")
	}
	if config.VirtualAddrNetwork == "" {
		return config, fmt.Errorf("Tor VirtualAddrNetwork not found")
	}
	return config, nil
}

func portValue(value string) string {
	if before, _, ok := strings.Cut(value, "/"); ok {
		value = before
	}
	if index := strings.LastIndex(value, ":"); index >= 0 {
		return value[index+1:]
	}
	return value
}

func stripComment(line string) string {
	if index := strings.Index(line, "#"); index >= 0 {
		return line[:index]
	}
	return line
}
