package aiproxy

import "github.com/w-run/one-api/relay/adaptor/openai"

var ModelList = []string{""}

func init() {
	ModelList = openai.ModelList
}
