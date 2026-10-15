package main

import (
	"errors"
	"flag"
	"io"
	"testing"
)

func TestFlagsAreReadBeforeOrAfterTheCommand(t *testing.T) {
	cases := [][]string{
		{"--hard-reset", "init"},
		{"-hard-reset", "init"},
		{"init", "--hard-reset"},
		{"init", "-hard-reset"},
		{"--dir", "/srv/x", "init", "--hard-reset"},
		{"init", "--dir=/srv/x", "-hard-reset"},
	}
	for _, args := range cases {
		o, err := parseArgs(args, io.Discard)
		if err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		if o.command != "init" || !o.hardReset {
			t.Errorf("%v: command %q, hard reset %v; want init, true", args, o.command, o.hardReset)
		}
	}
	o, _ := parseArgs([]string{"up", "--dir", "/srv/x", "-url", "http://box:8080"}, io.Discard)
	if o.command != "up" || o.dir != "/srv/x" || o.url != "http://box:8080" {
		t.Errorf("up with flags after it: %+v", o)
	}
}

func TestNoFlagsLeavesTheDefaults(t *testing.T) {
	o, err := parseArgs(nil, io.Discard)
	if err != nil || o.command != "" || o.dir != "." || o.url != "http://127.0.0.1:8080" || o.hardReset || o.showVersion {
		t.Errorf("no arguments: %+v, %v", o, err)
	}
}

func TestEveryWayOfAskingForTheVersion(t *testing.T) {
	for _, args := range [][]string{{"version"}, {"--version"}, {"-version"}, {"-v"}, {"--v"}, {"--dir", "x", "version"}} {
		o, err := parseArgs(args, io.Discard)
		if err != nil || !o.showVersion {
			t.Errorf("%v: showVersion %v, %v", args, o.showVersion, err)
		}
	}
	o, _ := parseArgs([]string{"up"}, io.Discard)
	if o.showVersion {
		t.Error("up asked for the version")
	}
}

func TestMoreThanOneCommandOrAnUnknownFlagIsRefused(t *testing.T) {
	for _, args := range [][]string{{"init", "up"}, {"up", "--frobnicate"}, {"--frobnicate"}} {
		if _, err := parseArgs(args, io.Discard); err == nil {
			t.Errorf("%v: no error", args)
		}
	}
}

func TestHelpIsAskedFor(t *testing.T) {
	for _, args := range [][]string{{"-h"}, {"--help"}, {"up", "-h"}} {
		if _, err := parseArgs(args, io.Discard); !errors.Is(err, flag.ErrHelp) {
			t.Errorf("%v: %v, want flag.ErrHelp", args, err)
		}
	}
}

func TestTheBannerNamesTheVersion(t *testing.T) {
	saved := version
	defer func() { version = saved }()
	version = "9.8.7"
	if got := banner(); got != "streampack-setup 9.8.7" {
		t.Errorf("banner %q", got)
	}
}

func TestCommandsItKnows(t *testing.T) {
	for _, c := range []string{"", "init", "up", "console", "secrets"} {
		if !knownCommand(c) {
			t.Errorf("%q unknown", c)
		}
	}
	if knownCommand("start") {
		t.Error("start known")
	}
}
