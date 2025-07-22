package parser

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
)

func FetchHTML(url string, attempts int, delay time.Duration) (*html.Node, error) {
	var err error
	for i := 0; i < attempts; i++ {
		resp, err := http.Get(url)
		if err == nil {
			defer resp.Body.Close()
			doc, parseErr := htmlquery.Parse(resp.Body)
			if parseErr == nil {
				return doc, nil
			}
			err = fmt.Errorf("failed to parse HTML: %w", parseErr)
		}
		fmt.Printf("Fetch failed for %s (attempt %d/%d): %v\n", url, i+1, attempts, err)
		time.Sleep(delay)
	}
	return nil, err
}

func ExtractGiftField(doc *html.Node, field string) string {
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
		info[field] = ExtractGiftField(doc, field)
	}

	return info
}

func CleanQuantity(q string) int {
	cleaned := strings.Split(q, `/`)
	num, _ := strconv.Atoi(strings.ReplaceAll(strings.ReplaceAll(cleaned[0], "\u00A0", ""), " ", ""))
	return num
}
