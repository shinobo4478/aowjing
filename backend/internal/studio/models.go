package studio

import (
	"math"
	"strconv"
)

// ModelKey names a video model, as stored on a Profile (profiles.video_model).
// Like a style, only the name is stored; what the model means lives below.
type ModelKey string

const (
	// ModelKlingV3 is Kling v3 Standard, the first model used.
	ModelKlingV3 ModelKey = "kling-v3"
	// ModelVeo31 is Google's Veo 3.1 on fal.
	ModelVeo31 ModelKey = "veo-3.1"
	// ModelVeo31Fast is the cheaper, faster Veo 3.1.
	ModelVeo31Fast ModelKey = "veo-3.1-fast"

	// DefaultModel is what a profile gets when none is given.
	DefaultModel = ModelKlingV3
)

// Wire says how fal's inputs are shaped for a model family. The shapes differ
// (field names, how the length is written, how an end picture is given), so the
// request builder in internal/generate switches on it.
type Wire string

const (
	WireKling Wire = "kling"
	WireVeo   Wire = "veo"
)

// VideoModel is everything the app needs to know about one video model.
type VideoModel struct {
	Key   ModelKey
	Label string
	Wire  Wire

	// TextModel makes a clip from a prompt alone ("" = not supported: the clip
	// must start from a picture). ImageModel starts from a picture. Both are fal
	// model ids.
	TextModel  string
	ImageModel string
	// FirstLastModel is the endpoint for a clip that also has an END picture,
	// for models that take it as a separate endpoint (Veo). "" with EndPicture
	// true means the end picture is a field on ImageModel (Kling's
	// end_image_url).
	FirstLastModel string
	// EndPicture reports whether a clip can be given an end picture at all.
	EndPicture bool

	// Seconds lists the clip lengths the model accepts, and DefaultSeconds is
	// what a shot gets when no length is chosen.
	Seconds        []int
	DefaultSeconds int

	// USDPerSecond is the price of a silent clip, per second; USDPerSecondAudio
	// with sound. 0 means not known (the screen then says so rather than guess).
	USDPerSecond      float64
	USDPerSecondAudio float64
	// ConfirmedSilent: a real bill matched USDPerSecond. Kling's did (a 5 s clip
	// cost $0.42). Veo's are fal's listed prices, not yet seen on a bill.
	ConfirmedSilent bool
}

func seq(from, to int) []int {
	out := make([]int, 0, to-from+1)
	for n := from; n <= to; n++ {
		out = append(out, n)
	}
	return out
}

// models is the registry, in the order shown in the UI. To add a model, add a
// ModelKey and an entry here (plus how fal wants its inputs, in internal/
// generate, if it is a new Wire). Prices are fal's list prices read on
// 2026-11-26; they are checked against a bill before they are marked
// confirmed.
var models = []VideoModel{
	{
		Key:   ModelKlingV3,
		Label: "Kling v3 Standard",
		Wire:  WireKling,

		TextModel:  "fal-ai/kling-video/v3/standard/text-to-video",
		ImageModel: "fal-ai/kling-video/v3/standard/image-to-video",
		EndPicture: true,

		Seconds:        seq(MinClipSeconds, MaxClipSeconds),
		DefaultSeconds: DefaultClipSeconds,

		USDPerSecond:      0.084,
		USDPerSecondAudio: 0.126,
		ConfirmedSilent:   true,
	},
	{
		Key:   ModelVeo31,
		Label: "Veo 3.1",
		Wire:  WireVeo,

		TextModel:      "fal-ai/veo3.1",
		ImageModel:     "fal-ai/veo3.1/image-to-video",
		FirstLastModel: "fal-ai/veo3.1/first-last-frame-to-video",
		EndPicture:     true,

		Seconds:        []int{4, 6, 8},
		DefaultSeconds: 4,

		USDPerSecond: 0.20,
		// With-sound price not read yet.
	},
	{
		Key:   ModelVeo31Fast,
		Label: "Veo 3.1 Fast",
		Wire:  WireVeo,

		// Only the image-to-video endpoint of Fast has been read. Its text and
		// first-last-frame endpoints have not, so they are not offered: a clip
		// must start from a picture, and there is no end picture.
		ImageModel: "fal-ai/veo3.1/fast/image-to-video",

		Seconds:        []int{4, 6, 8},
		DefaultSeconds: 4,

		USDPerSecond: 0.10,
	},
}

// Models lists every video model in display order. The slice is a copy.
func Models() []VideoModel { return append([]VideoModel(nil), models...) }

// ValidModel reports whether s names a known video model.
func ValidModel(s string) bool {
	_, ok := lookupModel(ModelKey(s))
	return ok
}

// ModelFor returns the model for a key. An unknown key (which the API never
// stores) falls back to the default, so a clip can always be made.
func ModelFor(s string) VideoModel {
	if m, ok := lookupModel(ModelKey(s)); ok {
		return m
	}
	m, _ := lookupModel(DefaultModel)
	return m
}

func lookupModel(k ModelKey) (VideoModel, bool) {
	for _, m := range models {
		if m.Key == k {
			return m, true
		}
	}
	return VideoModel{}, false
}

// ValidSeconds reports whether n is a length this model accepts.
func (m VideoModel) ValidSeconds(n int) bool {
	for _, s := range m.Seconds {
		if s == n {
			return true
		}
	}
	return false
}

// SupportsText reports whether the model can make a clip from a prompt alone.
func (m VideoModel) SupportsText() bool { return m.TextModel != "" }

// ClipUSD is the price of one image-to-video clip of the given length, with or
// without sound, rounded to the fraction of a cent fal bills in. 0 means the
// price is not known. That the price scales linearly with the length is an
// assumption, only checked at 5 s on Kling so far.
func (m VideoModel) ClipUSD(seconds int, audio bool) float64 {
	per := m.USDPerSecond
	if audio {
		per = m.USDPerSecondAudio
	}
	return math.Round(per*float64(seconds)*10000) / 10000
}

// SecondsList renders the allowed lengths for a message: "4, 6 or 8".
func (m VideoModel) SecondsList() string {
	switch len(m.Seconds) {
	case 0:
		return ""
	case 1:
		return strconv.Itoa(m.Seconds[0])
	}
	s := ""
	for i, n := range m.Seconds[:len(m.Seconds)-1] {
		if i > 0 {
			s += ", "
		}
		s += strconv.Itoa(n)
	}
	return s + " or " + strconv.Itoa(m.Seconds[len(m.Seconds)-1])
}
