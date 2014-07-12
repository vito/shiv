package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/kr/pty"
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

var create = flag.Bool(
	"create",
	false,
	"create a new container",
)

func main() {
	flag.Parse()

	handle := flag.Arg(0)

	client := wclient.New(wconnection.New(*wardenNetwork, *wardenAddr))

	var container warden.Container
	var err error

	if *create {
		container, err = client.Create(warden.ContainerSpec{
			RootFSPath: *rootfs,
		})
	} else if handle == "" {
		containers, err := client.Containers(nil)
		if err != nil {
			log.Fatalln("failed to get containers")
		}

		if len(containers) == 0 {
			log.Fatalln("no containers")
		}

		container = containers[len(containers)-1]
	} else {
		container, err = client.Lookup(handle)
	}

	if err != nil {
		log.Fatalln("failed to lookup container:", err)
	}

	term, err := term.Open(os.Stdin.Name())
	if err != nil {
		log.Fatalln("failed to open terminal:", err)
	}

	err = term.SetRaw()
	if err != nil {
		log.Fatalln("failed to thaw terminal:", err)
	}

	process, err := container.Run(warden.ProcessSpec{
		Path:       "bash",
		Args:       []string{"-l"},
		Env:        []string{"TERM=" + os.Getenv("TERM")},
		TTY:        true,
		Privileged: true,
	}, warden.ProcessIO{
		Stdin:  term,
		Stdout: term,
		Stderr: term,
	})
	if err != nil {
		term.Restore()
		log.Fatalln("failed to run:", err)
	}

	rows, cols, err := pty.Getsize(os.Stdin)
	if err == nil {
		process.SetWindowSize(cols, rows)
	}

	resized := make(chan os.Signal, 10)
	signal.Notify(resized, syscall.SIGWINCH)

	go func() {
		for {
			<-resized

			rows, cols, err := pty.Getsize(os.Stdin)
			if err == nil {
				process.SetWindowSize(cols, rows)
			}
		}
	}()

	process.Wait()
	term.Restore()
}
