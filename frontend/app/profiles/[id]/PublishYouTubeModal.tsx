"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { App, Button, Form, Input, Modal, Select, Spin, Typography } from "antd";
import { getProductBlock, publishToYouTube } from "@/lib/api";
import type { YouTubeAccount, YouTubePrivacy } from "@/lib/types";
import { mobileModal } from "@/lib/ui";

interface Values {
  channelId: string;
  title: string;
  description?: string;
  productBlock?: string;
  privacy: YouTubePrivacy;
}

/** YouTube's description limit; the API enforces it too. */
const MAX_DESCRIPTION = 5000;

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
  return (
    <Modal
      title="Publish to YouTube"
      open={open}
      onCancel={onClose}
      footer={null}
      destroyOnHidden
      {...mobileModal}
    >
      {/* Mounted only while open, so each publish starts fresh and fetches
          the clip's product list when the dialog opens. */}
      {generationId && (
        <PublishForm
          generationId={generationId}
          accounts={accounts}
          defaultTitle={defaultTitle}
          onPublished={onPublished}
          onClose={onClose}
        />
      )}
    </Modal>
  );
}

function PublishForm({
  generationId,
  accounts,
  defaultTitle,
  onPublished,
  onClose,
}: {
  generationId: string;
  accounts: YouTubeAccount[];
  defaultTitle: string;
  onPublished: () => void;
  onClose: () => void;
}) {
  const { message } = App.useApp();
  const [form] = Form.useForm<Values>();
  const [busy, setBusy] = useState(false);
  const [block, setBlock] = useState<{ text: string; count: number } | null>(
    null,
  );

  useEffect(() => {
    let cancelled = false;
    getProductBlock(generationId)
      .then((r) => {
        if (!cancelled) setBlock({ text: r.block, count: r.count });
      })
      .catch(() => {
        // Publishing still works without the product list.
        if (!cancelled) setBlock({ text: "", count: 0 });
      });
    return () => {
      cancelled = true;
    };
  }, [generationId]);

  if (block === null) {
    return (
      <div style={{ padding: 32, textAlign: "center" }}>
        <Spin />
      </div>
    );
  }

  async function onFinish(v: Values) {
    // The product list goes after whatever the description already says.
    const description = [v.description?.trim(), v.productBlock?.trim()]
      .filter(Boolean)
      .join("\n\n");
    if (description.length > MAX_DESCRIPTION) {
      message.error(
        `The description and product list are ${description.length} characters; YouTube allows ${MAX_DESCRIPTION}. Shorten one of them.`,
      );
      return;
    }

    setBusy(true);
    try {
      await publishToYouTube({
        generationId,
        channelId: v.channelId,
        title: v.title.trim(),
        description,
        privacy: v.privacy,
      });
      message.success("Uploaded to YouTube.");
      onPublished();
      onClose();
    } catch (err) {
      message.error(err instanceof Error ? err.message : "Publish failed.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <Form
      form={form}
      layout="vertical"
      requiredMark={false}
      initialValues={{
        channelId: accounts[0]?.channelId,
        title: defaultTitle,
        description: "",
        productBlock: block.text,
        privacy: "private" as YouTubePrivacy,
      }}
      onFinish={onFinish}
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

      <Form.Item label="Description" name="description">
        <Input.TextArea rows={4} />
      </Form.Item>

      {block.count > 0 ? (
        <Form.Item
          label="Pinned products"
          name="productBlock"
          extra="Added to the end of the description. Edit or clear it before you publish. Tagging the products on the video itself is done afterwards in YouTube Studio."
        >
          <Input.TextArea rows={7} />
        </Form.Item>
      ) : (
        <Typography.Paragraph type="secondary" style={{ marginTop: -8 }}>
          No products are pinned to this clip. Pin up to two from its row, or
          add some in the <Link href="/products">catalog</Link>.
        </Typography.Paragraph>
      )}

      <Form.Item label="Visibility" name="privacy" style={{ marginBottom: 8 }}>
        <Select
          options={[
            { value: "private", label: "Private" },
            { value: "unlisted", label: "Unlisted" },
            { value: "public", label: "Public" },
          ]}
        />
      </Form.Item>

      <Typography.Paragraph type="secondary" style={{ fontSize: 13 }}>
        An unverified Google project always uploads as private. Change the
        visibility afterwards in YouTube Studio.
      </Typography.Paragraph>

      <Button type="primary" htmlType="submit" loading={busy}>
        Publish
      </Button>
    </Form>
  );
}
