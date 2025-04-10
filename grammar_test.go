package context_free_grammar

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func testPositiveParse(t *testing.T, state MatchState) {
	require.Equal(t, true, state.HasMatch())
	require.Equal(t, 0, len(state.RemainingTokens()))
}

func testNegativeParse(t *testing.T, state MatchState) {
	require.Equal(t, false, state.HasMatch())
	require.True(t, len(state.RemainingTokens()) > 0)
}

func testDictParserResult(t *testing.T, state MatchState, expected AttrValues) {
	require.Equal(t, state.Memory().GetStorage(), expected)
}

func getTokens(query string) []string {
	return strings.Split(query, " ")
}

func TestAllowedWord_Match_Positive(t *testing.T) {
	allowedWord := NewAllowedWordMatcher("слово")
	state := NewInitialState(getTokens("слово"))
	res := allowedWord.Match(state)
	testPositiveParse(t, res)
}

func TestAllowedWord_Match_Negative(t *testing.T) {
	allowedWord := NewAllowedWordMatcher("слово")
	state := NewInitialState(getTokens("неслово"))
	res := allowedWord.Match(state)
	testNegativeParse(t, res)
}

func TestAllowedWord_Match_Empty(t *testing.T) {
	allowedWord := NewAllowedWordMatcher("слово")
	state := NewInitialState(getTokens(""))
	res := allowedWord.Match(state)
	testNegativeParse(t, res)
}

func TestDictMatcher_Match_Positive(t *testing.T) {
	sourceDict := make(map[string][]ValueID, 3)
	sourceDict["lorem"] = []ValueID{1}
	sourceDict["ipsum"] = []ValueID{2}
	sourceDict["dolor"] = []ValueID{3}
	dictMatcher := NewDictMatcher(sourceDict, 100500)
	state := NewInitialState(getTokens("dolor"))
	res := dictMatcher.Match(state)
	testPositiveParse(t, res)
	expected := AttrValues{
		100500: {3},
	}
	testDictParserResult(t, res, expected)
}

func TestDictMatcher_Match_Negative(t *testing.T) {
	sourceDict := make(map[string][]ValueID, 3)
	sourceDict["lorem"] = []ValueID{1}
	sourceDict["ipsum"] = []ValueID{2}
	sourceDict["dolor"] = []ValueID{3}
	dictMatcher := NewDictMatcher(sourceDict, 100500)
	state := NewInitialState(getTokens("wrong"))
	res := dictMatcher.Match(state)
	testNegativeParse(t, res)
}

func TestDictMatcher_LongestInput(t *testing.T) {
	sourceDict := make(map[string][]ValueID, 3)
	sourceDict["word"] = []ValueID{1}
	sourceDict["another word"] = []ValueID{2}
	sourceDict["complex key number three"] = []ValueID{3}
	sourceDict["rare key in the end"] = []ValueID{3}
	dictMatcher := NewDictMatcher(sourceDict, 100500, CalculateNeedleLength())
	state := NewInitialState(getTokens("complex key number three or what"))
	res := dictMatcher.Match(state)
	require.True(t, res.HasMatch())
	require.Equal(t, res.RemainingTokens(), []string{"or", "what"})
}

func TestAtLeastOneNode_Match_Positive(t *testing.T) {
	lorem := NewAllowedWordMatcher("lorem")
	ipsum := NewAllowedWordMatcher("ipsum")
	dolor := NewAllowedWordMatcher("dolor")
	atLeastOneNode := NewFullTextMatcher([]Matcher{
		lorem,
		ipsum,
		dolor,
	})
	state := NewInitialState(strings.Split("lorem ipsum dolor", " "))
	res := atLeastOneNode.Match(state)
	testPositiveParse(t, res)
}

