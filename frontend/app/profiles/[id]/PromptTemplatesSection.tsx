"use client";

import { useEffect, useState } from "react";
import { App, Button, Empty, Flex, Modal, Popconfirm, Space, Table } from "antd";
import type { TableColumnsType } from "antd";
import {
  ApiError,
  createPromptTemplate,
  deletePromptTemplate,
  listPromptTemplates,
  runGeneration,
  updatePromptTemplate,
} from "@/lib/api";
import { formatDateTime } from "@/lib/format";
import { productSlotsNeeded } from "@/lib/template";
import type { ProviderKey, PromptTemplate } from "@/lib/types";
import { mobileModal } from "@/lib/ui";
import PromptTemplateForm from "../PromptTemplateForm";
import GenerateModal from "./GenerateModal";

export default function PromptTemplatesSection({
  profileId,
  provider,
  onGenerated,
}: {
  profileId: string;
  /** The profile's generator; "fal" lets Generate take a starting image. */
  provider: ProviderKey;
  /** Called after a template is run, so the Generations list can refresh. */
  onGenerated: () => void;
}) {
  const { message } = App.useApp();
  const [templates, setTemplates] = useState<PromptTemplate[]>([]);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [editing, setEditing] = useState<PromptTemplate | null>(null);
  const [running, setRunning] = useState<string | null>(null);
  const [picking, setPicking] = useState<PromptTemplate | null>(null);

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

  // Ask first when there is something to choose: a template that mentions
  // products, or a fal profile (which can start from an image). Anything else
  // runs straight away.
  function handleGenerate(tpl: PromptTemplate) {
    if (productSlotsNeeded(tpl.body) > 0 || provider === "fal") setPicking(tpl);
    else run(tpl.id);
  }

  async function run(
    id: string,
    opts: { productIds?: string[]; imageUrl?: string; imageId?: string } = {},
  ) {
    setRunning(id);
    try {
      await runGeneration(id, opts);
      message.success("Generation queued.");
      onGenerated(); // the Generations list picks it up and polls for the result
    } catch (err) {
      // A 422 carries a message per field; show the first one.
      const field =
        err instanceof ApiError && err.fieldErrors
          ? Object.values(err.fieldErrors)[0]
          : undefined;
      message.error(
        field ?? (err instanceof Error ? err.message : "Failed to generate."),
      );
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
      render: (v: string) => formatDateTime(v),
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
            onClick={() => handleGenerate(row)}
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
    <section>
      <Flex justify="end">
        <Button onClick={() => setCreating(true)}>New template</Button>
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
        locale={{
          emptyText: (
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description="No prompt templates yet. Write one, then press Generate to run it."
            >
              <Button type="primary" onClick={() => setCreating(true)}>
                New template
              </Button>
            </Empty>
          ),
        }}
      />

      <GenerateModal
        template={picking}
        askImage={provider === "fal"}
        onClose={() => setPicking(null)}
        onGenerate={async (choices) => {
          const id = picking!.id;
          setPicking(null);
          await run(id, choices);
        }}
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
