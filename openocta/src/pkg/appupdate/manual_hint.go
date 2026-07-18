package appupdate

import (
	"fmt"
	"path/filepath"
	"strings"
)

func linuxManualInstallHint(urls map[string]string) string {
	if len(urls) == 0 {
		return ""
	}
	deb := urls[string(FormatDEB)]
	rpm := urls[string(FormatRPM)]
	tarball := urls[string(FormatTarGz)]

	var b strings.Builder
	b.WriteString("若自动安装失败（例如需要 sudo 密码），可在本机终端手动执行：\n\n")

	if deb != "" {
		debFile := filepath.Base(deb)
		b.WriteString("Debian / Ubuntu:\n")
		b.WriteString(fmt.Sprintf("  curl -fLO %s\n", deb))
		b.WriteString("  sudo systemctl stop openocta\n")
		b.WriteString(fmt.Sprintf("  sudo dpkg -i %s\n", debFile))
		b.WriteString("  sudo systemctl start openocta\n\n")
	}
	if rpm != "" {
		rpmFile := filepath.Base(rpm)
		b.WriteString("RHEL / CentOS / Fedora:\n")
		b.WriteString(fmt.Sprintf("  curl -fLO %s\n", rpm))
		b.WriteString("  sudo systemctl stop openocta\n")
		b.WriteString(fmt.Sprintf("  sudo rpm -Uvh %s\n", rpmFile))
		b.WriteString("  sudo systemctl start openocta\n\n")
	}
	if tarball != "" {
		tarFile := filepath.Base(tarball)
		b.WriteString("tar.gz 手动部署:\n")
		b.WriteString(fmt.Sprintf("  curl -fLO %s\n", tarball))
		b.WriteString("  sudo systemctl stop openocta\n")
		b.WriteString(fmt.Sprintf("  tar -xzf %s\n", tarFile))
		b.WriteString("  sudo install -m 0755 openocta_*/openocta /usr/bin/openocta\n")
		b.WriteString("  sudo systemctl start openocta\n")
	}
	return strings.TrimSpace(b.String())
}
