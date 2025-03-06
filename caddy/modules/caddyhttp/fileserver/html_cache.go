package fileserver

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"io/fs"
	"slices"
	"strings"

	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func findTags(root *html.Node, tags []atom.Atom) []*html.Node {
	current := root.FirstChild
	var foundTags []*html.Node
	for current != nil {
		if slices.Contains(tags, current.DataAtom) {
			foundTags = append(foundTags, current)
		} else {
			foundTags = append(foundTags, findTags(current, tags)...)
		}
		current = current.NextSibling
	}
	return foundTags
}

func isLocalfile(uri string) bool {
	s := strings.ToLower(uri)
	return strings.Contains(s, "localhost") || !strings.Contains(s, "http")
}

func normalizeFilename(s string) string {
	idx := strings.Index(s, "?")
	if idx == -1 {
		return s
	}

	return s[:idx]
}

func getEtagJsonAndRegisterServiceWorker(fsys fs.FS, rootDir string, h io.Reader) (string, string, error) {
	m := make(map[string]string)
	root, err := html.Parse(h)
	if err != nil {
		return "", "", err
	}

	tags := findTags(root, []atom.Atom{atom.Img, atom.Link, atom.Script})
	for _, node := range tags {
		var src *html.Attribute
		for _, attr := range node.Attr {
			a := attr
			if attr.Key == "src" {
				src = &a
				break
			} else if attr.Key == "href" {
				src = &a
				break
			}
		}

		if src == nil || !isLocalfile(src.Val) {
			continue
		}

		fileName := strings.TrimSuffix(caddyhttp.SanitizedPathJoin(rootDir, normalizeFilename(src.Val)), "/")
		stat, err := fs.Stat(fsys, fileName)
		if err != nil {
			continue
		}

		m[src.Val] = calculateEtag(stat)
	}

	jsCode := `
if ('serviceWorker' in navigator) {
    navigator.serviceWorker.register('/sw.js').then(function() {
        return navigator.serviceWorker.ready;
    }).catch(function(error) {
        console.log('Error : ', error);
    });
}
`
	body := findTags(root, []atom.Atom{atom.Body})[0]
	script := &html.Node{
		Type:     html.ElementNode,
		Data:     "script",
		DataAtom: atom.Script,
	}
	script.AppendChild(&html.Node{
		Type: html.TextNode,
		Data: jsCode,
	})
	if body.FirstChild != nil {
		body.InsertBefore(script, body.FirstChild)
	} else {
		body.AppendChild(script)
	}

	buf := new(bytes.Buffer)
	w := bufio.NewWriter(buf)
	_ = html.Render(w, root)
	err = w.Flush()
	if err != nil {
		return "", "", err
	}

	etagJson, err := json.Marshal(m)
	if err != nil {
		return "", "", err
	}

	return buf.String(), string(etagJson), nil
}
