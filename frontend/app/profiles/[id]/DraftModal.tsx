"use client";

import { useEffect, useState } from "react";
import { Alert, App, Button, Input, InputNumber, Modal, Segmented, Select, Typography } from "antd";
import SecondsField from "@/components/SecondsField";
import { ApiError, draftVideo, getStudioPricing, listCast, listPickableProducts } from "@/lib/api";
import { formatUsd } from "@/lib/format";
import { CAST_ROLES, pricingFor, type CastMember, type PickableProduct, type StudioPricing, type Video } from "@/lib/types";
import { mobileModal } from "@/lib/ui";
import { refreshCredits } from "../../CreditsBadge";

/** The kinds of video the AI can draft. Mirrors scriptwriter.Mode. */
type DraftMode = "general" | "product" | "film";

const MODE_OPTIONS: { value: DraftMode; label: string; hint: string }[] = [
  {
    value: "general",
    label: "ทั่วไป · General",
    hint: "เรื่องสั้นจากหัวข้อที่ให้ (เหมือนเดิม)",
  },
  {
    value: "product",
    label: "สินค้า · Product",
    hint: "คลิปเกี่ยวกับสินค้าจริงที่ต้องเหมือนของจริง: ฮุก → ใช้งานในชีวิตจริง → ช็อตสินค้านิ่งๆ ตอนจบ ไม่ซูมโลโก้ ไม่อ้างอะไรเกินโน้ตที่คุณเขียน",
  },
  {
    value: "film",
    label: "หนังสั้น · Film",
    hint: "เรื่องสั้นตลกเสียดสี เล่าด้วยภาพ ไม่มีบทพูด: ตั้งเรื่อง → ตึงขึ้น → ตบมุก → ปิดท้าย (ใส่คำบรรยายภายหลังในโปรแกรมตัดต่อ)",
  },
];

/**
 * Asks the script helper to draft a whole video: the owner gives the topic and
 * how many shots of how many seconds, and gets a new Video with the scene and
 * each shot's picture prompt, motion prompt and length filled in, to read and
 * change. It makes no pictures and no clips, so the only cost is the model's
 * reply; fal bills per token, so the exact cost is shown afterwards.
 */
export default function DraftModal({
  open,
  profileId,
  videoModel,
  onClose,
  onDrafted,
}: {
  open: boolean;
  profileId: string;
  videoModel: string;
  onClose: () => void;
  onDrafted: (video: Video) => void;
}) {
  return (
    <Modal
      title="Draft a video with AI"
      open={open}
      onCancel={onClose}
      footer={null}
      width={640}
      destroyOnHidden
      {...mobileModal}
    >
      {open && <DraftForm profileId={profileId} videoModel={videoModel} onDrafted={onDrafted} />}
    </Modal>
  );
}

