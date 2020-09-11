// +build windows

// Encoding: UTF-8
//
// AWS CloudWatch Logs - Configuration via Tags
//
// Copyright © 2020 Brian Dwyer - Intelligent Digital Services
//

package main

import (
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

func RestartService() {
	m, err := mgr.Connect()
	if err != nil {
		log.Fatal("SCM connection failed: ", err)
	}
	defer m.Disconnect()

	svcName := "AmazonCloudWatchAgent"

	s, err := m.OpenService(svcName)
	if err != nil {
		log.Printf("Could not open service %v: %v", svcName, err)
		return
	}
	defer s.Close()

	status, err := s.Query()
	if err != nil {
		log.Printf("Could not query status of service %v: %v", svcName, err)
		return
	}

	if status.State != windows.SERVICE_RUNNING {
		log.Errorf("%v was not running...", svcName)
	}

	// Stop the Service
	status, err = s.Control(svc.Stop)
	if err != nil {
		log.Printf("Could not stop service %v: %v", svcName, err)
		return
	}

	timeDuration := time.Millisecond * 50

	timeout := time.After(getStopTimeout() + (timeDuration * 2))
	tick := time.NewTicker(timeDuration)
	defer tick.Stop()

	for status.State != svc.Stopped {
		select {
		case <-tick.C:
			status, err = s.Query()
			if err != nil {
				log.Fatal(err)
			}
		case <-timeout:
			log.Fatal("Timed out waiting for service %v to stop", svcName)
		}
	}

	// Start the Service
	s.Start()
}

// getStopTimeout fetches the time before windows will kill the service.
func getStopTimeout() time.Duration {
	// For default and paths see https://support.microsoft.com/en-us/kb/146092
	defaultTimeout := time.Millisecond * 20000
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control`, registry.READ)
	if err != nil {
		return defaultTimeout
	}
	sv, _, err := key.GetStringValue("WaitToKillServiceTimeout")
	if err != nil {
		return defaultTimeout
	}
	v, err := strconv.Atoi(sv)
	if err != nil {
		return defaultTimeout
	}
	return time.Millisecond * time.Duration(v)
}