func TestAtLeastOneNode_Match_Negative(t *testing.T) {
	lorem := NewAllowedWordMatcher("lorem")
	ipsum := NewAllowedWordMatcher("ipsum")
	dolor := NewAllowedWordMatcher("dolor")
	atLeastOneNode := NewFullTextMatcher([]Matcher{
		lorem,
		ipsum,
		dolor,
	})
	state := NewInitialState(strings.Split("lorem unknown dolor", " "))
	res := atLeastOneNode.Match(state)
	testNegativeParse(t, res)
}

func TestAtLeastOneDictMatcher_Multiple_Positive(t *testing.T) {
	sourceDict := make(map[string][]ValueID, 3)
	sourceDict["lorem"] = []ValueID{1}
	sourceDict["ipsum"] = []ValueID{2}
	sourceDict["dolor"] = []ValueID{3}
	dictMatcher := NewDictMatcher(sourceDict, 100500)
	atLeastOneNode := NewFullTextMatcher([]Matcher{
		dictMatcher,
	})
	state := NewInitialState(getTokens("dolor lorem ipsum"))
	res := atLeastOneNode.Match(state)
	testPositiveParse(t, res)
	expected := AttrValues{
		100500: {3, 1, 2},
	}
	testDictParserResult(t, res, expected)
}

func TestSequence_Positive(t *testing.T) {
	lorem := NewAllowedWordMatcher("lorem")
	ipsum := NewAllowedWordMatcher("ipsum")
	dolor := NewAllowedWordMatcher("dolor")
	sequence := NewSequenceMatcher([]Matcher{
		lorem,
		ipsum,
		dolor,
	})
	state := NewInitialState(strings.Split("lorem ipsum dolor", " "))
	res := sequence.Match(state)
	testPositiveParse(t, res)
}

func TestSequence_Negative(t *testing.T) {
	lorem := NewAllowedWordMatcher("lorem")
	ipsum := NewAllowedWordMatcher("ipsum")
	dolor := NewAllowedWordMatcher("dolor")
	sequence := NewSequenceMatcher([]Matcher{
		lorem,
		ipsum,
		dolor,
	})
	state := NewInitialState(strings.Split("lorem dolor negative", " "))
	res := sequence.Match(state)
	testNegativeParse(t, res)
}

func TestOnceMatcher_Match(t *testing.T) {
	lorem := NewOnceMatcher(NewAllowedWordMatcher("lorem"))
	state := NewInitialState(getTokens("lorem lorem"))
	res := lorem.Match(state)
	require.Equal(t, res.HasMatch(), true)
	require.Equal(t, len(res.RemainingTokens()), 1)
	res = lorem.Match(res)
	testNegativeParse(t, res)
}

func Test_OnceMatcher_Match_Fail(t *testing.T) {
	allowedWordA := NewAllowedWordMatcher("A")
	allowedWordB := NewAllowedWordMatcher("B")
	allowedWordC := NewAllowedWordMatcher("C")
	a := NewOnceMatcher(allowedWordA)
	b := NewOnceMatcher(allowedWordB)
	c := NewOnceMatcher(allowedWordC)

	ftAB := NewFullTextMatcher([]Matcher{a, b})
	ftAC := NewFullTextMatcher([]Matcher{a, c})

	root := NewOneOfMatcher([]Matcher{ftAB, ftAC})

	query := "A C"
	tokens := getTokens(query)
	state := NewInitialState(tokens)

	state = root.Match(state)
	require.False(t, state.HasMatch())
}

func TestAllowedWordsMatcher_Match(t *testing.T) {
	allowedDictMatcher := NewAllowedWordsMatcher([]string{"goes brr", "matcher"})
	sequence := NewSequenceMatcher(
		[]Matcher{
			allowedDictMatcher,
			allowedDictMatcher,
		})
	state := NewInitialState(getTokens("matcher goes brr awesome"))
	res := sequence.Match(state)
	require.Equal(t, res.HasMatch(), true)
	require.Equal(t, 1, len(res.RemainingTokens()))
	res = allowedDictMatcher.Match(res)
	testNegativeParse(t, res)
}

