package tty

import (
	"context"
	"os"
	goSignal "os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/zeromake/docker-debug/pkg/stream"
)

// ResizeTtyTo re sizes tty to specific height and width
func ResizeTtyTo(ctx context.Context, client client.ContainerAPIClient, id string, height, width uint, isExec bool) {
	if height == 0 && width == 0 {
		return
	}

	options := container.ResizeOptions{
		Height: height,
		Width:  width,
	}

	var err error
	if isExec {
		err = client.ContainerExecResize(ctx, id, options)
	} else {
		err = client.ContainerResize(ctx, id, options)
	}

	if err != nil {
		logrus.Debugf("Error resize: %s", err)
	}
}

// MonitorTtySize updates the container tty size when the terminal tty changes size
func MonitorTtySize(ctx context.Context, client client.ContainerAPIClient, out *stream.OutStream, id string, isExec bool) error {
	resizeTty := func() {
		height, width := out.GetTtySize()
		ResizeTtyTo(ctx, client, id, height, width, isExec)
	}

	resizeTty()

	if runtime.GOOS == "windows" {
		go func() {
			prevH, prevW := out.GetTtySize()
			for {
				time.Sleep(time.Millisecond * 250)
				h, w := out.GetTtySize()

				if prevW != w || prevH != h {
					resizeTty()
				}
				prevH = h
				prevW = w
			}
		}()
	} else {
		sigChan := make(chan os.Signal, 1)
		goSignal.Notify(sigChan, syscall.Signal(0x1c))
		go func() {
			for range sigChan {
				resizeTty()
			}
		}()
	}
	return nil
}
