package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

func removeCommentsFromJson(json string) (string, error) {
	jsonRunes := []rune(json)

	var builder strings.Builder
	inString := false
	escaped := false

	for i := 0; i < len(jsonRunes); {
		c := jsonRunes[i]

		if c == '"' && !escaped {
			inString = !inString
		}

		if c == '\\' && inString {
			escaped = !escaped
		} else {
			escaped = false
		}

		if !inString && c == '/' && i+1 < len(jsonRunes) {
			next := jsonRunes[i+1]

			if next == '/' {
				i += 2
				for i < len(jsonRunes) && jsonRunes[i] != '\n' && jsonRunes[i] != '\r' {
					i++
				}
				continue
			}

			if next == '*' {
				i += 2
				for {
					if i+1 >= len(jsonRunes) {
						return "", fmt.Errorf("Unclosed comment")
					}
					if jsonRunes[i] == '*' && jsonRunes[i+1] == '/' {
						i += 2
						break
					}
					i++
				}
				continue
			}
		}

		builder.WriteRune(c)
		i++
	}

	return builder.String(), nil
}

func UnmarshalJsonWithComments(s string, v any) error {
	sWithoutComments, err := removeCommentsFromJson(s)
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(sWithoutComments), v)
}
