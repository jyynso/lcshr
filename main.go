package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type FileEntry struct {
	Name    string
	IsImage bool
	IsVideo bool
	IsDir   bool
	ModTime time.Time
	Size    int64
}

type Breadcrumb struct {
	Name string
	Path string
}

//go:embed tmplt.html
var tmpltHTML string

var tmpl = template.Must(template.New("index").Funcs(template.FuncMap{
	"urlenc": url.PathEscape,
}).Parse(tmpltHTML))

var imageExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true,
}

var videoExts = map[string]bool{
	".mp4": true, ".webm": true, ".mov": true, ".mkv": true,
}

var thumbSem = make(chan struct{}, 4)

func handler(root string) http.HandlerFunc {
	fs := http.FileServer(http.Dir(root))

	return func(w http.ResponseWriter, r *http.Request) {
		reqPath := filepath.Join(root, filepath.Clean(r.URL.Path))
		if info, err := os.Stat(reqPath); err == nil && !info.IsDir() {
			fs.ServeHTTP(w, r)
			return
		}

		entries, err := os.ReadDir(reqPath)
		if err != nil {
			http.Error(w, "cannot read folder", 500)
			return
		}

		var files []FileEntry
		for _, e := range entries {
			ext := strings.ToLower(filepath.Ext(e.Name()))
			info, err := e.Info()
			if err != nil {
				continue
			}
			files = append(files, FileEntry{
				Name:    e.Name(),
				IsImage: imageExts[ext],
				IsVideo: videoExts[ext],
				IsDir:   e.IsDir(),
				ModTime: info.ModTime(),
				Size:    info.Size(),
			})
		}

		var lightboxItems []map[string]interface{}
		for _, f := range files {
			if f.IsImage || f.IsVideo {
				lightboxItems = append(lightboxItems, map[string]interface{}{
					"name":  f.Name,
					"video": f.IsVideo,
				})
			}
		}
		itemsJSON, _ := json.Marshal(lightboxItems)

		sortBy := r.URL.Query().Get("sort")
		switch sortBy {
		case "size":
			sort.Slice(files, func(i, j int) bool {
				return (files[i].Size > files[j].Size)
			})
		case "date":
			sort.Slice(files, func(i, j int) bool {
				return files[i].ModTime.After(files[j].ModTime)
			})
		default:
			sort.Slice(files, func(i, j int) bool {
				return files[i].Name < files[j].Name
			})
		}

		urlPath := strings.Trim(r.URL.Path, "/")
		var crumbs []Breadcrumb
		if urlPath != "" {
			parts := strings.Split(urlPath, "/")
			accum := ""
			for _, p := range parts {
				accum += "/" + p
				crumbs = append(crumbs, Breadcrumb{Name: p, Path: accum})
			}
		}

		tmpl.Execute(w, map[string]interface{}{
			"Title":       filepath.Base(root),
			"Files":       files,
			"Sort":        sortBy,
			"ItemsJSON":   template.JS(itemsJSON),
			"Breadcrumbs": crumbs,
			"CurrentPath": urlPath,
		})
	}
}

func thumbHandler(root, thumbDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/thumbs/")
		videoPath := filepath.Join(root, strings.TrimSuffix(name, ".jpg"))
		thumbPath := filepath.Join(thumbDir, name)

		if _, err := os.Stat(thumbPath); os.IsNotExist(err) {
			os.MkdirAll(filepath.Dir(thumbPath), 0755)

			thumbSem <- struct{}{}
			defer func() { <-thumbSem }()

			cmd := exec.Command("ffmpeg", "-ss", "00:00:01", "-i", videoPath,
				"-frames:v", "1", "-vf", "scale=320:-1", "-y", thumbPath)
			out, err := cmd.CombinedOutput()
			if err != nil {
				log.Printf("ffmpeg failed for %s: %v\n%s", videoPath, err, out)
				http.Error(w, "thumb generation failed", 500)
				return
			}
		}
		http.ServeFile(w, r, thumbPath)
	}
}

func main() {
	dir := flag.String("dir", ".", "folder to serve")
	port := flag.String("port", "8080", "port to listen on")
	flag.Parse()

	absDir, _ := filepath.Abs(*dir)
	http.HandleFunc("/", handler(absDir))
	http.HandleFunc("/thumbs/", thumbHandler(absDir, filepath.Join(absDir, ".thumbs")))

	addr := "0.0.0.0:" + *port
	log.Printf("Serving %s at http://%s", absDir, addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
