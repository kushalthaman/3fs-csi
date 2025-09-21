package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultDriverName     = "fs.3fs.dev"
	DefaultGlobalMountBase = "/var/lib/3fs/mnt"
	DefaultConfigDir       = "/var/lib/3fs/etc"
	DefaultTokenFile       = "/var/lib/3fs/token.txt"
	DefaultHF3FSBinaryPath = "/opt/3fs/bin/hf3fs_fuse_main"
	DefaultLogLevel        = "info"
)

type Config struct {
	DriverName      string
	ClusterID       string
	MgmtdAddresses  []string
	GlobalMountBase string
	ConfigDir       string
	TokenFile       string
	HF3FSBinaryPath string
	LogLevel        string
	NodeID          string
}

func FromEnv() (*Config, error) {
	c := &Config{
		DriverName:      getEnv("CSI_DRIVER_NAME", DefaultDriverName),
		GlobalMountBase: getEnv("GLOBAL_MOUNT_BASE", DefaultGlobalMountBase),
		ConfigDir:       getEnv("CONFIG_DIR", DefaultConfigDir),
		TokenFile:       getEnv("TOKEN_FILE", DefaultTokenFile),
		HF3FSBinaryPath: getEnv("HF3FS_BINARY_PATH", DefaultHF3FSBinaryPath),
		LogLevel:        getEnv("LOG_LEVEL", DefaultLogLevel),
	}

	c.ClusterID = os.Getenv("CLUSTER_ID")
	if c.ClusterID == "" {
		return nil, errors.New("CLUSTER_ID is required")
	}

	mgmtdJSON := os.Getenv("MGMtd_ADDRESSES")
	if mgmtdJSON == "" {
		mgmtdJSON = os.Getenv("MGMTPD_ADDRESSES")
	}
	if mgmtdJSON == "" {
		mgmtdJSON = os.Getenv("MGMtd_ADDRESSES_JSON")
	}
	if mgmtdJSON == "" {
		mgmtdJSON = os.Getenv("MGMTD_ADDRESSES")
	}
	if mgmtdJSON == "" {
		return nil, errors.New("MGMTD_ADDRESSES is required (JSON array string)")
	}
	if err := json.Unmarshal([]byte(mgmtdJSON), &c.MgmtdAddresses); err != nil {
		// try comma separated
		parts := strings.Split(mgmtdJSON, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		if len(parts) == 0 || parts[0] == "" {
			return nil, fmt.Errorf("failed to parse MGMTD_ADDRESSES: %v", err)
		}
		c.MgmtdAddresses = parts
	}

	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		if h, err := os.Hostname(); err == nil {
			nodeID = h
		} else {
			nodeID = randomNodeID()
		}
	}
	c.NodeID = nodeID

	return c, nil
}

func (c *Config) GlobalMountPoint() string {
	return filepath.Join(c.GlobalMountBase, c.ClusterID)
}

func (c *Config) LauncherPath() string {
	return filepath.Join(c.ConfigDir, "hf3fs_fuse_main_launcher.toml")
}

func (c *Config) PluginSocketPath() string {
	return filepath.Join("/var/lib/kubelet/plugins", c.DriverName, "csi.sock")
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func randomNodeID() string {
	addrs, _ := net.InterfaceAddrs()
	var parts []string
	for _, a := range addrs {
		parts = append(parts, a.String())
	}
	if len(parts) == 0 {
		return "node-unknown"
	}
	return fmt.Sprintf("node-%x", strings.Join(parts, ","))
}


