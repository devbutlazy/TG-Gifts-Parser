package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
)

func fetchHTML(url string) (*html.Node, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	doc, err := htmlquery.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}
	return doc, nil
}

func extractField(doc *html.Node, field string) string {
	td := htmlquery.FindOne(doc, fmt.Sprintf(`//th[text()="%s"]/following-sibling::td`, field))
	if td == nil {
		return "N/A"
	}

	text := strings.TrimSpace(htmlquery.InnerText(td))
	mark := htmlquery.FindOne(td, "mark")
	if mark != nil {
		return fmt.Sprintf("%s (%s)", strings.TrimSpace(htmlquery.InnerText(td)), strings.TrimSpace(htmlquery.InnerText(mark)))
	}
	return text
}

func parseInfo(doc *html.Node) map[string]string {
	info := make(map[string]string)

	ownerNode := htmlquery.FindOne(doc, `//th[text()="Owner"]/following-sibling::td/a`)
	if ownerNode != nil {
		nameNode := htmlquery.FindOne(ownerNode, "span")
		name := strings.TrimSpace(htmlquery.InnerText(nameNode))
		href := strings.TrimSpace(htmlquery.SelectAttr(ownerNode, "href"))
		info["Owner"] = fmt.Sprintf("%s (%s)", name, href)
	} else {
		info["Owner"] = "N/A"
	}

	for _, field := range []string{"Model", "Backdrop", "Symbol", "Quantity"} {
		info[field] = extractField(doc, field)
	}

	return info
}

func main() {
	url := "https://t.me/nft/BowTie-5443"
	doc, err := fetchHTML(url)

	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	info := parseInfo(doc)
	for key, value := range info {
		fmt.Printf("%s: %s\n", key, value)
	}
}
