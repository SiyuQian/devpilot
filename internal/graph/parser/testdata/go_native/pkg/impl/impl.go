package impl

type Console struct{}

func (Console) Speak() string {
	return "console"
}

// PartialSpeaker exercises the negative implements case: missing Speak().
type PartialSpeaker struct{}

// Whisper is not part of the Speaker interface, so PartialSpeaker does not
// implement Speaker.
func (PartialSpeaker) Whisper() string { return "psst" }
