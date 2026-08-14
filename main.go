package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type FileEntry struct {
	Name    string
	IsImage bool
}

var tmpl = template.Must(template.New("index").Parse(`
<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}}</title>
<style>
  * { box-sizing: border-box; }
  body {
    font-family: sans-serif;
    background: #111;
    color: #eee;
    padding: 16px;
    margin: 0;
  }
  h2 { font-size: 1.2rem; margin-bottom: 16px; }

  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
    gap: 12px;
  }

  .item img {
    width: 100%;
    aspect-ratio: 1 / 1;
    object-fit: cover;
    border-radius: 8px;
    display: block;
    cursor: pointer;
  }
  .item a.filename {
    color: #eee;
    text-decoration: none;
    font-size: 0.8rem;
    word-break: break-word;
  }
  .item {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  @media (max-width: 600px) {
    .grid { grid-template-columns: repeat(2, 1fr); gap: 10px; }
    body { padding: 10px; }
  }
  @media (min-width: 1200px) {
    .grid { grid-template-columns: repeat(auto-fill, minmax(180px, 1fr)); }
  }

  /* Lightbox */
  #lightbox {
    display: none;
    position: fixed;
    inset: 0;
    background: rgba(0,0,0,0.95);
    z-index: 1000;
    touch-action: pan-y;
    user-select: none;
  }
  #lightbox.open { display: flex; align-items: center; justify-content: center; }
  #lightbox img {
    max-width: 100%;
    max-height: 100%;
    object-fit: contain;
    pointer-events: none;
  }
  #lightbox .close {
    position: absolute;
    top: 16px; right: 20px;
    font-size: 2rem;
    color: #eee;
    cursor: pointer;
    z-index: 1001;
  }
  #lightbox .nav {
    position: absolute;
    top: 0; bottom: 0;
    width: 25%;
    display: flex;
    align-items: center;
    font-size: 2.5rem;
    color: rgba(255,255,255,0.6);
    cursor: pointer;
    user-select: none;
  }
  #lightbox .prev { left: 0; justify-content: flex-start; padding-left: 12px; }
  #lightbox .next { right: 0; justify-content: flex-end; padding-right: 12px; }
</style>
</head>
<body>
<h2>{{.Title}}</h2>
<div class="grid">
{{range $i, $f := .Files}}
  <div class="item">
    {{if $f.IsImage}}
      <img src="{{$f.Name}}" loading="lazy" onclick="openLightbox('{{$f.Name}}')">
    {{end}}
    <a class="filename" href="{{$f.Name}}">{{$f.Name}}</a>
  </div>
{{end}}
</div>

<div id="lightbox">
  <span class="close" onclick="closeLightbox()">&times;</span>
  <div class="nav prev" onclick="navigate(-1)">&#10094;</div>
  <img id="lightbox-img" src="">
  <div class="nav next" onclick="navigate(1)">&#10095;</div>
</div>

<script>
  const images = [{{range .Files}}{{if .IsImage}}"{{.Name}}",{{end}}{{end}}];
  let currentIndex = 0;

  function openLightbox(name) {
    currentIndex = images.indexOf(name);
    updateImage();
    document.getElementById('lightbox').classList.add('open');
  }
  function closeLightbox() {
    document.getElementById('lightbox').classList.remove('open');
  }
  function updateImage() {
    document.getElementById('lightbox-img').src = images[currentIndex];
  }
  function navigate(dir) {
    currentIndex = (currentIndex + dir + images.length) % images.length;
    updateImage();
  }

  // keyboard nav
  document.addEventListener('keydown', (e) => {
    if (!document.getElementById('lightbox').classList.contains('open')) return;
    if (e.key === 'ArrowRight') navigate(1);
    if (e.key === 'ArrowLeft') navigate(-1);
    if (e.key === 'Escape') closeLightbox();
  });

  // swipe nav
  let touchStartX = 0;
  const lb = document.getElementById('lightbox');
  lb.addEventListener('touchstart', (e) => {
    touchStartX = e.changedTouches[0].screenX;
  });
  lb.addEventListener('touchend', (e) => {
    const dx = e.changedTouches[0].screenX - touchStartX;
    if (Math.abs(dx) > 50) navigate(dx < 0 ? 1 : -1);
  });
</script>
</body>
</html>
`))

var imageExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true,
}

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
			files = append(files, FileEntry{
				Name:    e.Name(),
				IsImage: imageExts[ext],
			})
		}

		tmpl.Execute(w, map[string]interface{}{
			"Title": filepath.Base(root),
			"Files": files,
		})
	}
}

func main() {
	dir := flag.String("dir", ".", "folder to serve")
	port := flag.String("port", "8080", "port to listen on")
	flag.Parse()

	absDir, _ := filepath.Abs(*dir)
	http.HandleFunc("/", handler(absDir))

	addr := "0.0.0.0:" + *port
	log.Printf("Serving %s at http://%s", absDir, addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
