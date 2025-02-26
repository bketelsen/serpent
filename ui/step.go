package ui

import (
	"fmt"
	"io"
)

/*
	s := ui.Step("Creating Instance")
	s.Warn(...)
	s.Info(...)

	ui.Step("One Shot").Info(...)


*/

type stepWriter struct {
	Name string
	wtr  io.Writer
}

func Step(name string, writer io.Writer) stepWriter {
	return stepWriter{
		Name: name,
		wtr:  writer,
	}
}

// Info writes a log to the writer provided.
func (s stepWriter) Debug(header string, lines ...string) {
	_, _ = fmt.Fprint(s.wtr, cliMessage{
		Prefix: "[" + s.Name + "] " + "DEBUG: ",
		Header: header,
		Lines:  lines,
	}.String())
}

// Info writes a log to the writer provided.
func (s stepWriter) Info(header string, lines ...string) {
	_, _ = fmt.Fprint(s.wtr, cliMessage{
		Prefix: "[" + s.Name + "] ",
		Header: header,
		Lines:  lines,
	}.String())
}

// Warn writes a log to the writer provided.
func (s stepWriter) Warn(header string, lines ...string) {
	_, _ = fmt.Fprint(s.wtr, cliMessage{
		Style:  DefaultStyles.Warn,
		Prefix: "[" + s.Name + "] " + "WARNING: ",
		Header: header,
		Lines:  lines,
	}.String())
}

// Error writes a log to the writer provided.
func (s stepWriter) Error(header string, lines ...string) {
	_, _ = fmt.Fprint(s.wtr, cliMessage{
		Style:  DefaultStyles.Error,
		Prefix: "[" + s.Name + "] " + "ERROR: ",
		Header: header,
		Lines:  lines,
	}.String())
}
