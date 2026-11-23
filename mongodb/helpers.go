package mongodb

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
)

func ParseId(id string, expectedParts int) ([]string, error) {
	result, errEncoding := base64.StdEncoding.DecodeString(id)
	if errEncoding != nil {
		return nil, fmt.Errorf("unexpected format of ID Error : %s", errEncoding)
	}
	parts := strings.SplitN(string(result), ".", expectedParts)
	if len(parts) != expectedParts {
		return nil, fmt.Errorf("unexpected format of ID (%s), expected attribute1.attributeN", id)
	}

	for _, part := range parts {
		if part == "" {
			return nil, fmt.Errorf("invalid ID format: %s", result)
		}
	}

	return parts, nil
}

var iamARNRegex = regexp.MustCompile(`^arn:aws:iam::\d{12}:(user|role)/[\w+=,.@/-]+$`)
