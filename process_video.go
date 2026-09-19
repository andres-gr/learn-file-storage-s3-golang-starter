package main

import (
	"log"
	"os/exec"
	"strings"
)

const (
	cmdFfmpeg = "ffmpeg"
)

func processVideoForFastStart(filePath string) (path string, err error) {
	out := filePath + ".processing"
	cmdArgs := [9]string{
		"-i", filePath,
		"-c", "copy",
		"-movflags", "faststart",
		"-f", "mp4",
		out,
	}

	log.Printf("Running: %s %s", cmdFfmpeg, strings.Join(cmdArgs[:], " "))

	cmd := exec.Command(cmdFfmpeg, cmdArgs[:]...)
	err = cmd.Run()
	if err != nil {
		log.Printf("Error running ffmpeg: %s", err)
		return
	}

	path = out

	return
}
