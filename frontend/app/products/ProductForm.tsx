"use client";

import { useState } from "react";
import { Alert, Button, Form, Input, InputNumber } from "antd";
import { ApiError } from "@/lib/api";
import type { ProductInput } from "@/lib/types";

interface Props {
  initial?: ProductInput;
  submitLabel: string;
  onSubmit: (input: ProductInput) => Promise<void>;
}

const EMPTY: ProductInput = {
  name: "",
  shopeeUrl: "",
  affiliateUrl: "",
  priceThb: 0,
  note: "",
};

export default function ProductForm({ initial, submitLabel, onSubmit }: Props) {
  const [form] = Form.useForm<ProductInput>();
  const [formError, setFormError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function handleFinish(values: ProductInput) {
    setBusy(true);
    setFormError(null);
    try {
      await onSubmit({
        name: values.name.trim(),
        shopeeUrl: values.shopeeUrl.trim(),
        affiliateUrl: (values.affiliateUrl ?? "").trim(),
        priceThb: values.priceThb ?? 0,
        note: (values.note ?? "").trim(),
      });
    } catch (err) {
      if (err instanceof ApiError && err.fieldErrors) {
        form.setFields(
          Object.entries(err.fieldErrors).map(([name, message]) => ({
            name: name as keyof ProductInput,
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
        <Input placeholder="Power strip, 4 outlets" />
      </Form.Item>

      <Form.Item
        label="Shopee link"
        name="shopeeUrl"
        rules={[{ required: true, message: "Paste the product's Shopee link." }]}
        extra="The product page on shopee.co.th. Tracking parameters are removed when you save."
      >
        <Input placeholder="https://shopee.co.th/…" autoComplete="off" />
      </Form.Item>

      <Form.Item
        label="Affiliate link"
        name="affiliateUrl"
        extra="Your own tracked link from the Shopee Affiliate app. Leave empty until you have one."
      >
        <Input placeholder="https://s.shopee.co.th/…" autoComplete="off" />
      </Form.Item>

      <Form.Item label="Price (baht)" name="priceThb">
        <InputNumber min={0} max={10000000} precision={0} style={{ width: 160 }} />
      </Form.Item>

      <Form.Item
        label="Note"
        name="note"
        rules={[{ max: 600, message: "Note must be 600 characters or fewer." }]}
      >
        <Input.TextArea rows={2} placeholder="Why it fits, commission rate, variants…" />
      </Form.Item>

      <Form.Item style={{ marginBottom: 0 }}>
        <Button type="primary" htmlType="submit" loading={busy}>
          {submitLabel}
        </Button>
      </Form.Item>
    </Form>
  );
}
