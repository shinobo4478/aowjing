"use client";

import { useState, type CSSProperties } from "react";
import { Button, Image } from "antd";

/**
 * A picture that opens large when clicked (antd's Image preview: full screen,
 * zoom, close with Esc). The pictures are served by the API with the session
 * cookie, so this is a plain <img> under the hood, not next/image.
 */
export function ZoomableImage({
  src,
  alt,
  width,
  height,
  style,
}: {
  src: string;
  alt: string;
  width: number;
  height: number;
  style?: CSSProperties;
}) {
  return (
    <Image
      src={src}
      alt={alt}
      width={width}
      height={height}
      style={{ objectFit: "cover", borderRadius: 10, display: "block", ...style }}
      preview={{ cover: "ขยาย" }}
    />
  );
}

/**
 * A "ขยาย" button that opens a picture large, for a picture that already has
 * another click action (a candidate the owner picks by clicking it), where
 * making the picture itself zoomable would clash with that.
 */
export function ZoomLink({ src, label = "ขยาย" }: { src: string; label?: string }) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button
        type="link"
        size="small"
        onClick={(e) => {
          e.stopPropagation();
          setOpen(true);
        }}
        style={{ padding: 0, height: "auto", fontSize: 12 }}
      >
        {label}
      </Button>
      {/* The preview opens from this hidden image; it renders in a portal. */}
      <Image
        src={src}
        alt=""
        style={{ display: "none" }}
        preview={{ open, onOpenChange: (v: boolean) => setOpen(v) }}
      />
    </>
  );
}
