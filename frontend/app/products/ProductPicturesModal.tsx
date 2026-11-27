"use client";

import { useCallback, useEffect, useState } from "react";
import { App, Button, Empty, Input, Modal, Popconfirm, Typography, Upload } from "antd";
import {
  addProductPicture,
  ApiError,
  imageSrc,
  listProductPictures,
  MAX_REFERENCE_BYTES,
  removeProductPicture,
  updateProductPictureLabel,
} from "@/lib/api";
import type { ProductPicture, ProductWithUse } from "@/lib/types";

/** The first validation message of an API error, else its own message. */
function errorText(err: unknown, fallback: string): string {
  if (err instanceof ApiError && err.fieldErrors) {
    const first = Object.values(err.fieldErrors)[0];
    if (first) return first;
  }
  return err instanceof Error ? err.message : fallback;
}

/**
 * The pictures of one product as it really looks: several angles and states.
 * They go to the image model when a shot's picture is made, so the product on
 * screen is the real one. Because an edit model copies the STATE of its
 * reference (a closed bottle stays closed), give a picture for each state a shot
 * needs and say which in the label ("cap open", "in hand").
 */
export default function ProductPicturesModal({
  product,
  onClose,
}: {
  product: ProductWithUse | null;
  /** Called when the dialog closes, with the product's picture count so the list
   *  can update without reloading. */
  onClose: (productId: string, pictureCount: number) => void;
}) {
  const { message } = App.useApp();
  const productId = product?.id ?? null;
  const [pictures, setPictures] = useState<ProductPicture[]>([]);
  const [max, setMax] = useState(4);
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  // The label for the NEXT upload.
  const [newLabel, setNewLabel] = useState("");
  // Labels being edited, by image id (saved when the box loses focus).
  const [drafts, setDrafts] = useState<Record<string, string>>({});

  const load = useCallback(
    (id: string) => {
      setLoading(true);
      listProductPictures(id)
        .then((r) => {
          setPictures(r.pictures);
          setMax(r.max);
          setDrafts({});
        })
        .catch((err) => message.error(errorText(err, "Failed to load the pictures.")))
        .finally(() => setLoading(false));
    },
    [message],
  );

  useEffect(() => {
    if (productId) load(productId);
  }, [productId, load]);

  async function upload(file: File) {
    if (!productId) return;
    if (file.type !== "image/jpeg" && file.type !== "image/png") {
      message.error("Use a JPG or PNG image.");
      return;
    }
    if (file.size > MAX_REFERENCE_BYTES) {
      message.error(`The picture is too large; at most ${MAX_REFERENCE_BYTES / 1024 / 1024} MB.`);
      return;
    }
    setUploading(true);
    try {
      await addProductPicture(productId, file, newLabel.trim());
      setNewLabel("");
      load(productId);
    } catch (err) {
      message.error(errorText(err, "Upload failed."));
    } finally {
      setUploading(false);
    }
  }

  async function saveLabel(p: ProductPicture) {
    if (!productId) return;
    const text = (drafts[p.imageId] ?? p.label).trim();
    if (text === p.label) return;
    try {
      const saved = await updateProductPictureLabel(productId, p.imageId, text);
      setPictures((ps) => ps.map((x) => (x.imageId === p.imageId ? saved : x)));
      setDrafts((d) => {
        const { [p.imageId]: _gone, ...rest } = d;
        void _gone;
        return rest;
      });
    } catch (err) {
      message.error(errorText(err, "Could not save the label."));
    }
  }

  async function remove(p: ProductPicture) {
    if (!productId) return;
    try {
      await removeProductPicture(productId, p.imageId);
      setPictures((ps) => ps.filter((x) => x.imageId !== p.imageId));
    } catch (err) {
      message.error(errorText(err, "Failed to remove."));
    }
  }

  const full = pictures.length >= max;

  return (
    <Modal
      open={product !== null}
      title={product ? `Pictures: ${product.name}` : ""}
      footer={null}
      onCancel={() => product && onClose(product.id, pictures.length)}
      width={620}
      destroyOnHidden
    >
      <Typography.Paragraph type="secondary">
        รูปจริงของสินค้า หลายมุมและหลายสภาพ (เช่น ฝาปิด ฝาเปิด ถืออยู่ในมือ) ตอนสร้างภาพของช็อต
        เลือกได้ว่าจะส่งรูปไหนให้โมเดล โมเดลคัดลอกสภาพจากรูปด้วย ถ้าช็อตต้องการฝาเปิดต้องมีรูปที่ฝาเปิด
        ใส่ชื่อบอกสภาพของรูปไว้ใต้รูป ใช้ได้ถึง {max} รูป (JPG หรือ PNG ขั้นต่ำ 300×300 px
        อัตราส่วน 0.4–2.5 ไม่เกิน {MAX_REFERENCE_BYTES / 1024 / 1024} MB) สินค้าสีเดียวกันทุกรูป
      </Typography.Paragraph>

      {!loading && pictures.length === 0 ? (
        <Empty description="No pictures yet" style={{ margin: "16px 0" }} />
      ) : (
        <div style={{ display: "flex", flexWrap: "wrap", gap: 16, margin: "12px 0" }}>
          {pictures.map((p) => (
            <div key={p.imageId} style={{ width: 120 }}>
              {/* Served by the API with the session cookie, so next/image cannot optimize it. */}
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img
                src={imageSrc(p.imageId)}
                alt={p.label || "Product picture"}
                style={{ width: 120, height: 150, objectFit: "contain", borderRadius: 8, display: "block", background: "rgba(128,128,128,0.12)" }}
              />
              <Input
                size="small"
                value={drafts[p.imageId] ?? p.label}
                placeholder="เช่น ฝาเปิด"
                maxLength={60}
                onChange={(e) => setDrafts((d) => ({ ...d, [p.imageId]: e.target.value }))}
                onBlur={() => void saveLabel(p)}
                onPressEnter={() => void saveLabel(p)}
                style={{ marginTop: 4 }}
              />
              <Popconfirm
                title="Remove this picture?"
                okText="Remove"
                okButtonProps={{ danger: true }}
                onConfirm={() => remove(p)}
              >
                <Button type="text" size="small" danger style={{ marginTop: 2 }}>
                  Remove
                </Button>
              </Popconfirm>
            </div>
          ))}
        </div>
      )}

      <div style={{ display: "flex", gap: 8, alignItems: "center", flexWrap: "wrap" }}>
        <Input
          value={newLabel}
          onChange={(e) => setNewLabel(e.target.value)}
          placeholder="ชื่อบอกสภาพ เช่น ฝาเปิด (ไม่บังคับ)"
          maxLength={60}
          disabled={full}
          style={{ width: 260 }}
        />
        <Upload
          accept="image/png,image/jpeg"
          showUploadList={false}
          disabled={full || uploading}
          beforeUpload={(file) => {
            void upload(file);
            return false; // we upload ourselves, not antd
          }}
        >
          <Button type="primary" disabled={full} loading={uploading}>
            {full ? `All ${max} used` : "Add a picture…"}
          </Button>
        </Upload>
        <Typography.Text type="secondary">
          {pictures.length} of {max}
        </Typography.Text>
      </div>
    </Modal>
  );
}
