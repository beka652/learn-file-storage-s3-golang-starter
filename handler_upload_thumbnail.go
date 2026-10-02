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
	const maxMemory = 10 << 20 
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "unable to parse form file", err )
		return 
	}
	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "unable to parse form file", err )
		return 
	}
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		respondWithError(w, http.StatusBadRequest, "Content-Type missing", nil)
		return 
	}
	mediatype, _, err := mime.ParseMediaType(contentType)
	if (mediatype != "image/jpeg" && mediatype != "image/png") || err != nil  {
		respondWithError(w, http.StatusUnsupportedMediaType, "Only jpeg or png allowed", nil )
		return 
	}
	fileformat := strings.Split(mediatype, "/")
	extension := fileformat[1]
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "unable to find video", err )
		return 
	}
	if video.UserID != userID {
		w.WriteHeader(http.StatusUnauthorized)
		return 
	}
	videoName := fmt.Sprintf("%s.%s", videoIDString, extension)
	videoPath := filepath.Join(cfg.assetsRoot, videoName)
	fileOs, err := os.Create(videoPath)
	if err != nil { 
		w.WriteHeader(http.StatusInternalServerError)
		return 
	}
	_ , err = io.Copy(fileOs, file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to copy thumbnail", err )
		return 
	}
	dataUrl := fmt.Sprintf("http://localhost:%s/assets/%s",cfg.port, videoName)
	video.ThumbnailURL = &dataUrl
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "internal error", err)
		return 
	}
	respondWithJSON(w, http.StatusOK, video)
}
