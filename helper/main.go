package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
)

type helperStatus struct {
	PostmasterPid    int
	DataDirectory    string
	XlogDirectory    string
	XlogUsedBytes    uint64
	SystemIdentifier string
}

func getStatus() {
	var postmasterPidStr, pgControldataOut, xlogUsageBytesStr []byte
	var pgControldataBinary string
	var status helperStatus
	var err error

	postmasterPidStr, err = exec.Command("pgrep", "-U", "postgres", "-o").Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to find Postgres Postmaster Pid: %s\n", err)
	} else {
		status.PostmasterPid, err = strconv.Atoi(string(postmasterPidStr[:len(postmasterPidStr)-1]))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Postgres Pid is not an integer: %s\n", err)
		} else {
			status.DataDirectory = os.Getenv("PGDATA")
			if status.DataDirectory == "" {
				status.DataDirectory, err = filepath.EvalSymlinks("/proc/" + strconv.Itoa(status.PostmasterPid) + "/cwd")
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to resolve data directory path: %s\n", err)
				}
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

			var cmdOut []byte
			cmdOut, err = exec.Command("locate", "-r", "bin/pg_controldata$").Output()
			if err != nil {
				pgControldataBinary = "pg_controldata"
			} else {
				pgControldataBinary = string(cmdOut[:len(cmdOut)-1])
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
	}

	out, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not marshal JSON: %s", err)
	}

	fmt.Printf("%s\n", out)
}

func getLogDirectory() {
	postgresUser, err := user.Lookup("postgres")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error retrieving postgres UID: %s", err)
	}
	postgresUID, _ := strconv.Atoi(postgresUser.Uid)

	cmd := exec.Command("psql", "-t", "-A", "-c", "SHOW log_directory")
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uint32(postgresUID)}}
	logDirectory, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running psql: %s", err)
	} else {
		fmt.Printf("%s", logDirectory)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Please pass a command to run as the first argument - valid choices are: status, log_directory\n")
		return
	}

	switch os.Args[1] {
	case "status":
		getStatus()
	case "log_directory":
		getLogDirectory()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
	}
}
