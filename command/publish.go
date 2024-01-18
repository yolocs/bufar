package command

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/abcxyz/pkg/cli"
)

const (
	dockerfile = `FROM bufbuild/buf:latest

  ARG PROTOS_DIR
  ARG PACKAGE_DIR

  WORKDIR /workspace

  COPY $PROTOS_DIR $PACKAGE_DIR

  RUN /usr/local/bin/buf lint

  ENTRYPOINT [ "/usr/local/bin/buf" ]
`
)

var _ cli.Command = (*PublishCommand)(nil)

type PublishCommand struct {
	cli.BaseCommand

	flagRegistry    string
	flagPackageName string
	flagProtosDir   string
}

func (c *PublishCommand) Desc() string {
	return `Publish protos to the registry`
}

func (c *PublishCommand) Help() string {
	return `
Usage: {{ COMMAND }} [options]
`
}

func (c *PublishCommand) Flags() *cli.FlagSet {
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

	f.StringVar(&cli.StringVar{
		Name:   "protos",
		Target: &c.flagProtosDir,
		Usage:  `Input dir that contains protos.`,
	})

	// TODO: add tag.

	return set
}

func (c *PublishCommand) Run(ctx context.Context, args []string) error {
	f := c.Flags()
	if err := f.Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	if err := os.WriteFile("Dockerfile.proto_pub", []byte(dockerfile), 0o600); err != nil {
		return fmt.Errorf("failed to write Dockerfile.proto_pub: %w", err)
	}
	defer func() {
		if err := os.Remove("Dockerfile.proto_pub"); err != nil {
			c.Outf("Failed to remove Dockerfile.proto_pub; please remove manually: %w", err)
		}
	}()

	packageDir := strings.ReplaceAll(c.flagPackageName, ".", "/")
	registryPath := fmt.Sprintf("%s/%s:latest", c.flagRegistry, packageDir)

	dockerArgs := []string{
		"buildx", "build", "--push", "-t", registryPath,
		"--build-arg", fmt.Sprintf("PROTOS_DIR=%s", c.flagProtosDir),
		"--build-arg", fmt.Sprintf("PACKAGE_DIR=%s", packageDir), "-f", "Dockerfile.proto_pub", ".",
	}

	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)
	cmd.Stdout = c.Stdout()
	cmd.Stderr = c.Stderr()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to publish protos: %w", err)
	}

	c.Outf("Published proto package: %s", registryPath)
	return nil
}
