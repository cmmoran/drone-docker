package docker

import (
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/dchest/uniuri"
)

func TestCommandBuild(t *testing.T) {
	tempTag := strings.ToLower(uniuri.New())
	tcs := []struct {
		name  string
		build Build
		want  *exec.Cmd
	}{
		{
			name: "secret from env var",
			build: Build{
				Name:       "plugins/drone-docker:latest",
				TempTag:    tempTag,
				Dockerfile: "Dockerfile",
				Context:    ".",
				SecretEnvs: []string{
					"foo_secret=FOO_SECRET_ENV_VAR",
				},
			},
			want: exec.Command(
				dockerExe,
				"build",
				"--rm=true",
				"-f",
				"Dockerfile",
				"-t",
				tempTag,
				".",
				"--secret id=foo_secret,env=FOO_SECRET_ENV_VAR",
			),
		},
		{
			name: "secret from file",
			build: Build{
				Name:       "plugins/drone-docker:latest",
				TempTag:    tempTag,
				Dockerfile: "Dockerfile",
				Context:    ".",
				SecretFiles: []string{
					"foo_secret=/path/to/foo_secret",
				},
			},
			want: exec.Command(
				dockerExe,
				"build",
				"--rm=true",
				"-f",
				"Dockerfile",
				"-t",
				tempTag,
				".",
				"--secret id=foo_secret,src=/path/to/foo_secret",
			),
		},
		{
			name: "multiple mixed secrets",
			build: Build{
				Name:       "plugins/drone-docker:latest",
				TempTag:    tempTag,
				Dockerfile: "Dockerfile",
				Context:    ".",
				SecretEnvs: []string{
					"foo_secret=FOO_SECRET_ENV_VAR",
					"bar_secret=BAR_SECRET_ENV_VAR",
				},
				SecretFiles: []string{
					"foo_secret=/path/to/foo_secret",
					"bar_secret=/path/to/bar_secret",
				},
			},
			want: exec.Command(
				dockerExe,
				"build",
				"--rm=true",
				"-f",
				"Dockerfile",
				"-t",
				tempTag,
				".",
				"--secret id=foo_secret,env=FOO_SECRET_ENV_VAR",
				"--secret id=bar_secret,env=BAR_SECRET_ENV_VAR",
				"--secret id=foo_secret,src=/path/to/foo_secret",
				"--secret id=bar_secret,src=/path/to/bar_secret",
			),
		},
		{
			name: "invalid mixed secrets",
			build: Build{
				Name:       "plugins/drone-docker:latest",
				TempTag:    tempTag,
				Dockerfile: "Dockerfile",
				Context:    ".",
				SecretEnvs: []string{
					"foo_secret=",
					"=FOO_SECRET_ENV_VAR",
					"",
				},
				SecretFiles: []string{
					"foo_secret=",
					"=/path/to/bar_secret",
					"",
				},
			},
			want: exec.Command(
				dockerExe,
				"build",
				"--rm=true",
				"-f",
				"Dockerfile",
				"-t",
				tempTag,
				".",
			),
		},
		{
			name: "platform argument",
			build: Build{
				Name:       "plugins/drone-docker:latest",
				TempTag:    tempTag,
				Dockerfile: "Dockerfile",
				Context:    ".",
				Platform:   "test/platform",
			},
			want: exec.Command(
				dockerExe,
				"build",
				"--rm=true",
				"-f",
				"Dockerfile",
				"-t",
				tempTag,
				".",
				"--platform",
				"test/platform",
			),
		},
		{
			name: "ssh agent",
			build: Build{
				Name:       "plugins/drone-docker:latest",
				TempTag:    tempTag,
				Dockerfile: "Dockerfile",
				Context:    ".",
				SSHKeyPath: "id_rsa=/root/.ssh/id_rsa",
			},
			want: exec.Command(
				dockerExe,
				"build",
				"--rm=true",
				"-f",
				"Dockerfile",
				"-t",
				tempTag,
				".",
				"--ssh id_rsa=/root/.ssh/id_rsa",
			),
		},
	}

	for _, tc := range tcs {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			cmd := commandBuild(tc.build)

			if !reflect.DeepEqual(cmd.String(), tc.want.String()) {
				t.Errorf("Got cmd %v, want %v", cmd, tc.want)
			}
		})
	}
}

