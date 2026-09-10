import { useEffect, useState, type ChangeEvent, type FormEvent } from "react";

import { publishRichMenu } from "../services/adminService";

const TARGET_WIDTH = 2500;
const TARGET_HEIGHT = 843;
const MAX_BYTES = 1024 * 1024;

function canvasToBlob(canvas: HTMLCanvasElement, quality: number): Promise<Blob> {
  return new Promise((resolve, reject) => {
    canvas.toBlob(
      (blob) => (blob ? resolve(blob) : reject(new Error("Could not prepare the rich menu image."))),
      "image/jpeg",
      quality,
    );
  });
}

async function prepareRichMenu(file: File): Promise<File> {
  if (!/^image\/(png|jpeg)$/.test(file.type)) {
    throw new Error("Choose a PNG or JPEG image.");
  }

  const bitmap = await createImageBitmap(file);
  try {
    const canvas = document.createElement("canvas");
    canvas.width = TARGET_WIDTH;
    canvas.height = TARGET_HEIGHT;

    const context = canvas.getContext("2d");
    if (!context) throw new Error("Your browser cannot prepare this image.");

    context.fillStyle = "#ffffff";
    context.fillRect(0, 0, TARGET_WIDTH, TARGET_HEIGHT);

    // Cover the LINE canvas without stretching the artwork. Only overflow at
    // the outer edges is cropped when the source aspect ratio is different.
    const scale = Math.max(TARGET_WIDTH / bitmap.width, TARGET_HEIGHT / bitmap.height);
    const width = bitmap.width * scale;
    const height = bitmap.height * scale;
    context.drawImage(bitmap, (TARGET_WIDTH - width) / 2, (TARGET_HEIGHT - height) / 2, width, height);

    let quality = 0.92;
    let blob = await canvasToBlob(canvas, quality);
    while (blob.size > MAX_BYTES && quality > 0.4) {
      quality -= 0.08;
      blob = await canvasToBlob(canvas, quality);
    }

    if (blob.size > MAX_BYTES) {
      throw new Error("The prepared image is still larger than LINE's 1 MB limit.");
    }

    return new File([blob], "alert-bot-rich-menu.jpg", {
      type: "image/jpeg",
      lastModified: Date.now(),
    });
  } finally {
    bitmap.close();
  }
}

export function RichMenuPublisher() {
  const [image, setImage] = useState<File | null>(null);
  const [preview, setPreview] = useState("");
  const [preparing, setPreparing] = useState(false);
  const [publishing, setPublishing] = useState(false);
  const [notice, setNotice] = useState("");
  const [error, setError] = useState("");

  useEffect(() => () => {
    if (preview) URL.revokeObjectURL(preview);
  }, [preview]);

  async function chooseImage(event: ChangeEvent<HTMLInputElement>) {
    const source = event.target.files?.[0];
    if (!source) return;

    setPreparing(true);
    setImage(null);
    setNotice("");
    setError("");

    try {
      const prepared = await prepareRichMenu(source);
      setImage(prepared);
      setPreview(URL.createObjectURL(prepared));
    } catch (caught) {
      setPreview("");
      setError(caught instanceof Error ? caught.message : "Could not prepare the image.");
      event.target.value = "";
    } finally {
      setPreparing(false);
    }
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!image || preparing || publishing) return;

    setPublishing(true);
    setNotice("");
    setError("");
    try {
      const result = await publishRichMenu(image);
      setNotice(`Published ${result.rich_menu_id} as the default LINE rich menu.`);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Could not publish the rich menu.");
    } finally {
      setPublishing(false);
    }
  }

  return (
    <form className="rich-menu-publisher" onSubmit={submit}>
      <div>
        <p>LINE RICH MENU</p>
        <h2>Publish default menu</h2>
        <span>The browser crops and compresses your image to LINE's required 2500 × 843 JPEG.</span>
        {image && <small>READY · 2500 × 843 · {Math.ceil(image.size / 1024)} KB · JPEG</small>}
      </div>

      <label className="rich-menu-upload">
        {preview ? <img src={preview} alt="Prepared LINE rich menu preview" /> : <span>{preparing ? "Preparing image…" : "Choose PNG or JPEG"}</span>}
        <input type="file" accept="image/png,image/jpeg" onChange={chooseImage} disabled={preparing || publishing} />
      </label>

      <button type="submit" disabled={!image || preparing || publishing}>
        {publishing ? "Publishing…" : "Publish to LINE"}
      </button>

      {notice && <p className="success">{notice}</p>}
      {error && <p className="error">{error}</p>}
    </form>
  );
}
