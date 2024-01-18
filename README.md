# bufar

POC only depends on Docker.

0. Build the binary

`go build .`

1. Create an Artifact Registry (Docker) named as:

`us-docker.pkg.dev/[YOUR_PROJECT]/protos`

2. Publish the protos:

`./bufar publish --registry=us-docker.pkg.dev/[YOUR_PROJECT]/protos --protos=./protos --package="bufarexample.v1"`

This command publishes the protos defined in [protos](/protos/).

3. Generate the code from the published protos:

First, you need a [`buf.gen.yaml`](https://buf.build/docs/configuration/v1/buf-gen-yaml) where you need to call the tool.
See the example in the repo.

`./bufar generate --registry=us-docker.pkg.dev/gochen/protos --package="burarexample.v1"`

[Folder gen](./gen/) is generated from the command above.