func TestGetProxyValue(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		envVars  map[string]string
		expected string
	}{
		{
			name:     "lowercase env var set",
			key:      "http_proxy",
			envVars:  map[string]string{"http_proxy": "http://proxy:8080"},
			expected: "http://proxy:8080",
		},
		{
			name:     "uppercase env var set",
			key:      "http_proxy",
			envVars:  map[string]string{"HTTP_PROXY": "http://proxy:8080"},
			expected: "http://proxy:8080",
		},
		{
			name:     "HARNESS prefixed env var set",
			key:      "http_proxy",
			envVars:  map[string]string{"HARNESS_HTTP_PROXY": "http://harness-proxy:8080"},
			expected: "http://harness-proxy:8080",
		},
		{
			name: "standard takes precedence over HARNESS",
			key:  "http_proxy",
			envVars: map[string]string{
				"HTTP_PROXY":         "http://standard:8080",
				"HARNESS_HTTP_PROXY": "http://harness:8080",
			},
			expected: "http://standard:8080",
		},
		{
			name: "lowercase takes precedence over uppercase",
			key:  "no_proxy",
			envVars: map[string]string{
				"no_proxy":         "localhost,127.0.0.1",
				"NO_PROXY":         "*.example.com",
				"HARNESS_NO_PROXY": "*.local",
			},
			expected: "localhost,127.0.0.1",
		},
		{
			name: "lowercase takes precedence over HARNESS",
			key:  "https_proxy",
			envVars: map[string]string{
				"https_proxy":         "https://standard:8080",
				"HARNESS_HTTPS_PROXY": "https://harness:8080",
			},
			expected: "https://standard:8080",
		},
		{
			name:     "no env var set",
			key:      "http_proxy",
			envVars:  map[string]string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean env
			lowercaseKey := tt.key
			uppercaseKey := strings.ToUpper(tt.key)
			harnessKey := "HARNESS_" + strings.ToUpper(tt.key)

			os.Unsetenv(lowercaseKey)
			os.Unsetenv(uppercaseKey)
			os.Unsetenv(harnessKey)

			// Set test environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			// Execute and verify
			result := getProxyValue(tt.key)
			if result != tt.expected {
				t.Errorf("getProxyValue(%q) = %q, want %q", tt.key, result, tt.expected)
			}
		})
	}
}

func TestResolveOutputs(t *testing.T) {
	plugin := Plugin{
		Outputs: []string{
			"tags",
			"digest",
			"image_refs",
			"primary_image_ref",
			"image_with_digest",
			"labels",
			"outputs.image_repo=settings.repo",
			"outputs.publish_enabled=settings.dry_run",
		},
		Build: Build{
			Repo:   "octocat/hello-world",
			Tags:   []string{"latest", "1.2.3"},
			Labels: []string{"org.opencontainers.image.title=hello-world"},
		},
		Dryrun: true,
	}

	got, err := plugin.resolveOutputs(buildOutputContext(plugin.Build.Repo, plugin.Build.Tags, "sha256:abc123"))
	if err != nil {
		t.Fatal(err)
	}

	want := []exportedOutput{
		{Key: "tags", Value: []string{"latest", "1.2.3"}},
		{Key: "digest", Value: "sha256:abc123"},
		{Key: "image_refs", Value: []string{"octocat/hello-world:latest", "octocat/hello-world:1.2.3"}},
		{Key: "primary_image_ref", Value: "octocat/hello-world:latest"},
		{Key: "image_with_digest", Value: "octocat/hello-world@sha256:abc123"},
		{Key: "labels", Value: []string{"org.opencontainers.image.title=hello-world"}},
		{Key: "outputs.image_repo", Value: "octocat/hello-world"},
		{Key: "outputs.publish_enabled", Value: true},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resolveOutputs() got %#v want %#v", got, want)
	}
}

func TestResolveOutputsUnknownSource(t *testing.T) {
	plugin := Plugin{Outputs: []string{"outputs.foo=does_not_exist"}}
	_, err := plugin.resolveOutputs(outputContext{})
	if err == nil || !strings.Contains(err.Error(), "unsupported output source") {
		t.Fatalf("expected unsupported output source error, got %v", err)
	}
}

