"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Alert, Button, Divider, Form, Input, Typography } from "antd";
import Stripe from "@/components/Stripe";
import { ApiError } from "@/lib/api";
import { login } from "@/lib/auth";
import {
  DEV_LOGIN_ENABLED,
  DEV_LOGIN_PASSWORD,
  DEV_LOGIN_USERNAME,
} from "@/lib/devLogin";
import { INK } from "@/lib/tape";

export default function LoginPage() {
  const router = useRouter();
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function signIn(values: { username: string; password: string }) {
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
    <div style={{ position: "relative" }}>
      <Stripe
        color={INK.chalk}
        size={14}
        style={{ position: "absolute", top: 0, left: 0, right: 0, height: 20 }}
      />

      <div className="acmp-login">
        <div
          style={{
            display: "flex",
            flexDirection: "column",
            justifyContent: "flex-end",
            gap: 20,
            padding: "clamp(56px, 8vw, 96px) clamp(20px, 5vw, 64px) clamp(24px, 5vw, 64px)",
          }}
        >
          <Typography.Paragraph
            type="secondary"
            style={{ margin: 0, maxWidth: "34ch", fontSize: 17 }}
          >
            Make AI video, then publish it to your channels, from one place.
          </Typography.Paragraph>
          <h1 className="acmp-login-mark">ACMP</h1>
        </div>

        <div
          style={{
            display: "flex",
            alignItems: "center",
            padding: "clamp(24px, 5vw, 64px)",
            borderLeft: "1px solid #1c1c20",
          }}
        >
          <div style={{ width: "100%", maxWidth: 380 }}>
            <Typography.Title level={3} style={{ marginTop: 0, marginBottom: 24 }}>
              Sign in
            </Typography.Title>

            <Form layout="vertical" onFinish={signIn} requiredMark={false}>
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

            {DEV_LOGIN_ENABLED && (
              <>
                <Divider plain style={{ marginBlock: 20, fontSize: 12 }}>
                  dev only
                </Divider>
                <Button
                  block
                  loading={busy}
                  onClick={() =>
                    signIn({
                      username: DEV_LOGIN_USERNAME,
                      password: DEV_LOGIN_PASSWORD,
                    })
                  }
                >
                  Log in as admin
                </Button>
              </>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
