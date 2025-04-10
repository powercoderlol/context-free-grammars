package context_free_grammar

import (
	"strings"
)

func countKeyTokens(key string) int {
	keyTokens := strings.Split(key, " ")
	return len(keyTokens)
}

type memoryState struct {
	dict AttrValues
}

func (m *memoryState) GetStorage() AttrValues {
	return m.dict
}

type MemoryState interface {
	GetStorage() AttrValues
}

func NewMemoryState(memory AttrValues) MemoryState {
	return &memoryState{
		memory,
	}
}

type matchState struct {
	hasMatch        bool
	remainingTokens []string
	matchedTokens   []string
	memory          MemoryState
}

func (ms *matchState) HasMatch() bool {
	return ms.hasMatch
}

func (ms *matchState) RemainingTokens() []string {
	return ms.remainingTokens
}

func (ms *matchState) MatchedTokens() []string {
	return ms.matchedTokens
}

func (ms *matchState) Memory() MemoryState {
	return ms.memory
}

type MatchState interface {
	HasMatch() bool
	RemainingTokens() []string
	MatchedTokens() []string
	Memory() MemoryState
}

func Copy(state MatchState) MatchState {
	storage := state.Memory().GetStorage()
	newMemory := make(AttrValues, len(storage))
	for k, v := range storage {
		newMemory[k] = v
	}
	return &matchState{
		state.HasMatch(),
		state.RemainingTokens(),
		state.MatchedTokens(),
		NewMemoryState(newMemory),
	}
}

func NewMatchState(hasMatch bool, remainTokens, matchedTokens []string, memory MemoryState) MatchState {
	return &matchState{
		hasMatch:        hasMatch,
		remainingTokens: remainTokens,
		memory:          memory,
		matchedTokens:   matchedTokens,
	}
}

func NewInitialState(tokens []string) MatchState {
	return &matchState{
		hasMatch:        false,
		remainingTokens: tokens,
		matchedTokens:   make([]string, 0),
		memory:          NewMemoryState(make(AttrValues)),
	}
}

type Matcher interface {
	Match(input MatchState) MatchState
}

type allowedWordMatcher struct {
	word string
}

func (w *allowedWordMatcher) Match(input MatchState) MatchState {
	tokens := input.RemainingTokens()

	if len(tokens) == 0 {
		return NewMatchState(false, tokens, nil, nil)
	}

	if tokens[0] == w.word {
		return NewMatchState(true, tokens[1:], nil, input.Memory())
	}
	return NewMatchState(false, tokens, nil, nil)
}

func NewAllowedWordMatcher(word string) Matcher {
	return &allowedWordMatcher{
		word,
	}
}

type allowedWordsMatcher struct {
	words        map[string]struct{}
	maxKeyLength int
	o            options
}

func (w *allowedWordsMatcher) Match(input MatchState) MatchState {
	tokens := input.RemainingTokens()
	if len(tokens) == 0 {
		return NewMatchState(false, tokens, nil, nil)
	}
	needleBorder := min(w.maxKeyLength, len(tokens))
	var matchedTokens []string
	for i := needleBorder; i > 0; i-- {
		lookup := strings.Join(tokens[:i], " ")
		if _, ok := w.words[lookup]; !ok {
			continue
		}
		if w.o.keepMatchedTokens {
			matchedTokens = []string{lookup}
		}
		return NewMatchState(true, tokens[i:], matchedTokens, input.Memory())
	}
	return NewMatchState(false, tokens, nil, input.Memory())
}

func NewAllowedWordsMatcher(words []string, opts ...Option) Matcher {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	res := &allowedWordsMatcher{
		words: make(map[string]struct{}, len(words)),
		o:     *o,
	}
	maxKeyLength := 0
	for _, key := range words {
		keyLength := countKeyTokens(key)
		if keyLength > maxKeyLength {
			maxKeyLength = keyLength
		}
		res.words[key] = struct{}{}
	}
	res.maxKeyLength = maxKeyLength
	return res
}

type sequenceMatcher struct {
	words []Matcher
	o     options
}

