"use client";

import { useEffect, useState } from "react";
import { App, Button, Form, Grid, Input, Spin, Tabs, Typography } from "antd";
import { getSettings, updateSettings } from "@/lib/api";
import type { Settings } from "@/lib/types";
import { pageLede, pageTitle } from "@/lib/ui";
import VoiceSamples from "./VoiceSamples";

/** A section's title and a line saying what it holds. */
function SectionHead({ title, children }: { title: string; children?: React.ReactNode }) {
  return (
    <div style={{ marginBottom: 20 }}>
      <Typography.Title level={4} style={{ marginTop: 0, marginBottom: 4 }}>
        {title}
      </Typography.Title>
      {children && (
        <Typography.Paragraph type="secondary" style={{ margin: 0 }}>
          {children}
        </Typography.Paragraph>
      )}
    </div>
  );
}

export default function SettingsPage() {
  const { message } = App.useApp();
  const [form] = Form.useForm<Settings>();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [tab, setTab] = useState("keys");
  // Vertical tabs need width: on a narrow screen they go along the top instead.
  const screens = Grid.useBreakpoint();
  const wide = screens.md ?? true;

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

  const saveButton = (
    <Form.Item style={{ marginBottom: 0 }}>
      <Button type="primary" htmlType="submit" loading={saving}>
        Save changes
      </Button>
    </Form.Item>
  );

  // One form holds every field of the first three sections, so Save always saves
  // what was typed in any of them (the panes are all rendered, `forceRender`, so
  // a section that was never opened still keeps its loaded values). The voice
  // section is not part of the form: it has its own buttons.
  const items = [
    {
      key: "keys",
      label: "Keys",
      forceRender: true,
      children: (
        <div style={{ maxWidth: 520 }}>
          <SectionHead title="fal.ai keys">
            Credentials are stored once and shared by every profile.
          </SectionHead>
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
          {saveButton}
        </div>
      ),
    },
    {
      key: "models",
      label: "AI models",
      forceRender: true,
      children: (
        <div style={{ maxWidth: 520 }}>
          <SectionHead title="Script and translation models">
            Both run through fal with the API key and are billed per token.
          </SectionHead>
          <Form.Item
            label="Script model (optional)"
            name="scriptModel"
            extra={'The AI model that drafts shot lists, as an OpenRouter-style id, e.g. "anthropic/claude-sonnet-5" or "google/gemini-2.5-flash". It runs through fal with the key above and is billed per token. Leave empty for the built-in default.'}
          >
            <Input autoComplete="off" placeholder="anthropic/claude-sonnet-5" />
          </Form.Item>

          <Form.Item
            label="Translation model (optional)"
            name="translateModel"
            extra={'The AI model that translates your Thai prompts into the English the picture and video models receive, and back into Thai so you can read them. It runs through fal with the key above and is billed per token (a fraction of a cent per prompt). A smaller, cheaper model is usually enough, e.g. "google/gemini-2.5-flash". Leave empty to use the script model above.'}
          >
            <Input autoComplete="off" placeholder="google/gemini-2.5-flash" />
          </Form.Item>
          {saveButton}
        </div>
      ),
    },
    {
      key: "spending",
      label: "Spending",
      forceRender: true,
      children: (
        <div style={{ maxWidth: 520 }}>
          <SectionHead title="Daily spending limit" />
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
          {saveButton}
        </div>
      ),
    },
    {
      key: "voice",
      label: "Voice",
      forceRender: false,
      children: <VoiceSamples />,
    },
  ];

  return (
    <>
      <Typography.Title level={1} style={pageTitle}>
        Settings
      </Typography.Title>
      <Typography.Paragraph type="secondary" style={pageLede}>
        Credentials, models, the spending limit, and the Thai voice samples.
      </Typography.Paragraph>

      <div style={{ marginTop: 36 }}>
        <Spin spinning={loading}>
          <Form form={form} layout="vertical" onFinish={onFinish}>
            <Tabs
              tabPlacement={wide ? "start" : "top"}
              activeKey={tab}
              onChange={setTab}
              items={items}
              style={{ minHeight: 360 }}
            />
          </Form>
        </Spin>
      </div>
    </>
  );
}
