package canton

import "strings"

const (
	MCMSTemplateKey = "MCMS.Main:MCMS"
)

func NormalizeTemplateKey(tid string) string {
	tid = strings.TrimPrefix(tid, "#")
	parts := strings.Split(tid, ":")
	if len(parts) < 3 {
		return tid
	}

	return parts[len(parts)-2] + ":" + parts[len(parts)-1]
}
