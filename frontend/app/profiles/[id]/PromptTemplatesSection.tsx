"use client";

import { useEffect, useState } from "react";
import { App, Button, Flex, Modal, Popconfirm, Space, Table, Typography } from "antd";
import type { TableColumnsType } from "antd";
import {
  createPromptTemplate,
  deletePromptTemplate,
  listPromptTemplates,
  runGeneration,
  updatePromptTemplate,
} from "@/lib/api";
import type { PromptTemplate } from "@/lib/types";
import { mobileModal } from "@/lib/ui";
import PromptTemplateForm from "../PromptTemplateForm";

export default function PromptTemplatesSection({
  profileId,
  onGenerated,
}: {
  profileId: string;
  /** Called after a template is run, so the Generations list can refresh. */
  onGenerated: () => void;
}) {
  const { message } = App.useApp();
  const [templates, setTemplates] = useState<PromptTemplate[]>([]);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [editing, setEditing] = useState<PromptTemplate | null>(null);
  const [running, setRunning] = useState<string | null>(null);

  useEffect(() => {
    listPromptTemplates(profileId)
      .then(setTemplates)
      .catch((err) =>
        message.error(
          err instanceof Error ? err.message : "Failed to load prompt templates.",
        ),
      )
      .finally(() => setLoading(false));
  }, [profileId, message]);

  async function reload() {
    setLoading(true);
    try {
      setTemplates(await listPromptTemplates(profileId));
    } catch (err) {
      message.error(
        err instanceof Error ? err.message : "Failed to load prompt templates.",
      );
    } finally {
      setLoading(false);
    }
  }

  async function handleDelete(id: string) {
    try {
      await deletePromptTemplate(id);
      message.success("Prompt template deleted.");
      await reload();
    } catch (err) {
      message.error(err instanceof Error ? err.message : "Failed to delete.");
    }
  }

  async function handleGenerate(id: string) {
    setRunning(id);
    try {
      await runGeneration(id);
      message.success("Generation queued.");
      onGenerated(); // the Generations list picks it up and polls for the result
    } catch (err) {
      message.error(err instanceof Error ? err.message : "Failed to generate.");
    } finally {
      setRunning(null);
    }
  }

  const columns: TableColumnsType<PromptTemplate> = [
    { title: "Name", dataIndex: "name" },
    {
      title: "Updated",
      dataIndex: "updatedAt",
      width: 190,
      render: (v: string) => new Date(v).toLocaleString(),
    },
    {
      title: "",
      key: "actions",
      width: 230,
      render: (_, row) => (
        <Space>
          <Button
            size="small"
            type="primary"
            loading={running === row.id}
            onClick={() => handleGenerate(row.id)}
          >
            Generate
          </Button>
          <Button size="small" onClick={() => setEditing(row)}>
            Edit
          </Button>
          <Popconfirm
            title="Delete this template?"
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
          Prompt templates
        </Typography.Title>
        <Button type="primary" size="small" onClick={() => setCreating(true)}>
          New template
        </Button>
      </Flex>

      <Table<PromptTemplate>
        rowKey="id"
        columns={columns}
        dataSource={templates}
        loading={loading}
        pagination={false}
        style={{ marginTop: 12 }}
        scroll={{ x: "max-content" }}
        expandable={{
          expandedRowRender: (row) => (
            <pre
              style={{
                margin: 0,
                whiteSpace: "pre-wrap",
                wordBreak: "break-word",
                fontFamily: "inherit",
              }}
            >
              {row.body}
            </pre>
          ),
        }}
        locale={{ emptyText: "No prompt templates yet." }}
      />

      <Modal
        title="New prompt template"
        open={creating}
        onCancel={() => setCreating(false)}
        footer={null}
        width={640}
        destroyOnHidden
        {...mobileModal}
      >
        <PromptTemplateForm
          submitLabel="Create"
          onSubmit={async (input) => {
            await createPromptTemplate(profileId, input);
            setCreating(false);
            message.success("Prompt template created.");
            await reload();
          }}
        />
      </Modal>

      <Modal
        title="Edit prompt template"
        open={!!editing}
        onCancel={() => setEditing(null)}
        footer={null}
        width={640}
        destroyOnHidden
        {...mobileModal}
      >
        {editing && (
          <PromptTemplateForm
            initial={{
              name: editing.name,
              body: editing.body,
              description: editing.description,
            }}
            submitLabel="Save changes"
            onSubmit={async (input) => {
              await updatePromptTemplate(editing.id, input);
              setEditing(null);
              message.success("Prompt template updated.");
              await reload();
            }}
          />
        )}
      </Modal>
    </section>
  );
}
