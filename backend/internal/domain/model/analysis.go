package model

// Analysis is an immutable DTO snapshot of one engine run (text + config → flags + scores).
// Used as the cache value in CacheRepository (A33).
type Analysis struct {
	textHash         uint64
	configHash       uint64
	flags            []*Flag
	cleanlinessScore Score
	readabilityScore Score
}

// NewAnalysis creates a new Analysis.
func NewAnalysis(textHash, configHash uint64, flags []*Flag, cleanlinessScore, readabilityScore Score) *Analysis {
	return &Analysis{
		textHash:         textHash,
		configHash:       configHash,
		flags:            flags,
		cleanlinessScore: cleanlinessScore,
		readabilityScore: readabilityScore,
	}
}

// TextHash returns the text hash.
func (a *Analysis) TextHash() uint64 { return a.textHash }

// ConfigHash returns the config hash.
func (a *Analysis) ConfigHash() uint64 { return a.configHash }

// Flags returns the detected flags.
func (a *Analysis) Flags() []*Flag { return a.flags }

// CleanlinessScore returns the cleanliness score.
func (a *Analysis) CleanlinessScore() Score { return a.cleanlinessScore }

// ReadabilityScore returns the readability score.
func (a *Analysis) ReadabilityScore() Score { return a.readabilityScore }
