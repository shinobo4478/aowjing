"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Alert, Button, Card, Form, Input, Typography } from "antd";
import { ApiError } from "@/lib/api";
import { login } from "@/lib/auth";

export default function LoginPage() {
  const router = useRouter();
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function onFinish(values: { username: string; password: string }) {
    setBusy(true);
    setError(null);
    try {
      await login(values.username, values.password);
      router.replace("/profiles");
    } catch (err) {
      setError(
        err instanceof ApiError
          ? err.message
          : "Could not reach the server. Is the API running?",
      );
      setBusy(false);
    }
  }

  return (
    <div style={{ display: "grid", placeItems: "center", minHeight: "100vh", padding: 24 }}>
      <Card style={{ width: 360 }}>
        <Typography.Title level={4} style={{ marginTop: 0 }}>
          Sign in
        </Typography.Title>
        <Typography.Paragraph type="secondary" style={{ marginTop: -4 }}>
          AI Content Management Platform
        </Typography.Paragraph>

        <Form layout="vertical" onFinish={onFinish} requiredMark={false}>
          {error && (
            <Form.Item>
              <Alert type="error" showIcon title={error} />
            </Form.Item>
          )}

          <Form.Item
            label="Username"
            name="username"
            rules={[{ required: true, message: "Enter your username." }]}
          >
            <Input autoFocus autoComplete="username" />
          </Form.Item>

          <Form.Item
            label="Password"
            name="password"
            rules={[{ required: true, message: "Enter your password." }]}
          >
            <Input.Password autoComplete="current-password" />
          </Form.Item>

          <Form.Item style={{ marginBottom: 0 }}>
            <Button type="primary" htmlType="submit" block loading={busy}>
              Sign in
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}
