package main

import (
	"strings"
	"time"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

func (cfg *apiConfig) dbVideoToSignedVideo(vid database.Video) (video database.Video, err error) {
	keys := strings.Split(*vid.VideoURL, ",")
	uri, err := generatePresignedURL(cfg.s3Client, keys[0], keys[1], 5*time.Minute)
	if err != nil {
		return
	}

	vid.VideoURL = &uri
	video = vid

	return
}
