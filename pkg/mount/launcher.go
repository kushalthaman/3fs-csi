package mount

import (
	"fmt"
	"strings"

	"github.com/3fs-csi/3fs-csi/pkg/config"
)

func renderLauncherTOML(cfg *config.Config) string {
	var b strings.Builder
	b.WriteString("cluster_id = \"")
	b.WriteString(cfg.ClusterID)
	b.WriteString("\"\n")
	b.WriteString("mountpoint = \"")
	b.WriteString(cfg.GlobalMountPoint())
	b.WriteString("\"\n")
	b.WriteString("token_file = \"")
	b.WriteString(cfg.TokenFile)
	b.WriteString("\"\n\n")
	b.WriteString("[mgmtd_client]\n")
	b.WriteString("mgmtd_server_addresses = ")
	b.WriteString(formatStringArray(cfg.MgmtdAddresses))
	b.WriteString("\n")
	return b.String()
}

func formatStringArray(items []string) string {
	var parts []string
	for _, s := range items {
		parts = append(parts, fmt.Sprintf("\"%s\"", s))
	}
	return fmt.Sprintf("[%s]", strings.Join(parts, ", "))
}


