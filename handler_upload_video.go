package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	videoIdStr := r.PathValue("videoID")
	videoId, err := uuid.Parse(videoIdStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	uId, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	video, err := cfg.db.GetVideo(videoId)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't get video", err)
		return
	}

	if video.UserID != uId {
		respondWithError(w, http.StatusUnauthorized, "You can't upload to this video", err)
		return
	}

	fmt.Println("uploading video...", videoId, "by user", uId)

	const maxMem = 1 << 30
	r.Body = http.MaxBytesReader(w, r.Body, maxMem)

	if err := r.ParseMultipartForm(maxMem); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			respondWithError(w, http.StatusRequestEntityTooLarge, "File too large", err)
			return
		}

		respondWithError(w, http.StatusBadRequest, "Couldn't parse form", err)
		return
	}

	vid, vidHead, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't get file", err)
		return
	}

	defer func() {
		if err := vid.Close(); err != nil {
			log.Printf("Error closing file: %s", err)
			return
		}
	}()

	media := vidHead.Header.Get("Content-Type")
	medType, _, err := mime.ParseMediaType(media)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid media type", err)
		return
	}

	if medType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Invalid media type", err)
		return
	}

	ext := strings.Split(media, "/")[1]

	temp, err := os.CreateTemp("", "tmp-tubely-upload-video."+ext)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create temp file", err)
		return
	}

	defer func() {
		if err := os.Remove(temp.Name()); err != nil {
			log.Printf("Error removing temp file: %s", err)
			return
		}
	}()

	defer func() {
		if err := temp.Close(); err != nil {
			log.Printf("Error closing file: %s", err)
			return
		}
	}()

	_, err = io.Copy(temp, vid)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't copy file", err)
		return
	}

	_, err = temp.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't seek file", err)
		return
	}

	aspect, err := getVideoAspectRatio(temp.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get video aspect ratio", err)
		return
	}

	key := make([]byte, 32)
	rand.Read(key)

	videoName := base64.RawURLEncoding.EncodeToString(key)

	_, err = cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket:      aws.String(cfg.s3Bucket),
		Key:         aws.String(aspect + "/" + videoName + "." + ext),
		Body:        temp,
		ContentType: aws.String(medType),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't upload file", err)
		return
	}

	uri := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, aspect+"/"+videoName+"."+ext)
	video.VideoURL = &uri
	video.UpdatedAt = time.Now()

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
