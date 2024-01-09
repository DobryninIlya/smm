package page_constructor

import (
	"log"
	"path/filepath"
	"strings"
)

func getMainTemplate() string {
	data, err := readFile(filepath.Join(htmlPath, "main.html"))
	if err != nil {
		log.Println(err)
		return ""
	}
	return strings.Join(data, "\n")
}
