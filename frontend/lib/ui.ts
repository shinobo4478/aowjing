import type { ModalProps } from "antd";

/** antd Modal props that keep a fixed-width modal inside a phone viewport. */
export const mobileModal: Pick<ModalProps, "style"> = {
  style: { maxWidth: "calc(100vw - 16px)" },
};
