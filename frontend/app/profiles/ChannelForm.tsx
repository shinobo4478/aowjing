"use client";

import { useState } from "react";
import { Alert, Button, Form, Input } from "antd";
import { ApiError } from "@/lib/api";
import type { ChannelInput } from "@/lib/types";

interface Props {
  initial?: ChannelInput;
  submitLabel: string;
  onSubmit: (input: ChannelInput) => Promise<void>;
}

const EMPTY: ChannelInput = {
  name: "",
  platform: "",
  handle: "",
  description: "",
};

export default function ChannelForm({ initial, submitLabel, onSubmit }: Props) {
  const [form] = Form.useForm<ChannelInput>();
  const [formError, setFormError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function handleFinish(values: ChannelInput) {
    setBusy(true);
    setFormError(null);
    try {
      await onSubmit({
        name: values.name.trim(),
        platform: values.platform.trim(),
        handle: (values.handle ?? "").trim(),
        description: (values.description ?? "").trim(),
      });
    } catch (err) {
      if (err instanceof ApiError && err.fieldErrors) {
        form.setFields(
          Object.entries(err.fieldErrors).map(([name, message]) => ({
            name: name as keyof ChannelInput,
            errors: [message],
          })),
        );
      } else {
        setFormError(err instanceof Error ? err.message : "Something went wrong.");
      }
    } finally {
      setBusy(false);
    }
  }

  return (
    <Form
      form={form}
      layout="vertical"
      requiredMark="optional"
      initialValues={initial ?? EMPTY}
      onFinish={handleFinish}
    >
      {formError && (
        <Form.Item>
          <Alert type="error" showIcon title={formError} />
        </Form.Item>
      )}

      <Form.Item
        label="Name"
        name="name"
        rules={[
          { required: true, message: "Name is required." },
          { min: 2, message: "Name must be at least 2 characters." },
          { max: 80, message: "Name must be 80 characters or fewer." },
        ]}
      >
        <Input placeholder="Main YouTube" />
      </Form.Item>

      <Form.Item
        label="Platform"
        name="platform"
        rules={[
          { required: true, message: "Platform is required." },
          { min: 2, message: "Platform must be at least 2 characters." },
          { max: 40, message: "Platform must be 40 characters or fewer." },
        ]}
      >
        <Input placeholder="youtube, tiktok, instagram…" />
      </Form.Item>

      <Form.Item
        label="Handle"
        name="handle"
        rules={[{ max: 80, message: "Handle must be 80 characters or fewer." }]}
      >
        <Input placeholder="@retroarcadevault" />
      </Form.Item>

      <Form.Item
        label="Description"
        name="description"
        rules={[{ max: 600, message: "Description must be 600 characters or fewer." }]}
      >
        <Input.TextArea rows={3} placeholder="What this channel is for…" />
      </Form.Item>

      <Form.Item style={{ marginBottom: 0 }}>
        <Button type="primary" htmlType="submit" loading={busy}>
          {submitLabel}
        </Button>
      </Form.Item>
    </Form>
  );
}
