"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { App, Button, Modal, Spin, Typography } from "antd";
import Stripe from "@/components/Stripe";
import { createProfile, listProfiles } from "@/lib/api";
import { formatDate } from "@/lib/format";
import { tapeColor } from "@/lib/tape";
import { providerShort, type Profile } from "@/lib/types";
import { mobileModal, pageLede, pageTitle } from "@/lib/ui";
import ProfileForm from "./ProfileForm";

export default function ProfilesPage() {
  const { message } = App.useApp();
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);

  // Load once on mount. `loading` already starts true, so no synchronous
  // setState here — state only changes inside the async callbacks.
  useEffect(() => {
    listProfiles()
      .then(setProfiles)
      .catch((err) =>
        message.error(
          err instanceof Error ? err.message : "Failed to load profiles.",
        ),
      )
      .finally(() => setLoading(false));
  }, [message]);

  // Re-fetch after a mutation. Only ever called from event handlers.
  async function reload() {
    setLoading(true);
    try {
      setProfiles(await listProfiles());
    } catch (err) {
      message.error(err instanceof Error ? err.message : "Failed to load profiles.");
    } finally {
      setLoading(false);
    }
  }

  const isEmpty = !loading && profiles.length === 0;

  return (
    <>
      <Typography.Title level={1} style={pageTitle}>
        Profiles
      </Typography.Title>
      <Typography.Paragraph type="secondary" style={pageLede}>
        One profile per channel persona. Each keeps its own prompt templates,
        generations and channels.
      </Typography.Paragraph>

      <div style={{ marginTop: 44 }}>
        {loading ? (
          <div style={{ padding: 48, textAlign: "center" }}>
            <Spin />
          </div>
        ) : (
          <>
            {isEmpty && (
              <Typography.Paragraph
                type="secondary"
                style={{ maxWidth: "52ch", marginBottom: 20 }}
              >
                A profile is a channel persona, like &ldquo;Retro Arcade
                Vault&rdquo;. Give it a niche and a generator, then write
                prompt templates inside it.
              </Typography.Paragraph>
            )}
            <ul className="acmp-roster">
              {profiles.map((p) => (
                <li key={p.id}>
                  <Link href={`/profiles/${p.id}`} className="acmp-roster-row">
                    <Stripe color={tapeColor(p.id)} size={6} />
                    <div className="acmp-roster-main">
                      <div className="acmp-roster-name">{p.name}</div>
                      <div className="acmp-roster-sub">{p.niche}</div>
                    </div>
                    <div className="acmp-roster-meta">
                      {providerShort(p.provider)}
                    </div>
                    <div className="acmp-roster-meta is-quiet">
                      {formatDate(p.updatedAt)}
                    </div>
                  </Link>
                </li>
              ))}
              <li>
                <Button
                  type="dashed"
                  block
                  onClick={() => setCreating(true)}
                  style={{
                    height: 72,
                    justifyContent: "flex-start",
                    paddingInline: 24,
                    fontSize: 16,
                  }}
                >
                  {isEmpty ? "Create your first profile" : "New profile"}
                </Button>
              </li>
            </ul>
          </>
        )}
      </div>

      <Modal
        title="New profile"
        open={creating}
        onCancel={() => setCreating(false)}
        footer={null}
        destroyOnHidden
        {...mobileModal}
      >
        <ProfileForm
          submitLabel="Create profile"
          onSubmit={async (input) => {
            await createProfile(input);
            setCreating(false);
            message.success("Profile created.");
            await reload();
          }}
        />
      </Modal>
    </>
  );
}
