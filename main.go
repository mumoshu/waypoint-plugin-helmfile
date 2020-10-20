package main

import (
	sdk "github.com/hashicorp/waypoint-plugin-sdk"
	"github.com/mumoshu/waypoint-plugin-helmfile/platform"
)

//go:generate protoc -I ./ --go_opt=plugins=grpc --go_out=./ ./platform/plugin.proto

func main() {
	sdk.Main(sdk.WithComponents(
		&platform.Platform{},
		// NOTE:
		// The following error indicates that DockerImageMapper is missing in here:
		//
		// Generated new Docker image: web:latest
		//
		// » Deploying...
		// ! 1 error occurred:
		//        * argument cannot be satisfied: type *anypb.Any (subtype: "platform.Input")
		//sdk.WithMappers(platform.DockerImageMapper),
	))
}
