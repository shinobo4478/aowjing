"use client";

import Link from "next/link";
import { Button, Card, Flex, Typography } from "antd";

const capabilities = [
  {
    title: "Profiles",
    text: "One channel persona / content niche. Each picks a default generator.",
  },
  {
    title: "Channels",
    text: "Distribution targets under a profile (YouTube, TikTok, …).",
  },
  {
    title: "Prompt templates",
    text: "Reusable prompt text, tied to a profile.",
  },
  {
    title: "Generate",
    text: "Run a template through its generator — text now, fal.ai video with a key. Runs on a background worker.",
  },
  {
    title: "Settings",
    text: "Provider credentials, stored once and shared across profiles.",
  },
];

export default function HomePage() {
  return (
    <>
      <Typography.Title level={3} style={{ marginTop: 0 }}>
        AI Content Management Platform
      </Typography.Title>
      <Typography.Paragraph type="secondary">
        Create and organise AI-generated video content across profiles and
        channels.
      </Typography.Paragraph>

      <Card
        title="What you can do"
        extra={
          <Link href="/profiles">
            <Button type="primary">Open Profiles</Button>
          </Link>
        }
      >
        <Flex vertical gap={18}>
          {capabilities.map((c) => (
            <div key={c.title}>
              <Typography.Text strong>{c.title}</Typography.Text>
              <br />
              <Typography.Text type="secondary">{c.text}</Typography.Text>
            </div>
          ))}
        </Flex>
      </Card>
    </>
  );
}
