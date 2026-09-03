"use client";

import Link from "next/link";
import { Card, Flex, Tag, Typography } from "antd";

const features = [
  {
    title: "Profiles",
    done: true,
    text: "Full CRUD — list, create, edit, delete.",
    href: "/profiles",
  },
  { title: "Channels", done: false, text: "Nested under a Profile (not built yet)." },
  {
    title: "Prompt templates",
    done: false,
    text: "Associated with a Profile (not built yet).",
  },
];

export default function HomePage() {
  return (
    <>
      <Typography.Title level={3} style={{ marginTop: 0 }}>
        AI Content Management Platform
      </Typography.Title>
      <Typography.Paragraph type="secondary">
        Phase 1 frontend shell — Ant Design components on Next.js App Router.
      </Typography.Paragraph>

      <Card title="What's wired up">
        <Flex vertical gap={16}>
          {features.map((f) => (
            <Flex key={f.title} justify="space-between" align="flex-start" gap={16}>
              <div>
                <Typography.Text strong>
                  {f.href && f.done ? (
                    <Link href={f.href}>{f.title}</Link>
                  ) : (
                    f.title
                  )}
                </Typography.Text>
                <br />
                <Typography.Text type="secondary">{f.text}</Typography.Text>
              </div>
              <Tag color={f.done ? "green" : "default"}>
                {f.done ? "ready" : "todo"}
              </Tag>
            </Flex>
          ))}
        </Flex>
      </Card>

      <Typography.Paragraph type="secondary" style={{ marginTop: 16 }}>
        Data lives in a temporary in-memory store on the Next.js server and
        resets when the dev server restarts. The Go backend replaces it later.
      </Typography.Paragraph>
    </>
  );
}
