package model

import "github.com/oklog/ulid/v2"

// NewSessionID creates a unique identifier for a live editing session.
func NewSessionID() SessionID {
	return SessionID{value: ulid.Make().String()}
}

// Session — aggregate root, in-memory live editing session (A33).
type Session struct {
	id               SessionID
	text             *Text
	configHash       ConfigHash
	flags            map[FlagID]*Flag
	cleanlinessScore Score
	readabilityScore Score
	wordCount        WordCount
}

// NewSession from raw values.
func NewSession(id string, text *Text, configHash uint64) (*Session, error) {
	sid, err := SessionIDFromString(id)
	if err != nil {
		return nil, err
	}
	return &Session{
		id: sid, text: text, configHash: ConfigHashFromUint64(configHash),
		flags:            make(map[FlagID]*Flag),
		cleanlinessScore: NewScoreUnsafe(MaxScore),
		readabilityScore: NewScoreUnsafe(MaxScore),
		wordCount:        WordCountFromInt(text.WordCount()),
	}, nil
}

func (s *Session) ID() SessionID           { return s.id }
func (s *Session) Text() *Text             { return s.text }
func (s *Session) ConfigHash() ConfigHash  { return s.configHash }
func (s *Session) CleanlinessScore() Score { return s.cleanlinessScore }
func (s *Session) ReadabilityScore() Score { return s.readabilityScore }
func (s *Session) WordCount() WordCount    { return s.wordCount }

func (s *Session) Flags() []*Flag {
	out := make([]*Flag, 0, len(s.flags))
	for _, f := range s.flags {
		out = append(out, f)
	}
	return out
}

func (s *Session) ActiveFlags() []*Flag {
	var out []*Flag
	for _, f := range s.flags {
		if f.IsPending() || f.State() == FlagStateAccepted {
			out = append(out, f)
		}
	}
	return out
}

func (s *Session) AddFlag(f *Flag)      { s.flags[f.ID()] = f }
func (s *Session) SetText(t *Text)      { s.text = t; s.wordCount = WordCountFromInt(t.WordCount()) }
func (s *Session) SetScores(c, r Score) { s.cleanlinessScore, s.readabilityScore = c, r }
func (s *Session) ReplaceFlags(flags []*Flag) {
	s.flags = make(map[FlagID]*Flag, len(flags))
	for _, f := range flags {
		s.flags[f.ID()] = f
	}
}

func (s *Session) AcceptFlag(id FlagID) error {
	f, ok := s.flags[id]
	if !ok {
		return &DomainError{Op: "Session.AcceptFlag", Message: "flag not found"}
	}
	return f.Accept()
}

func (s *Session) RejectFlag(id FlagID) error {
	f, ok := s.flags[id]
	if !ok {
		return &DomainError{Op: "Session.RejectFlag", Message: "flag not found"}
	}
	return f.Reject()
}

func (s *Session) ApplyFlag(id FlagID) (Suggestion, error) {
	f, ok := s.flags[id]
	if !ok {
		return Suggestion{}, &DomainError{Op: "Session.ApplyFlag", Message: "flag not found"}
	}
	if err := f.Apply(); err != nil {
		return Suggestion{}, err
	}
	return f.Suggestion(), nil
}
