package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
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

	data, err := io.ReadAll(thumb)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't read thumbnail", err)
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

	thumbnail := thumbnail{
		data:      data,
		mediaType: media,
	}

	videoThumbnails[videoID] = thumbnail
	url := fmt.Sprintf("http://localhost:%s/api/thumbnails/%s", cfg.port, videoIDString)

	err = cfg.db.UpdateVideo(database.Video{
		ID:           video.ID,
		ThumbnailURL: &url,
		VideoURL:     video.VideoURL,
		UpdatedAt:    time.Now(),
		CreateVideoParams: database.CreateVideoParams{
			Title:       video.Title,
			Description: video.Description,
			UserID:      video.UserID,
		},
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, database.Video{
		ID:           videoID,
		ThumbnailURL: &url,
		VideoURL:     video.VideoURL,
		CreateVideoParams: database.CreateVideoParams{
			Title:       video.Title,
			Description: video.Description,
			UserID:      video.UserID,
		},
	})
}
