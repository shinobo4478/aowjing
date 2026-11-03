import type { ReactNode } from "react";
import { Typography } from "antd";
import Stripe from "@/components/Stripe";
import { formatDate } from "@/lib/format";
import { tapeColor } from "@/lib/tape";
import { providerShort, type Profile } from "@/lib/types";
import { pageTitle } from "@/lib/ui";

function Field({
  label,
  children,
  wide,
  note,
}: {
  label: string;
  children: ReactNode;
  wide?: boolean;
  note?: boolean;
}) {
  return (
    <div className={wide ? "acmp-slate-field is-wide" : "acmp-slate-field"}>
      <div className="acmp-field-label">{label}</div>
      <div className={note ? "acmp-field-value is-note" : "acmp-field-value"}>
        {children}
      </div>
    </div>
  );
}

/**
 * The profile's header, laid out like a production slate: a clapper stripe in
 * the profile's own tape colour, the name as a title card, then ruled cells of
 * label/value fields. The ruling is real structure — these are fields.
 */
export default function ProfileSlate({
  profile,
  actions,
}: {
  profile: Profile;
  actions: ReactNode;
}) {
  return (
    <div className="acmp-slate">
      <Stripe color={tapeColor(profile.id)} size={9} style={{ height: 16 }} />
      <div className="acmp-slate-head">
        <Typography.Title
          level={1}
          style={{
            ...pageTitle,
            fontSize: "clamp(30px, 5vw, 50px)",
            flex: "1 1 280px",
            minWidth: 0,
            overflowWrap: "anywhere",
          }}
        >
          {profile.name}
        </Typography.Title>
        <div>{actions}</div>
      </div>
      <div className="acmp-slate-fields">
        <Field label="Niche">{profile.niche}</Field>
        <Field label="Generator">{providerShort(profile.provider)}</Field>
        <Field label="Updated">{formatDate(profile.updatedAt)}</Field>
        {profile.description && (
          <Field label="Notes" wide note>
            {profile.description}
          </Field>
        )}
      </div>
    </div>
  );
}
