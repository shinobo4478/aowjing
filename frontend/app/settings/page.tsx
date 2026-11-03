"use client";

import { useEffect, useState } from "react";
import { App, Button, Form, Input, Spin, Typography } from "antd";
import { getSettings, updateSettings } from "@/lib/api";
import type { Settings } from "@/lib/types";
import { pageLede, pageTitle } from "@/lib/ui";

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
      <Typography.Title level={1} style={pageTitle}>
        Settings
      </Typography.Title>
      <Typography.Paragraph type="secondary" style={pageLede}>
        Credentials are stored once and shared by every profile.
      </Typography.Paragraph>

      <div style={{ marginTop: 44, maxWidth: 520 }}>
        <Typography.Title level={4} style={{ marginTop: 0, marginBottom: 20 }}>
          Video generation
        </Typography.Title>

        <Spin spinning={loading}>
          <Form form={form} layout="vertical" onFinish={onFinish}>
            <Form.Item
              label="fal.ai API key"
              name="falApiKey"
              extra="Needed to generate video with Kling 3.0. Create one in your fal.ai dashboard under API keys. Without it, profiles can only generate text."
            >
              <Input.Password autoComplete="off" placeholder="fal-…" />
            </Form.Item>

            <Form.Item style={{ marginBottom: 0 }}>
              <Button type="primary" htmlType="submit" loading={saving}>
                Save changes
              </Button>
            </Form.Item>
          </Form>
        </Spin>
      </div>
    </>
  );
}
