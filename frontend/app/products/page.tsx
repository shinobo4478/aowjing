"use client";

import { useCallback, useEffect, useState } from "react";
import {
  App,
  Button,
  Empty,
  Flex,
  Input,
  Modal,
  Popconfirm,
  Space,
  Table,
  Typography,
} from "antd";
import type { TableColumnsType } from "antd";
import {
  createProduct,
  deleteProduct,
  listProducts,
  updateProduct,
} from "@/lib/api";
import { formatDate } from "@/lib/format";
import { INK, STATUS } from "@/lib/tape";
import type { ProductWithUse } from "@/lib/types";
import { mobileModal, pageLede, pageTitle } from "@/lib/ui";
import ProductForm from "./ProductForm";
import ProductUsageModal from "./ProductUsageModal";

export default function ProductsPage() {
  const { message } = App.useApp();
  const [products, setProducts] = useState<ProductWithUse[]>([]);
  const [loading, setLoading] = useState(true);
  const [query, setQuery] = useState("");
  const [creating, setCreating] = useState(false);
  const [usageFor, setUsageFor] = useState<ProductWithUse | null>(null);
  const [editing, setEditing] = useState<ProductWithUse | null>(null);

  const load = useCallback(
    (q: string) =>
      listProducts(q)
        .then(setProducts)
        .catch((err) =>
          message.error(
            err instanceof Error ? err.message : "Failed to load products.",
          ),
        )
        .finally(() => setLoading(false)),
    [message],
  );

  // Fetch on mount and, after a short pause, whenever the search text changes.
  useEffect(() => {
    const t = setTimeout(() => load(query), query ? 250 : 0);
    return () => clearTimeout(t);
  }, [query, load]);

  async function handleDelete(id: string) {
    try {
      await deleteProduct(id);
      message.success("Product deleted.");
      await load(query);
    } catch (err) {
      message.error(err instanceof Error ? err.message : "Failed to delete.");
    }
  }

  const columns: TableColumnsType<ProductWithUse> = [
    {
      title: "Product",
      dataIndex: "name",
      render: (name: string, row) => (
        <div style={{ minWidth: 200 }}>
          <a href={row.shopeeUrl} target="_blank" rel="noreferrer">
            {name}
          </a>
          {row.note && (
            <div>
              <Typography.Text type="secondary">{row.note}</Typography.Text>
            </div>
          )}
          {!row.idsKnown && (
            <div>
              <Typography.Text type="secondary" style={{ fontSize: 13 }}>
                Short link: duplicates can&apos;t be detected.
              </Typography.Text>
            </div>
          )}
        </div>
      ),
    },
    {
      title: "Price",
      dataIndex: "priceThb",
      width: 110,
      align: "right",
      render: (v: number) =>
        v > 0 ? `฿${v.toLocaleString("en-US")}` : (
          <Typography.Text type="secondary">Not set</Typography.Text>
        ),
    },
    {
      title: "Affiliate link",
      dataIndex: "affiliateUrl",
      width: 160,
      render: (v: string) => (
        <span className="acmp-tally">
          <span
            aria-hidden
            className="acmp-tally-dot"
            style={{ background: v ? STATUS.succeeded : INK.dim }}
          />
          {v ? "Added" : "Not added"}
        </span>
      ),
    },
    {
      title: "Pinned to",
      dataIndex: "usedCount",
      width: 120,
      render: (n: number, row) =>
        n > 0 ? (
          <Button type="link" size="small" onClick={() => setUsageFor(row)}>
            {n} {n === 1 ? "clip" : "clips"}
          </Button>
        ) : (
          <Typography.Text type="secondary">No clips</Typography.Text>
        ),
    },
    {
      title: "Updated",
      dataIndex: "updatedAt",
      width: 130,
      render: (v: string) => formatDate(v),
    },
    {
      title: "",
      key: "actions",
      width: 150,
      render: (_, row) => (
        <Space>
          <Button size="small" onClick={() => setEditing(row)}>
            Edit
          </Button>
          <Popconfirm
            title="Delete this product?"
            okText="Delete product"
            okButtonProps={{ danger: true }}
            onConfirm={() => handleDelete(row.id)}
          >
            <Button size="small" danger>
              Delete
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const searching = query.trim() !== "";

  return (
    <>
      <Typography.Title level={1} style={pageTitle}>
        Products
      </Typography.Title>
      <Typography.Paragraph type="secondary" style={pageLede}>
        Shopee products you can pin to a clip, up to two per clip. One list,
        shared by every profile.
      </Typography.Paragraph>

      <Flex
        justify="space-between"
        align="center"
        wrap
        gap={12}
        style={{ marginTop: 44 }}
      >
        <Input.Search
          allowClear
          placeholder="Search products"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onSearch={(v) => setQuery(v)}
          style={{ maxWidth: 360, flex: "1 1 220px" }}
        />
        <Button onClick={() => setCreating(true)}>New product</Button>
      </Flex>

      <Table<ProductWithUse>
        rowKey="id"
        columns={columns}
        dataSource={products}
        loading={loading}
        pagination={false}
        style={{ marginTop: 16 }}
        scroll={{ x: "max-content" }}
        locale={{
          emptyText: loading ? (
            <span />
          ) : searching ? (
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description={`No products match “${query.trim()}”.`}
            />
          ) : (
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description="No products yet. Paste a Shopee link to add the first one."
            >
              <Button type="primary" onClick={() => setCreating(true)}>
                Add your first product
              </Button>
            </Empty>
          ),
        }}
      />

      <ProductUsageModal product={usageFor} onClose={() => setUsageFor(null)} />

      <Modal
        title="New product"
        open={creating}
        onCancel={() => setCreating(false)}
        footer={null}
        width={560}
        destroyOnHidden
        {...mobileModal}
      >
        <ProductForm
          submitLabel="Add product"
          onSubmit={async (input) => {
            await createProduct(input);
            setCreating(false);
            message.success("Product added.");
            await load(query);
          }}
        />
      </Modal>

      <Modal
        title="Edit product"
        open={!!editing}
        onCancel={() => setEditing(null)}
        footer={null}
        width={560}
        destroyOnHidden
        {...mobileModal}
      >
        {editing && (
          <ProductForm
            initial={{
              name: editing.name,
              shopeeUrl: editing.shopeeUrl,
              affiliateUrl: editing.affiliateUrl,
              priceThb: editing.priceThb,
              note: editing.note,
            }}
            submitLabel="Save changes"
            onSubmit={async (input) => {
              await updateProduct(editing.id, input);
              setEditing(null);
              message.success("Product updated.");
              await load(query);
            }}
          />
        )}
      </Modal>
    </>
  );
}
