//go:generate sh -c "if [ ! -f go.mod ]; then echo 'Initializing go.mod...'; go mod init .containifyci; else echo 'go.mod already exists. Skipping initialization.'; fi"
//go:generate go get github.com/containifyci/engine-ci/protos2
//go:generate go get github.com/containifyci/engine-ci/client
//go:generate go mod tidy

package main

import (
	"os"

	"github.com/containifyci/engine-ci/client/pkg/build"
	"github.com/containifyci/engine-ci/protos2"
)

func main() {
	os.Chdir("../")

	// Build Group 1
	rodwer := build.NewGoLibraryBuild("rodwer")
	rodwer.Folder = "."
	rodwer.Properties = map[string]*build.ListValue{
		"go_type": build.NewList("chromium"),
	}
	rodwer.ContainerFiles = map[string]*protos2.ContainerFile{
		"build": DockerFile(),
	}

	// Build Group 0
	examples := build.NewServiceBuild("examples", protos2.BuildType_GoLang)
	examples.Folder = "examples"
	examples.File = "basic_example.go"
	examples.Image = ""

	//TODO: adjust the registries to your own container registry
	build.BuildGroups(
		&protos2.BuildArgsGroup{
			Args: []*protos2.BuildArgs{
				rodwer, examples,
			},
		},
	)
}

func DockerFile() *protos2.ContainerFile {
	return &protos2.ContainerFile{
		Name: "golang-1.26.2-alpine-chromium",
		Content: `FROM golang:1.26.2-alpine

RUN apk --no-cache add git openssh-client chromium xvfb-run && \
	rm -rf /var/cache/apk/*

RUN go install github.com/wadey/gocovmerge@latest && \
	go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest && \
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.3 && \
	go clean -cache && \
	go clean -modcache
		`,
	}
}

