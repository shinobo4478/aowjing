"use client";

import { useState } from "react";
import { App, Form, Input, Modal, Select, Typography } from "antd";
import { publishToYouTube } from "@/lib/api";
import type { YouTubeAccount, YouTubePrivacy } from "@/lib/types";
import { mobileModal } from "@/lib/ui";

export default function PublishYouTubeModal({
  open,
  onClose,
  onPublished,
  generationId,
  accounts,
  defaultTitle,
}: {
  open: boolean;
  onClose: () => void;
  onPublished: () => void;
  generationId: string | null;
  /** Connected YouTube accounts to choose between. */
  accounts: YouTubeAccount[];
  defaultTitle: string;
}) {
  const { message } = App.useApp();
  const [busy, setBusy] = useState(false);
  const [form] = Form.useForm();

  return (
    <Modal
      title="Publish to YouTube"
      open={open}
      onCancel={onClose}
      okText="Publish"
      confirmLoading={busy}
      onOk={() => form.submit()}
      destroyOnHidden
      {...mobileModal}
    >
      <Form
        form={form}
        layout="vertical"
        requiredMark={false}
        preserve={false}
        initialValues={{
          channelId: accounts[0]?.channelId,
          title: defaultTitle,
          description: "",
          privacy: "private" as YouTubePrivacy,
        }}
        onFinish={async (v: {
          channelId: string;
          title: string;
          description?: string;
          privacy: YouTubePrivacy;
        }) => {
          if (!generationId) return;
          setBusy(true);
          try {
            await publishToYouTube({
              generationId,
              channelId: v.channelId,
              title: v.title.trim(),
              description: v.description ?? "",
              privacy: v.privacy,
            });
            message.success("Uploaded to YouTube.");
            onPublished();
            onClose();
          } catch (err) {
            message.error(
              err instanceof Error ? err.message : "Publish failed.",
            );
          } finally {
            setBusy(false);
          }
        }}
      >
        <Form.Item label="Channel" name="channelId" rules={[{ required: true }]}>
          <Select
            options={accounts.map((a) => ({
              value: a.channelId,
              label: a.youtubeTitle || "Connected channel",
            }))}
          />
        </Form.Item>

        <Form.Item
          label="Title"
          name="title"
          rules={[
            { required: true, message: "A title is required." },
            { max: 100, message: "Max 100 characters." },
          ]}
        >
          <Input maxLength={100} />
        </Form.Item>

        <Form.Item
          label="Description"
          name="description"
          rules={[{ max: 5000, message: "Max 5000 characters." }]}
        >
          <Input.TextArea rows={4} maxLength={5000} />
        </Form.Item>

        <Form.Item label="Visibility" name="privacy" style={{ marginBottom: 8 }}>
          <Select
            options={[
              { value: "private", label: "Private" },
              { value: "unlisted", label: "Unlisted" },
              { value: "public", label: "Public" },
            ]}
          />
        </Form.Item>

        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          An unverified Google project always uploads as private — flip the
          visibility afterwards in YouTube Studio.
        </Typography.Text>
      </Form>
    </Modal>
  );
}
