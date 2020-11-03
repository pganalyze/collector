package log

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

type Logger struct {
	StandardOutput io.Writer
	VerboseOutput  io.Writer
	indent         int
}

func NewLogger() Logger {
	return Logger{
		StandardOutput: os.Stdout,
		VerboseOutput:  ioutil.Discard,
	}
}

func (l *Logger) StartStep(description string) {
	line := fmt.Sprintf("\033[1m*\033[0m %s", description)
	l.Verbose(line)
	l.indent += 1
}

func (l *Logger) EndStep() {
	l.indent -= 1
	l.Verbose("") // add a blank line after every step
}

func (l *Logger) Log(line string, params ...interface{}) error {
	return l.write(l.StandardOutput, line, params...)
}

func (l *Logger) Verbose(line string, params ...interface{}) error {
	return l.write(l.VerboseOutput, line, params...)
}

func (l *Logger) write(to io.Writer, line string, params ...interface{}) error {
	var prefix = ""
	for i := 0; i < l.indent; i++ {
		prefix += "  "
	}

	_, err := to.Write([]byte(prefix + fmt.Sprintf(line, params...) + "\n"))
	return err
}
