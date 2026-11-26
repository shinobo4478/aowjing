"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { App, Button, Input, Modal, Select, Spin, Typography } from "antd";
import { listProducts } from "@/lib/api";
import { productSlotsNeeded } from "@/lib/template";
import { MAX_PRODUCTS_PER_CLIP, type Product, type PromptTemplate } from "@/lib/types";
import { mobileModal } from "@/lib/ui";

/** What the owner chose in the dialog. `imageUrl` is "" for a text-only run. */
export interface GenerateChoices {
  productIds: string[];
  imageUrl: string;
}

/** Mirrors the backend check (generations.validateImageURL): an absolute
 *  http(s) link. "" is fine — it means no image. */
export function imageUrlError(raw: string): string | null {
  const s = raw.trim();
  if (!s) return null;
  try {
    const u = new URL(s);
    if (u.protocol !== "https:" && u.protocol !== "http:") throw new Error();
  } catch {
    return "Paste the full image link, starting with https://.";
  }
  return s.length > 1000 ? "Image link must be 1000 characters or fewer." : null;
}

/**
 * Asked before generating when there is something to choose: which products a
 * template that mentions them should use ({{product.name}} …; the first pick is
 * {{product.*}}, the second {{product2.*}}, and they are pinned to the new
 * clip), and — for a profile that makes video with fal — an optional image to
 * start the clip from.
 */
export default function GenerateModal({
  template,
  askImage,
  onClose,
  onGenerate,
}: {
  template: PromptTemplate | null;
  /** The profile's provider is fal, so an image can be given. */
  askImage: boolean;
  onClose: () => void;
  onGenerate: (choices: GenerateChoices) => Promise<void>;
}) {
  return (
    <Modal
      title={template ? `Generate “${template.name}”` : "Generate"}
      open={!!template}
      onCancel={onClose}
      footer={null}
      destroyOnHidden
      {...mobileModal}
    >
      {template && (
        <Choices template={template} askImage={askImage} onGenerate={onGenerate} />
      )}
    </Modal>
  );
}

function Choices({
  template,
  askImage,
  onGenerate,
}: {
  template: PromptTemplate;
  askImage: boolean;
  onGenerate: (choices: GenerateChoices) => Promise<void>;
}) {
  const { message } = App.useApp();
  const need = productSlotsNeeded(template.body);
  // null = still loading; stays [] when no products are needed.
  const [catalog, setCatalog] = useState<Product[] | null>(need > 0 ? null : []);
  const [chosen, setChosen] = useState<string[]>([]);
  const [imageUrl, setImageUrl] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (need === 0) return;
    let cancelled = false;
    listProducts()
      .then((p) => {
        if (!cancelled) setCatalog(p);
      })
      .catch((err) => {
        message.error(
          err instanceof Error ? err.message : "Failed to load products.",
        );
        if (!cancelled) setCatalog([]);
      });
    return () => {
      cancelled = true;
    };
  }, [need, message]);

  if (catalog === null) {
    return (
      <div style={{ padding: 32, textAlign: "center" }}>
        <Spin />
      </div>
    );
  }

  if (need > 0 && catalog.length === 0) {
    return (
      <Typography.Paragraph type="secondary">
        This template mentions a product, but the catalog is empty.{" "}
        <Link href="/products">Add a product</Link> first.
      </Typography.Paragraph>
    );
  }

  const imageError = askImage ? imageUrlError(imageUrl) : null;

  async function go() {
    setBusy(true);
    try {
      await onGenerate({
        productIds: chosen,
        imageUrl: askImage ? imageUrl.trim() : "",
      });
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      {need > 0 && (
        <>
          <Typography.Paragraph type="secondary">
            {need === 1
              ? "This template mentions a product. Choose one."
              : "This template mentions two products. Choose two: the first fills {{product.…}}, the second {{product2.…}}."}
          </Typography.Paragraph>
          <Select
            mode="multiple"
            maxCount={MAX_PRODUCTS_PER_CLIP}
            value={chosen}
            onChange={setChosen}
            optionFilterProp="label"
            placeholder={`Choose ${need} product${need === 1 ? "" : "s"}`}
            options={catalog.map((p) => ({ value: p.id, label: p.name }))}
            style={{ width: "100%" }}
          />
        </>
      )}

      {askImage && (
        <div style={{ marginTop: need > 0 ? 20 : 0 }}>
          <Typography.Text>Starting image (optional)</Typography.Text>
          <Input
            value={imageUrl}
            onChange={(e) => setImageUrl(e.target.value)}
            placeholder="https://…"
            status={imageError ? "error" : undefined}
            allowClear
            style={{ marginTop: 4 }}
          />
          <Typography.Paragraph
            type={imageError ? "danger" : "secondary"}
            style={{ margin: "4px 0 0" }}
          >
            {imageError ??
              "A public link, at least 300×300 px. With an image the clip is silent and 5 seconds long; leave it empty to generate from the prompt alone."}
          </Typography.Paragraph>
        </div>
      )}

      <Button
        type="primary"
        style={{ marginTop: 20 }}
        disabled={chosen.length < need || !!imageError}
        loading={busy}
        onClick={go}
      >
        Generate
      </Button>
    </>
  );
}
