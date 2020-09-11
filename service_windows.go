// +build windows

// Encoding: UTF-8
//
// AWS CloudWatch Logs - Configuration via Tags
//
// Copyright © 2020 Brian Dwyer - Intelligent Digital Services
//

package main

import (
	log "github.com/sirupsen/logrus"

	"golang.org/x/sys/windows"
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

	status, err = s.Control(mgr.ServiceRestart)
	if err != nil {
		log.Printf("Could not restart service %v: %v", svcName, err)
		return
	}
}
