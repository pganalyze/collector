package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type helperStatus struct {
	PostmasterPid    int
	DataDirectory    string
	XlogDirectory    string
	XlogUsedBytes    uint64
	SystemIdentifier string
}

func getPostmasterPid() (int, error) {
	pgPidStr, err := exec.Command("pgrep", "-U", "postgres", "-o", "postgres").Output()
	if err != nil {
		// on some systems (e.g., RHEL), the postgres process uses the name "postmaster",
		// so try that as a fallback
		pgPidStr, err = exec.Command("pgrep", "-U", "postgres", "-o", "postmaster").Output()
	}
	if err != nil {
		return -1, fmt.Errorf("Failed to find Postgres Postmaster Pid: %s", err)
	}

	pgPid, err := strconv.Atoi(string(pgPidStr[:len(pgPidStr)-1]))
	if err != nil {
		return -1, fmt.Errorf("Postgres Pid is not an integer: %s", err)
	}

	return pgPid, nil
}

func getStatus(dataDirectory string) {
	var pgControldataOut, xlogUsageBytesStr []byte
	var pgControldataBinary string
	var status helperStatus
	var err error

	status.PostmasterPid, err = getPostmasterPid()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
	} else {
		status.DataDirectory = dataDirectory
		if status.DataDirectory == "" {
			status.DataDirectory = os.Getenv("PGDATA")
		}

		if status.DataDirectory == "" {
			status.DataDirectory = "/proc/" + strconv.Itoa(status.PostmasterPid) + "/cwd"
		}

		status.DataDirectory, err = filepath.EvalSymlinks(status.DataDirectory)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to resolve data directory path: %s\n", err)
		}

		xlogDirectoryName := "pg_wal"
		if _, err = os.Stat(status.DataDirectory + "/" + xlogDirectoryName); os.IsNotExist(err) {
			xlogDirectoryName = "pg_xlog"
		}

		status.XlogDirectory, err = filepath.EvalSymlinks(status.DataDirectory + "/" + xlogDirectoryName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to resolve xlog path: %s\n", err)
			if status.DataDirectory != "" {
				status.XlogDirectory = status.DataDirectory + "/" + xlogDirectoryName
			}
		}

		xlogUsageBytesStr, err = exec.Command("du", "-b", "-s", status.XlogDirectory).Output()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to determine xlog disk usage: %s\n", err)
		} else {
			status.XlogUsedBytes, err = strconv.ParseUint(strings.Fields(string(xlogUsageBytesStr))[0], 10, 64)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Xlog disk usage is not an integer: %s\n", err)
			}
		}

		postmasterBin, err := filepath.EvalSymlinks(filepath.Join("/proc", strconv.Itoa(status.PostmasterPid), "exe"))
		if err != nil {
			var cmdOut []byte
			cmdOut, err = exec.Command("locate", "-r", "bin/pg_controldata$").Output()
			if err != nil {
				pgControldataBinary = "pg_controldata"
			} else {
				pgControldataBinary = string(cmdOut[:len(cmdOut)-1])
			}
		} else {
			pgControldataBinary = filepath.Join(filepath.Dir(postmasterBin), "pg_controldata")
		}

		pgControldataOut, err = exec.Command(pgControldataBinary, status.DataDirectory).Output()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to run pg_controldata: %s\n", err)
		} else {
			re := regexp.MustCompile("Database system identifier:\\s+(\\d+)")
			match := re.FindStringSubmatch(string(pgControldataOut))
			if len(match) > 1 {
				status.SystemIdentifier = match[1]
			}
		}
	}

	out, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not marshal JSON: %s", err)
	}

	fmt.Printf("%s\n", out)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Please pass a command to run as the first argument - valid choices are: status\n")
		return
	}

	switch os.Args[1] {
	case "status":
		var dataDir string
		if len(os.Args) == 3 {
			dataDir = os.Args[2]
		}
		getStatus(dataDir)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
	}
}
