"use client";

import { useCallback, useEffect, useState } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import {
  App,
  Button,
  Flex,
  Modal,
  Popconfirm,
  Space,
  Table,
  Tag,
  Typography,
} from "antd";
import type { TableColumnsType } from "antd";
import {
  createChannel,
  deleteChannel,
  disconnectYouTube,
  listChannels,
  listYouTubeAccounts,
  updateChannel,
  youtubeConnectUrl,
} from "@/lib/api";
import type { Channel, YouTubeAccount } from "@/lib/types";
import { mobileModal } from "@/lib/ui";
import ChannelForm from "../ChannelForm";

export default function ChannelsSection({ profileId }: { profileId: string }) {
  const { message } = App.useApp();
  const router = useRouter();
  const pathname = usePathname();
  const search = useSearchParams();
  const [channels, setChannels] = useState<Channel[]>([]);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [editing, setEditing] = useState<Channel | null>(null);

  // YouTube connection state, one fetch for the whole table.
  const [ytConfigured, setYtConfigured] = useState(false);
  const [ytByChannel, setYtByChannel] = useState<Record<string, YouTubeAccount>>(
    {},
  );

  const loadYouTube = useCallback(() => {
    listYouTubeAccounts(profileId)
      .then(({ configured, accounts }) => {
        setYtConfigured(configured);
        setYtByChannel(
          Object.fromEntries(accounts.map((a) => [a.channelId, a])),
        );
      })
      .catch(() => {
        /* non-fatal — the column just won't show */
      });
  }, [profileId]);

  useEffect(() => {
    listChannels(profileId)
      .then(setChannels)
      .catch((err) =>
        message.error(
          err instanceof Error ? err.message : "Failed to load channels.",
        ),
      )
      .finally(() => setLoading(false));
    loadYouTube();
  }, [profileId, message, loadYouTube]);

  // The OAuth callback lands back here with ?yt=connected | ?yt=error&reason=…
  useEffect(() => {
    const yt = search.get("yt");
    if (!yt) return;
    if (yt === "connected") {
      message.success("YouTube channel connected.");
      loadYouTube();
    } else {
      message.error(
        `YouTube connection failed (${search.get("reason") ?? "unknown"}).`,
      );
    }
    router.replace(pathname);
  }, [search, pathname, router, message, loadYouTube]);

  async function reload() {
    setLoading(true);
    try {
      setChannels(await listChannels(profileId));
    } catch (err) {
      message.error(
        err instanceof Error ? err.message : "Failed to load channels.",
      );
    } finally {
      setLoading(false);
    }
  }

  async function handleDelete(id: string) {
    try {
      await deleteChannel(id);
      message.success("Channel deleted.");
      await reload();
    } catch (err) {
      message.error(err instanceof Error ? err.message : "Failed to delete.");
    }
  }

  async function handleDisconnect(channelId: string) {
    try {
      await disconnectYouTube(channelId);
      message.success("YouTube disconnected.");
      loadYouTube();
    } catch (err) {
      message.error(
        err instanceof Error ? err.message : "Failed to disconnect.",
      );
    }
  }

  const columns: TableColumnsType<Channel> = [
    { title: "Name", dataIndex: "name" },
    { title: "Platform", dataIndex: "platform" },
    { title: "Handle", dataIndex: "handle" },
    ...(ytConfigured
      ? ([
          {
            title: "YouTube",
            key: "youtube",
            width: 220,
            render: (_, row) => {
              const acct = ytByChannel[row.id];
              if (acct) {
                return (
                  <Space size="small" wrap>
                    <Tag color="red" style={{ marginInlineEnd: 0 }}>
                      ▶ {acct.youtubeTitle || "connected"}
                    </Tag>
                    <Popconfirm
                      title="Disconnect this YouTube account?"
                      okText="Disconnect"
                      okButtonProps={{ danger: true }}
                      onConfirm={() => handleDisconnect(row.id)}
                    >
                      <Button size="small" type="text">
                        Disconnect
                      </Button>
                    </Popconfirm>
                  </Space>
                );
              }
              return (
                <Button
                  size="small"
                  onClick={() => {
                    window.location.href = youtubeConnectUrl(row.id);
                  }}
                >
                  Connect
                </Button>
              );
            },
          },
        ] as TableColumnsType<Channel>)
      : []),
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
            title="Delete this channel?"
            okText="Delete"
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

  return (
    <section style={{ marginTop: 40 }}>
      <Flex justify="space-between" align="center" wrap gap={12}>
        <Typography.Title level={4} style={{ margin: 0 }}>
          Channels
        </Typography.Title>
        <Button type="primary" size="small" onClick={() => setCreating(true)}>
          New channel
        </Button>
      </Flex>

      <Table<Channel>
        rowKey="id"
        columns={columns}
        dataSource={channels}
        loading={loading}
        pagination={false}
        style={{ marginTop: 12 }}
        scroll={{ x: "max-content" }}
        locale={{ emptyText: "No channels yet." }}
      />

      <Modal
        title="New channel"
        open={creating}
        onCancel={() => setCreating(false)}
        footer={null}
        destroyOnHidden
        {...mobileModal}
      >
        <ChannelForm
          submitLabel="Create"
          onSubmit={async (input) => {
            await createChannel(profileId, input);
            setCreating(false);
            message.success("Channel created.");
            await reload();
          }}
        />
      </Modal>

      <Modal
        title="Edit channel"
        open={!!editing}
        onCancel={() => setEditing(null)}
        footer={null}
        destroyOnHidden
        {...mobileModal}
      >
        {editing && (
          <ChannelForm
            initial={{
              name: editing.name,
              platform: editing.platform,
              handle: editing.handle,
              description: editing.description,
            }}
            submitLabel="Save changes"
            onSubmit={async (input) => {
              await updateChannel(editing.id, input);
              setEditing(null);
              message.success("Channel updated.");
              await reload();
            }}
          />
        )}
      </Modal>
    </section>
  );
}
