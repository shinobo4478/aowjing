"use client";

import { use, useEffect, useState } from "react";
import { useRouter, usePathname, useSearchParams } from "next/navigation";
import Link from "next/link";
import {
  App,
  Breadcrumb,
  Button,
  Modal,
  Popconfirm,
  Result,
  Space,
  Spin,
  Tabs,
} from "antd";
import { deleteProfile, getProfile, updateProfile } from "@/lib/api";
import type { Profile } from "@/lib/types";
import { mobileModal } from "@/lib/ui";
import ProfileForm from "../ProfileForm";
import ChannelsSection from "./ChannelsSection";
import GenerationsSection from "./GenerationsSection";
import ProfileSlate from "./ProfileSlate";
import PromptTemplatesSection from "./PromptTemplatesSection";

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
            <Button type="primary">Back to profiles</Button>
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
      <Breadcrumb
        style={{ marginBottom: 20 }}
        items={[
          { title: <Link href="/profiles">Profiles</Link> },
          { title: profile.name },
        ]}
      />

      <ProfileSlate
        profile={profile}
        actions={
          <Space wrap>
            <Button onClick={() => setEditing(true)}>Edit</Button>
            <Popconfirm
              title="Delete this profile?"
              description="Its channels, templates and generations are deleted too. This can't be undone."
              okText="Delete profile"
              okButtonProps={{ danger: true, loading: deleting }}
              onConfirm={handleDelete}
            >
              <Button danger>Delete</Button>
            </Popconfirm>
          </Space>
        }
      />

      <Tabs
        activeKey={activeTab}
        onChange={setTab}
        style={{ marginTop: 32 }}
        items={[
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
            children: <GenerationsSection profileId={id} refreshTick={genTick} />,
          },
          {
            key: "channels",
            label: "Channels",
            children: <ChannelsSection profileId={id} />,
          },
        ]}
      />

      <Modal
        title="Edit profile"
        open={editing}
        onCancel={() => setEditing(false)}
        footer={null}
        destroyOnHidden
        {...mobileModal}
      >
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
      </Modal>
    </>
  );
}
