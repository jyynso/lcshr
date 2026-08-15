# lcshr
A simple local LAN file server. Run in any folder and browse its contents from any device on the same network.

## Building and Running
Requires [Go](https://go.dev/dl/) installed.
```bash
git clone https://github.com/jyynso/lcshr.git
cd lcshr
go build -o lcshr.exe .
./lcshr.exe -dir "C:\path\to\folder" -port 8080
```
or run the exe on your chosen folder.

## Stack
- Go (`net/http`, `html/template`, `embed`)
- JS and CSS
- `ffmpeg` for video thumbnails

In progress, built as a learning project.