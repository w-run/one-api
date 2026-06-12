package aiproxy

import "github.com/w-run/mimi-router/relay/adaptor/openai"

var ModelList = []string{""}

func init() {
	ModelList = openai.ModelList
}
