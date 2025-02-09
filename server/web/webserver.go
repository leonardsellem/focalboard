package web

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"text/template"

	"github.com/gorilla/mux"
)

// RoutedService defines the interface that is needed for any service to
// register themself in the web server to provide new endpoints. (see
// AddRoutes).
type RoutedService interface {
	RegisterRoutes(*mux.Router)
}

// Server is the structure responsible for managing our http web server.
type Server struct {
	http.Server

	baseURL   string
	rootPath  string
	port      int
	ssl       bool
	localOnly bool
}

// NewServer creates a new instance of the webserver.
func NewServer(rootPath string, serverRoot string, port int, ssl, localOnly bool) *Server {
	r := mux.NewRouter()

	var addr string
	if localOnly {
		addr = fmt.Sprintf(`localhost:%d`, port)
	} else {
		addr = fmt.Sprintf(`:%d`, port)
	}

	baseURL := ""
	url, err := url.Parse(serverRoot)
	if err != nil {
		log.Printf("Invalid ServerRoot setting: %v\n", err)
	}
	baseURL = url.Path

	ws := &Server{
		Server: http.Server{
			Addr:    addr,
			Handler: r,
		},
		baseURL:  baseURL,
		rootPath: rootPath,
		port:     port,
		ssl:      ssl,
	}

	return ws
}

func (ws *Server) Router() *mux.Router {
	return ws.Server.Handler.(*mux.Router)
}

// AddRoutes allows services to register themself in the webserver router and provide new endpoints.
func (ws *Server) AddRoutes(rs RoutedService) {
	rs.RegisterRoutes(ws.Router())
}

func (ws *Server) registerRoutes() {
	ws.Router().PathPrefix("/static").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(filepath.Join(ws.rootPath, "static")))))
	ws.Router().PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		indexTemplate, err := template.New("index").ParseFiles(path.Join(ws.rootPath, "index.html"))
		if err != nil {
			log.Printf("Unable to serve the index.html fil, err: %v\n", err)
			w.WriteHeader(500)
			return
		}
		err = indexTemplate.ExecuteTemplate(w, "index.html", map[string]string{"BaseURL": ws.baseURL})
		if err != nil {
			log.Printf("Unable to serve the index.html fil, err: %v\n", err)
			w.WriteHeader(500)
			return
		}
	})
}

// Start runs the web server and start listening for charsetnnections.
func (ws *Server) Start() {
	ws.registerRoutes()

	isSSL := ws.ssl && fileExists("./cert/cert.pem") && fileExists("./cert/key.pem")
	if isSSL {
		log.Printf("https server started on :%d\n", ws.port)
		go func() {
			if err := ws.ListenAndServeTLS("./cert/cert.pem", "./cert/key.pem"); err != nil {
				log.Fatalf("ListenAndServeTLS: %v", err)
			}
		}()

		return
	}

	log.Printf("http server started on :%d\n", ws.port)
	go func() {
		if err := ws.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe: %v", err)
		}
		log.Println("http server stopped")
	}()
}

func (ws *Server) Shutdown() error {
	return ws.Close()
}

// fileExists returns true if a file exists at the path.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}

	return err == nil
}
