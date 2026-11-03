"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { App, Button, Modal, Select, Spin, Typography } from "antd";
import { listProducts } from "@/lib/api";
import { productSlotsNeeded } from "@/lib/template";
import { MAX_PRODUCTS_PER_CLIP, type Product, type PromptTemplate } from "@/lib/types";
import { mobileModal } from "@/lib/ui";

/**
 * Asked before generating from a template that mentions products
 * ({{product.name}} …): which products to use. The pick order matters — the
 * first is {{product.*}}, the second {{product2.*}}. They are also pinned to
 * the new clip.
 */
export default function GenerateWithProductsModal({
  template,
  onClose,
  onGenerate,
}: {
  template: PromptTemplate | null;
  onClose: () => void;
  onGenerate: (productIds: string[]) => Promise<void>;
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
      {template && <Picker template={template} onGenerate={onGenerate} />}
    </Modal>
  );
}

function Picker({
  template,
  onGenerate,
}: {
  template: PromptTemplate;
  onGenerate: (productIds: string[]) => Promise<void>;
}) {
  const { message } = App.useApp();
  const need = productSlotsNeeded(template.body);
  const [catalog, setCatalog] = useState<Product[] | null>(null);
  const [chosen, setChosen] = useState<string[]>([]);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
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
  }, [message]);

  if (catalog === null) {
    return (
      <div style={{ padding: 32, textAlign: "center" }}>
        <Spin />
      </div>
    );
  }

  if (catalog.length === 0) {
    return (
      <Typography.Paragraph type="secondary">
        This template mentions a product, but the catalog is empty.{" "}
        <Link href="/products">Add a product</Link> first.
      </Typography.Paragraph>
    );
  }

  async function go() {
    setBusy(true);
    try {
      await onGenerate(chosen);
    } finally {
      setBusy(false);
    }
  }

  return (
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
      <Button
        type="primary"
        style={{ marginTop: 20 }}
        disabled={chosen.length < need}
        loading={busy}
        onClick={go}
      >
        Generate
      </Button>
    </>
  );
}
