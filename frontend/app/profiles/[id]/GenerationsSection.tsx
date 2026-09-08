"use client";

import { useCallback, useEffect, useState } from "react";
import { App, Button, Popconfirm, Table, Tag, Typography } from "antd";
import type { TableColumnsType } from "antd";
import {
  deleteGeneration,
  listGenerations,
  listYouTubeAccounts,
  listYouTubeUploads,
} from "@/lib/api";
import type { Generation, YouTubeAccount, YouTubeUpload } from "@/lib/types";
import PublishYouTubeModal from "./PublishYouTubeModal";

const STATUS_COLOR: Record<Generation["status"], string> = {
  pending: "blue",
  succeeded: "green",
  failed: "red",
};

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

  const [ytAccounts, setYtAccounts] = useState<YouTubeAccount[]>([]);
  const [uploads, setUploads] = useState<YouTubeUpload[]>([]);
  const [publishing, setPublishing] = useState<Generation | null>(null);

  const loadYouTube = useCallback(() => {
    listYouTubeAccounts(profileId)
      .then(({ accounts }) => setYtAccounts(accounts))
      .catch(() => {
        /* non-fatal */
      });
    listYouTubeUploads(profileId)
      .then(setUploads)
      .catch(() => {
        /* non-fatal */
      });
  }, [profileId]);

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
    loadYouTube();
    return () => {
      cancelled = true;
    };
  }, [profileId, refreshTick, message, loadYouTube]);

  // While a generation is pending, poll for the worker to finish it. Stops
  // when nothing is pending or after ~1 minute.
  const hasPending = generations.some((g) => g.status === "pending");
  useEffect(() => {
    if (!hasPending) return;
    let tries = 0;
    const iv = setInterval(() => {
      tries += 1;
      listGenerations(profileId)
        .then((fresh) => {
          setGenerations(fresh);
          if (tries >= 30) clearInterval(iv);
        })
        .catch(() => clearInterval(iv));
    }, 2000);
    return () => clearInterval(iv);
  }, [hasPending, profileId]);

  async function handleDelete(id: string) {
    try {
      await deleteGeneration(id);
      message.success("Generation deleted.");
      setGenerations((gs) => gs.filter((g) => g.id !== id));
    } catch (err) {
      message.error(err instanceof Error ? err.message : "Failed to delete.");
    }
  }

  const uploadsFor = (genId: string) =>
    uploads.filter((u) => u.generationId === genId);

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
        <Tag color={STATUS_COLOR[v]}>{v}</Tag>
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
        scroll={{ x: "max-content" }}
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
                {row.status === "succeeded" && row.outputKind === "video" ? (
                  <div style={{ marginTop: 4 }}>
                    <video
                      src={row.output}
                      controls
                      style={{ maxWidth: "100%", maxHeight: 320, display: "block" }}
                    />
                    <a href={row.output} target="_blank" rel="noreferrer">
                      Open video ↗
                    </a>
                  </div>
                ) : (
                  <pre style={preStyle}>
                    {row.status === "pending"
                      ? "Waiting for the worker…"
                      : row.status === "failed"
                        ? row.error
                        : row.output}
                  </pre>
                )}
              </div>

              {row.status === "succeeded" && row.outputKind === "video" && (
                <div style={{ display: "grid", gap: 6 }}>
                  <Typography.Text type="secondary">YouTube</Typography.Text>
                  {uploadsFor(row.id).map((u) => (
                    <a
                      key={u.id}
                      href={u.url}
                      target="_blank"
                      rel="noreferrer"
                    >
                      ▶ {u.title} — {u.privacy} ↗
                    </a>
                  ))}
                  <div>
                    <Button
                      size="small"
                      disabled={ytAccounts.length === 0}
                      onClick={() => setPublishing(row)}
                    >
                      Publish to YouTube
                    </Button>
                    {ytAccounts.length === 0 && (
                      <Typography.Text
                        type="secondary"
                        style={{ fontSize: 12, marginLeft: 8 }}
                      >
                        Connect a channel to YouTube first.
                      </Typography.Text>
                    )}
                  </div>
                </div>
              )}
            </div>
          ),
        }}
        locale={{ emptyText: "Nothing generated yet." }}
      />

      <PublishYouTubeModal
        open={!!publishing}
        onClose={() => setPublishing(null)}
        onPublished={loadYouTube}
        generationId={publishing?.id ?? null}
        accounts={ytAccounts}
        defaultTitle={publishing?.templateName ?? ""}
      />
    </section>
  );
}

const preStyle: React.CSSProperties = {
  margin: "4px 0 0",
  whiteSpace: "pre-wrap",
  wordBreak: "break-word",
  fontFamily: "inherit",
};
