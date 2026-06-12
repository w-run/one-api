package relay

import (
	"github.com/w-run/mimi-router/relay/adaptor"
	"github.com/w-run/mimi-router/relay/adaptor/aiproxy"
	"github.com/w-run/mimi-router/relay/adaptor/ali"
	"github.com/w-run/mimi-router/relay/adaptor/anthropic"
	"github.com/w-run/mimi-router/relay/adaptor/aws"
	"github.com/w-run/mimi-router/relay/adaptor/baidu"
	"github.com/w-run/mimi-router/relay/adaptor/cloudflare"
	"github.com/w-run/mimi-router/relay/adaptor/cohere"
	"github.com/w-run/mimi-router/relay/adaptor/coze"
	"github.com/w-run/mimi-router/relay/adaptor/deepl"
	"github.com/w-run/mimi-router/relay/adaptor/gemini"
	"github.com/w-run/mimi-router/relay/adaptor/ollama"
	"github.com/w-run/mimi-router/relay/adaptor/openai"
	"github.com/w-run/mimi-router/relay/adaptor/palm"
	"github.com/w-run/mimi-router/relay/adaptor/proxy"
	"github.com/w-run/mimi-router/relay/adaptor/replicate"
	"github.com/w-run/mimi-router/relay/adaptor/tencent"
	"github.com/w-run/mimi-router/relay/adaptor/vertexai"
	"github.com/w-run/mimi-router/relay/adaptor/xunfei"
	"github.com/w-run/mimi-router/relay/adaptor/zhipu"
	"github.com/w-run/mimi-router/relay/apitype"
)

func GetAdaptor(apiType int) adaptor.Adaptor {
	switch apiType {
	case apitype.AIProxyLibrary:
		return &aiproxy.Adaptor{}
	case apitype.Ali:
		return &ali.Adaptor{}
	case apitype.Anthropic:
		return &anthropic.Adaptor{}
	case apitype.AwsClaude:
		return &aws.Adaptor{}
	case apitype.Baidu:
		return &baidu.Adaptor{}
	case apitype.Gemini:
		return &gemini.Adaptor{}
	case apitype.OpenAI:
		return &openai.Adaptor{}
	case apitype.PaLM:
		return &palm.Adaptor{}
	case apitype.Tencent:
		return &tencent.Adaptor{}
	case apitype.Xunfei:
		return &xunfei.Adaptor{}
	case apitype.Zhipu:
		return &zhipu.Adaptor{}
	case apitype.Ollama:
		return &ollama.Adaptor{}
	case apitype.Coze:
		return &coze.Adaptor{}
	case apitype.Cohere:
		return &cohere.Adaptor{}
	case apitype.Cloudflare:
		return &cloudflare.Adaptor{}
	case apitype.DeepL:
		return &deepl.Adaptor{}
	case apitype.VertexAI:
		return &vertexai.Adaptor{}
	case apitype.Proxy:
		return &proxy.Adaptor{}
	case apitype.Replicate:
		return &replicate.Adaptor{}
	}
	return nil
}