func TestAll(t *testing.T) {
	allowedWord := NewAllowedWordMatcher("allowed")
	lorem := NewAllowedWordMatcher("lorem")
	ipsum := NewAllowedWordMatcher("ipsum")
	dolor := NewAllowedWordMatcher("dolor")
	sequence := NewSequenceMatcher([]Matcher{
		lorem,
		ipsum,
		dolor,
	})
	sourceDict := make(map[string][]ValueID, 3)
	sourceDict["abra"] = []ValueID{1}
	sourceDict["cadabra"] = []ValueID{2}
	dictMatcher := NewDictMatcher(sourceDict, 100500)

	allowedWordsMatcher := NewAllowedWordsMatcher([]string{"awesome", "matcher"})

	root := NewFullTextMatcher([]Matcher{
		allowedWord,
		NewOnceMatcher(NewOneOfMatcher([]Matcher{
			sequence,
			lorem,
		})),
		dictMatcher,
		allowedWordsMatcher,
	})
	state := NewInitialState(getTokens("awesome abra cadabra allowed lorem ipsum dolor matcher"))
	res := root.Match(state)
	testPositiveParse(t, res)
	expected := AttrValues{
		100500: {1, 2},
	}
	testDictParserResult(t, res, expected)
}

func TestEmptyQuery(t *testing.T) {
	allowedWord := NewAllowedWordMatcher("allowed")
	lorem := NewAllowedWordMatcher("lorem")
	ipsum := NewAllowedWordMatcher("ipsum")
	dolor := NewAllowedWordMatcher("dolor")
	sequence := NewSequenceMatcher([]Matcher{
		lorem,
		ipsum,
		dolor,
	})
	sourceDict := make(map[string][]ValueID, 3)
	sourceDict["abra"] = []ValueID{1}
	sourceDict["cadabra"] = []ValueID{2}
	dictMatcher := NewDictMatcher(sourceDict, 100500)

	allowedWordsMatcher := NewAllowedWordsMatcher([]string{"awesome", "matcher"})

	root := NewFullTextMatcher([]Matcher{
		allowedWord,
		NewOnceMatcher(NewOneOfMatcher([]Matcher{
			sequence,
			lorem,
		})),
		dictMatcher,
		allowedWordsMatcher,
	})
	state := NewInitialState(getTokens(""))
	res := root.Match(state)
	testNegativeParse(t, res)
}

func TestKeepMatchedTokens(t *testing.T) {
	sourceDict := make(map[string][]ValueID, 2)
	sourceDict["lorem"] = []ValueID{1}
	sourceDict["ipsum"] = []ValueID{2}
	dictWordsMatcher := NewDictMatcher(sourceDict, 100500, KeepMatchedTokens())

	allowedWordsMatcher := NewAllowedWordsMatcher([]string{"dolor dalar"}, KeepMatchedTokens())

	dictInSequence := make(map[string][]ValueID, 1)
	dictInSequence["this"] = []ValueID{3}
	sequenceWordsMatcher := NewSequenceMatcher([]Matcher{
		NewDictMatcher(
			dictInSequence,
			111555,
			KeepMatchedTokens(),
		),
		NewAllowedWordMatcher("is"),
	}, KeepMatchedTokens())

	rootNode := NewFullTextMatcher([]Matcher{
		sequenceWordsMatcher,
		dictWordsMatcher,
		allowedWordsMatcher,
	})

	queryString := "this is lorem dolor dalar"
	queryTokens := getTokens(queryString)
	state := NewInitialState(queryTokens)

	expectedString := "this lorem"
	expectedMatchedTokens := getTokens(expectedString)
	expectedMatchedTokens = append(expectedMatchedTokens, "dolor dalar")

	state = rootNode.Match(state)
	require.True(t, state.HasMatch())
	require.Equal(t, expectedMatchedTokens, state.MatchedTokens())
}

