package page_constructor

import (
	"bufio"
	"log"
	"os"
	"path/filepath"
)

var (
	Path     = filepath.Join("internal", "telegram_app", "templates")
	htmlPath = filepath.Join(Path, "html")
)

func readFile(path string) ([]string, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("file does not exist")
			return nil, err
		}
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var rows []string
	sc := bufio.NewScanner(file)
	for sc.Scan() {
		rows = append(rows, sc.Text())
	}
	return rows, nil

}
