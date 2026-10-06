"use client";

import { use, useEffect, useState } from "react";
import { useRouter, usePathname, useSearchParams } from "next/navigation";
import Link from "next/link";
import {
  App,
  Button,
  Card,
  Flex,
  Popconfirm,
  Result,
  Space,
  Spin,
  Tabs,
  Tag,
  Typography,
} from "antd";
import { deleteProfile, getProfile, updateProfile } from "@/lib/api";
import { PROVIDER_OPTIONS, type Profile } from "@/lib/types";
import ProfileForm from "../ProfileForm";
import ChannelsSection from "./ChannelsSection";
import PromptTemplatesSection from "./PromptTemplatesSection";
import GenerationsSection from "./GenerationsSection";

const DEFAULT_TAB = "templates";

export default function ProfileDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const router = useRouter();
  const pathname = usePathname();
  const search = useSearchParams();
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

  const activeTab = search.get("tab") ?? DEFAULT_TAB;

  // Switches tab via the URL (the single source of truth for `activeTab`),
  // dropping any one-off query params like the OAuth callback's ?yt=/&reason=.
  function setTab(key: string) {
    router.replace(`${pathname}?tab=${key}`);
  }

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
          <Button style={{ marginTop: 12 }} onClick={() => setEditing(false)}>
            Cancel
          </Button>
        </Card>
      ) : (
        <>
          <Flex justify="space-between" align="flex-start" wrap gap={12} style={{ marginTop: 16 }}>
            <div style={{ minWidth: 0 }}>
              <Typography.Title level={3} style={{ marginTop: 0, marginBottom: 6 }}>
                {profile.name}
              </Typography.Title>
              <Space size={[8, 8]} wrap style={{ marginBottom: 8 }}>
                <Tag>{profile.niche}</Tag>
                <Tag color="purple">
                  {PROVIDER_OPTIONS.find((o) => o.value === profile.provider)
                    ?.label ?? profile.provider}
                </Tag>
              </Space>
              {profile.description && (
                <Typography.Paragraph
                  type="secondary"
                  style={{ marginBottom: 4, maxWidth: 640, whiteSpace: "pre-wrap" }}
                >
                  {profile.description}
                </Typography.Paragraph>
              )}
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                Updated {new Date(profile.updatedAt).toLocaleString()}
              </Typography.Text>
            </div>
            <Space wrap>
              <Button onClick={() => setEditing(true)}>Edit</Button>
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
          </Flex>

          <Tabs
            activeKey={activeTab}
            onChange={setTab}
            style={{ marginTop: 16 }}
            items={[
              {
                key: "channels",
                label: "Channels",
                children: <ChannelsSection profileId={id} />,
              },
              {
                key: "templates",
                label: "Prompt templates",
                children: (
                  <PromptTemplatesSection
                    profileId={id}
                    onGenerated={() => {
                      setGenTick((t) => t + 1);
                      setTab("generations");
                    }}
                  />
                ),
              },
              {
                key: "generations",
                label: "Generations",
                children: (
                  <GenerationsSection profileId={id} refreshTick={genTick} />
                ),
              },
            ]}
          />
        </>
      )}
    </>
  );
}
