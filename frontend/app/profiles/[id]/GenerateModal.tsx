"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import {
  App,
  Button,
  Input,
  Modal,
  Segmented,
  Select,
  Spin,
  Typography,
  Upload,
} from "antd";
import {
  ApiError,
  getStudioPricing,
  listProducts,
  MAX_IMAGE_BYTES,
  uploadImage,
} from "@/lib/api";
import { formatUsd } from "@/lib/format";
import { productSlotsNeeded } from "@/lib/template";
import {
  MAX_PRODUCTS_PER_CLIP,
  type Product,
  type PromptTemplate,
  type StudioPricing,
} from "@/lib/types";
import { mobileModal } from "@/lib/ui";
import KeyframePicker from "./KeyframePicker";
import PastPictures from "./PastPictures";

/** What the owner chose in the dialog. At most one of `imageUrl` (a pasted
 *  link) and `imageId` (a picture already uploaded) is set; both "" means a
 *  text-only run. */
export interface GenerateChoices {
  productIds: string[];
  imageUrl: string;
  imageId: string;
}

/** Where the starting image comes from: a pasted link, a file from this
 *  computer, a picture made with an image model ("ai", see KeyframePicker), or
 *  one an earlier clip of this profile already used ("past", see PastPictures). */
type ImageSource = "link" | "file" | "ai" | "past";

/** Checks a picked file before uploading it; null = fine. The backend checks
 *  the same things (and the pixel size, which the browser can't read cheaply). */
export function imageFileError(file: File): string | null {
  if (file.type !== "image/jpeg" && file.type !== "image/png") {
    return "Use a JPG or PNG image.";
  }
  if (file.size > MAX_IMAGE_BYTES) {
    return `The image is too large; the limit is ${MAX_IMAGE_BYTES / 1024 / 1024} MB.`;
  }
  return null;
}

/** Mirrors the backend check (generations.validateImageURL): an absolute
 *  http(s) link. "" is fine — it means no image. */
export function imageUrlError(raw: string): string | null {
  const s = raw.trim();
  if (!s) return null;
  try {
    const u = new URL(s);
    if (u.protocol !== "https:" && u.protocol !== "http:") throw new Error();
  } catch {
    return "Paste the full image link, starting with https://.";
  }
  return s.length > 1000 ? "Image link must be 1000 characters or fewer." : null;
}

/**
 * Asked before generating when there is something to choose: which products a
 * template that mentions them should use ({{product.name}} …; the first pick is
 * {{product.*}}, the second {{product2.*}}, and they are pinned to the new
 * clip), and — for a profile that makes video with fal — an optional image to
 * start the clip from.
 */
export default function GenerateModal({
  template,
  askImage,
  videoStyle,
  onClose,
  onGenerate,
}: {
  template: PromptTemplate | null;
  /** The profile's provider is fal, so an image can be given (and the run
   *  costs money, so it is confirmed first). */
  askImage: boolean;
  /** The profile's video style, which decides the price of a clip. */
  videoStyle: string;
  onClose: () => void;
  onGenerate: (choices: GenerateChoices) => Promise<void>;
}) {
  return (
    <Modal
      title={template ? `Generate “${template.name}”` : "Generate"}
      open={!!template}
      onCancel={onClose}
      footer={null}
      width={760}
      destroyOnHidden
      {...mobileModal}
    >
      {template && (
        <Choices
          template={template}
          askImage={askImage}
          videoStyle={videoStyle}
          onGenerate={onGenerate}
        />
      )}
    </Modal>
  );
}

