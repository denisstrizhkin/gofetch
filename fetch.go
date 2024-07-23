package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

const (
	COLOR_RESET = "\033[0m"
	COLOR_GREEN = "\033[0;32m"
)

func GetPatternsFromFile(path string, patterns []string) []string {
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

func GetPatternFromFile(path string, pattern string) string {
	return GetPatternsFromFile(path, []string{pattern})[0]
}

func ParseInt(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		log.Fatalln("parsing int:", err)
	}
	return n
}

func GetOS() string {
	return GetPatternFromFile("/etc/os-release", "PRETTY_NAME=\"(.+)\"")
}

func GetKernel() string {
	return GetPatternFromFile("/proc/version", "Linux version (.+?)\\s")
}

func GetCPU() string {
	return GetPatternFromFile("/proc/cpuinfo", "model name\\s+: (.+)")
}

func GetMem() string {
	s_mem := GetPatternsFromFile(
		"/proc/meminfo", []string{"MemTotal:\\s+([0-9]+)", "MemAvailable:\\s+([0-9]+)"},
	)
	mem_total := float64(ParseInt(s_mem[0]))
	mem_used := mem_total - float64(ParseInt(s_mem[1]))
	mem_total /= 1024.0 * 1024.0
	mem_used /= 1024.0 * 1024.0
	mem_percent := mem_used * 100.0 / mem_total

	return fmt.Sprintf("%.2f / %.2f Gi (%.0f%%)", mem_used, mem_total, mem_percent)
}

func GetUptime() string {
	s_seconds := GetPatternFromFile("/proc/uptime", "^([0-9]+)")
	seconds := ParseInt(s_seconds)
	mins := seconds / 60 % 60
	hours := seconds / 60 / 60 % 24
	days := seconds / 60 / 60 / 24

	if days == 0 && hours == 0 {
		return fmt.Sprintf("%d mins", mins)
	} else if days == 0 {
		return fmt.Sprintf("%d hours, %d mins", hours, mins)
	}
	return fmt.Sprintf("%d days, %d hours, %d mins", days, hours, mins)
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
