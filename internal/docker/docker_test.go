package docker

import "testing"

func TestProjectNameIsNormalizedAsComposeDoes(t *testing.T) {
	for dir, want := range map[string]string{
		"q":                "q",
		"Streampack Trial": "streampacktrial",
		"my.deploy_1":      "mydeploy_1",
		"-x":               "x",
	} {
		if got := ProjectName(dir); got != want {
			t.Errorf("%q: got %q, want %q", dir, got, want)
		}
	}
}

func TestPrintedCommandsQuoteWhatTheShellWouldSplit(t *testing.T) {
	got := shellJoin([]string{"run", "--entrypoint", "sh", "img", "-c", "rm -rf /clear/*", "it's"})
	want := `run --entrypoint sh img -c 'rm -rf /clear/*' 'it'\''s'`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}
