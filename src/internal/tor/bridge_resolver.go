package tor

import (
	"bufio"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"os/exec"
	"strings"
)

const BridgesPath = "/etc/shadow21/bridges.txt"

var ErrNoBridgeEntries = errors.New("no bridge entries found")

type BridgeResolver struct{}

func parseBridgeEntries(reader io.Reader) ([]string, error) {
	var entries []string
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		entries = append(entries, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, ErrNoBridgeEntries
	}

	return entries, nil
}

func loadBridgeEntries(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return parseBridgeEntries(file)
}

func selectRandomBridges(entries []string, count int) ([]string, error) {
	if len(entries) == 0 {
		return nil, ErrNoBridgeEntries
	}
	if count > len(entries) {
		count = len(entries)
	}

	remaining := append([]string{}, entries...)
	selected := make([]string, 0, count)
	for len(selected) < count {
		index, err := rand.Int(rand.Reader, big.NewInt(int64(len(remaining))))
		if err != nil {
			return nil, err
		}
		position := int(index.Int64())
		selected = append(selected, remaining[position])
		remaining = append(remaining[:position], remaining[position+1:]...)
	}
	return selected, nil
}

func (BridgeResolver) Resolve(options TorrcOptions) (TorrcOptions, error) {
	if !options.AutoBridge && strings.TrimSpace(options.BridgeAddress) == "" {
		return options, nil
	}

	if options.AutoBridge {
		entries, err := loadBridgeEntries(BridgesPath)
		if err != nil {
			return options, fmt.Errorf("load bridge entries: %w", err)
		}
		selected, err := selectRandomBridges(entries, 3)
		if err != nil {
			return options, fmt.Errorf("select bridges: %w", err)
		}
		options.BridgeAddresses = selected
	} else {
		options.BridgeAddresses = []string{options.BridgeAddress}
	}

	for _, address := range options.BridgeAddresses {
		transport, err := bridgeTransport(address)
		if err != nil {
			return options, err
		}
		switch transport {
		case "obfs4":
			if strings.TrimSpace(options.Obfs4ProxyPath) == "" {
				path, err := lookupExecutablePath("obfs4proxy")
				if err != nil {
					return options, fmt.Errorf("find obfs4proxy: %w", err)
				}
				options.Obfs4ProxyPath = path
			}
		case "snowflake":
			if strings.TrimSpace(options.SnowflakeClientPath) == "" {
				path, err := lookupExecutablePath("snowflake-client")
				if err != nil {
					return options, fmt.Errorf("find snowflake-client: %w", err)
				}
				options.SnowflakeClientPath = path
			}
		}
	}

	return options, nil
}

func lookupExecutablePath(name string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("%s path is empty", name)
	}
	return path, nil
}
