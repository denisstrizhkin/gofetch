package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
)

const (
	COLOR_RESET = "\033[0m"
	COLOR_GREEN = "\033[0;32m"
)

func GetPatternsFromFile(path string, patterns ...string) []string {
	f, err := os.Open(path)
	if err != nil {
		log.Fatalln("opening file:", err)
	}
	defer f.Close()

	res := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		res = append(res, regexp.MustCompile(pattern))
	}

	matches := make([]string, len(patterns))
	bs := bufio.NewScanner(f)
	for bs.Scan() {
		for i, re := range res {
			match := re.FindStringSubmatch(bs.Text())
			if len(match) > 0 {
				matches[i] = match[1]
			}
		}
	}

	if err := bs.Err(); err != nil {
		log.Fatalln("reading file:", err)
	}
	return matches
}

func RunCMD(name string, args ...string) string {
	cmd := exec.Command(name, args...)
	output, err := cmd.Output()
	if err != nil {
		log.Fatalln("running command", err)
	}
	return string(output)
}

func ParseInt(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		log.Fatalln("parsing int:", err)
	}
	return n
}

func GetOS() string {
	s_os := GetPatternsFromFile(
		"/etc/os-release",
		"^NAME=\"?([^\"]+)", "^VERSION=\"?([^\"]+)", "^VERSION_ID=\"?([^\"]+)",
	)
	s_arch := RunCMD("uname", "-m")
	s_arch = s_arch[:len(s_arch)-1]
	if len(s_os[1]) > 0 {
		return fmt.Sprintf("%s %s %s", s_os[0], s_os[1], s_arch)
	}
	return fmt.Sprintf("%s %s %s", s_os[0], s_os[2], s_arch)
}

func GetKernel() string {
	s_kernel := RunCMD("uname", "-sr")
	return s_kernel[:len(s_kernel)-1]
}

func GetCPU() string {
	return GetPatternsFromFile("/proc/cpuinfo", "model name\\s+: (.+)")[0]
}

func GetMem() string {
	s_mem := GetPatternsFromFile(
		"/proc/meminfo", "MemTotal:\\s+([0-9]+)", "MemAvailable:\\s+([0-9]+)",
	)
	mem_total := float64(ParseInt(s_mem[0]))
	mem_used := mem_total - float64(ParseInt(s_mem[1]))
	mem_total /= 1024.0 * 1024.0
	mem_used /= 1024.0 * 1024.0
	mem_percent := mem_used * 100.0 / mem_total

	return fmt.Sprintf("%.2f / %.2f Gi (%.0f%%)", mem_used, mem_total, mem_percent)
}

func GetUptime() string {
	s_seconds := GetPatternsFromFile("/proc/uptime", "^([0-9]+)")[0]
	seconds := ParseInt(s_seconds)
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
	defer w.Flush()
	PrintLine(w, "OS:", GetOS())
	PrintLine(w, "Kernel:", GetKernel())
	PrintLine(w, "CPU:", GetCPU())
	PrintLine(w, "Memory:", GetMem())
	PrintLine(w, "Uptime:", GetUptime())
	PrintLine(w, "Shell:", GetShell())
	PrintLine(w, "Packages:", GetPortage())
}
