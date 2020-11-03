package query

import (
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"strconv"
	"strings"
	"syscall"
)

type Runner struct {
	User     string
	Host     string
	Port     int
	Password string
	Database string

	csvChecked bool
	csv        bool
	separator  rune
}

func NewRunner(user, host string, port int) *Runner {
	return &Runner{User: user, Host: host, Port: port, Password: "", Database: "", separator: '\t', csv: false}
}

func (qr *Runner) InDB(dbname string) *Runner {
	var newRunner Runner = *qr
	newRunner.Database = dbname
	return &newRunner
}

// N.B.: older Postgres version will follow with another trailing dot and the point
// release number, but we don't care about that so we can match just the first two.
// This format goes back to at least Postgres 10.0, our oldest supported version
// right now.
var versionRE = regexp.MustCompile("psql \\(PostgreSQL\\) (\\d+\\.\\d+)")

func (qr *Runner) checkUseCSV() {
	if qr.csvChecked {
		return
	}
	// the check is best-effort, so just flag it as done now:
	qr.csvChecked = true

	// we could use \show :VERSION_NUM here, but this allows us to check the version without
	// requiring a valid connection, so we can avoid error handling for that here
	output, err := exec.Command("psql", "--no-psqlrc", "--version").CombinedOutput()
	if err != nil {
		return
	}
	verMatches := versionRE.FindStringSubmatch(string(output))
	if len(verMatches) != 2 {
		return
	}
	verNum, err := strconv.ParseFloat(verMatches[1], 32)
	if err != nil {
		return
	}
	if verNum >= 12 {
		qr.csv = true
		qr.separator = ','
	}
}

func (qr *Runner) PingSuper() error {
	// TODO: we should account for cloud provider faux superusers (since we may
	// want a consistent interface for this even if users have to enter credentials)
	row, err := qr.QueryRow("SELECT usesuper FROM pg_user WHERE usename = current_user")
	if err != nil {
		return err
	}
	if !row.GetBool(0) {
		return fmt.Errorf("user %s is not a superuser; Postgres superuser is required for setup", qr.User)
	}
	return nil
}

func (qr *Runner) runSQL(sql string) (string, error) {
	qr.checkUseCSV()
	// N.B.: we pass search_path as a separate command, since otherwise psql sends the whole
	// thing to the server as a single query, which forces a single transaction, which is not
	// supported for some things we want to run like ALTER SYSTEM
	args := []string{
		"--no-psqlrc", "--tuples-only", "--command", "SET search_path = pg_catalog", "--command", sql,
	}
	if qr.csv {
		args = append(args, "--csv")
	} else {
		args = append(args, "--no-align", "--field-separator", string(qr.separator))
	}

	cmd := exec.Command("psql", args...)
	cmd.Env = os.Environ()
	// N.B.: if there are conflicts, these later values override what's in os.Environ()
	if qr.Host != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("PGHOST=%s", qr.Host))
	}
	if qr.Port != 0 {
		cmd.Env = append(cmd.Env, fmt.Sprintf("PGPORT=%d", qr.Port))
	}
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
	pgUser, err := user.Lookup(qr.User)
	if err != nil {
		return "", err
	}
	var pgUserUid uint64
	pgUserUid, err = strconv.ParseUint(pgUser.Uid, 10, 32)
	if err != nil {
		return "", err
	}
	var pgUserGid uint64
	pgUserGid, err = strconv.ParseUint(pgUser.Gid, 10, 32)
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
	// See below: we need to ignore the first row, the result
	// of SET search_path, which may produce a different number
	// of columns (1) than our actual query
	r.FieldsPerRecord = -1
	r.Comma = qr.separator
	data, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	var rows []Row
	// N.B.: we ignore the first row, because that will
	// always be the result of SET search_path
	for _, row := range data[1:] {
		rows = append(rows, row)
	}
	return rows, err
}

func (qr *Runner) Exec(sql string) error {
	_, err := qr.runSQL(sql)
	return err
}