func (s *sequenceMatcher) Match(state MatchState) MatchState {
	tokens := state.RemainingTokens()
	if len(tokens) == 0 {
		return NewMatchState(false, tokens, nil, nil)
	}

	var matchedTokens []string
	for _, matcher := range s.words {
		state = matcher.Match(Copy(state))
		if !state.HasMatch() {
			return NewMatchState(false, tokens, nil, nil)
		}
		if !s.o.keepMatchedTokens {
			continue
		}
		if matchedTokens == nil {
			matchedTokens = make([]string, 0, len(s.words))
		}
		matchedTokens = append(matchedTokens, state.MatchedTokens()...)
	}
	if len(matchedTokens) > 0 {
		return NewMatchState(
			state.HasMatch(),
			state.RemainingTokens(),
			matchedTokens,
			state.Memory(),
		)
	}
	return state
}

func NewSequenceMatcher(matchers []Matcher, opts ...Option) Matcher {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}
	return &sequenceMatcher{matchers, *o}
}

type dictMatcher struct {
	dict         map[string][]ValueID
	attributeId  AttributeID
	o            options
	maxKeyLength int
}

func (m *dictMatcher) Match(state MatchState) MatchState {
	tokens := state.RemainingTokens()
	if len(tokens) == 0 {
		return NewMatchState(false, tokens, nil, nil)
	}
	memory := state.Memory().GetStorage()

	needleBorder := len(tokens)
	if m.o.calculateNeedleLength {
		needleBorder = min(m.maxKeyLength, needleBorder)
	}
	for i := needleBorder; i > 0; i-- {
		needle := strings.Join(tokens[:i], " ")
		if valueIds, ok := m.dict[needle]; ok {
			memory[m.attributeId] = append(memory[m.attributeId], valueIds...)
			var matchedTokens []string
			if m.o.keepMatchedTokens {
				matchedTokens = tokens[:i]
			}
			return NewMatchState(true, tokens[i:], matchedTokens, NewMemoryState(memory))
		}
	}

	return NewMatchState(false, tokens, nil, nil)
}

func (m *dictMatcher) Match_v1(state MatchState) MatchState {
	tokens := state.RemainingTokens()
	if len(tokens) == 0 {
		return NewMatchState(false, tokens, nil, nil)
	}
	memory := state.Memory().GetStorage()

	needleBorder := len(tokens)
	if m.o.calculateNeedleLength {
		needleBorder = min(m.maxKeyLength, needleBorder)
	}
	for i := needleBorder; i > 0; i-- {
		needle := strings.Join(tokens[:i], " ")
		if valueIds, ok := m.dict[needle]; ok {
			memory[m.attributeId] = append(memory[m.attributeId], valueIds...)
			var matchedTokens []string
			if m.o.keepMatchedTokens {
				matchedTokens = tokens[:i]
			}
			return NewMatchState(true, tokens[i:], matchedTokens, NewMemoryState(memory))
		}
	}

	return NewMatchState(false, tokens, nil, nil)
}

func NewDictMatcher(srcDictionary map[string][]ValueID, attributeId AttributeID, opts ...Option) Matcher {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}
	matcher := &dictMatcher{
		srcDictionary,
		attributeId,
		*o,
		0,
	}
	if !o.calculateNeedleLength {
		return matcher
	}
	maxKeyLength := 0
	for key := range srcDictionary {
		keyLength := countKeyTokens(key)
		if keyLength > maxKeyLength {
			maxKeyLength = keyLength
		}
	}
	matcher.maxKeyLength = maxKeyLength
	return matcher
}

type fullTextMatcher struct {
	nodes []Matcher
}

func (or *fullTextMatcher) Match(state MatchState) MatchState {
	for {
		hasMatch := false
		for _, node := range or.nodes {
			newState := node.Match(Copy(state))
			hasMatch = newState.HasMatch()
			if newState.HasMatch() {
				matchedTokens := make([]string, 0, len(state.MatchedTokens())+len(newState.MatchedTokens()))
				matchedTokens = append(append(matchedTokens, state.MatchedTokens()...), newState.MatchedTokens()...)
				state = NewMatchState(true, newState.RemainingTokens(), matchedTokens, newState.Memory())
				break
			}
		}
		if !hasMatch {
			break
		}
		if len(state.RemainingTokens()) == 0 {
			return state
		}
	}
	return NewMatchState(false, state.RemainingTokens(), nil, nil)
}

