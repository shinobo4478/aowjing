"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { App, Button, Modal, Table, Typography } from "antd";
import type { TableColumnsType } from "antd";
import { createProfile, listProfiles } from "@/lib/api";
import type { Profile } from "@/lib/types";
import ProfileForm from "./ProfileForm";

const columns: TableColumnsType<Profile> = [
  {
    title: "Name",
    dataIndex: "name",
    render: (name: string, row) => <Link href={`/profiles/${row.id}`}>{name}</Link>,
  },
  { title: "Niche", dataIndex: "niche" },
  {
    title: "Updated",
    dataIndex: "updatedAt",
    width: 200,
    render: (v: string) => new Date(v).toLocaleString(),
  },
];

export default function ProfilesPage() {
  const { message } = App.useApp();
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);

  // Load once on mount. `loading` already starts true, so no synchronous
  // setState here — state only changes inside the async callbacks.
  useEffect(() => {
    listProfiles()
      .then(setProfiles)
      .catch((err) =>
        message.error(
          err instanceof Error ? err.message : "Failed to load profiles.",
        ),
      )
      .finally(() => setLoading(false));
  }, [message]);

  // Re-fetch after a mutation. Only ever called from event handlers.
  async function reload() {
    setLoading(true);
    try {
      setProfiles(await listProfiles());
    } catch (err) {
      message.error(err instanceof Error ? err.message : "Failed to load profiles.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <>
      <Typography.Title level={3} style={{ marginTop: 0 }}>
        Profiles
      </Typography.Title>
      <Typography.Paragraph type="secondary">
        Each profile is one channel persona / content niche.
      </Typography.Paragraph>

      <Button
        type="primary"
        onClick={() => setCreating(true)}
        style={{ marginBottom: 16 }}
      >
        New profile
      </Button>

      <Table<Profile>
        rowKey="id"
        columns={columns}
        dataSource={profiles}
        loading={loading}
        pagination={false}
        locale={{ emptyText: "No profiles yet. Create the first one." }}
      />

      <Modal
        title="New profile"
        open={creating}
        onCancel={() => setCreating(false)}
        footer={null}
        destroyOnHidden
      >
        <ProfileForm
          submitLabel="Create"
          onSubmit={async (input) => {
            await createProfile(input);
            setCreating(false);
            message.success("Profile created.");
            await reload();
          }}
        />
      </Modal>
    </>
  );
}
