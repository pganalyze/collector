package service

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	s "github.com/pganalyze/collector/setup/state"
)

func RestartPostgres(state *s.SetupState) error {
	usePgCtl := os.Getenv("PGA_SETUP_USE_PG_CTL")

	if usePgCtl != "" {
		return restartPostgresPgCtl(state)
	} else {
		return restartPostgresSystemd()
	}
}

func restartPostgresSystemd() error {
	cmd := exec.Command("systemctl", "restart", "postgresql")
	out, err := cmd.CombinedOutput()
	if err != nil {
		var errInfo = err.Error()
		if len(out) > 0 {
			errInfo += "; " + string(out)
		}
		return fmt.Errorf("failed to restart: %s", errInfo)
	}
	return nil
}

func restartPostgresPgCtl(state *s.SetupState) error {
	row, err := state.QueryRunner.QueryRow("SHOW data_directory")
	if err != nil {
		return err
	}
	dataDir := row.GetString(0)
	dataDirInfo, err := os.Stat(dataDir)

	var uid uint32
	var gid uint32
	if stat, ok := dataDirInfo.Sys().(*syscall.Stat_t); ok {
		uid = stat.Uid
		gid = stat.Gid
	} else {
		return errors.New("could not determine data directory ownership")
	}
	datDirOwner, err := user.LookupId(fmt.Sprintf("%d", uid))
	if err != nil {
		return err
	}
	gids, err := datDirOwner.GroupIds()
	if err != nil {
		return fmt.Errorf("could not determine data directory ownership: %s", err)
	}

	// N.B.: need to fetch user's additional groups, since e.g. in the default Ubuntu
	// install, the Postgres user has access to /etc/ssl/private/ssl-cert-snakeoil.key
	// through the ssl-cert group, and reading that is required during restart
	var numGids []uint32
	for _, gidStr := range gids {
		gidNum, err := strconv.ParseUint(gidStr, 10, 32)
		if err != nil {
			return fmt.Errorf("could not determine data directory ownership: user group %s could not be parsed: %s", gidStr, err)
		}
		gidNum32 := uint32(gidNum)
		if gidNum32 != gid {
			numGids = append(numGids, gidNum32)
		}
	}

	pgCtlPath, err := getPgCtlLocation()
	cmd := exec.Command(pgCtlPath, "--pgdata", dataDir, "--wait", "--mode", "fast", "restart")
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Credential = &syscall.Credential{
		Uid:    uid,
		Gid:    gid,
		Groups: numGids,
	}
	cmd.Stdin = nil

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err != nil {
		return err
	}
	// N.B.: this is a bit weird because since cmd.Wait waits for the invoked process to close stdout
	// and stderr, but pg_ctl does not do that, possibly related to: https://github.com/golang/go/issues/13155 .
	// Instead, we copy what is available, then wait and if we hit os.ErrClosed on either stream we
	// ignore it
	stdoutCh := make(chan OutputResult)
	stderrCh := make(chan OutputResult)
	go getOutput(stdout, stdoutCh)
	go getOutput(stderr, stderrCh)

	err = cmd.Wait()
	outResult := <-stdoutCh
	errResult := <-stderrCh
	if err != nil || outResult.err != nil || errResult.err != nil {
		return fmt.Errorf(
			"cmd err: %s\nstdout: %s\nstdout err: %s\nstderr: %s\nstderr err: %s",
			err,
			outResult.content,
			outResult.err,
			errResult.content,
			errResult.err,
		)
	}

	return nil
}

type OutputResult struct {
	content string
	err     error
}

func getOutput(stream io.ReadCloser, ch chan<- OutputResult) {
	var result strings.Builder
	var buf = make([]byte, 1024)
	for {
		n, err := stream.Read(buf)
		if n > 0 {
			result.Write(buf[0:n])
		}
		if err != nil {
			var actualErr error
			if err == io.EOF || err == os.ErrClosed || errors.Unwrap(err) == os.ErrClosed {
				actualErr = nil
			} else {
				actualErr = err
			}
			ch <- OutputResult{result.String(), actualErr}
			break
		}
	}
}

func getPgCtlLocation() (string, error) {
	_, err := exec.Command("pg_ctl", "--help").CombinedOutput()
	if err == nil {
		// it's in PATH, no need to look for it
		return "pg_ctl", nil
	}
	cmd := exec.Command("pg_config")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}
	err = cmd.Start()
	if err != nil {
		return "", err
	}

	stdoutBytes, err := ioutil.ReadAll(stdout)
	if err != nil {
		return "", err
	}
	stderrBytes, err := ioutil.ReadAll(stderr)
	if err != nil {
		return "", err
	}

	err = cmd.Wait()
	if err != nil {
		return "", fmt.Errorf("%s\n%s", err, string(stderrBytes))
	}

	scanner := bufio.NewScanner(bytes.NewReader(stdoutBytes))
	for scanner.Scan() {
		line := scanner.Text()
		keyVal := strings.Split(line, "=")
		if len(keyVal) != 2 {
			continue
		}

		key := strings.TrimSpace(keyVal[0])
		if key != "BINDIR" {
			continue
		}

		val := strings.TrimSpace(keyVal[1])
		return filepath.Join(val, "pg_ctl"), nil
	}

	return "", errors.New("could not find pg_ctl")
}
