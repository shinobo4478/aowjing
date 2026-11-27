import type { ReactNode } from "react";
import Link from "next/link";
import { Card, Col, Row, Statistic, Tag, Typography } from "antd";
import { formatDateTime, formatUsd } from "@/lib/format";
import type { Dashboard as DashboardData } from "@/lib/types";

// Pieces shared by the home page (app/page.tsx) and the full dashboard
// (app/Dashboard.tsx).

// How often the page re-reads itself while it is open: clips finish and bills
// arrive on their own.
export const REFRESH_MS = 30_000;

/** "5 min ago", "3 h ago", else the date. */
export function ago(iso: string): string {
  const mins = Math.max(0, Math.round((Date.now() - new Date(iso).getTime()) / 60_000));
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins} min ago`;
  if (mins < 24 * 60) return `${Math.round(mins / 60)} h ago`;
  return formatDateTime(iso);
}

export const profileLink = (id: string, tab?: string) =>
  `/profiles/${id}${tab ? `?tab=${tab}` : ""}`;

export function Section({
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

export function Stat({
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
export function ComingSoon({ title, needs }: { title: string; needs: string }) {
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

export function RowLink({ href, children }: { href: string; children: ReactNode }) {
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

/** The clips being made right now and the latest failures, as two cards. */
export function ActiveJobs({ active }: { active: DashboardData["active"] }) {
  const needsAttention = active.pending.length + active.failed.length;
  return (
    <>
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
    </>
  );
}
