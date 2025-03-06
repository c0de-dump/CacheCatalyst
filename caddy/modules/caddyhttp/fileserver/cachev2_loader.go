package fileserver

import (
	"io"
	"os"

	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func getServiceWorkerFullPath(root string) string {
	return caddyhttp.SanitizedPathJoin(root, "/sw.js")
}

func getServiceWorkerRelativePath() string {
	return "modules/caddyhttp/fileserver/sw.js"
}

func loadCacheV2ServiceWorker(root string) error {
	dst := caddyhttp.SanitizedPathJoin(root, "sw.js")
	return copyFile("modules/caddyhttp/fileserver/sw.js", dst)
}

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}
