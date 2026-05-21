package impl

type Console struct{}

func (Console) Speak() string {
	return "console"
}

// PartialSpeaker exercises the negative implements case and method key collision.
type PartialSpeaker struct{}

// Speak with extra args — does not satisfy the Speaker interface,
// but the method name collides with Console.Speak. Exercises the
// receiver-aware objIndex key.
func (PartialSpeaker) Speak(volume int) string { return "psst" }

// ExerciseSpeakers calls both Speak methods to ensure the parser
// resolves them to distinct node IDs.
func ExerciseSpeakers() string {
	c := Console{}
	p := PartialSpeaker{}
	return c.Speak() + p.Speak(1)
}
