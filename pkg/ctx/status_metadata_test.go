package ctx

import "testing"

func TestParseDocumentMetadata(t *testing.T) {
	content := []byte("# Architecture\n\n<!-- ctx:doc {\"status\":\"verified\",\"verifiedAt\":\"abc1234\",\"sources\":[\"cmd/\",\"pkg/\"]} -->\n")
	metadata, found, err := parseDocumentMetadata(content)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("metadata was not found")
	}
	if metadata.Status != "verified" || metadata.VerifiedAt != "abc1234" || len(metadata.Sources) != 2 {
		t.Fatalf("metadata = %+v", metadata)
	}
}

func TestParseDocumentMetadataRejectsMalformedAndDuplicateComments(t *testing.T) {
	tests := map[string][]byte{
		"malformed": []byte("<!-- ctx:doc not-json -->\n"),
		"duplicate": []byte("<!-- ctx:doc {\"status\":\"draft\",\"verifiedAt\":\"\",\"sources\":[]} -->\n<!-- ctx:doc {\"status\":\"draft\",\"verifiedAt\":\"\",\"sources\":[]} -->\n"),
		"unknown":   []byte("<!-- ctx:doc {\"status\":\"draft\",\"verifiedAt\":\"\",\"sources\":[],\"source\":\"typo\"} -->\n"),
	}
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			if _, _, err := parseDocumentMetadata(content); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestDocumentWordWarningLimit(t *testing.T) {
	tests := map[string]int{
		"INDEX.md":                    500,
		"local/CONTINUE.md":           600,
		"context/overview.md":         1000,
		"context/architecture/api.md": 1600,
		"README.md":                   0,
	}
	for path, want := range tests {
		if got := documentWordWarningLimit(path); got != want {
			t.Errorf("documentWordWarningLimit(%q) = %d, want %d", path, got, want)
		}
	}
}
