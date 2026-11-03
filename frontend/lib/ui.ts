import type { CSSProperties } from "react";
import type { ModalProps } from "antd";

/** antd Modal props that keep a fixed-width modal inside a phone viewport. */
export const mobileModal: Pick<ModalProps, "style"> = {
  style: { maxWidth: "calc(100vw - 16px)" },
};

// Page titles are set big and light — a title card, not a heading. Applied
// inline (antd's Typography.Title carries its own selectors, which would
// out-rank a class).
export const pageTitle: CSSProperties = {
  margin: 0,
  fontSize: "clamp(38px, 6vw, 60px)",
  fontWeight: 300,
  letterSpacing: "-0.025em",
  lineHeight: 1.2,
};

/** The sentence under a page title. Pair with Typography.Paragraph type="secondary". */
export const pageLede: CSSProperties = {
  margin: "12px 0 0",
  maxWidth: "52ch",
  fontSize: 16,
};
