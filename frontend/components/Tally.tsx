import type { CSSProperties } from "react";
import { STATUS } from "@/lib/tape";
import type { Generation } from "@/lib/types";

const LABEL: Record<Generation["status"], string> = {
  pending: "Rendering",
  succeeded: "Done",
  failed: "Failed",
};

/**
 * A generation's status as a tally light: a glowing dot plus a word, in a
 * capsule tinted with the status colour. Pending pulses (the only motion in the
 * app that isn't a response to a click), done and failed sit still. The word is
 * always shown, so colour is never the only cue.
 */
export default function Tally({
  status,
  label,
}: {
  status: Generation["status"];
  /** Replaces the usual word, e.g. "2 rendering". */
  label?: string;
}) {
  return (
    <span className="acmp-tally" style={{ "--tally": STATUS[status] } as CSSProperties}>
      <span
        aria-hidden
        className={status === "pending" ? "acmp-tally-dot is-live" : "acmp-tally-dot"}
      />
      {label ?? LABEL[status]}
    </span>
  );
}
