// Encoding: UTF-8
//
// AWS CloudWatch Logs - Configuration via Tags
//
// Copyright © 2020 Brian Dwyer - Intelligent Digital Services
//

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/Jeffail/gabs/v2"
)

// https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/CloudWatch-Agent-Configuration-File-Details.html

type CwlLogFile struct {
	Encoding              string `json:"encoding,omitempty"`
	FilePath              string `json:"file_path"`
	LogGroupName          string `json:"log_group_name"`
	MultiLineStartPattern string `json:"multi_line_start_pattern,omitempty"`
}

func main() {
	// Parse Flags
	flag.Parse()

	if versionFlag {
		showVersion()
		os.Exit(0)
	}

	var LogsFromConfig, LogsFromTags []CwlLogFile

	// LogGroup-Friendly Replacer
	lgfriendly := strings.NewReplacer(":", "", " ", "_", "*", "#", "\\", "/")

	for _, tag := range getTags() {
		if strings.HasPrefix(*tag.Key, "cwl:") {
			f := CwlLogFile{
				Encoding:     "utf-8",
				FilePath:     *tag.Value,
				LogGroupName: fmt.Sprintf("/aws/ec2/%s", lgfriendly.Replace(strings.TrimPrefix(*tag.Value, "/"))),
			}
			LogsFromTags = append(LogsFromTags, f)
		}
	}

	if len(LogsFromTags) == 0 {
		log.Fatal("No logs")
	}

	//
	// CloudWatch Agent - Configuration Path
	//
	var cwlConfigFile string
	if runtime.GOOS == "windows" {
		cwlConfigFile = "C:\\ProgramData\\Amazon\\AmazonCloudWatchAgent\\amazon-cloudwatch-agent.json"
	} else {
		cwlConfigFile = "/opt/aws/amazon-cloudwatch-agent/etc/amazon-cloudwatch-agent.json"
	}

	jsonData, err := ioutil.ReadFile(cwlConfigFile)
	if err != nil {
		log.Fatal(err)
	}

	jsonParsed, err := gabs.ParseJSON(jsonData)
	if err != nil {
		log.Fatal(err)
	}

	if jsonParsed.ExistsP("logs.logs_collected.files.collect_list") {
		logFilesJson := jsonParsed.Path("logs.logs_collected.files.collect_list").EncodeJSON()

		if err := json.Unmarshal(logFilesJson, &LogsFromConfig); err != nil {
			log.Fatal(err)
		}
	}

	modified := false
	for _, logFile := range LogsFromTags {
		preExisting := false
		for _, logFromConfig := range LogsFromConfig {
			if strings.EqualFold(logFromConfig.FilePath, logFile.FilePath) {
				preExisting = true
			}
		}
		if preExisting {
			log.Println("pre-existing")
			continue
		}
		modified = true

		err = jsonParsed.ArrayAppend(logFile, "logs", "logs_collected", "files", "collect_list")
		if err != nil {
			log.Fatal(err)
		}
	}

	if modified {
		fmt.Println(jsonParsed.StringIndent("", "  "))

		ioutil.WriteFile(cwlConfigFile, []byte(jsonParsed.StringIndent("", "  ")), 0644)

		RestartService()
	}
}
