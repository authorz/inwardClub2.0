package validation

import (
	stdhtml "html"
	"net/url"
	"regexp"
	"strings"

	xhtml "golang.org/x/net/html"
)

var safeClassPattern = regexp.MustCompile(`^[A-Za-z0-9 _-]{1,200}$`)

var allowedRichTags = map[string]bool{
	"a": true, "b": true, "blockquote": true, "br": true, "code": true,
	"div": true, "em": true, "h1": true, "h2": true, "h3": true,
	"h4": true, "h5": true, "h6": true, "hr": true, "i": true,
	"img": true, "li": true, "ol": true, "p": true, "pre": true,
	"span": true, "strong": true, "table": true, "tbody": true,
	"td": true, "th": true, "thead": true, "tr": true, "u": true, "ul": true,
}

var blockedRichTags = map[string]bool{
	"script": true, "style": true, "iframe": true, "object": true,
	"embed": true, "form": true, "input": true, "button": true,
	"textarea": true, "select": true, "option": true, "svg": true, "math": true,
}

// SanitizeRichHTML returns a conservative subset accepted by the mini-program
// rich-text component. Event attributes, inline styles and unsafe URL schemes
// are removed; script-like elements and their contents are discarded entirely.
func SanitizeRichHTML(input string) string {
	if strings.TrimSpace(input) == "" {
		return ""
	}
	z := xhtml.NewTokenizer(strings.NewReader(input))
	var out strings.Builder
	blockedDepth := 0
	for {
		tt := z.Next()
		switch tt {
		case xhtml.ErrorToken:
			return strings.TrimSpace(out.String())
		case xhtml.TextToken:
			if blockedDepth == 0 {
				out.WriteString(stdhtml.EscapeString(string(z.Text())))
			}
		case xhtml.StartTagToken, xhtml.SelfClosingTagToken:
			tok := z.Token()
			tag := strings.ToLower(tok.Data)
			if blockedRichTags[tag] {
				if tt == xhtml.StartTagToken {
					blockedDepth++
				}
				continue
			}
			if blockedDepth > 0 || !allowedRichTags[tag] {
				continue
			}
			out.WriteByte('<')
			out.WriteString(tag)
			for _, attr := range tok.Attr {
				if name, value, ok := safeRichAttribute(tag, attr.Key, attr.Val); ok {
					out.WriteByte(' ')
					out.WriteString(name)
					out.WriteString(`="`)
					out.WriteString(stdhtml.EscapeString(value))
					out.WriteByte('"')
				}
			}
			if tt == xhtml.SelfClosingTagToken || tag == "br" || tag == "hr" || tag == "img" {
				out.WriteString(" />")
			} else {
				out.WriteByte('>')
			}
		case xhtml.EndTagToken:
			tag := strings.ToLower(z.Token().Data)
			if blockedRichTags[tag] {
				if blockedDepth > 0 {
					blockedDepth--
				}
				continue
			}
			if blockedDepth == 0 && allowedRichTags[tag] && tag != "br" && tag != "hr" && tag != "img" {
				out.WriteString("</")
				out.WriteString(tag)
				out.WriteByte('>')
			}
		}
	}
}

func safeRichAttribute(tag, name, value string) (string, string, bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	value = strings.TrimSpace(value)
	if name == "class" && safeClassPattern.MatchString(value) {
		return name, value, true
	}
	if name == "title" || (tag == "img" && name == "alt") {
		if cleaned, err := PlainText(value, TextOptions{Label: "富文本属性", MaxRunes: 200, AllowEmpty: true}); err == nil {
			return name, cleaned, true
		}
	}
	if tag == "img" && name == "src" && safeRichURL(value, false) {
		return name, value, true
	}
	if tag == "a" && name == "href" && safeRichURL(value, true) {
		return name, value, true
	}
	return "", "", false
}

func safeRichURL(value string, allowContact bool) bool {
	if len(value) == 0 || len(value) > 2048 {
		return false
	}
	u, err := url.Parse(value)
	if err != nil {
		return false
	}
	if (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" && u.User == nil {
		return true
	}
	return allowContact && (u.Scheme == "tel" || u.Scheme == "mailto") && u.Opaque != ""
}