function DraftForm({
  profileId,
  videoModel,
  onDrafted,
}: {
  profileId: string;
  videoModel: string;
  onDrafted: (video: Video) => void;
}) {
  const { message, modal } = App.useApp();
  const [topic, setTopic] = useState("");
  const [shots, setShots] = useState(4);
  const [seconds, setSeconds] = useState(4);
  const [title, setTitle] = useState("");
  // What kind of video: the structure the AI is told to follow. Product needs a
  // catalog product that has pictures (so its real look can be kept).
  const [mode, setMode] = useState<DraftMode>("general");
  const [productId, setProductId] = useState("");
  const [products, setProducts] = useState<PickableProduct[] | null>(null);
  const [cast, setCast] = useState<CastMember[]>([]);
  const [busy, setBusy] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});

  const totalSeconds = shots * seconds;

  // The lengths the profile's video model takes (Kling: 3 to 15; Veo: 4, 6 or 8).
  const [pricing, setPricing] = useState<StudioPricing | null>(null);
  useEffect(() => {
    getStudioPricing()
      .then(setPricing)
      .catch(() => {
        /* the length box falls back to Kling's range */
      });
  }, []);
  const secondsOptions = pricingFor(pricing, videoModel)?.seconds ?? Array.from({ length: 13 }, (_, i) => i + 3);

  // The profile's cast (the AI is told who exists) and the products that have
  // pictures (a product video is about one of them).
  useEffect(() => {
    let cancelled = false;
    listCast(profileId)
      .then((r) => {
        if (!cancelled) setCast(r.cast);
      })
      .catch(() => {
        /* non-fatal: the note about the cast just stays hidden */
      });
    listPickableProducts()
      .then((p) => {
        if (!cancelled) setProducts(p);
      })
      .catch(() => {
        if (!cancelled) setProducts([]);
      });
    return () => {
      cancelled = true;
    };
  }, [profileId]);
  // A style reference (an example frame of the drawing style) is not a character;
  // the AI is not told about it.
  const storyCast = cast.filter((m) => m.role !== "style");
  const chosenProduct = products?.find((p) => p.id === productId);
  const modeHint = MODE_OPTIONS.find((m) => m.value === mode)?.hint ?? "";
  const needsProduct = mode === "product" && !chosenProduct;

  // The draft costs a few cents (fal bills per token), so ask first.
  function confirmDraft() {
    modal.confirm({
      title: "Draft this video?",
      content: (
        <>
          <p style={{ margin: "0 0 8px" }}>
            The AI writes the shot list for {shots} shot{shots === 1 ? "" : "s"} of{" "}
            {seconds} s ({totalSeconds} s in all). It costs a few cents at most,
            charged to your fal.ai credits; fal bills per token, so the exact cost
            is shown once it is done.
          </p>
          <p style={{ margin: 0 }}>
            It makes no pictures and no clips: you read and change the shots first.
          </p>
        </>
      ),
      okText: "Draft it",
      cancelText: "Cancel",
      onOk: () => draft(),
    });
  }

  async function draft() {
    setBusy(true);
    setErrors({});
    try {
      const r = await draftVideo({
        profileId,
        topic,
        shots,
        seconds,
        title: title.trim() || undefined,
        mode,
        productId: mode === "product" ? productId : undefined,
      });
      refreshCredits();
      message.success(`Drafted with ${r.model}. It cost ${formatUsd(r.costUsd)}.`);
      onDrafted(r.video);
    } catch (err) {
      if (err instanceof ApiError && err.fieldErrors) {
        setErrors(err.fieldErrors as Record<string, string>);
      } else {
        message.error(err instanceof Error ? err.message : "The draft failed.");
      }
    } finally {
      setBusy(false);
    }
  }

  return (
    <div>
      <Typography.Paragraph type="secondary">
        Say what the video is about. The AI writes the scene and, for each shot, what
        the starting picture shows and how the clip moves. You can write the topic in
        Thai or English; the prompts it writes are in English.
      </Typography.Paragraph>

      <Typography.Text>แบบวิดีโอ · Format</Typography.Text>
      <div style={{ marginTop: 4 }}>
        <Segmented<DraftMode>
          value={mode}
          onChange={setMode}
          options={MODE_OPTIONS.map((m) => ({ value: m.value, label: m.label }))}
        />
      </div>
      <Typography.Paragraph type="secondary" style={{ margin: "6px 0 16px", fontSize: 13 }}>
        {modeHint}
      </Typography.Paragraph>

      {mode === "product" && (
        <div style={{ marginBottom: 16 }}>
          <Typography.Text>สินค้า · Product</Typography.Text>
          {products !== null && products.length === 0 ? (
            <Alert
              type="warning"
              showIcon
              style={{ marginTop: 4 }}
              title="ยังไม่มีสินค้าที่มีรูป เพิ่มรูปสินค้าที่หน้า Products ก่อน เพื่อให้คงหน้าตาจริงของสินค้า"
            />
          ) : (
            <Select
              value={productId || undefined}
              onChange={setProductId}
              loading={products === null}
              placeholder="เลือกสินค้า"
              style={{ width: "100%", marginTop: 4 }}
              options={(products ?? []).map((p) => ({
                value: p.id,
                label: `${p.name} (${p.pictures.length} รูป)`,
              }))}
            />
          )}
          {chosenProduct && (
            <Typography.Text type="secondary" style={{ display: "block", marginTop: 4, fontSize: 13 }}>
              สภาพที่มีรูป:{" "}
              {chosenProduct.pictures.map((p) => p.label || "ไม่มีชื่อสภาพ").join(", ")} AI จะเขียนช็อตให้ใช้สินค้าเฉพาะในสภาพเหล่านี้
            </Typography.Text>
          )}
          {errors.productId && (
            <Typography.Text type="danger" style={{ display: "block", marginTop: 4 }}>
              {errors.productId}
            </Typography.Text>
          )}
        </div>
      )}

      {storyCast.length > 0 && (
        <Typography.Paragraph type="secondary" style={{ fontSize: 13 }}>
          AI รู้จักนักแสดงของช่องนี้:{" "}
          {storyCast
            .map((m) => `${m.name} (${CAST_ROLES.find((r) => r.value === m.role)?.label.split(" · ")[0] ?? m.role})`)
            .join(", ")}{" "}
          และจะเขียนช็อตให้มีพวกเขาเมื่อเรื่องต้องการ ตอนสร้างภาพคุณยังเป็นคนติ๊กเองว่าใครอยู่ในภาพ
        </Typography.Paragraph>
      )}

      <Typography.Text>Topic</Typography.Text>
      <Input.TextArea
        value={topic}
        onChange={(e) => setTopic(e.target.value)}
        autoSize={{ minRows: 4, maxRows: 10 }}
        maxLength={1000}
        showCount
        status={errors.topic ? "error" : undefined}
        placeholder="e.g. A cat is startled by a plastic bag, plays with it, and ends up proud of it. Funny, quick, loops."
      />
      {errors.topic && (
        <Typography.Text type="danger" style={{ display: "block", marginTop: 4 }}>
          {errors.topic}
        </Typography.Text>
      )}

      <div style={{ display: "flex", gap: 24, flexWrap: "wrap", marginTop: 28 }}>
        <div>
          <Typography.Text>Shots</Typography.Text>
          <div>
            <InputNumber min={1} max={12} precision={0} value={shots} onChange={(v) => setShots(typeof v === "number" ? v : 4)} />
          </div>
          {errors.shots && <Typography.Text type="danger">{errors.shots}</Typography.Text>}
        </div>
        <div>
          <Typography.Text>Seconds per shot</Typography.Text>
          <div>
            <SecondsField options={secondsOptions} value={seconds} onChange={setSeconds} />
          </div>
          {errors.seconds && <Typography.Text type="danger">{errors.seconds}</Typography.Text>}
        </div>
        <div style={{ alignSelf: "end" }}>
          <Typography.Text type="secondary">{totalSeconds} seconds in all</Typography.Text>
        </div>
      </div>

      <div style={{ marginTop: 16 }}>
        <Typography.Text>Title (optional)</Typography.Text>
        <Input
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          maxLength={120}
          status={errors.title ? "error" : undefined}
          placeholder="The AI suggests one if you leave it empty"
        />
        {errors.title && <Typography.Text type="danger">{errors.title}</Typography.Text>}
      </div>

      <Button
        type="primary"
        style={{ marginTop: 20 }}
        loading={busy}
        disabled={topic.trim().length < 3 || needsProduct}
        onClick={confirmDraft}
      >
        Draft the video
      </Button>
      {busy && (
        <Typography.Text type="secondary" style={{ marginLeft: 12 }}>
          Writing the draft… this can take up to a minute.
        </Typography.Text>
      )}
    </div>
  );
}
