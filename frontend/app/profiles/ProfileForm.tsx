"use client";

import { useState } from "react";
import { Alert, Button, Form, Input, Select } from "antd";
import { ApiError } from "@/lib/api";
import { PROVIDER_OPTIONS, type ProfileInput } from "@/lib/types";

interface Props {
  initial?: ProfileInput;
  submitLabel: string;
  onSubmit: (input: ProfileInput) => Promise<void>;
}

const EMPTY: ProfileInput = {
  name: "",
  niche: "",
  description: "",
  provider: "text",
};

export default function ProfileForm({ initial, submitLabel, onSubmit }: Props) {
  const [form] = Form.useForm<ProfileInput>();
  const [formError, setFormError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function handleFinish(values: ProfileInput) {
    setBusy(true);
    setFormError(null);
    try {
      await onSubmit({
        name: values.name.trim(),
        niche: values.niche.trim(),
        description: (values.description ?? "").trim(),
        provider: values.provider,
      });
    } catch (err) {
      if (err instanceof ApiError && err.fieldErrors) {
        form.setFields(
          Object.entries(err.fieldErrors).map(([name, message]) => ({
            name: name as keyof ProfileInput,
            errors: [message as string],
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
        <Input placeholder="Retro Arcade Vault" />
      </Form.Item>

      <Form.Item
        label="Niche"
        name="niche"
        rules={[
          { required: true, message: "Niche is required." },
          { min: 2, message: "Niche must be at least 2 characters." },
        ]}
      >
        <Input placeholder="retro gaming history" />
      </Form.Item>

      <Form.Item
        label="Description"
        name="description"
        rules={[{ max: 600, message: "Description must be 600 characters or fewer." }]}
      >
        <Input.TextArea
          rows={4}
          placeholder="Tone, audience, pacing, do/don't…"
        />
      </Form.Item>

      <Form.Item
        label="Generator"
        name="provider"
        rules={[{ required: true, message: "Pick a generator." }]}
        extra="What this profile generates with by default."
      >
        <Select options={PROVIDER_OPTIONS} />
      </Form.Item>

      <Form.Item style={{ marginBottom: 0 }}>
        <Button type="primary" htmlType="submit" loading={busy}>
          {submitLabel}
        </Button>
      </Form.Item>
    </Form>
  );
}
