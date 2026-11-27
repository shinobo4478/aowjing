"use client";

import Link from "next/link";
import { Alert, Button, Col, Row, Spin, Typography } from "antd";
import { formatUsd } from "@/lib/format";
import { pageLede, pageTitle } from "@/lib/ui";
import { ActiveJobs, Section, Stat } from "./dashboardParts";
import { useDashboard } from "./useDashboard";

/**
 * The first screen: only what you would want to know on opening the app —
 * what is being made, what failed, what is left on fal and what today cost.
 * Everything else is on the Dashboard page.
 */
export default function Home() {
  const { data, error, credits } = useDashboard();

  const balance =
    credits?.balance != null
      ? formatUsd(credits.balance)
      : credits && !credits.configured
        ? "Not set"
        : "—";

  return (
    <>
      <Typography.Title level={1} style={pageTitle}>
        Home
      </Typography.Title>
      <Typography.Paragraph type="secondary" style={pageLede}>
        What is being made, what went wrong, and what it costs.
      </Typography.Paragraph>

      {error && <Alert type="error" showIcon title={error} style={{ marginTop: 24 }} />}
      {!data && !error && (
        <div style={{ padding: 48, textAlign: "center" }}>
          <Spin />
        </div>
      )}

      {data && (
        <>
          <div style={{ marginTop: 32 }}>
            <Row gutter={[16, 16]}>
              <Col xs={12} md={6}>
                <Stat
                  title="Credits left"
                  value={balance}
                  note={credits && !credits.configured ? "Add a billing key in Settings" : "On fal.ai"}
                />
              </Col>
              <Col xs={12} md={6}>
                <Stat
                  title="Spent today"
                  value={
                    data.spend.todayBilled === 0 && data.spend.todayUnbilled > 0
                      ? "—"
                      : formatUsd(data.spend.todayCost)
                  }
                  note={
                    data.spend.todayUnbilled > 0
                      ? `${data.spend.todayUnbilled} ${data.spend.todayUnbilled === 1 ? "clip" : "clips"} not billed yet`
                      : `${data.spend.todayBilled} ${data.spend.todayBilled === 1 ? "clip" : "clips"}`
                  }
                />
              </Col>
              <Col xs={12} md={6}>
                <Stat
                  title="Rendering now"
                  value={data.active.pending.length}
                  note={data.active.pending.length === 0 ? "Nothing running" : "Clips being made"}
                />
              </Col>
              <Col xs={12} md={6}>
                <Stat
                  title="Failed"
                  value={data.active.failed.length >= 5 ? "5+" : data.active.failed.length}
                  note="Most recent"
                />
              </Col>
            </Row>
          </div>

          <Section title="In progress and failed">
            <ActiveJobs active={data.active} />
          </Section>

          <div style={{ marginTop: 40 }}>
            <Link href="/dashboard">
              <Button type="primary">Open the full dashboard</Button>
            </Link>
          </div>
        </>
      )}
    </>
  );
}