func TestResolveOutputsBlockedSource(t *testing.T) {
	t.Setenv("PLUGIN_FROM_SECRET_KEYS", "password")
	plugin := Plugin{
		Outputs: []string{"outputs.registry_password=settings.password"},
		Login: Login{
			Password: "super-secret",
		},
	}

	_, err := plugin.resolveOutputs(outputContext{})
	if err == nil || !strings.Contains(err.Error(), "refusing to export blocked output source") {
		t.Fatalf("expected blocked output source error, got %v", err)
	}
}

func TestResolveOutputsUnavailableRuntimeSource(t *testing.T) {
	plugin := Plugin{Outputs: []string{"digest"}}
	_, err := plugin.resolveOutputs(outputContext{})
	if err == nil || !strings.Contains(err.Error(), "unavailable for this run") {
		t.Fatalf("expected unavailable runtime output error, got %v", err)
	}
}

func TestOutputCommandArgs(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value any
		want  []string
	}{
		{
			name:  "string",
			key:   "repo",
			value: "octocat/hello-world",
			want:  []string{"set", "repo", "octocat/hello-world"},
		},
		{
			name:  "slice",
			key:   "tags",
			value: []string{"latest", "1.2.3"},
			want:  []string{"set", "--format", "json", "tags", "[\"latest\",\"1.2.3\"]"},
		},
		{
			name:  "bool",
			key:   "outputs.publish_enabled",
			value: true,
			want:  []string{"set", "--format", "json", "outputs.publish_enabled", "true"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := outputCommandArgs(tc.key, tc.value)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("outputCommandArgs() got %#v want %#v", got, tc.want)
			}
		})
	}
}

func TestWriteOutputs(t *testing.T) {
	plugin := Plugin{
		Outputs: []string{"tags", "digest", "outputs.image_repo=settings.repo"},
		Build: Build{
			Repo: "octocat/hello-world",
			Tags: []string{"latest", "1.2.3"},
		},
	}

	var commands [][]string
	original := outputCommand
	t.Setenv("PLUGIN_OUTPUT_HELPER_BIN", "/drone/bin/drone-output")
	outputCommand = func(name string, args ...string) *exec.Cmd {
		commands = append(commands, append([]string{name}, args...))
		return exec.Command("sh", "-c", "exit 0")
	}
	defer func() {
		outputCommand = original
	}()

	if err := plugin.writeOutputs(buildOutputContext(plugin.Build.Repo, plugin.Build.Tags, "sha256:abc123")); err != nil {
		t.Fatal(err)
	}

	want := [][]string{
		{"/drone/bin/drone-output", "set", "--format", "json", "tags", "[\"latest\",\"1.2.3\"]"},
		{"/drone/bin/drone-output", "set", "digest", "sha256:abc123"},
		{"/drone/bin/drone-output", "set", "outputs.image_repo", "octocat/hello-world"},
	}

	if !reflect.DeepEqual(commands, want) {
		t.Fatalf("writeOutputs() got %s want %s", fmt.Sprintf("%#v", commands), fmt.Sprintf("%#v", want))
	}
}

func TestResolveOutputHelperBinPrefersEnv(t *testing.T) {
	t.Setenv("PLUGIN_OUTPUT_HELPER_BIN", "/drone/bin/drone-output")
	got, err := resolveOutputHelperBin()
	if err != nil {
		t.Fatal(err)
	}
	if got != "/drone/bin/drone-output" {
		t.Fatalf("resolveOutputHelperBin() got %q want %q", got, "/drone/bin/drone-output")
	}
}

func TestResolveOutputHelperBinFromPath(t *testing.T) {
	t.Setenv("PLUGIN_OUTPUT_HELPER_BIN", "")
	tmp := t.TempDir()
	helper := tmp + "/drone-output"
	if err := os.WriteFile(helper, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", tmp)

	got, err := resolveOutputHelperBin()
	if err != nil {
		t.Fatal(err)
	}
	if got != helper {
		t.Fatalf("resolveOutputHelperBin() got %q want %q", got, helper)
	}
}
