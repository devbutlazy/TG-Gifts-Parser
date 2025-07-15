package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
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

func extractGiftField(doc *html.Node, field string) string {
	td := htmlquery.FindOne(doc, fmt.Sprintf(`//th[text()="%s"]/following-sibling::td`, field))
	if td == nil {
		return "Unknown"
	}

	text := strings.TrimSpace(htmlquery.InnerText(td))
	mark := htmlquery.FindOne(td, "mark")
	if mark != nil {
		return fmt.Sprintf("%s (%s)", strings.TrimSpace(htmlquery.InnerText(td)), strings.TrimSpace(htmlquery.InnerText(mark)))
	}
	return text
}

func parseGiftInfo(doc *html.Node) map[string]string {
	info := make(map[string]string)

	if ownerTd := htmlquery.FindOne(doc, `//th[text()="Owner"]/following-sibling::td`); ownerTd != nil {
		var (
			name string
			href string
		)

		if a := htmlquery.FindOne(ownerTd, `a`); a != nil {
			href = strings.TrimSpace(htmlquery.SelectAttr(a, "href"))

			if span := htmlquery.FindOne(a, `span`); span != nil {
				name = htmlquery.InnerText(span)
			} else {
				name = htmlquery.InnerText(a)
			}
		} else if span := htmlquery.FindOne(ownerTd, `span`); span != nil {
			name = htmlquery.InnerText(span)
		} else {
			name = htmlquery.InnerText(ownerTd)
		}

		info["Owner"] = strings.TrimSpace(name)
		if href != "" {
			info["Owner"] += fmt.Sprintf(" (%s)", href)
		}
	} else {
		info["Owner"] = "Unknown"
	}

	for _, field := range []string{"Model", "Backdrop", "Symbol", "Quantity"} {
		info[field] = extractGiftField(doc, field)
	}

	return info
}

func main() {
	var url string
	fmt.Print("Enter the gift url: ")
	fmt.Scan(&url)

	doc, err := fetchHTML(url)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	info := parseGiftInfo(doc)
	for key, value := range info {
		if key == "Quantity" {
			cleaned := strings.Split(value, `/`)
			num, _ := strconv.Atoi(strings.ReplaceAll(strings.ReplaceAll(cleaned[0], "\u00A0", ""), " ", ""))
			fmt.Printf("%s: %v\n", key, num)
		} else {
			fmt.Printf("%s: %s\n", key, value)
		}
	}
}
