package zai

import (
	. "github.com/router-for-me/CLIProxyAPI/v7/internal/constant"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/translator"
)

const Zai = translator.Format("zai")

func init() {
	translator.Register(
		OpenAI,
		Zai,
		NewZaiRequestTransform(),
		translator.ResponseTransform{
			Stream:     NewZaiResponseStreamTransform(),
			NonStream:  NewZaiResponseNonStreamTransform(),
			TokenCount: NewZaiTokenCountTransform(),
		},
	)
}
