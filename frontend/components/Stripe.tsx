import type { CSSProperties } from "react";

/**
 * The clapper-stick stripe: diagonal bands of a colour and black. It is the
 * app's one recurring mark — chalk-on-black as the brand, a profile's tape
 * colour on that profile's slate and roster spine, tungsten on the login.
 *
 * `size` is the width of one band in px; the pattern repeats every 2×size.
 */
export default function Stripe({
  color,
  size = 12,
  style,
}: {
  color: string;
  size?: number;
  style?: CSSProperties;
}) {
  return (
    <div
      aria-hidden
      className="acmp-stripe"
      style={
        {
          "--tape": color,
          "--band": `${size}px`,
          ...style,
        } as CSSProperties
      }
    />
  );
}
