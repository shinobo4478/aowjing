"use client";

import { use, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import {
  App,
  Button,
  Card,
  Descriptions,
  Popconfirm,
  Result,
  Space,
  Spin,
  Typography,
} from "antd";
import { deleteProfile, getProfile, updateProfile } from "@/lib/api";
import { PROVIDER_OPTIONS, type Profile } from "@/lib/types";
import ProfileForm from "../ProfileForm";
import ChannelsSection from "./ChannelsSection";
import PromptTemplatesSection from "./PromptTemplatesSection";
import GenerationsSection from "./GenerationsSection";

export default function ProfileDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const router = useRouter();
  const { message } = App.useApp();
  const [profile, setProfile] = useState<Profile | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [editing, setEditing] = useState(false);
  const [genTick, setGenTick] = useState(0);
  const [deleting, setDeleting] = useState(false);

  useEffect(() => {
    getProfile(id)
      .then(setProfile)
      .catch((err) =>
        setError(err instanceof Error ? err.message : "Failed to load."),
      );
  }, [id]);

  async function handleDelete() {
    setDeleting(true);
    try {
      await deleteProfile(id);
      message.success("Profile deleted.");
      router.push("/profiles");
    } catch (err) {
      message.error(err instanceof Error ? err.message : "Failed to delete.");
      setDeleting(false);
    }
  }

  if (error) {
    return (
      <Result
        status="404"
        title="Profile not available"
        subTitle={error}
        extra={
          <Link href="/profiles">
            <Button type="primary">All profiles</Button>
          </Link>
        }
      />
    );
  }

  if (!profile) {
    return (
      <div style={{ textAlign: "center", paddingTop: 80 }}>
        <Spin />
      </div>
    );
  }

  return (
    <>
      <Link href="/profiles">← All profiles</Link>

      {editing ? (
        <Card title="Edit profile" style={{ marginTop: 16 }}>
          <ProfileForm
            initial={{
              name: profile.name,
              niche: profile.niche,
              description: profile.description,
              provider: profile.provider,
            }}
            submitLabel="Save changes"
            onSubmit={async (input) => {
              setProfile(await updateProfile(id, input));
              setEditing(false);
              message.success("Profile updated.");
            }}
          />
          <Button
            style={{ marginTop: 12 }}
            onClick={() => setEditing(false)}
          >
            Cancel
          </Button>
        </Card>
      ) : (
        <>
          <Typography.Title level={3}>{profile.name}</Typography.Title>

          <Descriptions
            bordered
            column={1}
            style={{ marginTop: 8 }}
            items={[
              { key: "niche", label: "Niche", children: profile.niche },
              {
                key: "provider",
                label: "Generator",
                children:
                  PROVIDER_OPTIONS.find((o) => o.value === profile.provider)
                    ?.label ?? profile.provider,
              },
              {
                key: "description",
                label: "Description",
                children: (
                  <span style={{ whiteSpace: "pre-wrap" }}>
                    {profile.description || (
                      <Typography.Text type="secondary">
                        No description.
                      </Typography.Text>
                    )}
                  </span>
                ),
              },
              {
                key: "updated",
                label: "Updated",
                children: new Date(profile.updatedAt).toLocaleString(),
              },
            ]}
          />

          <Space style={{ marginTop: 16 }} wrap>
            <Button type="primary" onClick={() => setEditing(true)}>
              Edit
            </Button>
            <Popconfirm
              title="Delete this profile?"
              description="This deletes its channels too. Can't be undone."
              okText="Delete"
              okButtonProps={{ danger: true, loading: deleting }}
              onConfirm={handleDelete}
            >
              <Button danger>Delete</Button>
            </Popconfirm>
          </Space>

          <ChannelsSection profileId={id} />
          <PromptTemplatesSection
            profileId={id}
            onGenerated={() => setGenTick((t) => t + 1)}
          />
          <GenerationsSection profileId={id} refreshTick={genTick} />
        </>
      )}
    </>
  );
}
