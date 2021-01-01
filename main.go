package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"

	i3ipc "github.com/brunnre8/i3ipc-go"
)

func main() {
	trigger, err := change()
	if err != nil {
		log.Fatalf("failed to watch: %v", err)
	}

	listener, err := net.Listen("unix", "/tmp/.focus_last")
	if err != nil {
		log.Fatal("failed to create listener: %v", err)
	}

	if err := http.Serve(listener, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		trigger <- struct{}{}
		w.WriteHeader(http.StatusOK)
	})); err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
}

func change() (chan struct{}, error) {
	i3ipc.StartEventListener()
	events, err := i3ipc.Subscribe(i3ipc.I3WindowEvent)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to window events: %w", err)
	}

	var idLock sync.Mutex
	var previousContainerID, currentContainerId int64
	go func() {
		for event := range events {
			windowEvent, ok := event.Details.(i3ipc.WindowEvent)
			if !ok {
				log.Printf("ERROR: got unexecpted event type %T", event.Details)
				continue
			}
			if windowEvent.Change != "focus" {
				continue
			}
			idLock.Lock()
			previousContainerID = currentContainerId
			currentContainerId = windowEvent.Container.ID
			idLock.Unlock()
		}
	}()

	socket, err := i3ipc.GetIPCSocket()
	if err != nil {
		return nil, fmt.Errorf("failed to get i3ipc socket: %w", err)
	}

	trigger := make(chan struct{})
	go func() {
		for range trigger {
			idLock.Lock()
			success, err := socket.Command(fmt.Sprintf("[con_id=%d] focus", previousContainerID))
			idLock.Unlock()
			if err != nil {
				log.Printf("ERROR running command: %v\n", err)
				continue
			}
			if !success {
				log.Println("Running command wasn't successful")
			}
		}
	}()

	return trigger, nil
}
