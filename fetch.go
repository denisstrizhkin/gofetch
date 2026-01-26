package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	COLOR_RESET = "\033[0m"
	COLOR_GREEN = "\033[0;32m"
)

func GetPatternsFromFile(path string, patterns ...string) ([]string, error) {
	f, err := os.ReadFile(path)
	if err != nil {
		log.Printf("opening file - '%s': %v\n", path, err)
		return nil, errors.New("couldn't get patterns")
	}
	f_str := string(f)

	res := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		res = append(res, regexp.MustCompile(pattern))
	}

	matches := make([]string, len(patterns))
	for i, re := range res {
		match := re.FindStringSubmatch(f_str)
		if len(match) > 0 {
			matches[i] = match[1]
		}
	}
	return matches, nil
}

func RunCMD(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("running command - '%s': %v\n", name, err)
		return "", errors.New("couldn't run command")
	}
	return strings.TrimSpace(string(output)), nil
}

func ParseInt(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		log.Fatalln("parsing int:", err)
	}
	return n
}

func GetOS(path, os string) string {
	var osMatches []string
	var err error
	switch os {
	case "darwin":
		osMatches, err = GetPatternsFromFile(
			path,
			`(?s)<key>ProductName</key>\s*<string>(.*?)</string>`,
			`(?s)<key>ProductName</key>\s*<string>(.*?)</string>`,
			`(?s)<key>ProductUserVisibleVersion</key>\s*<string>(.*?)</string>`,
			`(?s)<key>ProductVersion</key>\s*<string>(.*?)</string>`,
		)
	default:
		osMatches, err = GetPatternsFromFile(
			path,
			"^NAME=\"?([^\"]+)", "^PRETTY_NAME=\"?([^\"]+)",
			"^VERSION=\"?([^\"]+)", "^VERSION_ID=\"?([^\"]+)",
		)
	}
	var sOs, sVer string
	if err != nil {
		sOs = "Uknown OS"
	} else {
		if len(osMatches[1]) == 1 {
			sOs = osMatches[1]
		} else {
			sOs = osMatches[0]
		}
		if len(osMatches[2]) == 1 {
			sVer = osMatches[2]
		} else {
			sVer = osMatches[3]
		}
	}
	sArch, err := RunCMD("uname", "-m")
	if err != nil {
		sArch = "Uknown Arch"
	}
	if len(sVer) > 0 {
		return fmt.Sprintf("%s %s %s", sOs, sVer, sArch)
	}
	return fmt.Sprintf("%s %s", sOs, sArch)
}

func GetKernel() string {
	s_kernel, err := RunCMD("uname", "-sr")
	if err != nil {
		return "Uknown Kernel"
	}
	return s_kernel
}

func GetCPU(os string) string {
	switch os {
	case "darwin":
		matches, err := RunCMD("sysctl", "-n", "machdep.cpu.brand_string")
		if err != nil {
			return "Uknown CPU"
		}
		return matches
	default:
		matches, err := GetPatternsFromFile("/proc/cpuinfo", "model name\\s+: (.+)")
		if err != nil {
			return "Uknown CPU"
		}
		return matches[0]
	}
}

