package parser

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
)

func FetchHTML(url string) (*html.Node, error) {
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
	td := htmlquery.FindOne(doc, fmt.Sprintf(`//th[contains(normalize-space(.), "%s")]/following-sibling::td`, field))
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

func ParseGiftInfo(doc *html.Node) map[string]string {
	info := make(map[string]string)

	if ownerTd := htmlquery.FindOne(doc, `//th[contains(normalize-space(.), "Owner")]/following-sibling::td`); ownerTd != nil {
		var name, href string
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

func CleanQuantity(q string) int {
	cleaned := strings.Split(q, `/`)
	num, _ := strconv.Atoi(strings.ReplaceAll(strings.ReplaceAll(cleaned[0], "\u00A0", ""), " ", ""))
	return num
}
