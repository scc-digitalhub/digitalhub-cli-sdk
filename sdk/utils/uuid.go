package utils

import (
	"github.com/google/uuid"
	"strings"
)

func UUIDv4NoDash() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}
