"use client";

import { useEffect, useState } from "react";
import { App, Button, Flex, Modal, Popconfirm, Space, Table, Typography } from "antd";
import type { TableColumnsType } from "antd";
import {
  createChannel,
  deleteChannel,
  listChannels,
  updateChannel,
} from "@/lib/api";
import type { Channel } from "@/lib/types";
import { mobileModal } from "@/lib/ui";
import ChannelForm from "../ChannelForm";

export default function ChannelsSection({ profileId }: { profileId: string }) {
  const { message } = App.useApp();
  const [channels, setChannels] = useState<Channel[]>([]);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [editing, setEditing] = useState<Channel | null>(null);

  useEffect(() => {
    listChannels(profileId)
      .then(setChannels)
      .catch((err) =>
        message.error(
          err instanceof Error ? err.message : "Failed to load channels.",
        ),
      )
      .finally(() => setLoading(false));
  }, [profileId, message]);

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

  const columns: TableColumnsType<Channel> = [
    { title: "Name", dataIndex: "name" },
    { title: "Platform", dataIndex: "platform" },
    { title: "Handle", dataIndex: "handle" },
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
