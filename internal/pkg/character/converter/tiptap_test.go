package converter

import "testing"

func TestTiptapToPlainText_Nil(t *testing.T) {
	result := TiptapToPlainText(nil)
	if result != "" {
		t.Errorf("expected empty string for nil, got %q", result)
	}
}

func TestTiptapToPlainText_PlainString(t *testing.T) {
	result := TiptapToPlainText("Hello world")
	if result != "Hello world" {
		t.Errorf("expected 'Hello world', got %q", result)
	}
}

func TestTiptapToPlainText_TiptapDoc(t *testing.T) {
	doc := map[string]interface{}{
		"type": "doc",
		"content": []interface{}{
			map[string]interface{}{
				"type": "paragraph",
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "Hello ",
					},
					map[string]interface{}{
						"type": "text",
						"text": "world",
						"marks": []interface{}{
							map[string]interface{}{"type": "bold"},
						},
					},
				},
			},
			map[string]interface{}{
				"type": "paragraph",
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "Second paragraph",
					},
				},
			},
		},
	}

	result := TiptapToPlainText(doc)
	expected := "Hello world\nSecond paragraph"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestTiptapToPlainText_HTMLString(t *testing.T) {
	// LSS sometimes stores HTML strings instead of Tiptap JSON.
	// When it's just a plain string, it should be returned as-is.
	html := "<p><strong>Bold</strong> text</p>"
	result := TiptapToPlainText(html)
	if result != html {
		t.Errorf("expected HTML to be returned as-is, got %q", result)
	}
}

func TestTiptapToPlainText_JSONString(t *testing.T) {
	// Tiptap doc as a JSON string (needs parsing)
	jsonStr := `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"parsed from string"}]}]}`
	result := TiptapToPlainText(jsonStr)
	expected := "parsed from string"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestTiptapToPlainText_HardBreak(t *testing.T) {
	doc := map[string]interface{}{
		"type": "doc",
		"content": []interface{}{
			map[string]interface{}{
				"type": "paragraph",
				"content": []interface{}{
					map[string]interface{}{"type": "text", "text": "Line 1"},
					map[string]interface{}{"type": "hardBreak"},
					map[string]interface{}{"type": "text", "text": "Line 2"},
				},
			},
		},
	}

	result := TiptapToPlainText(doc)
	expected := "Line 1\nLine 2"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestTiptapToPlainText_BulletList(t *testing.T) {
	doc := map[string]interface{}{
		"type": "doc",
		"content": []interface{}{
			map[string]interface{}{
				"type": "bulletList",
				"content": []interface{}{
					map[string]interface{}{
						"type": "listItem",
						"content": []interface{}{
							map[string]interface{}{
								"type": "paragraph",
								"content": []interface{}{
									map[string]interface{}{"type": "text", "text": "Item 1"},
								},
							},
						},
					},
					map[string]interface{}{
						"type": "listItem",
						"content": []interface{}{
							map[string]interface{}{
								"type": "paragraph",
								"content": []interface{}{
									map[string]interface{}{"type": "text", "text": "Item 2"},
								},
							},
						},
					},
				},
			},
		},
	}

	result := TiptapToPlainText(doc)
	expected := "- Item 1\n- Item 2"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestTiptapToPlainText_EmptyDoc(t *testing.T) {
	doc := map[string]interface{}{
		"type":    "doc",
		"content": []interface{}{},
	}
	result := TiptapToPlainText(doc)
	if result != "" {
		t.Errorf("expected empty string for empty doc, got %q", result)
	}
}

func TestTiptapToPlainText_EmptyString(t *testing.T) {
	result := TiptapToPlainText("")
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}
