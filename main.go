package main

import (
	"flag"
	"log"
	"os"

	"github.com/pkg/term"

	wclient "github.com/cloudfoundry-incubator/garden/client"
	wconnection "github.com/cloudfoundry-incubator/garden/client/connection"
	"github.com/cloudfoundry-incubator/garden/warden"
)

var wardenNetwork = flag.String(
	"wardenNetwork",
	"tcp",
	"warden server network (e.g. unix, tcp)",
)

var wardenAddr = flag.String(
	"wardenAddr",
	"127.0.0.1:7777",
	"warden server address",
)

var rootfs = flag.String(
	"rootfs",
	"docker:///ubuntu#14.04",
	"rootfs for the container to create",
)

func main() {
	flag.Parse()

	handle := flag.Arg(0)

	term, err := term.Open(os.Stdin.Name())
	if err != nil {
		log.Fatalln("failed to open terminal:", err)
	}

	err = term.SetRaw()
	if err != nil {
		log.Fatalln("failed to thaw terminal:", err)
	}

	client := wclient.New(wconnection.New(*wardenNetwork, *wardenAddr))

	var container warden.Container
	if handle == "" {
		container, err = client.Create(warden.ContainerSpec{
			RootFSPath: *rootfs,
		})
	} else {
		container, err = client.Lookup(handle)
	}

	if err != nil {
		term.Restore()
		log.Fatalln("failed to lookup container:", err)
	}

	process, err := container.Run(warden.ProcessSpec{
		Path: "bash",
		Args: []string{"-l"},
		TTY:  true,
	}, warden.ProcessIO{
		Stdin:  term,
		Stdout: term,
		Stderr: term,
	})
	if err != nil {
		term.Restore()
		log.Fatalln("failed to run:", err)
	}

	process.Wait()
	term.Restore()
}
