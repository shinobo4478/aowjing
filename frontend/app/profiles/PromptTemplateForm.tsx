"use client";

import { useState } from "react";
import { Alert, Button, Form, Input } from "antd";
import { ApiError } from "@/lib/api";
import type { PromptTemplateInput } from "@/lib/types";

interface Props {
  initial?: PromptTemplateInput;
  submitLabel: string;
  onSubmit: (input: PromptTemplateInput) => Promise<void>;
}

const EMPTY: PromptTemplateInput = { name: "", body: "", description: "" };

export default function PromptTemplateForm({
  initial,
  submitLabel,
  onSubmit,
}: Props) {
  const [form] = Form.useForm<PromptTemplateInput>();
  const [formError, setFormError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function handleFinish(values: PromptTemplateInput) {
    setBusy(true);
    setFormError(null);
    try {
      await onSubmit({
        name: values.name.trim(),
        body: values.body.trim(),
        description: (values.description ?? "").trim(),
      });
    } catch (err) {
      if (err instanceof ApiError && err.fieldErrors) {
        form.setFields(
          Object.entries(err.fieldErrors).map(([name, message]) => ({
            name: name as keyof PromptTemplateInput,
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
          { max: 120, message: "Name must be 120 characters or fewer." },
        ]}
      >
        <Input placeholder="Cabinet deep-dive" />
      </Form.Item>

      <Form.Item
        label="Template body"
        name="body"
        rules={[
          { required: true, message: "Template body is required." },
          { max: 5000, message: "Template body must be 5000 characters or fewer." },
        ]}
      >
        <Input.TextArea
          rows={8}
          placeholder="Write a 60s script about the arcade cabinet …"
        />
      </Form.Item>

      <Form.Item
        label="Description"
        name="description"
        rules={[{ max: 600, message: "Description must be 600 characters or fewer." }]}
      >
        <Input.TextArea rows={2} placeholder="When to use this template…" />
      </Form.Item>

      <Form.Item style={{ marginBottom: 0 }}>
        <Button type="primary" htmlType="submit" loading={busy}>
          {submitLabel}
        </Button>
      </Form.Item>
    </Form>
  );
}
