"use client";

import { useState } from "react";
import { App, Button, Card, Space, Typography } from "antd";
import { setVideoScene } from "@/lib/api";
import { usePromptPair } from "@/lib/promptPair";
import type { Video } from "@/lib/types";
import PromptPairField from "./PromptPairField";

/**
 * The video's scene: the place every shot happens in, kept ONCE here and put in
 * front of the picture prompt of each shot when its picture is made (so it is edited
 * in one place, not copied into every shot). Like a prompt it is Thai for the owner
 * and English for the picture model. Empty on older videos, whose shots carry the
 * scene inside their own picture prompts.
 */
export default function SceneBox({ video, onSaved }: { video: Video; onSaved: (v: Video) => void }) {
  const { message } = App.useApp();
  const pair = usePromptPair("picture", video.sceneTh, video.scene);
  const [saving, setSaving] = useState(false);
  const dirty = pair.th.trim() !== video.sceneTh || pair.en.trim() !== video.scene;

  async function save() {
    setSaving(true);
    try {
      // The English is what the picture model receives: bring it up to date first.
      const english = await pair.ensureEnglish();
      onSaved(await setVideoScene(video.id, english.trim(), pair.th.trim()));
      message.success("Scene saved.");
    } catch (err) {
      message.error(err instanceof Error ? err.message : "Failed to save the scene.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <Card size="small" style={{ marginBottom: 16 }} title="ฉาก · Scene">
      <Typography.Text type="secondary" style={{ fontSize: 12 }}>
        สถานที่ที่ทุกช็อตเกิดขึ้น ระบบเอาไปไว้หน้าพรอมต์ภาพของทุกช็อตให้ตอนสร้างภาพ แก้ที่นี่ที่เดียว
        {video.scene === "" && video.sceneTh === ""
          ? " (วิดีโอนี้ยังไม่มีฉากแยก: ถ้ามี ฉากอยู่ในพรอมต์ภาพของแต่ละช็อตเอง)"
          : ""}
      </Typography.Text>
      <PromptPairField
        pair={pair}
        kind="picture"
        title="ฉาก"
        hint="พื้นหลังหรือสถานที่เท่านั้น ไม่ใส่สิ่งของ ไอคอน หรือตัวละครที่เป็นของช็อตใดช็อตหนึ่ง"
        placeholder="เช่น พื้นหลังแบนสีพาสเทลเรียบง่าย แสงสม่ำเสมอนุ่มนวล"
        minRows={2}
        maxRows={6}
      />
      <Space style={{ marginTop: 8 }}>
        <Button disabled={!dirty || saving || pair.busy} loading={saving || pair.busy} onClick={() => void save()}>
          Save scene
        </Button>
        <Typography.Text type={dirty ? "warning" : "secondary"} style={{ fontSize: 12 }}>
          {dirty ? "มีการแก้ไขที่ยังไม่ได้บันทึก" : "บันทึกแล้ว"}
        </Typography.Text>
      </Space>
    </Card>
  );
}
