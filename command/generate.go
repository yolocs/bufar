package command

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/abcxyz/pkg/cli"
)

const (
	tempDockerfile = `FROM %s

WORKDIR /workspace

COPY buf.gen.yaml buf.gen.yaml

RUN /usr/local/bin/buf generate && rm buf.gen.yaml && rm -r %s

ENTRYPOINT [ "/usr/local/bin/buf" ]
`
)

var _ cli.Command = (*GenerateCommand)(nil)

type GenerateCommand struct {
	cli.BaseCommand

	flagRegistry    string
	flagPackageName string
}

func (c *GenerateCommand) Desc() string {
	return `Generate code based on the given proto package`
}

func (c *GenerateCommand) Help() string {
	return `
Usage: {{ COMMAND }} [options]
`
}

func (c *GenerateCommand) Flags() *cli.FlagSet {
	set := cli.NewFlagSet()

	// Command options
	f := set.NewSection("COMMAND OPTIONS")

	f.StringVar(&cli.StringVar{
		Name:    "registry",
		Target:  &c.flagRegistry,
		EnvVar:  "BUFAR_REGISTRY",
		Example: `us-docker.pkg.dev/project-foo/registry-bar`,
		Usage:   `Must be a container registry.`,
	})

	f.StringVar(&cli.StringVar{
		Name:    "package",
		Target:  &c.flagPackageName,
		Example: `mypackage.v1`,
		Usage:   `Must be a valid proto package name.`,
	})

	return set
}

func (c *GenerateCommand) Run(ctx context.Context, args []string) error {
	f := c.Flags()
	if err := f.Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	if _, err := os.Stat("buf.gen.yaml"); os.IsNotExist(err) {
		return fmt.Errorf("buf.gen.yaml not found")
	}

	packageDir := strings.ReplaceAll(c.flagPackageName, ".", "/")
	registryPath := fmt.Sprintf("%s/%s:latest", c.flagRegistry, packageDir)
	localPath := fmt.Sprintf("%s/%s:gen", c.flagRegistry, packageDir)

	dockerfileGen := fmt.Sprintf(tempDockerfile, registryPath, filepath.Dir(packageDir))
	if err := os.WriteFile("Dockerfile.proto_gen", []byte(dockerfileGen), 0o600); err != nil {
		return fmt.Errorf("failed to write Dockerfile.proto_gen: %w", err)
	}
	defer func() {
		if err := os.Remove("Dockerfile.proto_gen"); err != nil {
			c.Outf("Failed to remove Dockerfile.proto_gen; please remove manually: %w", err)
		}
	}()

	dockerArgs := []string{
		"buildx", "build", "-t", localPath, "-f", "Dockerfile.proto_gen", ".",
	}

	buildCmd := exec.CommandContext(ctx, "docker", dockerArgs...)
	buildCmd.Stdout = c.Stdout()
	buildCmd.Stderr = c.Stderr()
	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("failed to build codegen container: %w", err)
	}

	// Copy protos out.
	createContainerCmd := exec.CommandContext(ctx, "docker", "create", localPath)
	containerIDOut := bytes.NewBuffer(nil)
	createContainerCmd.Stdout = containerIDOut
	createContainerCmd.Stderr = c.Stderr()
	if err := createContainerCmd.Run(); err != nil {
		return fmt.Errorf("failed to start codegen container: %w", err)
	}
	containerID := strings.TrimSpace(containerIDOut.String())
	defer func() {
		removeContainerCmd := exec.CommandContext(ctx, "docker", "rm", "-v", containerID)
		if err := removeContainerCmd.Run(); err != nil {
			c.Errf("Failed to remove codegen container %q; please remove manually: %w", containerID, err)
		}
	}()

	cpProtosCmd := exec.CommandContext(ctx, "docker", "cp", fmt.Sprintf("%s:/workspace/.", containerID), ".")
	cpProtosCmd.Stdout = c.Stdout()
	cpProtosCmd.Stderr = c.Stderr()
	if err := cpProtosCmd.Run(); err != nil {
		return fmt.Errorf("failed to extract generated code: %w", err)
	}

	// pwd, err := os.Getwd()
	// if err != nil {
	// 	return fmt.Errorf("failed to get current working directory: %w", err)
	// }

	// dockerArgs := []string{
	// 	"run", "--volume", fmt.Sprintf("%s:/workspace", pwd),
	// 	"--workdir", "/workspace", registryPath, "generate",
	// 	"--path", packageDir,
	// }

	// cmd := exec.CommandContext(ctx, "docker", dockerArgs...)
	// cmd.Stdout = c.Stdout()
	// cmd.Stderr = c.Stderr()
	// if err := cmd.Run(); err != nil {
	// 	return fmt.Errorf("failed to generate code: %w", err)
	// }

	return nil
}
