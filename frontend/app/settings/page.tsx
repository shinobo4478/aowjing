"use client";

import { useEffect, useState } from "react";
import { App, Button, Card, Form, Input, Spin, Typography } from "antd";
import { getSettings, updateSettings } from "@/lib/api";
import type { Settings } from "@/lib/types";

export default function SettingsPage() {
  const { message } = App.useApp();
  const [form] = Form.useForm<Settings>();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    getSettings()
      .then((s) => form.setFieldsValue(s))
      .catch((err) =>
        message.error(
          err instanceof Error ? err.message : "Failed to load settings.",
        ),
      )
      .finally(() => setLoading(false));
  }, [form, message]);

  async function onFinish(values: Settings) {
    setSaving(true);
    try {
      const saved = await updateSettings(values);
      form.setFieldsValue(saved);
      message.success("Settings saved.");
    } catch (err) {
      message.error(err instanceof Error ? err.message : "Failed to save.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <>
      <Typography.Title level={3} style={{ marginTop: 0 }}>
        Settings
      </Typography.Title>
      <Typography.Paragraph type="secondary">
        Provider credentials, stored once and shared across all profiles.
      </Typography.Paragraph>

      <Card style={{ maxWidth: 520 }}>
        {loading ? (
          <div style={{ textAlign: "center", padding: 24 }}>
            <Spin />
          </div>
        ) : (
          <Form form={form} layout="vertical" onFinish={onFinish}>
            <Form.Item
              label="fal.ai API key"
              name="falApiKey"
              extra="Used by the fal.ai video generator (Kling 3.0)."
            >
              <Input.Password autoComplete="off" placeholder="fal-…" />
            </Form.Item>

            <Form.Item style={{ marginBottom: 0 }}>
              <Button type="primary" htmlType="submit" loading={saving}>
                Save
              </Button>
            </Form.Item>
          </Form>
        )}
      </Card>
    </>
  );
}
