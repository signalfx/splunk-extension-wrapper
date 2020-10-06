package ossignal

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func Watch() {
	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGKILL)

	log.Printf("awaiting os/signals")

	go func() {
		for s := range sigs {
			log.Printf("os/signal: %s\n", s)
		}
	}()
}
