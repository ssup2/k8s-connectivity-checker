package kcchecker

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func cmdPing(contRuntime string, contID string, ip string) (float64, error) {
	// Set command
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command("cnsenter", "-R", contRuntime, "-c", contID, "-n", "--", "ping", "-w", "1", "-c", "1", ip)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run command
	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return 0, fmt.Errorf("exit error:%s", stderr.String())
		}
		return 0, err
	}

	// Parse avg ms
	// stdout format : round-trip min/avg/max = 0.025/0.025/0.025 ms
	tokens := strings.Split(stdout.String(), "/")
	if len(tokens) < 4 {
		return 0, fmt.Errorf("failed to split stdout:%s", stdout.String())
	}
	avgMS, err := strconv.ParseFloat(tokens[3], 64)
	if err != nil {
		return 0, err
	}
	return avgMS, nil
}

func cmdNcatConn(contRuntime string, contID string, ip string, port int32) (float64, error) {
	// Set command
	var stderr bytes.Buffer
	cmd := exec.Command("cnsenter", "-R", contRuntime, "-c", contID, "-n", "--", "ncat", "-w", "5", "-z", "-v", ip, strconv.Itoa(int(port)))
	cmd.Stderr = &stderr

	// Run command
	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return 0, fmt.Errorf("exit error:%s", stderr.String())
		}
		return 0, err
	}

	// Parse time
	// stderr format : Ncat: 0 bytes sent, 0 bytes received in 0.01 seconds.
	tokens := strings.Split(stderr.String(), " ")
	if !strings.Contains(tokens[len(tokens)-1], "seconds.") {
		return 0, fmt.Errorf("failed to split stderr:%s", stderr.String())
	}
	avgSec, err := strconv.ParseFloat(tokens[len(tokens)-2], 64)
	if err != nil {
		return 0, err
	}
	return avgSec * 1000, nil // Convert to seconds to ms
}
