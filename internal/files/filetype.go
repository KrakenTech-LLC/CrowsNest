package files

import "strings"

type FileType int32

const (
	JSON FileType = iota
	XML
	YAML
	TEXT
	GREPPABLE
	UNKNOWN
)

func GetFileType(filetype string) FileType {
	switch strings.ToLower(strings.TrimSpace(filetype)) {
	case "json":
		return JSON
	case "xml":
		return XML
	case "yaml":
		return YAML
	case "txt", "text":
		return TEXT
	case "grep", "greppable":
		return GREPPABLE
	default:
		return JSON
	}
}

func (ft FileType) String() string {
	switch ft {
	case JSON:
		return "json"
	case XML:
		return "xml"
	case YAML:
		return "yaml"
	case TEXT:
		return "txt"
	case GREPPABLE:
		return "grep"
	default:
		return "json"
	}
}

func (ft FileType) Extension() string {
	return "." + ft.String()
}
