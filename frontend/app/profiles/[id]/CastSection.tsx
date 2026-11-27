"use client";

import { useCallback, useEffect, useState } from "react";
import { Alert, App, Button, Card, Empty, Form, Input, Modal, Popconfirm, Select, Tag, Typography, Upload } from "antd";
import {
  addCastImage,
  type CastInput,
  ApiError,
  createCastMember,
  deleteCastMember,
  imageSrc,
  listCast,
  MAX_REFERENCE_BYTES,
  removeCastImage,
  updateCastMember,
} from "@/lib/api";
import { CAST_ROLES, type CastList, type CastMember, type ProviderKey } from "@/lib/types";

/** Checks a picked file before uploading it; null = fine. The backend checks
 *  the same things, plus the pixel size. */
function pictureFileError(file: File): string | null {
  if (file.type !== "image/jpeg" && file.type !== "image/png") {
    return "Use a JPG or PNG image.";
  }
  if (file.size > MAX_REFERENCE_BYTES) {
    return `The picture is too large; a cast picture can be at most ${MAX_REFERENCE_BYTES / 1024 / 1024} MB.`;
  }
  return null;
}

/** The first validation message of an API error, else its own message. */
function errorText(err: unknown, fallback: string): string {
  if (err instanceof ApiError && err.fieldErrors) {
    const first = Object.values(err.fieldErrors)[0];
    if (first) return first;
  }
  return err instanceof Error ? err.message : fallback;
}

const roleLabel = (role: string) => CAST_ROLES.find((r) => r.value === role)?.label ?? role;

/**
 * The profile's cast: the recurring characters, objects and places its pictures
 * keep the same. Each has a role, a name (the prompt calls it by that name, so
 * the image model knows which picture is who), an optional note and 1-4 pictures
 * of itself. When a picture is made, the owner ticks who appears in it.
 */
