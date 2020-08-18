package test

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"os/exec"
	"testing"
	"time"
)

func AppName(t *testing.T) string {
	return fmt.Sprintf("itermctl_%s", t.Name())
}

func StartTakingScreenshots(ctx context.Context, dir string) error {

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(300 * time.Millisecond):
				file := fmt.Sprintf("%s/%d.jpg", dir, time.Now().UnixNano())
				cmd := exec.Command("/usr/sbin/screencapture", file)
				if err := cmd.Run(); err != nil {
					log.Errorf("screencapture: %s", err)
				}
			}
		}
	}()

	return nil
}
