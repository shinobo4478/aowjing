import { STATUS } from "@/lib/tape";
import type { Generation } from "@/lib/types";

const LABEL: Record<Generation["status"], string> = {
  pending: "Rendering",
  succeeded: "Done",
  failed: "Failed",
};

/**
 * A generation's status as a tally light: a dot plus a word. Pending pulses
 * (the only motion in the app that isn't a response to a click), done and
 * failed sit still. The word is always shown, so colour is never the only cue.
 */
export default function Tally({ status }: { status: Generation["status"] }) {
  return (
    <span className="acmp-tally">
      <span
        aria-hidden
        className={status === "pending" ? "acmp-tally-dot is-live" : "acmp-tally-dot"}
        style={{ background: STATUS[status] }}
      />
      {LABEL[status]}
    </span>
  );
}
