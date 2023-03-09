module go.githedgehog.com/dasboot

go 1.20

require (
	github.com/0x5a17ed/uefi v0.6.1-0.20221119083023-4a7cfcbe0439
	github.com/go-chi/chi/v5 v5.0.8
	github.com/golang/mock v1.6.0
	github.com/google/uuid v1.3.0
	github.com/urfave/cli/v2 v2.24.4
	github.com/vishvananda/netlink v1.1.0
	go.uber.org/zap v1.24.0
	golang.org/x/sys v0.5.0
	gopkg.in/yaml.v3 v3.0.1
)

replace github.com/0x5a17ed/uefi => github.com/githedgehog/uefi v0.0.0-20230222015501-96f18acf01ad

require (
	github.com/0x5a17ed/itkit v0.6.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/spf13/afero v1.9.2 // indirect
	github.com/vishvananda/netns v0.0.0-20191106174202-0a2b9b5464df // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	golang.org/x/exp v0.0.0-20221028150844-83b7d23a625f // indirect
	golang.org/x/text v0.4.0 // indirect
)
