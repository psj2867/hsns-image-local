package localhsns

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"slices"

	"github.com/rs/cors"
	"github.com/sa-/slicefunk"
)

type LocalHsnsHandler struct {
	path       string
	getHandler http.Handler
}

func NewHandler(path string) *LocalHsnsHandler {
	absPath, _ := filepath.Abs(path)
	fi, err := os.Stat(absPath)
	if err != nil {
		panic(err)
	}
	if !fi.IsDir() {
		panic(errors.New(path + " is not dir"))
	}

	getHandler := http.FileServer(http.Dir(absPath))
	return &LocalHsnsHandler{
		path:       absPath,
		getHandler: getHandler,
	}
}

func Default(path string) http.Handler {
	lc := NewHandler(path)
	return cors.Default().Handler(lc)
}

func (s *LocalHsnsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.GetFile(w, r)
	case http.MethodPost:
		s.PostFile(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("not supported"))
	}
}

func (s *LocalHsnsHandler) GetFile(w http.ResponseWriter, r *http.Request) {
	s.getHandler.ServeHTTP(w, r)
}

func (s *LocalHsnsHandler) PostFile(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	token := r.FormValue("token")
	var requestData map[string]any
	decodedToken, err := decodeRequestToken(token)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "json error")
		return
	}
	err = json.Unmarshal(decodedToken, &requestData)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "json error")
		return
	}
	imageUuids, ok := requestData["imageUuids"].([]any)
	if !ok {
		writeError(w, http.StatusInternalServerError, "json error")
		return
	}
	uploadedImages := s.uploadImages(r.MultipartForm.File, imageUuids)
	returnData, err := json.Marshal(map[string]any{
		"uuid":           requestData["uuid"],
		"requestImages":  imageUuids,
		"uploadedImages": uploadedImages,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "json error")
		return
	}
	returnToken := encodeReturnToken(returnData)
	w.WriteHeader(http.StatusOK)
	w.Write(returnToken)
}
func (s *LocalHsnsHandler) uploadImages(files map[string][]*multipart.FileHeader, requestUuids []any) []string {
	requestUuidsStrings := slicefunk.Map(requestUuids, func(v any) string {
		return v.(string)
	})
	result := []string{}
	for key, file := range files {
		if len(file) < 1 {
			continue
		}
		image := file[0]
		fileName := key
		if !slices.Contains(requestUuidsStrings, fileName) {
			continue
		}
		fs, err := os.OpenFile(filepath.Join(s.path, fileName), os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			continue
		}
		imageFile, _ := image.Open()
		io.Copy(fs, imageFile)
		result = append(result, fileName)
		imageFile.Close()
		fs.Close()
	}
	return result
}
func writeError(w http.ResponseWriter, code int, text string) {
	w.WriteHeader(code)
	w.Write([]byte(text))
}

func decodeRequestToken(src string) ([]byte, error) {
	dst := make([]byte, base64.StdEncoding.DecodedLen(len(src)))
	n, err := base64.StdEncoding.Decode(dst, []byte(src))
	return dst[:n], err
}

func encodeReturnToken(src []byte) []byte {
	dst := make([]byte, base64.StdEncoding.EncodedLen(len(src)))
	base64.StdEncoding.Encode(dst, src)
	return dst
}