export default function CastSection({
  profileId,
  provider,
}: {
  profileId: string;
  provider: ProviderKey;
}) {
  const { message } = App.useApp();
  const [list, setList] = useState<CastList | null>(null);
  const [loading, setLoading] = useState(true);
  // The member being edited; "new" = the form adds one; null = closed.
  const [editing, setEditing] = useState<CastMember | "new" | null>(null);
  const [saving, setSaving] = useState(false);
  const [uploadingFor, setUploadingFor] = useState<string | null>(null);
  const [form] = Form.useForm<CastInput>();

  const load = useCallback(() => {
    listCast(profileId)
      .then(setList)
      .catch((err) => message.error(errorText(err, "Failed to load the cast.")))
      .finally(() => setLoading(false));
  }, [profileId, message]);

  useEffect(() => {
    load();
  }, [load]);

  const members = list?.cast ?? [];
  const maxMembers = list?.maxMembers ?? 8;
  const maxPictures = list?.maxMemberPictures ?? 4;

  function openForm(target: CastMember | "new") {
    form.resetFields();
    form.setFieldsValue(
      target === "new"
        ? { role: members.some((m) => m.role === "hero") ? "costar" : "hero", name: "", note: "" }
        : { role: target.role, name: target.name, note: target.note },
    );
    setEditing(target);
  }

  async function submit(values: CastInput) {
    if (!editing) return;
    setSaving(true);
    try {
      if (editing === "new") {
        await createCastMember(profileId, values);
      } else {
        await updateCastMember(profileId, editing.id, values);
      }
      setEditing(null);
      load();
    } catch (err) {
      message.error(errorText(err, "Could not save."));
    } finally {
      setSaving(false);
    }
  }

  async function remove(m: CastMember) {
    try {
      await deleteCastMember(profileId, m.id);
      load();
    } catch (err) {
      message.error(errorText(err, "Failed to remove."));
    }
  }

  async function upload(m: CastMember, file: File) {
    const problem = pictureFileError(file);
    if (problem) {
      message.error(problem);
      return;
    }
    setUploadingFor(m.id);
    try {
      await addCastImage(profileId, m.id, file);
      load();
    } catch (err) {
      message.error(errorText(err, "Upload failed."));
    } finally {
      setUploadingFor(null);
    }
  }

  async function removePicture(m: CastMember, imageId: string) {
    try {
      await removeCastImage(profileId, m.id, imageId);
      load();
    } catch (err) {
      message.error(errorText(err, "Failed to remove."));
    }
  }

  return (
    <div>
      <Typography.Paragraph type="secondary" style={{ maxWidth: 680 }}>
        ทีมนักแสดงของช่องนี้: ตัวละครที่โผล่ซ้ำ ของประกอบ และสถานที่ ใส่รูปของแต่ละอย่าง
        1–{maxPictures} รูป (หลายมุมได้) แล้วตอนสร้างภาพของช็อต ติ๊กเลือกว่าใครอยู่ในภาพ
        ระบบจะบอกโมเดลว่ารูปไหนคือใคร เพื่อให้หน้าตาเหมือนเดิม รูปสินค้าไม่ต้องใส่ที่นี่
        เพราะสินค้าเปลี่ยนทุกคลิป (ใช้ได้ถึง {maxMembers} รายการ, JPG หรือ PNG, ขั้นต่ำ 300×300 px,
        ไม่เกิน {MAX_REFERENCE_BYTES / 1024 / 1024} MB ต่อรูป)
      </Typography.Paragraph>

      {provider !== "fal" && (
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 16, maxWidth: 680 }}
          title="This profile uses the text generator, so the cast is not used yet. Switch its generator to fal.ai video to make pictures and clips."
        />
      )}

      {!loading && members.length === 0 ? (
        <Empty description="No cast yet" style={{ margin: "24px 0" }} />
      ) : (
        <div style={{ display: "flex", flexDirection: "column", gap: 16, margin: "16px 0", maxWidth: 680 }}>
          {members.map((m) => {
            const full = m.imageIds.length >= maxPictures;
            return (
              <Card
                key={m.id}
                size="small"
                title={
                  <span>
                    <Tag color={m.role === "hero" ? "gold" : undefined}>{roleLabel(m.role)}</Tag>
                    {m.name}
                    {m.note && (
                      <Typography.Text type="secondary" style={{ marginLeft: 8, fontWeight: 400 }}>
                        {m.note}
                      </Typography.Text>
                    )}
                  </span>
                }
                extra={
                  <>
                    <Button type="link" size="small" onClick={() => openForm(m)}>
                      Edit
                    </Button>
                    <Popconfirm
                      title={`Remove ${m.name} and its pictures?`}
                      okText="Remove"
                      okButtonProps={{ danger: true }}
                      onConfirm={() => remove(m)}
                    >
                      <Button type="link" size="small" danger>
                        Remove
                      </Button>
                    </Popconfirm>
                  </>
                }
              >
                <div style={{ display: "flex", flexWrap: "wrap", gap: 12, alignItems: "flex-start" }}>
                  {m.imageIds.map((id, i) => (
                    <div key={id} style={{ width: 96 }}>
                      {/* Served by the API with the session cookie, so next/image cannot optimize it. */}
                      {/* eslint-disable-next-line @next/next/no-img-element */}
                      <img
                        src={imageSrc(id)}
                        alt={`${m.name}, picture ${i + 1}`}
                        style={{ width: 96, height: 128, objectFit: "cover", borderRadius: 8, display: "block" }}
                      />
                      <Popconfirm
                        title="Remove this picture?"
                        okText="Remove"
                        okButtonProps={{ danger: true }}
                        onConfirm={() => removePicture(m, id)}
                      >
                        <Button type="text" size="small" danger style={{ marginTop: 2 }}>
                          Remove
                        </Button>
                      </Popconfirm>
                    </div>
                  ))}
                  <Upload
                    accept="image/png,image/jpeg"
                    showUploadList={false}
                    disabled={full || uploadingFor !== null}
                    beforeUpload={(file) => {
                      void upload(m, file);
                      return false; // we upload ourselves, not antd
                    }}
                  >
                    <Button disabled={full} loading={uploadingFor === m.id}>
                      {full ? `All ${maxPictures} used` : m.imageIds.length === 0 ? "Add a picture…" : "Add another…"}
                    </Button>
                  </Upload>
                </div>
                {m.imageIds.length === 0 && (
                  <Typography.Text type="secondary" style={{ display: "block", marginTop: 8, fontSize: 13 }}>
                    ยังไม่มีรูป ต้องมีอย่างน้อย 1 รูปจึงจะเลือกใช้ตอนสร้างภาพได้
                  </Typography.Text>
                )}
              </Card>
            );
          })}
        </div>
      )}

      <Button type="primary" disabled={members.length >= maxMembers} onClick={() => openForm("new")}>
        {members.length >= maxMembers ? `All ${maxMembers} places used` : "Add to the cast…"}
      </Button>
      <Typography.Text type="secondary" style={{ marginLeft: 12 }}>
        {members.length} of {maxMembers}
      </Typography.Text>

      <Modal
        open={editing !== null}
        title={editing === "new" ? "Add to the cast" : "Edit"}
        okText="Save"
        confirmLoading={saving}
        onCancel={() => setEditing(null)}
        onOk={() => form.submit()}
        destroyOnHidden
        forceRender
      >
        <Form form={form} layout="vertical" onFinish={submit} requiredMark={false}>
          <Form.Item label="บทบาท · Role" name="role" rules={[{ required: true }]}>
            <Select options={CAST_ROLES.map((r) => ({ value: r.value, label: r.label }))} />
          </Form.Item>
          <Form.Item
            label="ชื่อ · Name"
            name="name"
            extra="ชื่อที่ใช้เรียกในคำสั่งถึงโมเดล เช่น “พี่แบด”, “น้องมะลิ (แมว)”, “กระติกน้ำสีม่วง”"
            rules={[{ required: true, whitespace: true, message: "Give it a name." }]}
          >
            <Input maxLength={60} autoComplete="off" />
          </Form.Item>
          <Form.Item
            label="โน้ต · Note (optional)"
            name="note"
            extra="สภาพหรือรายละเอียดที่รูปแสดง เช่น “ชุดแข่งสีแดง”, “ฝาเปิด”"
          >
            <Input maxLength={120} autoComplete="off" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
