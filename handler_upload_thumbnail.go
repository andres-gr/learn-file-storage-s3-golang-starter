package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't get video", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "You can't upload a thumbnail to this video", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	const maxMem = 10 << 20
	if err := r.ParseMultipartForm(maxMem); err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse form", err)
		return
	}

	thumb, thumbHead, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't get thumbnail", err)
		return
	}

	defer func() {
		if err := thumb.Close(); err != nil {
			log.Printf("Error closing file: %s", err)
			return
		}
	}()

	media := thumbHead.Header.Get("Content-Type")
	medType, _, err := mime.ParseMediaType(media)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid media type", err)
		return
	}

	if medType != "image/jpeg" && medType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "Invalid media type", err)
		return
	}

	ext := strings.Split(media, "/")[1]

	oldVideoFile := *video.ThumbnailURL

	key := make([]byte, 32)
	rand.Read(key)

	thumbName := base64.RawURLEncoding.EncodeToString(key)

	path := filepath.Join(cfg.assetsRoot, thumbName+"."+ext)
	thumbFile, err := os.Create(path)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create thumbnail file", err)
		return
	}

	defer func() {
		if err := thumbFile.Close(); err != nil {
			log.Printf("Error closing file: %s", err)
			return
		}
	}()

	_, err = io.Copy(thumbFile, thumb)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't copy thumbnail", err)
		return
	}

	thumbUrl := fmt.Sprintf("http://localhost:%s/assets/%s.%s", cfg.port, thumbName, ext)

	video.ThumbnailURL = &thumbUrl
	video.UpdatedAt = time.Now()

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
		return
	}

	if oldVideoFile != "" {
		oldVideoFile = strings.TrimPrefix(oldVideoFile, "http://localhost:"+cfg.port+"/")

		err = os.Remove("./" + oldVideoFile)
		if err != nil {
			log.Printf("Error removing old video file: %s", err)
			return
		}
	}

	respondWithJSON(w, http.StatusOK, video)
}