func GetMem(sOs string) string {
	switch sOs {
	case "darwin":
		s_mem_total, err := RunCMD("sysctl", "-n", "hw.memsize")
		if err != nil {
			return "Memory Info is Unavailable"
		}
		mem_total := float64(ParseInt(s_mem_total))

		page_size := os.Getpagesize()
		vmOut, err_vm := RunCMD("vm_stat")
		if err_vm != nil {
			return "Memory Info is Unavailable"
		}

		reWired := regexp.MustCompile(`Pages wired down:\s+(\d+)`)
		reActive := regexp.MustCompile(`Pages active:\s+(\d+)`)
		reOccupied := regexp.MustCompile(`Pages occupied by compressor:\s+(\d+)`)

		wiredMatch := reWired.FindStringSubmatch(vmOut)
		activeMatch := reActive.FindStringSubmatch(vmOut)
		occupiedMatch := reOccupied.FindStringSubmatch(vmOut)

		var totalUsedPages int

		if len(wiredMatch) > 1 {
			v := ParseInt(wiredMatch[1])
			totalUsedPages += v
		}
		if len(activeMatch) > 1 {
			v := ParseInt(activeMatch[1])
			totalUsedPages += v
		}
		if len(occupiedMatch) > 1 {
			v := ParseInt(occupiedMatch[1])
			totalUsedPages += v
		}
		mem_used := float64(totalUsedPages * page_size)

		mem_total /= 1024.0 * 1024.0 * 1024.0
		mem_used /= 1024.0 * 1024.0 * 1024.0
		mem_percent := mem_used * 100.0 / mem_total
		return fmt.Sprintf("%.2f / %.2f Gi (%.0f%%)", mem_used, mem_total, mem_percent)

	default:
		s_mem, err := GetPatternsFromFile(
			"/proc/meminfo", "MemTotal:\\s+([0-9]+)", "MemAvailable:\\s+([0-9]+)",
		)
		if err != nil {
			return "Memory Info is Unavailable"
		}
		mem_total := float64(ParseInt(s_mem[0]))
		mem_used := mem_total - float64(ParseInt(s_mem[1]))
		mem_total /= 1024.0 * 1024.0
		mem_used /= 1024.0 * 1024.0
		mem_percent := mem_used * 100.0 / mem_total
		return fmt.Sprintf("%.2f / %.2f Gi (%.0f%%)", mem_used, mem_total, mem_percent)
	}
}

func GetUptime(sOs string) string {
	var seconds int
	switch sOs {
	case "darwin":
		out, err := RunCMD("sysctl", "-n", "kern.boottime")
		if err != nil {
			return "Runtime Info is Unavailable"
		}
		re := regexp.MustCompile(`sec = (\d+)`)
		match := re.FindStringSubmatch(string(out))
		if len(match) > 1 {
			bootTimestamp := ParseInt(match[1])

			seconds = int(time.Now().Unix()) - bootTimestamp
		}
	default:
		s_seconds, err := GetPatternsFromFile("/proc/uptime", "^([0-9]+)")
		if err != nil {
			return "Runtime Info is Unavailable"
		}
		seconds = ParseInt(s_seconds[0])
	}
	mins := seconds / 60 % 60
	hours := seconds / 60 / 60 % 24
	days := seconds / 60 / 60 / 24

	uptime := ""
	if days > 0 {
		uptime = fmt.Sprintf("%d day", days)
		if days > 1 {
			uptime += "s"
		}
	}
	if hours > 0 {
		uptime = fmt.Sprintf("%s, %d hour", uptime, hours)
		if hours > 1 {
			uptime += "s"
		}
	}
	uptime = fmt.Sprintf("%s, %d min", uptime, mins)
	if mins > 1 {
		uptime += "s"
	}
	if uptime[0] == ',' {
		return uptime[2:]
	}
	return uptime
}

func GetShell() string {
	return os.Getenv("SHELL")
}

func GetPortage() string {
	pkgs, err := filepath.Glob("/var/db/pkg/*/*")
	if err != nil {
		log.Fatalln("listing glob", err)
	}
	return fmt.Sprintf("emerge (%d)", len(pkgs))
}

func PrintLine(w *bufio.Writer, key, val string) {
	_, err := w.WriteString(fmt.Sprintf("%s%s%s %s\n", COLOR_GREEN, key, COLOR_RESET, val))
	if err != nil {
		log.Fatalln("writing to stdout", err)
	}
}

func main() {
	w := bufio.NewWriter(os.Stdout)
	sOs := runtime.GOOS
	var os_path string
	defer w.Flush()
	switch sOs {
	case "linux":
		os_path = "/etc/os-release"
	case "darwin":
		os_path = "/System/Library/CoreServices/SystemVersion.plist"
	case "windows":
		os_path = "Registry"
	default:
		os_path = sOs
	}
	PrintLine(w, "OS:      ", GetOS(os_path, sOs))
	PrintLine(w, "Kernel:  ", GetKernel())
	PrintLine(w, "CPU:     ", GetCPU(sOs))
	PrintLine(w, "Memory:  ", GetMem(sOs))
	PrintLine(w, "Uptime:  ", GetUptime(sOs))
	PrintLine(w, "Shell:   ", GetShell())
	PrintLine(w, "Packages:", GetPortage())
}
