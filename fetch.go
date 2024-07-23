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

func ReadFile(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		log.Fatalln("opening file:", err)
	}
	defer f.Close()

	var lines []string
	bs := bufio.NewScanner(f)
	for bs.Scan() {
		lines = append(lines, bs.Text())
	}

	if err := bs.Err(); err != nil {
		log.Fatalln("reading file:", err)
	}
	return lines
}

func GetPatternFromFile(file, pattern string) string {
	lines := ReadFile(file)
	re := regexp.MustCompile(pattern)
	for _, line := range lines {
		match := re.FindStringSubmatch(line)
		if len(match) > 0 {
			return match[1]
		}
	}
	return "Uknown"
}

func ParseInt(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		log.Fatalln("parsing int:", err)
	}
	return n
}

func GetMem() string {
	s_mem_total := GetPatternFromFile("/proc/meminfo", "MemTotal:\\s+([0-9]+)")
	s_mem_avail := GetPatternFromFile("/proc/meminfo", "MemAvailable:\\s+([0-9]+)")
	mem_total := float64(ParseInt(s_mem_total)) / 1024.0 / 1024.0
	mem_avail := float64(ParseInt(s_mem_avail)) / 1024.0 / 1024.0
	return fmt.Sprintf("%.1f/%.1f GBs", mem_total-mem_avail, mem_total)
}

func GetUptime() string {
	s_seconds := GetPatternFromFile("/proc/uptime", "^([0-9]+)")
	seconds := ParseInt(s_seconds)
	return fmt.Sprintf("%dh %dm", seconds/60/60, seconds/60%60)
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
	PrintLine(w, "OS:", GetPatternFromFile("/etc/os-release", "PRETTY_NAME=\"(.+)\""))
	PrintLine(w, "Kernel:", GetPatternFromFile("/proc/version", "Linux version (.+?)\\s"))
	PrintLine(w, "CPU:", GetPatternFromFile("/proc/cpuinfo", "model name\\s+: (.+)"))
	PrintLine(w, "Memory:", GetMem())
	PrintLine(w, "Uptime:", GetUptime())
	PrintLine(w, "Packages:", GetPortage())
}
