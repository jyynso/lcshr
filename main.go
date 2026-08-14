package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"
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
	ModTime time.Time
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

  .item img,
  .item video {
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

  #lightbox {
    display: none;
    position: fixed;
    inset: 0;
    background: rgba(0,0,0,0.84);
    z-index: 1000;
    touch-action: pan-y;
    user-select: none;
    padding: 32px;
  }
  #lightbox.open { display: flex; align-items: center; justify-content: center; }
  #lightbox img {
    max-width: 100%;
    max-height: 100%;
    object-fit: contain;
  }
  #lightbox video { max-width: 100%; max-height: 100%; }
  #lightbox .close {
    position: absolute;
    top: 16px; right: 20px;
    font-size: 2rem;
    color: #eee;
    cursor: pointer;
    z-index: 1001;
  }
  #lightbox-img { transition: transform 0.15s ease; cursor: zoom-in; }
  #lightbox-img.zoomed { object-fit: none; max-width: none; max-height: none; cursor: zoom-out; }
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
  #lightbox .name {
    position: absolute;
    bottom: 16px;
    left: 0;
    right: 0;
    text-align: center;
    font-size: 0.9rem;
    color: #eee;
    z-index: 1001;
  }

</style>
</head>
<body>
<h2>{{.Title}}</h2>
<div class="grid">
{{range $i, $f := .Files}}
  <div class="item">
    {{if $f.IsImage}}
      <img src="{{$f.Name}}" loading="lazy" onclick="openLightbox('{{$f.Name}}')">
    {{else if $f.IsVideo}}
      <img src="/thumbs/{{$f.Name}}.jpg" muted onclick="openLightbox('{{$f.Name}}')">
    {{end}}
    <a class="filename" href="{{$f.Name}}">{{$f.Name}}</a>
  </div>
{{end}}
</div>

<div id="lightbox">
  <span class="close" onclick="closeLightbox()">&times;</span>
  <div class="nav prev" onclick="navigate(-1)">&#10094;</div>
  <img id="lightbox-img" src="">
  <video id="lightbox-video" controls></video>
  <div class="nav next" onclick="navigate(1)">&#10095;</div>
  <span class="name" id="lightbox-name"></span>
</div>

<select onchange="location.href='?sort='+this.value">
  <option value="name">Name</option>
  <option value="date">Date Modified</option>
</select>

<script>
  const items = [{{range .Files}}{{if or .IsImage .IsVideo}}{name:"{{.Name}}", video:{{.IsVideo}}},{{end}}{{end}}];
  let currentIndex = 0;
  let zoomed = false;

  const lbImg = document.getElementById('lightbox-img');
  lbImg.addEventListener('dblclick', () => {
    zoomed = !zoomed;
    lbImg.classList.toggle('zoomed', zoomed);
  });
  lbImg.addEventListener('wheel', (e) => {
    e.preventDefault();
    const scale = e.deltaY < 0 ? 1.1 : 0.9;
    const current = lbImg.style.transform.match(/scale\(([\d.]+)\)/);
    const newScale = Math.max(1, (current ? parseFloat(current[1]) : 1) * scale);
    lbImg.style.transform = 'scale(' + newScale + ')';
  });

  function openLightbox(name) {
    currentIndex = items.findIndex(i => i.name === name);
    updateImage();
    document.getElementById('lightbox').classList.add('open');
  }
  function closeLightbox() {
    document.getElementById('lightbox').classList.remove('open');
    document.getElementById('lightbox-video').pause();
  }
  function updateImage() {
    const item = items[currentIndex];
    const img = document.getElementById('lightbox-img');
    const vid = document.getElementById('lightbox-video');
    vid.pause();
    if (item.video) {
      img.style.display = 'none';
      vid.style.display = 'block';
      vid.src = item.name;
      vid.play();
    } else {
      vid.style.display = 'none';
      img.style.display = 'block';
      img.src = item.name;
    }
    document.getElementById('lightbox-name').textContent = item.name;
  }
  function navigate(dir) {
    currentIndex = (currentIndex + dir + items.length) % items.length;
    updateImage();
  }

  document.addEventListener('keydown', (e) => {
    if (!document.getElementById('lightbox').classList.contains('open')) return;
    if (e.key === 'ArrowRight') navigate(1);
    if (e.key === 'ArrowLeft') navigate(-1);
    if (e.key === 'Escape') closeLightbox();
  });

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

var videoExts = map[string]bool{
	".mp4": true, ".webm": true, ".mov": true, ".mkv": true,
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
			info, err := e.Info()
			if err != nil {
				continue
			}
			files = append(files, FileEntry{
				Name:    e.Name(),
				IsImage: imageExts[ext],
				IsVideo: videoExts[ext],
				ModTime: info.ModTime(),
			})
		}

		sortBy := r.URL.Query().Get("sort")
		switch sortBy {
		case "date":
			sort.Slice(files, func(i, j int) bool {
				return files[i].ModTime.After(files[j].ModTime)
			})
		default:
			sort.Slice(files, func(i, j int) bool {
				return files[i].Name < files[j].Name
			})
		}

		tmpl.Execute(w, map[string]interface{}{
			"Title": filepath.Base(root),
			"Files": files,
		})
	}
}

func thumbHandler(root, thumbDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/thumbs/")
		videoPath := filepath.Join(root, strings.TrimSuffix(name, ".jpg"))
		thumbPath := filepath.Join(thumbDir, name)

		if _, err := os.Stat(thumbPath); os.IsNotExist(err) {
			os.MkdirAll(thumbDir, 0755)
			cmd := exec.Command("ffmpeg", "-ss", "00:00:01", "-i", videoPath,
				"-frames:v", "1", "-vf", "scale=320:-1", thumbPath)
			if err := cmd.Run(); err != nil {
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
