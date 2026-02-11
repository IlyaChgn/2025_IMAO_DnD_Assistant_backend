package converter

import (
	"encoding/json"
	"strings"
)

// TiptapToPlainText extracts plain text from a Tiptap document or plain string.
// Handles both formats encountered in LSS exports:
//   - Tiptap JSON: {"type": "doc", "content": [{"type": "paragraph", "content": [...]}]}
//   - Plain string: "some text" (returned as-is)
//   - nil: returns ""
//
// Preserves paragraph breaks as \n. Strips all formatting marks (bold, italic, link, etc.).
func TiptapToPlainText(data interface{}) string {
	if data == nil {
		return ""
	}

	switch v := data.(type) {
	case string:
		// Try parsing as JSON in case it's a JSON-encoded Tiptap document
		v = strings.TrimSpace(v)
		if len(v) > 0 && v[0] == '{' {
			var doc map[string]interface{}
			if err := json.Unmarshal([]byte(v), &doc); err == nil {
				if docType, ok := doc["type"].(string); ok && docType == "doc" {
					return extractFromDoc(doc)
				}
			}
		}
		return v

	case map[string]interface{}:
		if docType, ok := v["type"].(string); ok && docType == "doc" {
			return extractFromDoc(v)
		}
		// Unknown structure — try to extract any text
		return extractTextFromNode(v)

	default:
		return ""
	}
}

// extractFromDoc extracts plain text from a parsed Tiptap document node.
func extractFromDoc(doc map[string]interface{}) string {
	content, ok := doc["content"].([]interface{})
	if !ok {
		return ""
	}

	var paragraphs []string
	for _, item := range content {
		node, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		text := extractTextFromNode(node)
		if text != "" {
			paragraphs = append(paragraphs, text)
		}
	}

	return strings.Join(paragraphs, "\n")
}

// extractTextFromNode recursively extracts text from a Tiptap node.
func extractTextFromNode(node map[string]interface{}) string {
	nodeType, _ := node["type"].(string)

	switch nodeType {
	case "text":
		text, _ := node["text"].(string)
		return text

	case "hardBreak":
		return "\n"

	case "paragraph", "heading", "blockquote":
		return extractFromContent(node)

	case "bulletList", "orderedList":
		return extractListItems(node)

	case "listItem":
		return extractFromContent(node)

	default:
		// For unknown node types, try to extract content recursively
		return extractFromContent(node)
	}
}

// extractFromContent extracts concatenated text from a node's "content" array.
func extractFromContent(node map[string]interface{}) string {
	content, ok := node["content"].([]interface{})
	if !ok {
		return ""
	}

	var parts []string
	for _, item := range content {
		child, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		text := extractTextFromNode(child)
		if text != "" {
			parts = append(parts, text)
		}
	}

	return strings.Join(parts, "")
}

// extractListItems extracts text from list items, prefixing each with "- ".
func extractListItems(node map[string]interface{}) string {
	content, ok := node["content"].([]interface{})
	if !ok {
		return ""
	}

	var items []string
	for _, item := range content {
		child, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		text := extractTextFromNode(child)
		if text != "" {
			items = append(items, "- "+text)
		}
	}

	return strings.Join(items, "\n")
}
