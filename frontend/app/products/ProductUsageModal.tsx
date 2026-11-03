"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { App, Modal, Spin, Typography } from "antd";
import { getProductUsage } from "@/lib/api";
import { formatDateTime } from "@/lib/format";
import type { ProductUsage, ProductWithUse } from "@/lib/types";
import { mobileModal } from "@/lib/ui";

/** Where a product is pinned. Unpin it from these clips before deleting it. */
export default function ProductUsageModal({
  product,
  onClose,
}: {
  product: ProductWithUse | null;
  onClose: () => void;
}) {
  return (
    <Modal
      title={product ? `Pinned: ${product.name}` : "Pinned"}
      open={!!product}
      onCancel={onClose}
      footer={null}
      destroyOnHidden
      {...mobileModal}
    >
      {product && <Clips productId={product.id} />}
    </Modal>
  );
}

function Clips({ productId }: { productId: string }) {
  const { message } = App.useApp();
  const [clips, setClips] = useState<ProductUsage[] | null>(null);

  useEffect(() => {
    let cancelled = false;
    getProductUsage(productId)
      .then((c) => {
        if (!cancelled) setClips(c);
      })
      .catch((err) => {
        message.error(
          err instanceof Error ? err.message : "Failed to load usage.",
        );
        if (!cancelled) setClips([]);
      });
    return () => {
      cancelled = true;
    };
  }, [productId, message]);

  if (clips === null) {
    return (
      <div style={{ padding: 32, textAlign: "center" }}>
        <Spin />
      </div>
    );
  }
  if (clips.length === 0) {
    return (
      <Typography.Text type="secondary">
        Not pinned to any clip.
      </Typography.Text>
    );
  }

  return (
    <div style={{ display: "grid", gap: 14 }}>
      {clips.map((c) => (
        <div key={c.generationId}>
          <Link href={`/profiles/${c.profileId}?tab=generations`}>
            {c.profileName}
          </Link>
          <div>
            <Typography.Text type="secondary">
              {c.templateName || "Deleted template"}, {formatDateTime(c.createdAt)},
              product {c.position}
            </Typography.Text>
          </div>
        </div>
      ))}
    </div>
  );
}
