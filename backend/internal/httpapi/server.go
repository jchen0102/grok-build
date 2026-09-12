package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/jchen0102/grok-build/internal/compute"
	"github.com/jchen0102/grok-build/internal/importdata"
	"github.com/jchen0102/grok-build/internal/store"
)

type Server struct {
	Store *store.Store
}

func New(st *store.Store) http.Handler {
	s := &Server{Store: st}
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(cors)
	r.Get("/health", s.health)
	r.Post("/api/imports", s.importCSV)
	r.Get("/api/overview", s.overview)
	return r
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) importCSV(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("file")
	if err != nil {
		file, _, err = r.FormFile("csv")
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing_file", "请上传 CSV 文件（字段名 file）")
		return
	}
	defer file.Close()

	rows, result, err := importdata.Parse(io.LimitReader(file, 20<<20))
	if err != nil {
		var headerErr *importdata.HeaderError
		if errors.As(err, &headerErr) {
			writeError(w, http.StatusBadRequest, "missing_header", headerErr.Message)
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_csv", err.Error())
		return
	}

	requests := make([]store.Request, 0, len(rows))
	for _, row := range rows {
		requests = append(requests, row.Request)
	}

	inserted, dbDups, err := s.Store.InsertRequests(r.Context(), requests)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "insert_failed", "写入数据库失败")
		return
	}
	importdata.ApplyDBDuplicates(result, rows, dbDups)
	result.Summary.Success = inserted
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) overview(w http.ResponseWriter, r *http.Request) {
	from, to, emptyDB, err := parseWindow(r, s.Store)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_window", err.Error())
		return
	}
	if emptyDB {
		writeJSON(w, http.StatusOK, compute.Overview{Empty: true, Trend: []compute.HourBucket{}})
		return
	}
	rows, err := s.Store.ListInWindow(r.Context(), from, to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query_failed", "读取请求失败")
		return
	}
	writeJSON(w, http.StatusOK, compute.BuildOverview(rows, from, to))
}

type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	var body errorBody
	body.Error.Code = code
	body.Error.Message = message
	writeJSON(w, status, body)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
