package dns

import (
	"bufio"
	"errors"
	"net"
	"os"
	"strings"

	"gitlab.com/parrotsec/packages/anonsurf/internal/fileops"
)

var ErrNoValidDNSAddress = errors.New("no valid DNS address found")

type AddrFromParams struct {
	HasDHCPFlag bool
	ListAddr    []string
	Invalid     []string
}

func HasOnlyLocalhost(listAddr []string) bool {
	if len(listAddr) == 1 && (listAddr[0] == "localhost" || listAddr[0] == "127.0.0.1") {
		return true
	}
	return false
}

func SystemDNSFileIsSymlink(resolvConfPath string) (bool, error) {
	fileInfo, err := os.Lstat(resolvConfPath)
	if err != nil {
		return false, err
	}
	return fileInfo.Mode()&os.ModeSymlink != 0, nil
}

func removeManagedResolver(path string) error {
	isSymlink, err := SystemDNSFileIsSymlink(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if isSymlink {
		return nil
	}
	return os.Remove(path)
}

func ParseDNSAddresses(resolvConfPath string) ([]string, error) {
	var result []string
	file, err := os.Open(resolvConfPath)
	if err != nil {
		return result, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "nameserver" {
			result = append(result, fields[1])
		}
	}
	if err := scanner.Err(); err != nil {
		return result, err
	}
	return result, nil
}

func RenderResolvConf(listDNSAddr []string) (string, error) {
	deduplicatedListAddr := GetListValidAddresses(listDNSAddr)
	if len(deduplicatedListAddr) == 0 {
		return "", ErrNoValidDNSAddress
	}

	var content strings.Builder
	content.WriteString("# Written by DNSTool\n")
	for _, address := range deduplicatedListAddr {
		content.WriteString("nameserver ")
		content.WriteString(address)
		content.WriteString("\n")
	}
	return content.String(), nil
}

func WriteDNSToSystem(listDNSAddr []string, resolvConfPath string) error {
	content, err := RenderResolvConf(listDNSAddr)
	if err != nil {
		return err
	}
	return fileops.WriteAtomic(resolvConfPath, []byte(content), 0644)
}

func GetListValidAddresses(listAddr []string) []string {
	var result []string
	seen := make(map[string]bool)
	for _, eachAddr := range listAddr {
		address := strings.TrimSpace(eachAddr)
		if net.ParseIP(address) != nil && !seen[address] {
			seen[address] = true
			result = append(result, address)
		}
	}
	return result
}

func ParseAddrFromParams(params []string) AddrFromParams {
	paramResult := AddrFromParams{
		HasDHCPFlag: false,
		ListAddr:    []string{},
	}

	for _, value := range params {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if value == "dhcp" {
			paramResult.HasDHCPFlag = true
		} else if net.ParseIP(value) != nil {
			paramResult.ListAddr = append(paramResult.ListAddr, value)
		} else {
			paramResult.Invalid = append(paramResult.Invalid, value)
		}
	}

	return paramResult
}
