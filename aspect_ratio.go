package main

import (
	"bytes"
	"encoding/json"
	"log"
	"os/exec"
)

type VideoInfo struct {
	Streams []struct {
		Height int `json:"height"`
		Width  int `json:"width"`
	} `json:"streams"`
}

const (
	aspectLandscape = "landscape"
	aspectOther     = "other"
	aspectPortrait  = "portrait"
	cmdProbe        = "ffprobe"
)

var args = [5]string{
	"-v",
	"error",
	"-print_format",
	"json",
	"-show_streams",
}

func getVideoAspectRatio(path string) (aspect string, err error) {
	cmd := exec.Command(cmdProbe, append(args[:], path)...)

	var out bytes.Buffer

	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		log.Printf("Error running ffprobe: %s", err)
		return
	}

	var info VideoInfo
	err = json.Unmarshal(out.Bytes(), &info)
	if err != nil {
		log.Printf("Error unmarshalling JSON: %s", err)
		return
	}

	if len(info.Streams) > 0 {
		aspect = aspectOther
		videoStream := info.Streams[0]

		if videoStream.Width > videoStream.Height {
			aspect = aspectLandscape
		} else if videoStream.Width < videoStream.Height {
			aspect = aspectPortrait
		}
	}

	log.Printf("Video aspect ratio: %s", aspect)

	return
}
