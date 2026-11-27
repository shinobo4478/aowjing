"use client";

import { useEffect, useMemo, useState, type MouseEvent } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { App, Button, Card, Col, Modal, Row, Segmented, Select, Spin, Typography } from "antd";
import Stripe from "@/components/Stripe";
import Tally from "@/components/Tally";
import { createProfile, listProfiles } from "@/lib/api";
import { formatDate } from "@/lib/format";
import { tapeColor } from "@/lib/tape";
import { providerShort, type Profile } from "@/lib/types";
import { mobileModal, pageLede, pageTitle } from "@/lib/ui";
import ProfileForm from "./ProfileForm";
import { clipsByProfile, useRunning } from "../useRunning";

type View = "list" | "cards";
type SortKey = "updated" | "created" | "name" | "rendering";

const SORT_OPTIONS: { value: SortKey; label: string }[] = [
  { value: "updated", label: "Recently updated" },
  { value: "created", label: "Newest first" },
  { value: "name", label: "Name A–Z" },
  { value: "rendering", label: "Rendering first" },
];

// The choices are remembered in this browser. They are read when the state is
// created; the controls only show once the profiles have loaded, so the server
// render (which shows a spinner) never disagrees with them.
const VIEW_KEY = "acmp.profiles.view";
const SORT_KEY = "acmp.profiles.sort";

function stored<T extends string>(key: string, allowed: readonly T[], fallback: T): T {
  if (typeof window === "undefined") return fallback;
  const v = window.localStorage.getItem(key);
  return allowed.includes(v as T) ? (v as T) : fallback;
}

function sortProfiles(list: Profile[], key: SortKey, running: Record<string, number>): Profile[] {
  const byUpdated = (a: Profile, b: Profile) => b.updatedAt.localeCompare(a.updatedAt);
  const out = [...list];
  switch (key) {
    case "created":
      return out.sort((a, b) => b.createdAt.localeCompare(a.createdAt));
    case "name":
      return out.sort((a, b) => a.name.localeCompare(b.name, undefined, { sensitivity: "base", numeric: true }));
    case "rendering":
      return out.sort((a, b) => (running[b.id] ?? 0) - (running[a.id] ?? 0) || byUpdated(a, b));
    default:
      return out.sort(byUpdated);
  }
}

/** What a profile is making right now, as a pulsing tally; a link to the
 *  Generations tab so the clips can be found among several profiles. */
function Rendering({ profileId, count }: { profileId: string; count: number }) {
  if (count === 0) return null;
  return (
    <Link
      href={`/profiles/${profileId}?tab=generations`}
      aria-label={`${count} rendering. Open this profile's generations`}
      style={{ display: "inline-block", textDecoration: "none", color: "inherit" }}
    >
      <Tally status="pending" label={`${count} rendering`} />
    </Link>
  );
}

/** The picture of a profile card. Profiles have no picture yet, so it shows the
 *  profile's tape stripe; when a profile gets one, pass its address as `imageUrl`. */
function Cover({ id, imageUrl }: { id: string; imageUrl?: string }) {
  return (
    <div style={{ height: 132, overflow: "hidden" }}>
      {imageUrl ? (
        // eslint-disable-next-line @next/next/no-img-element -- a user's own picture of unknown size
        <img src={imageUrl} alt="" style={{ width: "100%", height: "100%", objectFit: "cover", display: "block" }} />
      ) : (
        <Stripe color={tapeColor(id)} size={14} style={{ width: "100%", height: "100%" }} />
      )}
    </div>
  );
}

