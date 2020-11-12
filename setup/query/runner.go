package query

import (
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"syscall"
)

type Runner struct {
	User     string
	Password string
	Database string
}

func NewRunner() *Runner {
	return &Runner{User: "postgres", Password: "", Database: ""}
}

func (qr *Runner) Ping() error {
	// TODO: we should check that this is actually a superuser
	// and account for cloud provider faux superusers (since we
	// may want a consistent interface for this even if users
	// have to enter credentials)
	_, err := qr.runSQL("select 1")
	return err
}

func (qr *Runner) runSQL(sql string) (string, error) {
	// TODO: should we try to find the socket for psql here and pass it as -d or PGHOST,
	// rather than relying on it to do that itself?
	cmd := exec.Command("psql", "--no-psqlrc", "--csv", "--tuples-only", "--command", sql)
	cmd.Env = os.Environ()
	// N.B.: if there are conflicts, these later values override what's in os.Environ()
	if qr.User != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("PGUSER=%s", qr.User))
	}
	if qr.Password != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("PGPASSWORD=%s", qr.Password))
	}
	if qr.Database != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("PGDATABASE=%s", qr.Database))
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	pgUser, err := user.Lookup("postgres")
	if err != nil {
		return "", err
	}
	var pgUserUid int64
	pgUserUid, err = strconv.ParseInt(pgUser.Uid, 10, 64)
	if err != nil {
		return "", err
	}
	var pgUserGid int64
	pgUserGid, err = strconv.ParseInt(pgUser.Gid, 10, 64)
	if err != nil {
		return "", err
	}
	cmd.SysProcAttr.Credential = &syscall.Credential{
		Uid: uint32(pgUserUid),
		Gid: uint32(pgUserGid),
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, _ := cmd.StderrPipe()
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
	stderrBytes, _ := ioutil.ReadAll(stderr)
	if err != nil {
		return "", err
	}

	err = cmd.Wait()
	if err != nil {
		return "", fmt.Errorf("%s\n%s", err, string(stderrBytes))
	}

	return string(stdoutBytes), nil
}

func (qr *Runner) QueryRow(sql string) (Row, error) {
	rows, err := qr.Query(sql)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNoRows
	} else if len(rows) > 1 {
		return nil, fmt.Errorf("expected one row; got %d", len(rows))
	}
	return rows[0], err
}

func (qr *Runner) Query(sql string) ([]Row, error) {
	result, err := qr.runSQL(sql)
	if err != nil {
		return nil, err
	}

	r := csv.NewReader(strings.NewReader(result))
	data, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	var rows []Row
	for _, row := range data {
		rows = append(rows, row)
	}
	return rows, err
}

func (qr *Runner) Exec(sql string) error {
	_, err := qr.runSQL(sql)
	return err
}
