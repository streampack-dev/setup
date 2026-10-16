package docker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"testing"
)

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

// box is a fake machine: the commands installed, what each command line prints, and which fail.
// It records every command run, as typed.
type box struct {
	installed map[string]bool
	prints    map[string]string
	fails     map[string]bool
	ran       []string
}

func (b *box) lookPath(name string) (string, error) {
	if b.installed[name] {
		return "/usr/bin/" + name, nil
	}
	return "", exec.ErrNotFound
}

func (b *box) run(_ context.Context, _ string, stdout, _ io.Writer, name string, args ...string) error {
	line := strings.Join(append([]string{name}, args...), " ")
	b.ran = append(b.ran, line)
	if !b.installed[name] || b.fails[line] {
		return errors.New("exit status 1")
	}
	io.WriteString(stdout, b.prints[line])
	return nil
}

func installed(names ...string) map[string]bool {
	m := map[string]bool{}
	for _, n := range names {
		m[n] = true
	}
	return m
}

func TestDetectFindsTheEngineAndItsCompose(t *testing.T) {
	for _, c := range []struct {
		name string
		box  box
		want string
	}{
		{"docker with the compose plugin", box{installed: installed("docker")}, "docker, with docker compose"},
		{"docker with the old docker-compose",
			box{installed: installed("docker", "docker-compose"), fails: map[string]bool{"docker compose version": true}},
			"docker, with docker-compose"},
		{"podman with podman-compose", box{installed: installed("podman", "podman-compose"),
			prints: map[string]string{"podman-compose version": "podman-compose version 1.6.0\npodman version 5.4.0\n"}},
			"podman, with podman-compose"},
		{"podman behind the docker command", box{installed: installed("docker", "podman", "podman-compose"),
			prints: map[string]string{"docker --version": "podman version 5.4.0\n",
				"podman-compose version": "podman-compose version 1.3.0\n"}},
			"podman, with podman-compose"},
		{"podman's own compose, handing over to docker-compose", box{installed: installed("podman"),
			prints: map[string]string{"podman compose version": "Docker Compose version v2.39.1\n"}},
			"podman, with podman compose"},
		{"docker not running, podman is", box{installed: installed("docker", "podman", "podman-compose"),
			fails:  map[string]bool{"docker version --format {{.Server.Version}}": true},
			prints: map[string]string{"podman-compose version": "podman-compose version 1.2.0\n"}},
			"podman, with podman-compose"},
	} {
		t.Run(c.name, func(t *testing.T) {
			e, err := Detect(context.Background(), c.box.lookPath, c.box.run)
			if err != nil || e.String() != c.want {
				t.Errorf("got %q, %v; want %q", e, err, c.want)
			}
		})
	}
}

func TestDetectSaysWhatsMissing(t *testing.T) {
	for _, c := range []struct {
		name string
		box  box
		want string
	}{
		{"nothing", box{}, "install Docker or Podman"},
		{"docker stopped", box{installed: installed("docker"),
			fails: map[string]bool{"docker version --format {{.Server.Version}}": true}}, "start Docker"},
		{"podman machine stopped", box{installed: installed("podman"),
			fails: map[string]bool{"podman info": true}}, "podman machine start"},
		{"podman without compose", box{installed: installed("podman"),
			fails: map[string]bool{"podman compose version": true}}, "install podman-compose"},
		// Before 1.2.0, podman-compose handed .env to Podman, quotes and all.
		{"podman-compose 1.0.6", box{installed: installed("podman", "podman-compose"),
			prints: map[string]string{"podman-compose version": "podman-compose version: 1.0.6\n['podman', '--version', '']\n"}},
			"podman-compose 1.0.6 is too old"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := Detect(context.Background(), c.box.lookPath, c.box.run)
			if !errors.Is(err, ErrNoEngine) || !strings.Contains(err.Error(), c.want) {
				t.Errorf("got %v, want it to say %q", err, c.want)
			}
		})
	}
}

var (
	dockerEngine = Engine{Name: "docker", Compose: []string{"docker", "compose"}}
	podmanEngine = Engine{Name: "podman", Compose: []string{"podman-compose"}, PodmanCompose: true}
)

