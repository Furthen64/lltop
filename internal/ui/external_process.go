package ui

import (
	"os/exec"
	"strconv"
	"strings"
)

type externalProcess struct {
	PID     int
	Command string
}

func detectExternalLlamaServer(selfPID int) (externalProcess, error) {
	cmd := exec.Command("ps", "-eo", "pid=,comm=,args=")
	out, err := cmd.Output()
	if err != nil {
		return externalProcess{}, err
	}
	proc, ok := parseExternalLlamaServer(string(out), selfPID)
	if !ok {
		return externalProcess{}, nil
	}
	return proc, nil
}

func parseExternalLlamaServer(psOutput string, selfPID int) (externalProcess, bool) {
	lines := strings.Split(psOutput, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil || pid <= 0 || pid == selfPID {
			continue
		}
		comm := fields[1]
		if comm != "llama-server" {
			continue
		}
		return externalProcess{
			PID:     pid,
			Command: strings.Join(fields[2:], " "),
		}, true
	}
	return externalProcess{}, false
}
