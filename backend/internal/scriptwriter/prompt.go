// Package scriptwriter drafts a video's shot list with an LLM: from a topic it
// writes the scene, and for each shot what the starting picture shows, how the
// clip moves and, where the action carries straight on into the next shot, what
// changes by the end picture. The result fills in a new Video for the owner to
// review; it makes no pictures and no clips, so nothing but the draft itself is
// spent.
//
// The prompts here carry the rules this studio has learned the hard way (see
// CLAUDE.md, Phase 4): a clip prompt describes motion only, the pictures of one
// video must share a scene, AI pictures should hold no text, and a clip morphs
// whatever differs between its start and end pictures, so an end picture may
// change only a pose.
package scriptwriter

import (
	"fmt"
	"strings"

	"github.com/shinobo4478/aowjing/backend/internal/studio"
)

// systemPrompt is the standing instruction. It asks for JSON only and shows one
// worked example (the owner's cat-and-bag video, whose prompts gave good clips),
// because a model follows a shown format far better than a described one.
const systemPrompt = `You write shot lists for short vertical (9:16) videos that are made from still pictures: every shot is ONE picture that an AI turns into a clip of a few seconds. You reply with JSON only: no prose, no markdown, no code fences.

The JSON has this shape:
{"title": string, "scene": string, "sceneTh": string, "shots": [{"picturePrompt": string, "picturePromptTh": string, "endPicturePrompt": string, "endPicturePromptTh": string, "motionPrompt": string, "motionPromptTh": string}]}

Rules:
- "title": a short working title for the video.
- "scene": one or two sentences that describe the single place where ALL shots happen: the room or setting, where the main furniture or objects stand, the lighting and time of day. It is added in front of every picture prompt so the shots match, so make it concrete and visual. Describe the PLACE only: never a camera framing or shot size (no "wide view", "close-up", "medium shot", "showing the full room") and never a style phrase ("in the style of ..."), because each shot chooses its own framing.
- "picturePrompt": what the still picture of that shot shows at its START: who is in it, what they are doing, the camera framing (close-up, medium, wide), the mood. Do not repeat the scene description. Never use a style word (no "3D", "cartoon", "animated", "rendered", "Pixar", "illustration", "realistic", "photo" or similar): the studio adds the style itself. Never describe how the character looks (breed, colours, face shape, fur): reference pictures fix that. No text, signs, logos or readable writing anywhere in the picture.
  - The picture is the moment BEFORE the clip's action starts. If the motion raises a bottle to drink, the picture shows the bottle held below the mouth, not already drinking; if the motion opens a door, the picture shows it closed. A picture that already shows the action makes the clip ignore the motion prompt.
  - Never write a negative ("no", "not", "nothing", "without", "empty", "absent"): image models draw what is named. To leave something out, frame it out ("framed from the chest up") or simply do not mention it. State objects as they are, with their count when it matters ("three empty bottles stand on the floor").
  - Refer to a person only as a role in the story, never by a job they are not doing in this shot (a character who is only sitting is not "a badminton player").
- "motionPrompt": ONLY what moves and how the camera moves during the clip: the action, in order, and how it ends. Never describe appearance (the picture already fixes it). One clear action per shot; at most two beats for a clip of 4 to 5 seconds. Name people and things with plain nouns ("the person", "the player", "the cat", "the bottle") and never with a role word such as "hero", "costar", "lead character" or "main character": the clip model sees only the starting picture and this text. Do not mention things that merely stand in the picture; write only what moves. Show a feeling as a concrete visible action (a small smile, a slow exhale), not as a mood word.
  - End the prompt with ONE camera sentence. By default it is "Static camera, no zoom." Only if the brief asks for camera movement, give exactly one slow movement instead (for example "Very slow, slight pull-back; nothing else about the camera changes.") and never combine "static camera" with a movement. A camera move must not reveal anything outside the starting picture (the model would invent it): to reveal more of a place, make the next shot a wider picture and cut to it.
- "endPicturePrompt": the clip is made to flow from its starting picture to an END picture, and it morphs ANYTHING that differs between the two, so an end picture may change a pose or position and nothing else.
  - Fill it ONLY when this shot's action carries straight on into the NEXT shot (same place, same framing, the moment continues). Write ONLY what changes between the first picture and the last frame of this shot: the pose or position of the character, or the state or position of one object. Start with "Only the ...". Keep the change small enough to happen within the shot's few seconds. Never change the camera, the room or the lighting.
  - When you fill it, the next shot STARTS from that end picture, so the next shot's "picturePrompt" must describe exactly the state the end picture shows, with the same framing, and its "motionPrompt" must begin from that state.
  - When the next shot cuts to a new angle, a new moment or a different framing, set "endPicturePrompt" to "" (an empty string). Mix both: not every shot should chain. The last shot always has "".
- Refer to the main character with the same simple noun in every shot (for example "the cat").
- Avoid big jumps of framing between neighbouring shots (an extreme close-up followed by a wide shot) unless the jump is a deliberate cut; shots that chain must keep the same framing.
- Shot 1 is a hook that makes a viewer stop scrolling within a second or two. The middle shots pay it off. The last shot closes the story, and may lead back into shot 1 so the video loops.
- Every English text has a Thai twin in the field whose name ends in "Th": "sceneTh" for "scene", "picturePromptTh" for "picturePrompt", "endPicturePromptTh" for "endPicturePrompt", "motionPromptTh" for "motionPrompt". A Thai twin is a faithful, natural Thai rendering that says exactly what its English text says, nothing added and nothing left out: the owner reads Thai much better than English and will read and edit the Thai, and the English is what the picture and video models receive. Where an "endPicturePrompt" is "", its "endPicturePromptTh" is "" too.
- Everything except the fields ending in "Th" is written in English, whatever language the brief is in; the "Th" fields are in Thai.
- Nothing dangerous or that could be imitated harmfully (for example never a bag over an animal's head).

Example of a valid reply for a 3-shot video about a cat and a plastic bag. Shot 1 chains into shot 2 (the end picture is the state shot 2 starts from); shot 2 cuts to shot 3, so its endPicturePrompt is empty:
{"title":"The bag drop","scene":"A small sunny living room: a wooden bookshelf on the left, a beige armchair at the back, a large shaggy rug on a wooden floor, warm afternoon light from a window on the right.","sceneTh":"ห้องนั่งเล่นเล็กๆ ที่แดดส่อง: ชั้นหนังสือไม้อยู่ทางซ้าย เก้าอี้เท้าแขนสีเบจอยู่ด้านหลัง พรมขนฟูผืนใหญ่บนพื้นไม้ และแสงบ่ายอุ่นๆ จากหน้าต่างทางขวา","shots":[{"picturePrompt":"Medium shot of the cat sitting upright in the middle of the rug looking at the camera, a white plastic bag floating down toward its nose.","picturePromptTh":"ภาพระยะกลางของแมวที่นั่งตัวตรงกลางพรมมองกล้อง มีถุงพลาสติกสีขาวลอยลงมาทางจมูก","endPicturePrompt":"Only the cat's pose and the bag change: the cat is frozen low with wide eyes and its ears pointing forward, and the white plastic bag now rests on the rug right in front of its nose.","endPicturePromptTh":"เปลี่ยนเฉพาะท่าของแมวกับถุง: แมวนิ่งก้มตัวต่ำ ตาโต หูชี้ไปข้างหน้า และถุงพลาสติกสีขาววางอยู่บนพรมตรงหน้าจมูก","motionPrompt":"The plastic bag drops slowly in front of the cat's nose and the cat freezes with wide eyes, its ears snapping forward, ending crouched low with the bag resting just in front of its nose. Static camera, no zoom.","motionPromptTh":"ถุงพลาสติกค่อยๆ ตกลงตรงหน้าจมูกแมว แมวนิ่งตาโต หูกระดกไปข้างหน้า จบด้วยท่าก้มตัวต่ำโดยมีถุงวางอยู่หน้าจมูก กล้องนิ่ง ไม่ซูม"},{"picturePrompt":"Medium shot of the cat frozen low in the middle of the rug with wide eyes and ears pointing forward, a white plastic bag resting on the rug right in front of its nose.","picturePromptTh":"ภาพระยะกลางของแมวที่นิ่งก้มตัวต่ำกลางพรม ตาโต หูชี้ไปข้างหน้า มีถุงพลาสติกสีขาววางบนพรมตรงหน้าจมูก","endPicturePrompt":"","endPicturePromptTh":"","motionPrompt":"The cat wiggles its rear, pounces on the bag, tumbles into a clumsy somersault and bounces back up, the bag landing beside it. The cat stays in the same spot. Static camera, no zoom.","motionPromptTh":"แมวส่ายก้น กระโจนใส่ถุง ตีลังกาอย่างงุ่มง่ามแล้วเด้งลุกขึ้น ถุงตกลงข้างตัว แมวอยู่ที่เดิม กล้องนิ่ง ไม่ซูม"},{"picturePrompt":"The cat sitting upright on the rug looking straight at the camera with a smug expression, one plastic bag handle hooked over its ear, another bag just starting to fall at the top of the frame.","picturePromptTh":"แมวนั่งตัวตรงบนพรมมองตรงมาที่กล้องด้วยสีหน้าภูมิใจ มีหูหิ้วถุงใบหนึ่งคล้องหูอยู่ และมีถุงอีกใบเริ่มตกลงมาที่ด้านบนของภาพ","endPicturePrompt":"","endPicturePromptTh":"","motionPrompt":"The cat blinks slowly at the camera, smug and content, while another plastic bag drifts down from above toward its nose. Slow, slight zoom in; nothing else about the camera changes.","motionPromptTh":"แมวกะพริบตาช้าๆ ใส่กล้องอย่างภูมิใจและพอใจ ขณะที่ถุงอีกใบลอยลงมาจากด้านบนทางจมูก ซูมเข้าช้าๆ เล็กน้อย ไม่มีการเคลื่อนกล้องแบบอื่น"}]}`

// userPrompt is the task for one video: the topic, the shape asked for, and what
// the profile says about itself so the shots fit its channel.
func userPrompt(b Brief) string {
	var s strings.Builder
	s.WriteString("Write the shot list for this video.\n\n")
	fmt.Fprintf(&s, "Topic (the owner's words, any language): %s\n", strings.TrimSpace(b.Topic))
	fmt.Fprintf(&s, "Number of shots: %d\n", b.Shots)
	if b.Mode == ModeExplainer {
		// The narration sets the length of a shot, so the brief gives a ceiling
		// and the character budget that fits it.
		fmt.Fprintf(&s, "Each shot is at most %d seconds long. Its narrationTh has at most %d characters (spaces not counted); a shorter line gives a shorter shot, which is fine.\n",
			b.Seconds, studio.NarrationBudget(b.Seconds))
	} else {
		fmt.Fprintf(&s, "Each shot is about %d seconds long, so give each only as much action as fits.\n", b.Seconds)
	}
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
	s.WriteString(userCastBlock(b.Cast))
	s.WriteString(userProductBlock(b.Product))
	s.WriteString("\nReply with the JSON only.")
	return s.String()
}
