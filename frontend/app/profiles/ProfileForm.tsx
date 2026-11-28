"use client";

import { useState } from "react";
import { Alert, Button, Form, Input, Select, Switch } from "antd";
import { ApiError } from "@/lib/api";
import {
  DEFAULT_VIDEO_MODEL,
  DEFAULT_VIDEO_STYLE,
  PROVIDER_OPTIONS,
  VIDEO_MODEL_OPTIONS,
  VIDEO_STYLE_OPTIONS,
  type ProfileInput,
} from "@/lib/types";

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
  videoStyle: DEFAULT_VIDEO_STYLE,
  videoModel: DEFAULT_VIDEO_MODEL,
  videoAudio: false,
  lookNotes: "",
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
        // The style field is only on screen for fal profiles, and onFinish
        // leaves out fields that are not on screen; read the stored value so a
        // text profile keeps (and sends) its style.
        videoStyle: form.getFieldValue("videoStyle") ?? DEFAULT_VIDEO_STYLE,
        videoModel: form.getFieldValue("videoModel") ?? DEFAULT_VIDEO_MODEL,
        videoAudio: !!form.getFieldValue("videoAudio"),
        lookNotes: String(form.getFieldValue("lookNotes") ?? "").trim(),
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

      {/* Only a fal profile makes video, so only it has a video style. */}
      <Form.Item noStyle dependencies={["provider"]}>
        {({ getFieldValue }) =>
          getFieldValue("provider") === "fal" ? (
            <Form.Item
              label="Video style"
              name="videoStyle"
              rules={[{ required: true, message: "Pick a video style." }]}
              extra="How this profile's clips look. The style decides the wording added to prompts; the video model is chosen below."
            >
              <Select options={VIDEO_STYLE_OPTIONS} />
            </Form.Item>
          ) : null
        }
      </Form.Item>

      <Form.Item noStyle dependencies={["provider"]}>
        {({ getFieldValue }) =>
          getFieldValue("provider") === "fal" ? (
            <Form.Item
              label="Look notes (optional)"
              name="lookNotes"
              rules={[{ max: 1200, message: "Look notes must be 1200 characters or fewer." }]}
              extra="This profile's OWN look, written in English: outline weight, colours (names or hex codes), how things are drawn. Added to every picture prompt of this profile, on top of the style above. Every profile can look different. A picture of the look (a “style” member in the Cast tab) holds it better than words."
            >
              <Input.TextArea
                rows={4}
                placeholder="e.g. thick black outlines, warm pastel palette (peach #FFB88C, mint #A8E6CF, butter yellow #FFE29A), rounded friendly shapes, paper-cut feel"
              />
            </Form.Item>
          ) : null
        }
      </Form.Item>

      <Form.Item noStyle dependencies={["provider"]}>
        {({ getFieldValue }) =>
          getFieldValue("provider") === "fal" ? (
            <>
              <Form.Item
                label="Video model"
                name="videoModel"
                rules={[{ required: true, message: "Pick a video model." }]}
                extra="Which model makes this profile's clips. Models differ in price, in the clip lengths they take (Veo: 4, 6 or 8 seconds) and in how realistic they look."
              >
                <Select options={VIDEO_MODEL_OPTIONS} />
              </Form.Item>
              <Form.Item
                label="Sound"
                name="videoAudio"
                valuePropName="checked"
                extra="Ask the model to make sound too. Off by default: narration is added separately, and clips with sound cost more."
              >
                <Switch checkedChildren="On" unCheckedChildren="Off" />
              </Form.Item>
            </>
          ) : null
        }
      </Form.Item>

      <Form.Item style={{ marginBottom: 0 }}>
        <Button type="primary" htmlType="submit" loading={busy}>
          {submitLabel}
        </Button>
      </Form.Item>
    </Form>
  );
}
