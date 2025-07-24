package utils

import "regexp"

func RemovePercent(s string) string {
	re := regexp.MustCompile(` ?\(?\d+(\.\d+)?%\)?`)
	return re.ReplaceAllString(s, "")
}

func SanitizeGiftName(name string) string {
	re := regexp.MustCompile(`[^\w]`)
	return re.ReplaceAllString(name, "")
}
