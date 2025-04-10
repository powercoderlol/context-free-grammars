package context_free_grammar

type options struct {
	keepMatchedTokens     bool
	calculateNeedleLength bool
}

type Option func(opt *options)

func KeepMatchedTokens() Option {
	return func(opt *options) {
		opt.keepMatchedTokens = true
	}
}

func CalculateNeedleLength() Option {
	return func(opt *options) {
		opt.calculateNeedleLength = true
	}
}