func Test_AnyOrderDictMatcher_Match(t *testing.T) {
	matcher := NewAnyOrderDictMatcher(
		map[string][]ValueID{
			"1к":            {1},
			"однокомнатная": {1},

			"2к":            {2},
			"двухкомнатная": {2},

			"3 комнатная": {3},

			"2 или 3 комнатная": {2, 3},
		},
		1,
		KeepMatchedTokens(),
	)

	tests := []struct {
		name                    string
		state                   MatchState
		hasMatch                bool
		expectedParams          AttrValues
		expectedRemainingTokens []string
		expectedMatchedTokens   []string
	}{
		{
			name:     "Should match value 1 by `1к` when its in 1st position",
			state:    NewInitialState([]string{"1к", "квартира", "купить"}),
			hasMatch: true,
			expectedParams: AttrValues{
				1: {1},
			},
			expectedRemainingTokens: []string{"квартира", "купить"},
			expectedMatchedTokens:   []string{"1к"},
		},
		{
			name:     "Should match value 1 by `1к` when its in 2nd position",
			state:    NewInitialState([]string{"квартира", "1к", "купить"}),
			hasMatch: true,
			expectedParams: AttrValues{
				1: {1},
			},
			expectedRemainingTokens: []string{"квартира", "купить"},
			expectedMatchedTokens:   []string{"1к"},
		},
		{
			name:     "Should match value 1 by `1к` when its in 3rd position",
			state:    NewInitialState([]string{"квартира", "купить", "1к"}),
			hasMatch: true,
			expectedParams: AttrValues{
				1: {1},
			},
			expectedRemainingTokens: []string{"квартира", "купить"},
			expectedMatchedTokens:   []string{"1к"},
		},
		{
			name:     "Should match value 1 by `однокомнатная`",
			state:    NewInitialState([]string{"снять", "однокомнатная", "квартира"}),
			hasMatch: true,
			expectedParams: AttrValues{
				1: {1},
			},
			expectedRemainingTokens: []string{"снять", "квартира"},
			expectedMatchedTokens:   []string{"однокомнатная"},
		},
		{
			name:     "Should match full query",
			state:    NewInitialState([]string{"2", "или", "3", "комнатная"}),
			hasMatch: true,
			expectedParams: AttrValues{
				1: {2, 3},
			},
			expectedRemainingTokens: nil,
			expectedMatchedTokens:   []string{"2", "или", "3", "комнатная"},
		},
		{
			name:     "Should match long dict key",
			state:    NewInitialState([]string{"снять", "2", "или", "3", "комнатная", "у", "моря"}),
			hasMatch: true,
			expectedParams: AttrValues{
				1: {2, 3},
			},
			expectedRemainingTokens: []string{"снять", "у", "моря"},
			expectedMatchedTokens:   []string{"2", "или", "3", "комнатная"},
		},
		{
			name:     "Should match two word dict key",
			state:    NewInitialState([]string{"снять", "3", "комнатная", "у", "моря"}),
			hasMatch: true,
			expectedParams: AttrValues{
				1: {3},
			},
			expectedRemainingTokens: []string{"снять", "у", "моря"},
			expectedMatchedTokens:   []string{"3", "комнатная"},
		},
		{
			name: "Should save already matched attributes when matched",
			state: NewMatchState(
				true,
				[]string{"снять", "1к", "квартиру"},
				nil,
				NewMemoryState(AttrValues{
					100: {1000},
					200: {2000, 2001},
				}),
			),
			hasMatch: true,
			expectedParams: AttrValues{
				1:   {1},
				100: {1000},
				200: {2000, 2001},
			},
			expectedRemainingTokens: []string{"снять", "квартиру"},
			expectedMatchedTokens:   []string{"1к"},
		},
		{
			name:                    "Should not match when any matching key absent in dict",
			state:                   NewInitialState([]string{"дом"}),
			hasMatch:                false,
			expectedParams:          nil,
			expectedRemainingTokens: []string{"дом"},
			expectedMatchedTokens:   nil,
		},

		{
			name:     "Should firstly match long token keys",
			state:    NewInitialState([]string{"снять", "2", "или", "3", "комнатная", "у", "моря", "2к"}),
			hasMatch: true,
			expectedParams: AttrValues{
				1: {2, 3},
			},
			expectedRemainingTokens: []string{"снять", "у", "моря", "2к"},
			expectedMatchedTokens:   []string{"2", "или", "3", "комнатная"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultState := matcher.Match(tt.state)
			if resultState.HasMatch() != tt.hasMatch {
				t.Errorf("State.HasMatch() = %v, want %v", resultState.HasMatch(), tt.hasMatch)
			}

			if tt.expectedParams != nil {
				if !reflect.DeepEqual(resultState.Memory().GetStorage(), tt.expectedParams) {
					t.Errorf("State.Memory().GetStorage() = %v, want %v", resultState.Memory().GetStorage(), tt.expectedParams)
				}
			} else {
				if resultState.Memory() != nil {
					t.Errorf("State.Memory().GetStorage() = %v, want %v", resultState.Memory().GetStorage(), tt.expectedParams)
				}
			}

			if !reflect.DeepEqual(resultState.RemainingTokens(), tt.expectedRemainingTokens) {
				t.Errorf("State.RemainingTokens() = %v, want %v", resultState.RemainingTokens(), tt.expectedRemainingTokens)
			}

			if !reflect.DeepEqual(resultState.MatchedTokens(), tt.expectedMatchedTokens) {
				t.Errorf("State.RemainingTokens() = %v, want %v", resultState.MatchedTokens(), tt.expectedMatchedTokens)
			}
		})
	}
}

