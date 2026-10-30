package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	uploadLimit := 1 << 30 //1GB
	r.Body = http.MaxBytesReader(w, r.Body, int64(uploadLimit))

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
		respondWithError(w, http.StatusInternalServerError, "error fetching video", err)
		return
	}
	if userID != video.UserID {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	file, fileHeaders, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error with file", err)
	}
	defer file.Close()
	contentTypeHeader := fileHeaders.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentTypeHeader)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid content type header", err)
		return
	}
	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusUnsupportedMediaType, "wrong media type", err)
		return
	}

	tmpFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error during upload", err)
		return
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()
	written, err := io.Copy(tmpFile, file)
	fmt.Printf("%v bytes written\n", written)
	tmpFile.Seek(0, io.SeekStart)
	randBytes := make([]byte, 32)
	rand.Read(randBytes)
	fileName := base64.RawURLEncoding.EncodeToString(randBytes)
	videoFileName := fmt.Sprintf("%s.%s", fileName, strings.TrimPrefix(mediaType, "video/"))

	cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket:      aws.String(cfg.s3Bucket),
		Key:         aws.String(videoFileName),
		Body:        tmpFile,
		ContentType: aws.String(mediaType),
	})
	videoURL := cfg.getObjectURL(videoFileName)
	video.VideoURL = &videoURL
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error updating db", err)
	}
	respondWithJSON(w, http.StatusOK, video)
}
