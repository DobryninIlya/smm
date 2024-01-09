package site_api_parser

import (
	"bytes"
	"golang.org/x/net/html"
	"strings"
)

func deleteTagsWithClasses(htmlInput []byte, targetClasses []string) ([]byte, error) {
	doc, err := html.Parse(bytes.NewReader(htmlInput))
	if err != nil {
		return nil, err
	}

	var cleanedHTML []byte
	var buffer bytes.Buffer

	var filterNodes func(*html.Node)
	filterNodes = func(n *html.Node) {
		if n == nil {
			return
		}

		if n.Type == html.ElementNode {
			class := getClassAttributeValue(n)
			if class == "" || !containsAnyClass(class, targetClasses) {
				renderNode(&buffer, n)
			}
		}

		// Рекурсивно обрабатываем дочерние узлы
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			filterNodes(c)
		}
	}

	filterNodes(doc)

	cleanedHTML = buffer.Bytes()

	return cleanedHTML, nil
}

func renderNode(buffer *bytes.Buffer, n *html.Node) {
	var nodeBuffer bytes.Buffer
	html.Render(&nodeBuffer, n)
	buffer.WriteString(nodeBuffer.String())
}

func getClassAttributeValue(n *html.Node) string {
	for _, attr := range n.Attr {
		if attr.Key == "class" {
			return attr.Val
		}
	}
	return ""
}

func containsAnyClass(class string, targetClasses []string) bool {
	for _, targetClass := range targetClasses {
		if strings.Contains(class, targetClass) {
			return true
		}
	}
	return false
}
