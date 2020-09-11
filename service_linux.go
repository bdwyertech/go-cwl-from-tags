// +build linux

// Encoding: UTF-8
//
// AWS CloudWatch Logs - Configuration via Tags
//
// Copyright © 2020 Brian Dwyer - Intelligent Digital Services
//

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/coreos/go-systemd/v22/dbus"
)

func RestartService() (healthy bool, err error) {
	service := "amazon-cloudwatch-agent"
	switch {
	case isSystemD():
		return restartSystemD(service)
	case isRedhatSysV(), isDebianSysV():
		return restartSysV(service)
	default:
		log.Fatal("Service checks on this platform are not supported yet!")
	}

	return
}

func restartSystemD(service string) (healthy bool, err error) {
	m, err := dbus.New()

	if err != nil {
		log.Fatal("SCM connection failed: ", err)
	}
	defer m.Close()

	serviceUnit := fmt.Sprintf("%v.service", service)

	s, err := m.GetAllProperties(serviceUnit)
	if err != nil {
		log.Errorf("Could not open service %v: %v", service, err)
		return
	}

	healthy = true
	for _, v := range []string{"SubState", "StatusErrno", "StatusText"} {
		if v == "SubState" && s[v] != "running" {
			healthy = false
		}
	}

	reschan := make(chan string)
	_, err = m.ReloadOrRestartUnit(serviceUnit, "replace", reschan)
	if err != nil {
		log.Fatal(err)
	}

	job := <-reschan
	if job != "done" {
		log.Fatal("Job is not done:", job)
	}

	return
}

func restartSysV(service string) (healthy bool, err error) {
	timeout := 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "service", service, "status")
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr == context.DeadlineExceeded {
			log.Errorf("Command timed out: %v", cmd.String())
		} else {
			log.Errorf("%v failed: %v, %v", cmd.String(), err, output)
			return
		}
	}

	if strings.Contains(output, "is running") {
		healthy = true
	}

	cmd = exec.CommandContext(ctx, "service", service, "restart")
	out, err = cmd.CombinedOutput()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr == context.DeadlineExceeded {
			log.Errorf("Command timed out: %v", cmd.String())
		} else {
			log.Errorf("%v failed: %v, %v", cmd.String(), err, strings.TrimSpace(string(out)))
			return
		}
	}

	return
}

func isSystemD() bool {
	// https://www.freedesktop.org/software/systemd/man/sd_booted.html
	if _, err := os.Stat("/run/systemd/system/"); err != nil {
		return false
	}
	return true
}

func isDebianSysV() bool {
	if _, err := os.Stat("/lib/lsb/init-functions"); err != nil {
		return false
	}
	if _, err := os.Stat("/sbin/start-stop-daemon"); err != nil {
		return false
	}
	return true
}

func isRedhatSysV() bool {
	if _, err := os.Stat("/etc/rc.d/init.d/functions"); err != nil {
		return false
	}
	return true
}
