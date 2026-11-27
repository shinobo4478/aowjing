"use client";

import { useCallback, useEffect, useState, type ReactNode } from "react";
import Link from "next/link";
import { Alert, Button, Card, Col, Empty, Progress, Row, Spin, Statistic, Tag, Typography } from "antd";
import SpendChart from "@/components/SpendChart";
import { getDashboard, getFalCredits } from "@/lib/api";
import { formatDateTime, formatUsd } from "@/lib/format";
import type { Dashboard as DashboardData, FalCredits } from "@/lib/types";
import { pageLede, pageTitle } from "@/lib/ui";
import { CREDITS_REFRESH_EVENT } from "./CreditsBadge";

// How often the page re-reads itself while it is open: clips finish and bills
// arrive on their own.
const REFRESH_MS = 30_000;

/** "5 min ago", "3 h ago", else the date. */
function ago(iso: string): string {
  const mins = Math.max(0, Math.round((Date.now() - new Date(iso).getTime()) / 60_000));
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins} min ago`;
  if (mins < 24 * 60) return `${Math.round(mins / 60)} h ago`;
  return formatDateTime(iso);
}

const profileLink = (id: string, tab?: string) =>
  `/profiles/${id}${tab ? `?tab=${tab}` : ""}`;

function Section({
  title,
  hint,
  children,
}: {
  title: string;
  hint?: string;
  children: ReactNode;
}) {
  return (
    <section style={{ marginTop: 48 }}>
      <Typography.Title level={3} style={{ margin: 0, fontSize: 24, fontWeight: 600 }}>
        {title}
      </Typography.Title>
      {hint && (
        <Typography.Paragraph type="secondary" style={{ margin: "4px 0 0" }}>
          {hint}
        </Typography.Paragraph>
      )}
      <div style={{ marginTop: 16 }}>{children}</div>
    </section>
  );
}

function Stat({
  title,
  value,
  note,
  loading,
}: {
  title: string;
  value: ReactNode;
  note?: ReactNode;
  loading?: boolean;
}) {
  return (
    <Card size="small" loading={loading} style={{ height: "100%" }}>
      <Statistic title={title} value={value as string | number} />
      {note && (
        <Typography.Text type="secondary" style={{ fontSize: 13 }}>
          {note}
        </Typography.Text>
      )}
    </Card>
  );
}

/** A slot for something we cannot fill yet, so the layout is already in place. */
function ComingSoon({ title, needs }: { title: string; needs: string }) {
  return (
    <Card
      size="small"
      style={{ height: "100%", borderStyle: "dashed", background: "transparent" }}
    >
      <Typography.Text strong>{title}</Typography.Text> <Tag>Not connected</Tag>
      <Typography.Paragraph type="secondary" style={{ margin: "8px 0 0" }}>
        {needs}
      </Typography.Paragraph>
    </Card>
  );
}

function RowLink({ href, children }: { href: string; children: ReactNode }) {
  return (
    <Link
      href={href}
      style={{
        display: "block",
        padding: "10px 0",
        borderTop: "1px solid var(--acmp-rule-soft)",
        color: "inherit",
      }}
    >
      {children}
    </Link>
  );
}

export default function Dashboard() {
  const [data, setData] = useState<DashboardData | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [credits, setCredits] = useState<FalCredits | null>(null);

  const load = useCallback(() => {
    getDashboard()
      .then((d) => {
        setData(d);
        setError(null);
      })
      .catch((e) => setError(e instanceof Error ? e.message : "Could not load the dashboard."));
    getFalCredits()
      .then(setCredits)
      .catch(() => {
        /* the credits tile just stays empty */
      });
  }, []);

  useEffect(() => {
    load();
    const iv = setInterval(load, REFRESH_MS);
    window.addEventListener(CREDITS_REFRESH_EVENT, load);
    return () => {
      clearInterval(iv);
      window.removeEventListener(CREDITS_REFRESH_EVENT, load);
    };
  }, [load]);

  return (
    <>
      <Typography.Title level={1} style={pageTitle}>
        Dashboard
      </Typography.Title>
      <Typography.Paragraph type="secondary" style={pageLede}>
        What you have spent, what needs a look, and how far your videos are along.
      </Typography.Paragraph>

      {error && <Alert type="error" showIcon title={error} style={{ marginTop: 24 }} />}
      {!data && !error && (
        <div style={{ padding: 48, textAlign: "center" }}>
          <Spin />
        </div>
      )}
      {data && <Body d={data} credits={credits} />}
    </>
  );
}

function Body({ d, credits }: { d: DashboardData; credits: FalCredits | null }) {
  const { spend, active, pipeline, published, products } = d;
  const needsAttention = active.pending.length + active.failed.length;
  const shotsLeft = Math.max(0, pipeline.shots - pipeline.shotsMade);

  return (
    <>
      {/* --- Needs attention first: it is the only part that asks for action. --- */}
      <Section
        title="Needs a look"
        hint={needsAttention === 0 ? undefined : "Clips being made now and the latest that failed."}
      >
        {needsAttention === 0 ? (
          <Card size="small">
            <Typography.Text type="secondary">
              Nothing is running and nothing has failed.
            </Typography.Text>
          </Card>
        ) : (
          <Row gutter={[16, 16]}>
            <Col xs={24} md={12}>
              <Card size="small" title={`Rendering now (${active.pending.length})`}>
                {active.pending.length === 0 ? (
                  <Typography.Text type="secondary">No clip is rendering.</Typography.Text>
                ) : (
                  active.pending.map((p) => (
                    <RowLink key={p.id} href={profileLink(p.profileId, "generations")}>
                      <strong>{p.profileName}</strong>{" "}
                      <Typography.Text type="secondary">
                        · {p.seconds} s{p.fromShot ? " · video shot" : ""} · started {ago(p.createdAt)}
                      </Typography.Text>
                    </RowLink>
                  ))
                )}
              </Card>
            </Col>
            <Col xs={24} md={12}>
              <Card size="small" title={`Latest failures (${active.failed.length})`}>
                {active.failed.length === 0 ? (
                  <Typography.Text type="secondary">No failed clips.</Typography.Text>
                ) : (
                  active.failed.map((f) => (
                    <RowLink key={f.id} href={profileLink(f.profileId, "generations")}>
                      <div>
                        <strong>{f.profileName}</strong>{" "}
                        <Typography.Text type="secondary">
                          · {ago(f.createdAt)}
                          {f.costUsd != null && f.costUsd > 0 ? ` · billed ${formatUsd(f.costUsd)}` : ""}
                        </Typography.Text>
                      </div>
                      <Typography.Text type="danger" ellipsis style={{ display: "block", fontSize: 13 }}>
                        {f.error || "Failed"}
                      </Typography.Text>
                    </RowLink>
                  ))
                )}
              </Card>
            </Col>
          </Row>
        )}
      </Section>

      {/* --- Spend --- */}
      <Section
        title="Spend"
        hint="What fal has really charged, from its billing records."
      >
        <Row gutter={[16, 16]}>
          <Col xs={12} md={6}>
            <Stat
              title="This month"
              value={spend.monthBilled === 0 && spend.monthUnbilled > 0 ? "—" : formatUsd(spend.monthCost)}
              note={
                spend.monthUnbilled > 0
                  ? `${spend.monthUnbilled} ${spend.monthUnbilled === 1 ? "clip" : "clips"} not billed yet`
                  : `${spend.monthBilled} ${spend.monthBilled === 1 ? "clip" : "clips"}`
              }
            />
          </Col>
          <Col xs={12} md={6}>
            <Stat
              title="Credits left"
              value={
                credits?.balance != null
                  ? formatUsd(credits.balance)
                  : credits && !credits.configured
                    ? "Not set"
                    : "—"
              }
              note={credits && !credits.configured ? "Add a billing key in Settings" : "On fal.ai"}
            />
          </Col>
          <Col xs={12} md={6}>
            <Stat
              title="Average per clip"
              value={spend.avgPerClip != null ? formatUsd(spend.avgPerClip) : "—"}
              note="This month, billed clips"
            />
          </Col>
          <Col xs={12} md={6}>
            <Stat
              title="All time"
              value={spend.totalBilled === 0 && spend.totalUnbilled > 0 ? "—" : formatUsd(spend.totalCost)}
              note={
                spend.totalUnbilled > 0
                  ? `${spend.totalUnbilled} older clips have no bill on record`
                  : `${spend.totalBilled} clips`
              }
            />
          </Col>
        </Row>

        <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
          <Col xs={24} md={15}>
            <Card size="small" title="Last 30 days">
              <SpendChart days={spend.days} />
            </Card>
          </Col>
          <Col xs={24} md={9}>
            <Card size="small" title="By profile, this month">
              {spend.byProfile.length === 0 ? (
                <Typography.Text type="secondary">No billed clips this month.</Typography.Text>
              ) : (
                spend.byProfile.map((p) => (
                  <Link
                    key={p.profileId}
                    href={profileLink(p.profileId, "generations")}
                    style={{ display: "block", color: "inherit", marginBottom: 12 }}
                  >
                    <div style={{ display: "flex", justifyContent: "space-between", gap: 12 }}>
                      <span style={{ overflowWrap: "anywhere" }}>{p.name}</span>
                      <strong>{formatUsd(p.cost)}</strong>
                    </div>
                    <Progress
                      percent={spend.monthCost > 0 ? (p.cost / spend.monthCost) * 100 : 0}
                      showInfo={false}
                      size="small"
                      strokeColor="var(--acmp-tungsten)"
                    />
                  </Link>
                ))
              )}
            </Card>
          </Col>
        </Row>
      </Section>

      {/* --- Pipeline --- */}
      <Section title="Production" hint="How far the videos are along.">
        <Row gutter={[16, 16]}>
          <Col xs={12} md={6}>
            <Stat title="Videos" value={pipeline.videos} note={`${pipeline.shots} shots in all`} />
          </Col>
          <Col xs={12} md={6}>
            <Stat
              title="Shots made"
              value={`${pipeline.shotsMade} / ${pipeline.shots}`}
              note={shotsLeft > 0 ? `${shotsLeft} still to make` : "All shots made"}
            />
          </Col>
          <Col xs={12} md={6}>
            <Stat
              title="OK’d, not made yet"
              value={pipeline.approvedNotMade}
              note={pipeline.approvedNotMade > 0 ? "Ready to generate in bulk" : "Nothing waiting"}
            />
          </Col>
          <Col xs={12} md={6}>
            <Stat
              title="Not published"
              value={pipeline.unpublishedClips}
              note="Finished clips, retakes included"
            />
          </Col>
        </Row>
        {pipeline.shots > 0 && (
          <Progress
            percent={Math.round((pipeline.shotsMade / pipeline.shots) * 100)}
            strokeColor="var(--acmp-tungsten)"
            style={{ marginTop: 16 }}
          />
        )}
      </Section>

      {/* --- Published --- */}
      <Section title="Published" hint="Videos sent to YouTube from this app.">
        <Row gutter={[16, 16]}>
          <Col xs={8} md={6}>
            <Stat title="Last 7 days" value={published.last7Days} />
          </Col>
          <Col xs={8} md={6}>
            <Stat title="Last 30 days" value={published.last30Days} />
          </Col>
          <Col xs={8} md={6}>
            <Stat title="Total" value={published.total} />
          </Col>
        </Row>
        <Card size="small" title="Latest" style={{ marginTop: 16 }}>
          {published.recent.length === 0 ? (
            <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="Nothing published yet" />
          ) : (
            published.recent.map((u) => (
              <a
                key={u.id}
                href={u.url}
                target="_blank"
                rel="noreferrer"
                style={{
                  display: "block",
                  padding: "10px 0",
                  borderTop: "1px solid var(--acmp-rule-soft)",
                  color: "inherit",
                }}
              >
                <strong>{u.title}</strong> ↗
                <div>
                  <Typography.Text type="secondary" style={{ fontSize: 13 }}>
                    {u.profileName} · {u.channelName} · {u.privacy} · {ago(u.createdAt)}
                  </Typography.Text>
                </div>
              </a>
            ))
          )}
        </Card>
      </Section>

      {/* --- Products --- */}
      <Section title="Shopee products" hint="The catalog and where it is pinned.">
        <Row gutter={[16, 16]}>
          <Col xs={12} md={6}>
            <Stat title="In catalog" value={products.total} />
          </Col>
          <Col xs={12} md={6}>
            <Stat
              title="Pinned to a clip"
              value={`${products.pinned} / ${products.total}`}
              note={products.total - products.pinned > 0 ? `${products.total - products.pinned} not used yet` : undefined}
            />
          </Col>
        </Row>
        {products.unpinned.length > 0 && (
          <Card
            size="small"
            title="Not pinned to any clip"
            extra={
              <Link href="/products">
                <Button size="small">Open products</Button>
              </Link>
            }
            style={{ marginTop: 16 }}
          >
            {products.unpinned.map((p) => (
              <RowLink key={p.id} href="/products">
                {p.name}
              </RowLink>
            ))}
          </Card>
        )}
      </Section>

      {/* --- Not wired up yet: the places are reserved, the data is not there. --- */}
      <Section
        title="Results"
        hint="What the clips earn back. Needs connections that are not made yet."
      >
        <Row gutter={[16, 16]}>
          <Col xs={24} md={12}>
            <ComingSoon
              title="YouTube performance"
              needs="Views and likes per published video. Reading them needs a call to YouTube’s API for each upload; the channel connection you already made may be enough for views and likes, while watch time and revenue need a further permission."
            />
          </Col>
          <Col xs={24} md={12}>
            <ComingSoon
              title="Shopee commission"
              needs="Clicks, orders and commission per clip. Needs a Shopee Affiliate account and its Open API credentials (Phase 3 B)."
            />
          </Col>
        </Row>
      </Section>
    </>
  );
}
