package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

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

	// TODO: implement the upload here
	const maxMemory = 10 << 20
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse response  body to multipart/form-data", err)
		return
	}

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Content-Type", err)
		return
	}

	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "Thumbnail must be jpeg or png", err)
		return
	}

	exts, err := mime.ExtensionsByType(mediaType)
	if err != nil || len(exts) == 0 {
		respondWithError(w, http.StatusBadRequest, "Unsupported content type", err)
		return
	}

	videoMetaData, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to get video data from sql", err)
		return
	}

	if videoMetaData.UserID != userID {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	fileName := videoID.String() + exts[0]
	filePath := filepath.Join(cfg.assetsRoot, fileName)

	dst, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to create file", err)
		return
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to save file", err)
		return
	}

	thumbnailUrl := "/assets/" + fileName
	videoMetaData.ThumbnailURL = &thumbnailUrl

	if err := cfg.db.UpdateVideo(videoMetaData); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unbale to update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoMetaData)

	/* Bad practice
	*CPU performance: Base64 encoding is an expensive CPU-intensive operation. If we have a lot of uploads (I mean, we're planning on being a successful company, right?), we'll have some scaling issues.
	Storage costs: Base64 encoding bloats the size of the image data. We're using more disk space than we need to, which again, is expensive and slow.
	Database performance: Databases (especially relational databases like SQLite, Postgres and MySQL) are optimized for small, structured data, not giant blobs of binary. It will impact query performance in a non-trivial way.
	Caching: Base64 encoded images aren't as cache friendly as raw files, meaning slower load times and higher bandwidth costs.
		imageDataStr := base64.StdEncoding.EncodeToString(imageData)
			dataUrl := fmt.Sprintf("data:%v;base64,%v", mediaType, imageDataStr)

			videoMetaData, err := cfg.db.GetVideo(videoID)
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "Unable to get video data from sql", err)
				return
			}

			if videoMetaData.UserID != userID {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			videoMetaData.ThumbnailURL = &dataUrl

			if err := cfg.db.UpdateVideo(videoMetaData); err != nil {
				respondWithError(w, http.StatusInternalServerError, "Unable to update video", err)
				return
			}

			respondWithJSON(w, http.StatusOK, videoMetaData)
	*/

}
