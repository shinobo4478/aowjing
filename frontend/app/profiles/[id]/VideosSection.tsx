"use client";

import { useEffect, useState } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { Alert, App, Button, Empty, Flex, Form, Input, Modal, Popconfirm, Space, Table } from "antd";
import type { TableColumnsType } from "antd";
import { ApiError, createVideo, deleteVideo, listVideos } from "@/lib/api";
import { formatDateTime } from "@/lib/format";
import type { ProviderKey, Video } from "@/lib/types";
import { mobileModal } from "@/lib/ui";
import DraftModal from "./DraftModal";
import VideoEditor from "./VideoEditor";

/**
 * A profile's videos: each an ordered list of shots with its own prompts,
 * lengths and starting pictures. The open video is kept in the URL
 * (?tab=videos&video=<id>) so a reload or a link returns to it.
 */
export default function VideosSection({
  profileId,
  provider,
  videoModel,
  videoAudio,
}: {
  profileId: string;
  provider: ProviderKey;
  videoModel: string;
  videoAudio: boolean;
}) {
  const { message } = App.useApp();
  const router = useRouter();
  const pathname = usePathname();
  const search = useSearchParams();
  const openId = search.get("video");

  const [videos, setVideos] = useState<Video[]>([]);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [drafting, setDrafting] = useState(false);
  const [title, setTitle] = useState("");
  const [titleError, setTitleError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  // Reloads the list whenever it is shown again (it comes back from an open video).
  useEffect(() => {
    if (openId) return;
    let cancelled = false;
    listVideos(profileId)
      .then((v) => {
        if (!cancelled) setVideos(v);
      })
      .catch((err) =>
        message.error(err instanceof Error ? err.message : "Failed to load videos."),
      )
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [profileId, openId, message]);

  const open = (id: string) => router.replace(`${pathname}?tab=videos&video=${id}`);
  const backToList = () => router.replace(`${pathname}?tab=videos`);

  async function create() {
    setBusy(true);
    setTitleError(null);
    try {
      const v = await createVideo(profileId, title);
      setCreating(false);
      setTitle("");
      open(v.id);
    } catch (err) {
      const field =
        err instanceof ApiError && err.fieldErrors ? err.fieldErrors.title : undefined;
      if (field) setTitleError(field);
      else message.error(err instanceof Error ? err.message : "Failed to create the video.");
    } finally {
      setBusy(false);
    }
  }

  async function remove(id: string) {
    try {
      await deleteVideo(id);
      setVideos((all) => all.filter((v) => v.id !== id));
      message.success("Video deleted.");
    } catch (err) {
      message.error(err instanceof Error ? err.message : "Failed to delete.");
    }
  }

  if (openId) {
    return (
      <VideoEditor
        key={openId}
        videoId={openId}
        profileId={profileId}
        videoModel={videoModel}
        videoAudio={videoAudio}
        onBack={backToList}
      />
    );
  }

  const columns: TableColumnsType<Video> = [
    {
      title: "Title",
      dataIndex: "title",
      render: (v: string, row) => (
        <Button type="link" style={{ padding: 0, height: "auto" }} onClick={() => open(row.id)}>
          {v}
        </Button>
      ),
    },
    { title: "Shots", dataIndex: "shotCount", width: 90, responsive: ["md"] },
    {
      title: "Length",
      responsive: ["md"],
      dataIndex: "totalSeconds",
      width: 100,
      render: (v: number) => `${v} s`,
    },
    {
      title: "Created",
      responsive: ["md"],
      dataIndex: "createdAt",
      width: 180,
      render: (v: string) => formatDateTime(v),
    },
    {
      title: "",
      key: "actions",
      width: 170,
      render: (_, row) => (
        <Space>
          <Button size="small" onClick={() => open(row.id)}>
            Open
          </Button>
          <Popconfirm
            title="Delete this video?"
            description="Its shots are deleted. Clips already made stay in the Generations tab."
            okText="Delete"
            okButtonProps={{ danger: true }}
            onConfirm={() => remove(row.id)}
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
    <section>
      {provider !== "fal" && (
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 16, maxWidth: 640 }}
          title="Videos are made with fal.ai. Switch this profile's generator to fal.ai video to create one."
        />
      )}

      <Flex justify="end" gap={8}>
        <Button disabled={provider !== "fal"} onClick={() => setDrafting(true)}>
          Draft with AI
        </Button>
        <Button disabled={provider !== "fal"} onClick={() => setCreating(true)}>
          New video
        </Button>
      </Flex>

      <Table<Video>
        rowKey="id"
        columns={columns}
        dataSource={videos}
        loading={loading}
        pagination={false}
        style={{ marginTop: 12 }}
        scroll={{ x: "max-content" }}
        locale={{
          emptyText: (
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description="No videos yet. A video is a list of shots, each with its own prompt, length and starting picture."
            />
          ),
        }}
      />

      <DraftModal
        open={drafting}
        profileId={profileId}
        videoModel={videoModel}
        onClose={() => setDrafting(false)}
        onDrafted={(v) => {
          setDrafting(false);
          open(v.id);
        }}
      />

      <Modal
        title="New video"
        open={creating}
        onCancel={() => setCreating(false)}
        footer={null}
        destroyOnHidden
        {...mobileModal}
      >
        <Form layout="vertical" onFinish={() => void create()}>
          <Form.Item
            label="Title"
            validateStatus={titleError ? "error" : undefined}
            help={titleError ?? undefined}
          >
            <Input
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              maxLength={120}
              placeholder="The cat and the plastic bag"
              autoFocus
            />
          </Form.Item>
          <Button type="primary" htmlType="submit" loading={busy} disabled={title.trim() === ""}>
            Create
          </Button>
        </Form>
      </Modal>
    </section>
  );
}
