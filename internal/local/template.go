package local

import (
	"strings"
	"unicode/utf8"
)

// chatML formats messages in the ChatML layout, the fallback for model
// files that carry no chat template llama.cpp recognizes.
func chatML(msgs []TemplateMessage, addAssistant bool) string {
	var b strings.Builder
	for _, m := range msgs {
		b.WriteString("<|im_start|>" + m.Role + "\n" + m.Content + "<|im_end|>\n")
	}
	if addAssistant {
		b.WriteString("<|im_start|>assistant\n")
	}
	return b.String()
}

// splitValidUTF8 returns the longest valid UTF-8 prefix of b and the
// incomplete tail - tokens can end mid-character.
func splitValidUTF8(b []byte) (string, []byte) {
	end := len(b)
	for i := 0; i < utf8.UTFMax && end > 0; i++ {
		if utf8.Valid(b[:end]) {
			break
		}
		end--
	}
	if !utf8.Valid(b[:end]) {
		return strings.ToValidUTF8(string(b), "�"), nil
	}
	rest := append([]byte(nil), b[end:]...)
	return string(b[:end]), rest
}