// The commands a hard reset lists with use nothing one engine has and the other hasn't (setup#13).
func TestResourcesAskOnlyWhatEveryEngineAnswers(t *testing.T) {
	label := "--filter label=com.docker.compose.project=q"
	for _, c := range []struct {
		engine Engine
		prints map[string]string
		want   Resources
	}{
		{dockerEngine, map[string]string{
			"docker ps -a " + label + " --format {{.Names}}":                                                             "q-db-1\nq-server-streampack-1\n",
			"docker ps -a " + label + " --filter label=com.docker.compose.service=server-streampack --format {{.Image}}": "nexus.streampack.dev/images/server-streampack:1.16.0\n",
			"docker volume ls " + label + " --format {{.Name}}":                                                          "q_pgdata\n",
			"docker network ls " + label + " --format {{.Name}}":                                                         "q_default\n",
		}, Resources{Containers: []string{"q-db-1", "q-server-streampack-1"}, Volumes: []string{"q_pgdata"},
			Networks: []string{"q_default"}, ServerImage: "nexus.streampack.dev/images/server-streampack:1.16.0"}},
		{podmanEngine, map[string]string{
			"podman ps -a " + label + " --format {{.Names}}":                                                             "q_db_1\nq_server-streampack_1\n",
			"podman ps -a " + label + " --filter label=com.docker.compose.service=server-streampack --format {{.Image}}": "nexus.streampack.dev/images/server-streampack:1.16.0\n",
			"podman pod ls --format {{.Name}}":                                                                           "pod_other\npod_q\n",
			"podman volume ls " + label + " --format {{.Name}}":                                                          "q_pgdata\n",
			"podman network ls " + label + " --format {{.Name}}":                                                         "q_default\n",
		}, Resources{Containers: []string{"q_db_1", "q_server-streampack_1"}, Pods: []string{"pod_q"},
			Volumes: []string{"q_pgdata"}, Networks: []string{"q_default"},
			ServerImage: "nexus.streampack.dev/images/server-streampack:1.16.0"}},
	} {
		t.Run(c.engine.Name, func(t *testing.T) {
			b := &box{installed: installed(c.engine.Name), prints: c.prints}
			got, err := CLI{Dir: "/srv/q", Out: io.Discard, Engine: c.engine, Run: b.run}.Resources(context.Background())
			if err != nil || fmt.Sprint(got) != fmt.Sprint(c.want) {
				t.Errorf("got %+v, %v\nwant %+v\nran %q", got, err, c.want, b.ran)
			}
			for _, line := range b.ran {
				if strings.Contains(line, ".Label") {
					t.Errorf("%q uses a label template", line)
				}
			}
			var out bytes.Buffer
			CLI{Dir: "/srv/q", Out: &out, Engine: c.engine, Run: b.run}.RemoveResources(context.Background(), got)
			if c.engine.Name == "podman" && !strings.Contains(out.String(), "$ podman pod rm -f pod_q\n") {
				t.Errorf("the pod stays:\n%s", out.String())
			}
		})
	}
}

// podman-compose doesn't label a volume with its name, so it's found by its full name.
func TestVolumeIsFoundByItsFullName(t *testing.T) {
	b := &box{installed: installed("podman"), prints: map[string]string{
		"podman volume ls --filter label=com.docker.compose.project=q --format {{.Name}}": "q_other\nq_pgdata\n",
	}}
	c := CLI{Dir: "/srv/q", Out: io.Discard, Engine: podmanEngine, Run: b.run}
	if got, err := c.Volume(context.Background(), "pgdata"); err != nil || got != "q_pgdata" {
		t.Errorf("pgdata: got %q, %v", got, err)
	}
	if got, _ := c.Volume(context.Background(), "media"); got != "" {
		t.Errorf("media: got %q", got)
	}
}

func TestComposeCommandsAreTheEnginesOwn(t *testing.T) {
	for _, c := range []struct {
		engine Engine
		want   []string
	}{
		{dockerEngine, []string{"docker compose pull", "docker compose up -d",
			"docker compose exec -T server-streampack cat /run/streampack/admin-token"}},
		// podman-compose recreates nothing for a changed .env unless it's told to.
		{podmanEngine, []string{"podman-compose pull", "podman-compose up -d --force-recreate",
			"podman-compose exec -T server-streampack cat /run/streampack/admin-token"}},
	} {
		b := &box{installed: installed(c.engine.Compose[0])}
		var out bytes.Buffer
		cli := CLI{Dir: "/srv/q", Out: &out, Engine: c.engine, Run: b.run}
		cli.Pull(context.Background())
		cli.Up(context.Background())
		cli.Exec(context.Background(), "server-streampack", "cat", "/run/streampack/admin-token")
		if strings.Join(b.ran, "\n") != strings.Join(c.want, "\n") {
			t.Errorf("%s ran\n%s\nwant\n%s", c.engine.Name, strings.Join(b.ran, "\n"), strings.Join(c.want, "\n"))
		}
		if !strings.Contains(out.String(), "$ "+c.want[1]+"\n") {
			t.Errorf("printed %q", out.String())
		}
	}
}

func TestClearDirectoryMountsForSELinux(t *testing.T) {
	b := &box{installed: installed("podman")}
	CLI{Out: io.Discard, Engine: podmanEngine, Run: b.run}.ClearDirectory(context.Background(), "img", "/srv/q/media")
	if len(b.ran) != 1 || !strings.HasPrefix(b.ran[0], "podman run --rm -v /srv/q/media:/clear:z ") {
		t.Errorf("ran %q", b.ran)
	}
}
