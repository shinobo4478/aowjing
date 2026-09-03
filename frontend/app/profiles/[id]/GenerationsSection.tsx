"use client";

import { useEffect, useState } from "react";
import { App, Button, Popconfirm, Table, Tag, Typography } from "antd";
import type { TableColumnsType } from "antd";
import { deleteGeneration, listGenerations } from "@/lib/api";
import type { Generation } from "@/lib/types";

export default function GenerationsSection({
  profileId,
  refreshTick,
}: {
  profileId: string;
  refreshTick: number;
}) {
  const { message } = App.useApp();
  const [generations, setGenerations] = useState<Generation[]>([]);
  const [loading, setLoading] = useState(true);

  // Re-runs on mount and whenever refreshTick changes (a new generation).
  // `loading` starts true for the first paint; later refetches swap the rows in
  // without a spinner flash, so no synchronous setState here.
  useEffect(() => {
    let cancelled = false;
    listGenerations(profileId)
      .then((g) => {
        if (!cancelled) setGenerations(g);
      })
      .catch((err) => {
        if (!cancelled)
          message.error(
            err instanceof Error ? err.message : "Failed to load generations.",
          );
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [profileId, refreshTick, message]);

  async function handleDelete(id: string) {
    try {
      await deleteGeneration(id);
      message.success("Generation deleted.");
      setGenerations((gs) => gs.filter((g) => g.id !== id));
    } catch (err) {
      message.error(err instanceof Error ? err.message : "Failed to delete.");
    }
  }

  const columns: TableColumnsType<Generation> = [
    {
      title: "Template",
      dataIndex: "templateName",
      render: (v: string) => v || <Typography.Text type="secondary">—</Typography.Text>,
    },
    {
      title: "Provider",
      dataIndex: "provider",
      width: 120,
      render: (v: string, row) => (row.model ? `${v} / ${row.model}` : v),
    },
    {
      title: "Status",
      dataIndex: "status",
      width: 110,
      render: (v: Generation["status"]) => (
        <Tag color={v === "succeeded" ? "green" : "red"}>{v}</Tag>
      ),
    },
    {
      title: "Created",
      dataIndex: "createdAt",
      width: 180,
      render: (v: string) => new Date(v).toLocaleString(),
    },
    {
      title: "",
      key: "actions",
      width: 90,
      render: (_, row) => (
        <Popconfirm
          title="Delete this generation?"
          placement="topRight"
          okText="Delete"
          okButtonProps={{ danger: true }}
          onConfirm={() => handleDelete(row.id)}
        >
          <Button size="small" danger>
            Delete
          </Button>
        </Popconfirm>
      ),
    },
  ];

  return (
    <section style={{ marginTop: 40 }}>
      <Typography.Title level={4} style={{ marginTop: 0 }}>
        Generations
      </Typography.Title>
      <Typography.Paragraph type="secondary" style={{ marginTop: -8 }}>
        History of running a prompt template through the AI provider.
      </Typography.Paragraph>

      <Table<Generation>
        rowKey="id"
        columns={columns}
        dataSource={generations}
        loading={loading}
        pagination={false}
        scroll={{ x: 720 }}
        expandable={{
          expandedRowRender: (row) => (
            <div style={{ display: "grid", gap: 12 }}>
              <div>
                <Typography.Text type="secondary">Prompt sent</Typography.Text>
                <pre style={preStyle}>{row.inputPrompt}</pre>
              </div>
              <div>
                <Typography.Text type="secondary">
                  {row.status === "failed" ? "Error" : "Output"}
                </Typography.Text>
                <pre style={preStyle}>
                  {row.status === "failed" ? row.error : row.output}
                </pre>
              </div>
            </div>
          ),
        }}
        locale={{ emptyText: "Nothing generated yet." }}
      />
    </section>
  );
}

const preStyle: React.CSSProperties = {
  margin: "4px 0 0",
  whiteSpace: "pre-wrap",
  fontFamily: "inherit",
};
