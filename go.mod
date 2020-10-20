module github.com/mumoshu/waypoint-plugin-helmfile

go 1.14

require (
	github.com/containerd/continuity v0.0.0-20200928162600-f2cc35102c2a // indirect
	github.com/golang/protobuf v1.4.2
	github.com/hashicorp/waypoint v0.1.2
	github.com/hashicorp/waypoint-plugin-sdk v0.0.0-20201016002013-59421183d54f
	github.com/mumoshu/shoal v0.2.13
	github.com/opencontainers/runc v1.0.0-rc9 // indirect
	github.com/sirupsen/logrus v1.7.0 // indirect
	golang.org/x/net v0.0.0-20201016165138-7b1cca2348c0 // indirect
	golang.org/x/sys v0.0.0-20201017003518-b09fb700fbb7 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	google.golang.org/grpc v1.32.0
	google.golang.org/protobuf v1.25.0
)

// https://github.com/ory/dockertest/issues/208
replace golang.org/x/sys => golang.org/x/sys v0.0.0-20200826173525-f9321e4c35a6

replace github.com/fishworks/gofish => github.com/mumoshu/gofish v0.13.1-0.20200908033248-ab2d494fb15c
