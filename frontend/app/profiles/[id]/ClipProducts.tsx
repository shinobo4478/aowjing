"use client";

import { useState } from "react";
import Link from "next/link";
import { App, Select, Typography } from "antd";
import { setPlacements } from "@/lib/api";
import {
  MAX_PRODUCTS_PER_CLIP,
  type Placement,
  type Product,
} from "@/lib/types";

/**
 * Pick up to two catalog products to pin to a clip. Each change saves right
 * away (the whole list is replaced, so the order you pick in is the pin
 * order). The pins belong to the clip, not to a channel — every platform it is
 * published to renders them its own way.
 */
export default function ClipProducts({
  generationId,
  catalog,
  pinned,
  onChange,
}: {
  generationId: string;
  catalog: Product[];
  /** This clip's current pins, in position order. */
  pinned: Placement[];
  onChange: (next: Placement[]) => void;
}) {
  const { message } = App.useApp();
  const [busy, setBusy] = useState(false);

  async function save(ids: string[]) {
    setBusy(true);
    try {
      onChange(await setPlacements(generationId, ids));
      message.success(ids.length ? "Products updated." : "Products removed.");
    } catch (err) {
      message.error(
        err instanceof Error ? err.message : "Couldn't update the products.",
      );
    } finally {
      setBusy(false);
    }
  }

  if (catalog.length === 0) {
    return (
      <Typography.Text type="secondary">
        The catalog is empty. <Link href="/products">Add a product</Link> first,
        then pin it here.
      </Typography.Text>
    );
  }

  return (
    <Select
      mode="multiple"
      maxCount={MAX_PRODUCTS_PER_CLIP}
      value={pinned.map((p) => p.product.id)}
      onChange={save}
      loading={busy}
      disabled={busy}
      optionFilterProp="label"
      placeholder="Pin up to 2 products"
      options={catalog.map((p) => ({ value: p.id, label: p.name }))}
      style={{ width: "100%", maxWidth: 480 }}
    />
  );
}
