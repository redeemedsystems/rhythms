package domain

type HabitType string

const (
	YesNo     HabitType = "YES_NO"
	Numerical HabitType = "NUMERICAL"
)

type TargetType string

const (
	AtLeast TargetType = "AT_LEAST"
	AtMost  TargetType = "AT_MOST"
)

// Frequency represents "N times per D days", e.g. daily=1/1, weekly=1/7,
// 3x/week=3/7. M1 only ever uses DailyFrequency; full support lands in M2.
type Frequency struct {
	Numerator   int
	Denominator int
}

func DailyFrequency() Frequency { return Frequency{Numerator: 1, Denominator: 1} }

func (f Frequency) IsDaily() bool { return f.Numerator == f.Denominator }

type Habit struct {
	ID          int64
	UUID        string
	Name        string
	Question    string
	Description string
	Color       int // index into Palette
	Position    int
	Archived    bool
	Type        HabitType
	Unit        string
	TargetValue float64
	TargetType  TargetType
	Freq        Frequency
}

// Palette is a fixed set of colors habits can be tagged with in the UI.
//
// TODO(M4): swap for uHabits' exact upstream hex values before CSV export
// needs to reproduce the original app's per-color output byte-for-byte.
var Palette = [20]string{
	"#D32F2F", "#E64A19", "#F57C00", "#FFA000", "#FBC02D",
	"#AFB42B", "#7CB342", "#388E3C", "#00897B", "#00ACC1",
	"#0097A7", "#039BE5", "#1976D2", "#3949AB", "#5E35B1",
	"#8E24AA", "#D81B60", "#6D4C41", "#546E7A", "#AAAAAA",
}
