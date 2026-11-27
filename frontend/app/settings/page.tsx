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
      // The limit's "today" starts at this browser's midnight.
      const dailyCapTz = Intl.DateTimeFormat().resolvedOptions().timeZone ?? "";
      const saved = await updateSettings({ ...values, dailyCapTz });
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

            <Form.Item
              label="fal.ai billing key (optional)"
              name="falBillingKey"
              extra="Only used to show your remaining credit in the menu bar. Create a separate key with the BILLING preset in your fal.ai dashboard; the key above does not need billing access. Leave empty to hide the balance."
            >
              <Input.Password autoComplete="off" placeholder="fal-…" />
            </Form.Item>

            <Form.Item
              label="Script model (optional)"
              name="scriptModel"
              extra={'The AI model that drafts shot lists, as an OpenRouter-style id, e.g. "anthropic/claude-sonnet-5" or "google/gemini-2.5-flash". It runs through fal with the key above and is billed per token. Leave empty for the built-in default.'}
            >
              <Input autoComplete="off" placeholder="anthropic/claude-sonnet-5" />
            </Form.Item>

            <Form.Item
              label="Daily spending limit (optional)"
              name="dailyCapUsd"
              rules={[
                {
                  validator: (_, v: string | undefined) =>
                    !v?.trim() || Number(v) > 0
                      ? Promise.resolve()
                      : Promise.reject(new Error("Enter an amount above 0, or leave it empty.")),
                },
              ]}
              extra="Stops clips and pictures from being started once today's fal.ai spend (real bills, plus the estimated price of anything still rendering or not billed yet) would pass this amount. The day starts at midnight in this browser's time zone. Leave empty for no limit."
            >
              <Input addonBefore="$" inputMode="decimal" autoComplete="off" placeholder="e.g. 10" />
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
