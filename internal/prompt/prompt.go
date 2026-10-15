// Package prompt asks the person running the tool questions, in plain lines, with hidden input
// for secrets.
package prompt

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// Asker asks questions.
type Asker interface {
	// Ask returns the answer, or def when it's blank.
	Ask(question, def string) (string, error)
	// Secret returns an answer typed without being shown.
	Secret(question string) (string, error)
	// Confirm returns a yes or no, def when it's blank.
	Confirm(question string, def bool) (bool, error)
	// Line reads a line as it is; io.EOF when input ends.
	Line(prompt string) (string, error)
}

// Terminal asks on In and Out, hiding secrets when In is a terminal.
type Terminal struct {
	In  *os.File
	Out io.Writer
	r   *bufio.Reader
}

func (t *Terminal) reader() *bufio.Reader {
	if t.r == nil {
		t.r = bufio.NewReader(t.In)
	}
	return t.r
}

func (t *Terminal) Line(prompt string) (string, error) {
	fmt.Fprint(t.Out, prompt)
	line, err := t.reader().ReadString('\n')
	if err != nil && (line == "" || !errors.Is(err, io.EOF)) {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func (t *Terminal) Ask(question, def string) (string, error) {
	shown := question
	if def != "" {
		shown += " [" + def + "]"
	}
	answer, err := t.Line(shown + ": ")
	if err != nil {
		return "", err
	}
	if answer = strings.TrimSpace(answer); answer == "" {
		return def, nil
	}
	return answer, nil
}

func (t *Terminal) Secret(question string) (string, error) {
	fmt.Fprint(t.Out, question+": ")
	if !term.IsTerminal(int(t.In.Fd())) {
		line, err := t.reader().ReadString('\n')
		if err != nil && line == "" {
			return "", err
		}
		return strings.TrimSpace(line), nil
	}
	value, err := term.ReadPassword(int(t.In.Fd()))
	fmt.Fprintln(t.Out)
	return strings.TrimSpace(string(value)), err
}

func (t *Terminal) Confirm(question string, def bool) (bool, error) {
	hint := "y/N"
	if def {
		hint = "Y/n"
	}
	for {
		answer, err := t.Line(fmt.Sprintf("%s [%s]: ", question, hint))
		if err != nil {
			return false, err
		}
		switch strings.ToLower(strings.TrimSpace(answer)) {
		case "":
			return def, nil
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		}
		fmt.Fprintln(t.Out, "Please answer y or n.")
	}
}