function Choices({
  template,
  askImage,
  videoStyle,
  onGenerate,
}: {
  template: PromptTemplate;
  askImage: boolean;
  videoStyle: string;
  onGenerate: (choices: GenerateChoices) => Promise<void>;
}) {
  const { message, modal } = App.useApp();
  // What a clip costs, for the confirmation. null = not loaded (or not needed).
  const [pricing, setPricing] = useState<StudioPricing | null>(null);
  // The picture chosen among those earlier clips of this profile used.
  const [pastImageId, setPastImageId] = useState("");
  const need = productSlotsNeeded(template.body);
  // null = still loading; stays [] when no products are needed.
  const [catalog, setCatalog] = useState<Product[] | null>(need > 0 ? null : []);
  const [chosen, setChosen] = useState<string[]>([]);
  const [source, setSource] = useState<ImageSource>("link");
  const [imageUrl, setImageUrl] = useState("");
  // The file picked from this computer, with a local preview URL for it.
  const [picked, setPicked] = useState<{ file: File; preview: string } | null>(
    null,
  );
  // The picture chosen among the ones made with an image model.
  const [madeImageId, setMadeImageId] = useState("");
  const [busy, setBusy] = useState(false);

  // A preview URL holds the file in memory until revoked. Replace/remove
  // revoke it in pick(); this covers closing the dialog with a file still picked.
  const pickedRef = useRef(picked);
  useEffect(() => {
    pickedRef.current = picked; // kept current for the unmount cleanup below
  }, [picked]);
  useEffect(
    () => () => {
      if (pickedRef.current) URL.revokeObjectURL(pickedRef.current.preview);
    },
    [],
  );

  function pick(file: File | null) {
    if (picked) URL.revokeObjectURL(picked.preview);
    setPicked(file ? { file, preview: URL.createObjectURL(file) } : null);
  }

  useEffect(() => {
    if (need === 0) return;
    let cancelled = false;
    listProducts()
      .then((p) => {
        if (!cancelled) setCatalog(p);
      })
      .catch((err) => {
        message.error(
          err instanceof Error ? err.message : "Failed to load products.",
        );
        if (!cancelled) setCatalog([]);
      });
    return () => {
      cancelled = true;
    };
  }, [need, message]);

  // The clip price for the confirmation. If it cannot be loaded the dialog
  // still works and says the price is unknown.
  useEffect(() => {
    if (!askImage) return;
    let cancelled = false;
    getStudioPricing()
      .then((p) => {
        if (!cancelled) setPricing(p);
      })
      .catch(() => {
        /* the confirmation says the price is unknown */
      });
    return () => {
      cancelled = true;
    };
  }, [askImage]);

  if (catalog === null) {
    return (
      <div style={{ padding: 32, textAlign: "center" }}>
        <Spin />
      </div>
    );
  }

  if (need > 0 && catalog.length === 0) {
    return (
      <Typography.Paragraph type="secondary">
        This template mentions a product, but the catalog is empty.{" "}
        <Link href="/products">Add a product</Link> first.
      </Typography.Paragraph>
    );
  }

  const useLink = askImage && source === "link";
  const useFile = askImage && source === "file" && picked !== null;
  const useMade = askImage && source === "ai";
  const usePast = askImage && source === "past";
  const imageError = useLink ? imageUrlError(imageUrl) : null;

  // Whether the clip will start from a picture, whatever its source.
  const hasImage =
    (useLink && imageUrl.trim() !== "") ||
    useFile ||
    (useMade && madeImageId !== "") ||
    (usePast && pastImageId !== "");

  // A fal clip spends credits, so ask first, with the price. A profile that only
  // makes text (no askImage) costs nothing and goes straight through.
  function confirmGo() {
    if (!askImage) {
      void go();
      return;
    }
    const seconds = pricing?.defaultSeconds ?? 5;
    const perSecond = pricing?.imageUsdPerSecond[videoStyle];
    const price = perSecond === undefined ? undefined : perSecond * seconds;
    modal.confirm({
      title: "Generate this clip?",
      content: hasImage ? (
        <p style={{ margin: 0 }}>
          A {seconds}-second silent clip from the chosen picture
          {price !== undefined ? (
            <>
              : <strong>≈ {formatUsd(price)}</strong>, charged to your fal.ai
              credits.
            </>
          ) : (
            ". The price could not be loaded, so it is not shown here; it is charged to your fal.ai credits."
          )}
        </p>
      ) : (
        <p style={{ margin: 0 }}>
          A clip from the prompt alone. This is charged to your fal.ai credits,
          and the price of a clip without a starting picture has not been
          checked yet, so it may differ from a picture clip.
        </p>
      ),
      okText: hasImage && price !== undefined ? `Generate (≈ ${formatUsd(price)})` : "Generate",
      cancelText: "Cancel",
      onOk: () => go(),
    });
  }

  async function go() {
    setBusy(true);
    try {
      // Upload first, inside the dialog, so a refused picture (too small, …)
      // shows its message here and keeps what was chosen.
      let imageId = useMade ? madeImageId : usePast ? pastImageId : "";
      if (useFile) {
        try {
          imageId = (await uploadImage(picked!.file)).id;
        } catch (err) {
          const field =
            err instanceof ApiError && err.fieldErrors
              ? Object.values(err.fieldErrors)[0]
              : undefined;
          message.error(
            field ?? (err instanceof Error ? err.message : "Upload failed."),
          );
          return;
        }
      }
      await onGenerate({
        productIds: chosen,
        imageUrl: useLink ? imageUrl.trim() : "",
        imageId,
      });
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      {need > 0 && (
        <>
          <Typography.Paragraph type="secondary">
            {need === 1
              ? "This template mentions a product. Choose one."
              : "This template mentions two products. Choose two: the first fills {{product.…}}, the second {{product2.…}}."}
          </Typography.Paragraph>
          <Select
            mode="multiple"
            maxCount={MAX_PRODUCTS_PER_CLIP}
            value={chosen}
            onChange={setChosen}
            optionFilterProp="label"
            placeholder={`Choose ${need} product${need === 1 ? "" : "s"}`}
            options={catalog.map((p) => ({ value: p.id, label: p.name }))}
            style={{ width: "100%" }}
          />
        </>
      )}

      {askImage && (
        <div style={{ marginTop: need > 0 ? 20 : 0 }}>
          <Typography.Text>Starting image (optional)</Typography.Text>
          <div style={{ marginTop: 4 }}>
            <Segmented<ImageSource>
              size="small"
              value={source}
              onChange={setSource}
              options={[
                { value: "link", label: "Link" },
                { value: "file", label: "From this computer" },
                { value: "ai", label: "Make with AI" },
                { value: "past", label: "From an earlier clip" },
              ]}
            />
          </div>

          {source === "ai" ? (
            <KeyframePicker
              profileId={template.profileId}
              value={madeImageId}
              onChange={setMadeImageId}
            />
          ) : source === "past" ? (
            <PastPictures
              profileId={template.profileId}
              value={pastImageId}
              onChange={setPastImageId}
            />
          ) : source === "link" ? (
            <Input
              value={imageUrl}
              onChange={(e) => setImageUrl(e.target.value)}
              placeholder="https://…"
              status={imageError ? "error" : undefined}
              allowClear
              style={{ marginTop: 8 }}
            />
          ) : (
            <div style={{ marginTop: 8, display: "flex", gap: 12, alignItems: "center" }}>
              <Upload
                accept="image/png,image/jpeg"
                maxCount={1}
                showUploadList={false}
                beforeUpload={(file) => {
                  const problem = imageFileError(file);
                  if (problem) message.error(problem);
                  else pick(file);
                  return false; // never auto-upload: it happens when Generate is pressed
                }}
              >
                <Button>{picked ? "Choose another…" : "Choose a file…"}</Button>
              </Upload>
              {picked && (
                <>
                  {/* A local blob: URL, which next/image cannot optimize. */}
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img
                    src={picked.preview}
                    alt="Chosen starting image"
                    style={{ width: 56, height: 56, objectFit: "cover", borderRadius: 4 }}
                  />
                  <Typography.Text
                    type="secondary"
                    ellipsis
                    style={{ minWidth: 0, flex: 1 }}
                  >
                    {picked.file.name}
                  </Typography.Text>
                  <Button type="text" size="small" onClick={() => pick(null)}>
                    Remove
                  </Button>
                </>
              )}
            </div>
          )}

          <Typography.Paragraph
            type={imageError ? "danger" : "secondary"}
            style={{ margin: "4px 0 0" }}
          >
            {imageError ??
              (source === "link"
                ? "A public link, at least 300×300 px. With an image the clip is silent and 5 seconds long; leave it empty to generate from the prompt alone."
                : source === "file"
                  ? "A JPG or PNG, at least 300×300 px, up to 8 MB. With an image the clip is silent and 5 seconds long; choose nothing to generate from the prompt alone."
                  : source === "past"
                    ? "Reuse a picture this profile's earlier clips started from: only the prompt changes, and no new picture is paid for. The clip is silent and 5 seconds long."
                    : "Make a picture first, then pick one. The clip is silent and 5 seconds long, and starts from the picture you pick.")}
          </Typography.Paragraph>
        </div>
      )}

      <Button
        type="primary"
        style={{ marginTop: 20 }}
        disabled={
          chosen.length < need ||
          !!imageError ||
          (useMade && !madeImageId) ||
          (usePast && !pastImageId)
        }
        loading={busy}
        onClick={confirmGo}
      >
        Generate
      </Button>
    </>
  );
}