func Test_tryAllMatcher_Match(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		hasMatch       bool
		expectedParams AttrValues
	}{
		{
			name:     "Should match `снять квартиру посуточно`",
			query:    "снять квартиру посуточно",
			hasMatch: true,
			expectedParams: AttrValues{
				1: {1},
				2: {2},
			},
		},
		{
			name:     "Should match `купить квартиру вторичка`",
			query:    "купить квартиру вторичка",
			hasMatch: true,
			expectedParams: AttrValues{
				3: {3},
				4: {4},
			},
		},
		{
			name:           "Should NOT match `снять квартиру вторичка`",
			query:          "снять квартиру вторичка",
			hasMatch:       false,
			expectedParams: nil,
		},
		{
			name:           "Should NOT match `купить квартиру посуточно`",
			query:          "купить квартиру посуточно",
			hasMatch:       false,
			expectedParams: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rentMatcher := NewTryAllMatcher(
				[]Matcher{
					NewAnyOrderDictMatcher(
						map[string][]ValueID{
							"снять": {1},
						},
						1,
					),
					NewAnyOrderDictMatcher(
						map[string][]ValueID{
							"посуточно": {2},
						},
						2,
					),
				},
			)

			buyMatcher := NewTryAllMatcher(
				[]Matcher{
					NewAnyOrderDictMatcher(
						map[string][]ValueID{
							"купить": {3},
						},
						3,
					),
					NewAnyOrderDictMatcher(
						map[string][]ValueID{
							"вторичка": {4},
						},
						4,
					),
				},
			)

			oneOfMatcher := NewOnceMatcher(
				NewOneOfMatcher(
					[]Matcher{
						rentMatcher,
						buyMatcher,
					},
				),
			)

			matcher := NewFullTextMatcher([]Matcher{
				NewAllowedWordMatcher("квартиру"),
				oneOfMatcher,
			})

			initialState := NewInitialState(strings.Fields(tt.query))
			resultState := matcher.Match(initialState)
			if resultState.HasMatch() != tt.hasMatch {
				t.Errorf("Match() = %v, expected has match %v", resultState.HasMatch(), tt.hasMatch)
			} else {
				var actualMem AttrValues = nil
				if resultState.Memory() != nil {
					actualMem = resultState.Memory().GetStorage()
				}
				if !reflect.DeepEqual(actualMem, tt.expectedParams) {
					t.Errorf("Match() = %v, expectedRemainingTokens %v", actualMem, tt.expectedParams)
				}
			}
		})
	}
}

