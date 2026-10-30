package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
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

	maxMemory := 10 << 20
	err = r.ParseMultipartForm(int64(maxMemory))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error handling multipart data", err)
		return
	}

	file, fileHeaders, err := r.FormFile("thumbnail")
	contentTypeHeader := fileHeaders.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentTypeHeader)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid content type header", err)
		return
	}
	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusUnsupportedMediaType, "wrong media type", err)
		return
	}
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error fetching video", err)
		return
	}
	if userID != video.UserID {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	thumbnailPath := fmt.Sprintf("/%s.%s", videoID, strings.TrimPrefix(mediaType, "image/"))
	imagePath := filepath.Join(cfg.assetsRoot, thumbnailPath)
	assetFile, err := os.Create(imagePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error creating file", err)
		return
	}
	_, err = io.Copy(assetFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error saving file", err)
		return
	}
	dataURL := fmt.Sprintf("http://localhost:%s/%s", cfg.port, strings.TrimPrefix(imagePath, "./"))
	video.ThumbnailURL = &dataURL
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error updating video metadata", err)
		return
	}
	respondWithJSON(w, http.StatusOK, video)
}
