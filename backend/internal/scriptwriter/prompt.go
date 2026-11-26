// Package scriptwriter drafts a video's shot list with an LLM: from a topic it
// writes the scene, and for each shot what the starting picture shows and how
// the clip moves. The result fills in a new Video for the owner to review; it
// makes no pictures and no clips, so nothing but the draft itself is spent.
//
// The prompts here carry the rules this studio has learned the hard way (see
// CLAUDE.md, Phase 4): a clip prompt describes motion only, the pictures of one
// video must share a scene, and AI pictures should hold no text.
package scriptwriter

import (
	"fmt"
	"strings"
)

// systemPrompt is the standing instruction. It asks for JSON only and shows one
// worked example (the owner's cat-and-bag video, whose prompts gave good clips),
// because a model follows a shown format far better than a described one.
const systemPrompt = `You write shot lists for short vertical (9:16) videos that are made from still pictures: every shot is ONE picture that an AI turns into a clip of a few seconds. You reply with JSON only: no prose, no markdown, no code fences.

The JSON has this shape:
{"title": string, "scene": string, "shots": [{"picturePrompt": string, "motionPrompt": string}]}

Rules:
- "title": a short working title for the video.
- "scene": one or two sentences that describe the single place where ALL shots happen: the room or setting, where the main furniture or objects stand, the lighting and time of day. It is added in front of every picture prompt so the shots match, so make it concrete and visual.
- "picturePrompt": what the still picture of that shot shows at its START: who is in it, what they are doing, the camera framing (close-up, medium, wide), the mood. Do not repeat the scene description. Do not mention any art style or rendering style (the studio adds the style itself). No text, signs, logos or readable writing anywhere in the picture.
- "motionPrompt": ONLY what moves and how the camera moves during the clip: the action, in order, and how it ends. Never describe appearance (the picture already fixes it). One clear action per shot; at most two beats for a clip of 4 to 5 seconds. Use a static camera unless the brief asks for camera movement, and then end the prompt with "Static camera, no zoom."
- Refer to the main character with the same simple noun in every shot (for example "the cat") and never describe how it looks: reference pictures fix that.
- Shot 1 is a hook that makes a viewer stop scrolling within a second or two. The middle shots pay it off. The last shot closes the story, and may lead back into shot 1 so the video loops.
- Everything is written in English, whatever language the brief is in.
- Nothing dangerous or that could be imitated harmfully (for example never a bag over an animal's head).

Example of a valid reply for a 3-shot video about a cat and a plastic bag:
{"title":"The bag drop","scene":"A small sunny living room: a wooden bookshelf on the left, a beige armchair at the back, a large shaggy rug on a wooden floor, warm afternoon light from a window on the right.","shots":[{"picturePrompt":"Extreme close-up of the cat's face, wide shocked eyes, a white plastic bag floating down right in front of its nose, shallow depth of field.","motionPrompt":"The plastic bag drops slowly in front of the cat's nose and the cat freezes with wide eyes, its ears snapping forward, ending with the bag resting just in front of its nose. Static camera, no zoom."},{"picturePrompt":"The cat crouched low in the middle of the rug, rear end up and tail swishing, eyes locked on a crumpled plastic bag just in front of it, low-angle shot.","motionPrompt":"The cat wiggles its rear, pounces on the bag, tumbles into a clumsy somersault and bounces back up, the bag landing beside it. The cat stays in the same spot. Static camera, no zoom."},{"picturePrompt":"The cat sitting upright on the rug looking straight at the camera with a smug expression, one plastic bag handle hooked over its ear, another bag just starting to fall at the top of the frame.","motionPrompt":"The cat blinks slowly at the camera, smug and content, while another plastic bag drifts down from above toward its nose. Static camera, slight slow zoom in."}]}`

// userPrompt is the task for one video: the topic, the shape asked for, and what
// the profile says about itself so the shots fit its channel.
func userPrompt(b Brief) string {
	var s strings.Builder
	s.WriteString("Write the shot list for this video.\n\n")
	fmt.Fprintf(&s, "Topic (the owner's words, any language): %s\n", strings.TrimSpace(b.Topic))
	fmt.Fprintf(&s, "Number of shots: %d\n", b.Shots)
	fmt.Fprintf(&s, "Each shot is about %d seconds long, so give each only as much action as fits.\n", b.Seconds)
	if b.ProfileName != "" || b.Niche != "" || b.Description != "" {
		s.WriteString("\nThe channel this is for:\n")
		if b.ProfileName != "" {
			fmt.Fprintf(&s, "- Name: %s\n", b.ProfileName)
		}
		if b.Niche != "" {
			fmt.Fprintf(&s, "- Niche: %s\n", b.Niche)
		}
		if b.Description != "" {
			fmt.Fprintf(&s, "- Notes (tone, audience, do and don't): %s\n", b.Description)
		}
	}
	s.WriteString("\nReply with the JSON only.")
	return s.String()
}
