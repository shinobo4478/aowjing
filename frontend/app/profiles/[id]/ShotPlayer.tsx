"use client";

import { useRef } from "react";
import { Typography } from "antd";

/**
 * Plays a shot's clip together with its narration, so the owner can judge the
 * shot as it will sound before joining anything in the editor.
 *
 * The clip's own `<video>` keeps its normal controls and is the master: the
 * narration `<audio>` (hidden) follows it — it starts, pauses and seeks with the
 * video. The clip is silent anyway (or, when the profile has sound on, carries
 * Kling/Veo audio that is not the narration), so the clip is muted while a
 * narration plays.
 *
 * Drift is not tracked after the start: the two are short (a few seconds) and
 * both are re-aligned at every play and seek.
 */
export default function ShotPlayer({
  videoSrc,
  voiceSrc,
  voiceCurrent,
}: {
  videoSrc: string;
  /** The narration audio; undefined when the shot has none (then it is just the clip). */
  voiceSrc?: string;
  /** False when the narration text changed after this audio was made. */
  voiceCurrent?: boolean;
}) {
  const video = useRef<HTMLVideoElement>(null);
  const audio = useRef<HTMLAudioElement>(null);

  /** Put the narration at the clip's position (it is a no-op without audio). */
  function align() {
    const v = video.current;
    const a = audio.current;
    if (!v || !a) return;
    // Only touch the audio when it is more than a frame or two off: setting
    // currentTime on every event makes it stutter.
    if (Math.abs(a.currentTime - v.currentTime) > 0.05) a.currentTime = v.currentTime;
  }

  function onPlay() {
    const a = audio.current;
    if (!a) return;
    align();
    // A play() the browser refuses (no user gesture) is not worth a message: the
    // clip still plays and the owner can press play on the voice by hand.
    void a.play().catch(() => undefined);
  }

  function onPause() {
    audio.current?.pause();
  }

  return (
    <div>
      <video
        ref={video}
        controls
        preload="metadata"
        src={videoSrc}
        muted={voiceSrc !== undefined}
        onPlay={onPlay}
        onPause={onPause}
        onSeeked={align}
        onWaiting={onPause}
        onPlaying={onPlay}
        style={{ width: "100%", maxHeight: 360, borderRadius: 10, background: "#000" }}
      />
      {voiceSrc !== undefined && (
        <>
          {/* Hidden: it follows the video above. preload=auto so it is ready when
              play is pressed. */}
          <audio ref={audio} src={voiceSrc} preload="auto" />
          <Typography.Text
            type={voiceCurrent === false ? "warning" : "secondary"}
            style={{ display: "block", fontSize: 12, marginTop: 4 }}
          >
            {voiceCurrent === false
              ? "▶ เล่นพร้อมเสียงพากย์ · บทเปลี่ยนไปหลังสร้างเสียงนี้ เสียงที่ได้ยินเป็นบทเก่า"
              : "▶ เล่นพร้อมเสียงพากย์ (เสียงของคลิปถูกปิดไว้)"}
          </Typography.Text>
        </>
      )}
    </div>
  );
}