export default function ProfilesPage() {
  const { message } = App.useApp();
  const router = useRouter();
  const runningItems = useRunning();
  const running = useMemo(() => clipsByProfile(runningItems), [runningItems]);
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [view, setViewState] = useState<View>(() => stored(VIEW_KEY, ["list", "cards"], "list"));
  const [sort, setSortState] = useState<SortKey>(() =>
    stored(SORT_KEY, SORT_OPTIONS.map((o) => o.value), "updated"),
  );

  const setView = (v: View) => {
    setViewState(v);
    window.localStorage.setItem(VIEW_KEY, v);
  };
  const setSort = (s: SortKey) => {
    setSortState(s);
    window.localStorage.setItem(SORT_KEY, s);
  };

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

  const sorted = useMemo(() => sortProfiles(profiles, sort, running), [profiles, sort, running]);
  const isEmpty = !loading && profiles.length === 0;

  // The whole row or card opens the profile; a real link inside it (the name,
  // the rendering tag) keeps its own destination and middle-click.
  const open = (id: string) => (e: MouseEvent) => {
    if ((e.target as HTMLElement).closest("a")) return;
    router.push(`/profiles/${id}`);
  };

  const newProfileLabel = isEmpty ? "Create your first profile" : "New profile";

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

            {!isEmpty && (
              <div
                style={{
                  display: "flex",
                  flexWrap: "wrap",
                  gap: 12,
                  alignItems: "center",
                  justifyContent: "space-between",
                  marginBottom: 16,
                }}
              >
                <Segmented<View>
                  aria-label="Layout"
                  value={view}
                  onChange={setView}
                  options={[
                    { value: "list", label: "List" },
                    { value: "cards", label: "Cards" },
                  ]}
                />
                <Select<SortKey>
                  aria-label="Sort profiles"
                  value={sort}
                  onChange={setSort}
                  options={SORT_OPTIONS}
                  style={{ minWidth: 190 }}
                />
              </div>
            )}

            {view === "list" || isEmpty ? (
              <ul className="acmp-roster">
                {sorted.map((p) => (
                  <li key={p.id}>
                    <div className="acmp-roster-row" onClick={open(p.id)}>
                      <Stripe color={tapeColor(p.id)} size={6} />
                      <div className="acmp-roster-main">
                        <Link href={`/profiles/${p.id}`} className="acmp-roster-name">
                          {p.name}
                        </Link>
                        <div className="acmp-roster-sub">{p.niche}</div>
                        {running[p.id] ? (
                          <div style={{ marginTop: 8 }}>
                            <Rendering profileId={p.id} count={running[p.id]} />
                          </div>
                        ) : null}
                      </div>
                      <div className="acmp-roster-meta">
                        {providerShort(p.provider)}
                      </div>
                      <div className="acmp-roster-meta is-quiet">
                        {formatDate(p.updatedAt)}
                      </div>
                    </div>
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
                    {newProfileLabel}
                  </Button>
                </li>
              </ul>
            ) : (
              <Row gutter={[16, 16]}>
                {sorted.map((p) => (
                  <Col key={p.id} xs={24} sm={12} lg={8} xl={6}>
                    <Card
                      hoverable
                      onClick={open(p.id)}
                      cover={<Cover id={p.id} />}
                      styles={{ body: { padding: 16 } }}
                      style={{ height: "100%", overflow: "hidden" }}
                    >
                      <Link
                        href={`/profiles/${p.id}`}
                        className="acmp-roster-name"
                        style={{ fontSize: 20, display: "block" }}
                      >
                        {p.name}
                      </Link>
                      <div className="acmp-roster-sub">{p.niche}</div>
                      <div
                        style={{
                          display: "flex",
                          flexWrap: "wrap",
                          gap: 8,
                          alignItems: "center",
                          justifyContent: "space-between",
                          marginTop: 12,
                        }}
                      >
                        <span className="acmp-roster-meta is-quiet" style={{ fontSize: 14 }}>
                          {providerShort(p.provider)} · {formatDate(p.updatedAt)}
                        </span>
                        <Rendering profileId={p.id} count={running[p.id] ?? 0} />
                      </div>
                    </Card>
                  </Col>
                ))}
                <Col xs={24} sm={12} lg={8} xl={6}>
                  <Button
                    type="dashed"
                    block
                    onClick={() => setCreating(true)}
                    style={{ height: "100%", minHeight: 120, fontSize: 16, borderRadius: 16 }}
                  >
                    {newProfileLabel}
                  </Button>
                </Col>
              </Row>
            )}
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