func NewFullTextMatcher(matchers []Matcher, _ ...Option) Matcher {
	return &fullTextMatcher{matchers}
}

type oneOfMatcher struct {
	words []Matcher
}

func (o *oneOfMatcher) Match(state MatchState) MatchState {
	for _, matcher := range o.words {
		newState := matcher.Match(Copy(state))
		if newState.HasMatch() {
			return newState
		}
	}
	return NewMatchState(false, state.RemainingTokens(), state.MatchedTokens(), nil)
}

func NewOneOfMatcher(nodes []Matcher) Matcher {
	return &oneOfMatcher{
		nodes,
	}
}

type onceMatcher struct {
	matcher Matcher
	matched bool
}

func (om *onceMatcher) Match(state MatchState) MatchState {
	if om.matched {
		return NewMatchState(false, state.RemainingTokens(), state.MatchedTokens(), nil)
	}
	res := om.matcher.Match(Copy(state))
	if res.HasMatch() {
		om.matched = true
		return res
	}
	return NewMatchState(false, state.RemainingTokens(), state.MatchedTokens(), nil)
}

func NewOnceMatcher(matcher Matcher) Matcher {
	return &onceMatcher{
		matcher,
		false,
	}
}

type anyOrderDictMatcher struct {
	dict         map[string][]ValueID
	attributeId  AttributeID
	maxKeyLength int
	o            options
}

func NewAnyOrderDictMatcher(
	srcDictionary map[string][]ValueID,
	attributeId AttributeID,
	opts ...Option,
) Matcher {
	maxKeyLength := 0
	for key := range srcDictionary {
		keyLength := len(strings.Split(key, " "))
		if keyLength > maxKeyLength {
			maxKeyLength = keyLength
		}
	}

	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	return &anyOrderDictMatcher{
		dict:         srcDictionary,
		attributeId:  attributeId,
		maxKeyLength: maxKeyLength,
		o:            *o,
	}
}

func (m *anyOrderDictMatcher) Match(state MatchState) MatchState {
	tokens := state.RemainingTokens()
	if len(tokens) == 0 {
		return NewMatchState(false, tokens, nil, nil)
	}
	memory := state.Memory().GetStorage()

	maxNeedleLen := min(len(tokens), m.maxKeyLength)
	for length := maxNeedleLen; length >= 1; length-- {
		for offset := 0; offset+length <= len(tokens); offset++ {
			needleTokens := tokens[offset : offset+length]
			needle := strings.Join(needleTokens, " ")

			valueIds, dictContainsNeedle := m.dict[needle]
			if !dictContainsNeedle {
				continue
			}

			memory[m.attributeId] = append(memory[m.attributeId], valueIds...)
			remainingTokens := calculateRemainingTokens(tokens, offset, length)

			var matchedTokens []string
			if m.o.keepMatchedTokens {
				matchedTokens = needleTokens
			}

			return NewMatchState(true, remainingTokens, matchedTokens, NewMemoryState(memory))
		}
	}

	return NewMatchState(false, tokens, nil, nil)
}

func calculateRemainingTokens(tokens []string, matchedOffset, matchedLen int) []string {
	if matchedLen == len(tokens) {
		return nil
	}
	var result []string

	leftPartEndIndex := matchedOffset
	if leftPartEndIndex > 0 {
		result = append(result, tokens[:leftPartEndIndex]...)
	}

	rightPartStartIndex := matchedOffset + matchedLen
	if rightPartStartIndex < len(tokens) {
		result = append(result, tokens[rightPartStartIndex:]...)
	}

	return result
}

type tryAllMatcher struct {
	nodes []Matcher
}

func NewTryAllMatcher(nodes []Matcher) Matcher {
	return &tryAllMatcher{nodes: nodes}
}

func (rr *tryAllMatcher) Match(state MatchState) MatchState {
	hasAnyMatch := false
	for _, node := range rr.nodes {
		newState := node.Match(Copy(state))
		if newState.HasMatch() {
			state = newState
			hasAnyMatch = true
		}
	}

	if hasAnyMatch {
		return state
	}

	return NewMatchState(false, state.RemainingTokens(), nil, state.Memory())
}
