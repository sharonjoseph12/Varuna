#!/usr/bin/env python3
"""
sar_infer.py — SAR ship detector for Varuna corroboration jobs.

Called by corroboration/sar_job.go as a subprocess:
    python data/models/sar_infer.py --tile <path> --lat <lat> --lon <lon>

Outputs a single JSON line to stdout:
    {"found": true,  "confidence": 0.87, "bbox": [x, y, w, h], "model_version": "yolov8n-sar-ssdd-v1"}
    {"found": false, "confidence": 0.0,  "bbox": [],            "model_version": "yolov8n-sar-ssdd-v1"}

Exits 0 always — errors are reported as {"found": false, "error": "..."}.

Model: YOLOv8n fine-tuned on SSDD/HRSID SAR ship detection dataset, exported
       to ONNX. Place the model file at data/models/yolov8n_sar_ssdd.onnx.

       Download from HuggingFace:
         https://huggingface.co/models?search=sar+ship+detection+yolov8
       Or fine-tune yourself:
         pip install ultralytics
         yolo train model=yolov8n.pt data=ssdd.yaml epochs=50
         yolo export model=runs/detect/train/weights/best.pt format=onnx

NOTE: Do NOT use a COCO-pretrained checkpoint (yolov8n.pt without fine-tuning).
COCO has never seen SAR imagery and will detect nothing useful.
"""

import argparse
import json
import os
import sys

MODEL_VERSION = "yolov8n-sar-ssdd-v1"
MODEL_PATH = os.path.join(os.path.dirname(__file__), "yolov8n_sar_ssdd.onnx")

# Crop window around vessel position: 1.0 degree × 1.0 degree (~111 km at equator)
CROP_DEG = 0.5


def load_model():
    """Load ONNX model via onnxruntime."""
    import onnxruntime as ort
    sess = ort.InferenceSession(MODEL_PATH, providers=["CPUExecutionProvider"])
    return sess


def load_tile_crop(tile_path: str, lat: float, lon: float):
    """
    Load a crop of the Sentinel-1 GRD tile centred on (lat, lon).
    Returns a float32 numpy array of shape (1, 3, 640, 640) ready for YOLO.
    Returns None if the tile cannot be read.
    """
    try:
        from osgeo import gdal
        import numpy as np
        from PIL import Image

        ds = gdal.Open(tile_path)
        if ds is None:
            return None

        gt = ds.GetGeoTransform()  # (origin_x, pixel_w, 0, origin_y, 0, pixel_h)
        # Convert lat/lon to pixel coordinates
        px = int((lon - gt[0]) / gt[1])
        py = int((lat - gt[3]) / gt[5])
        half = int(CROP_DEG / abs(gt[1]) / 2)

        x0, y0 = max(0, px - half), max(0, py - half)
        x1 = min(ds.RasterXSize, px + half)
        y1 = min(ds.RasterYSize, py + half)

        band = ds.GetRasterBand(1).ReadAsArray(x0, y0, x1 - x0, y1 - y0)
        if band is None:
            return None

        # Normalise to 0–255, resize to 640×640 for YOLO input
        band_norm = ((band - band.min()) / (band.ptp() + 1e-8) * 255).astype("uint8")
        img = Image.fromarray(band_norm).convert("RGB").resize((640, 640))
        arr = np.array(img, dtype="float32") / 255.0
        return arr.transpose(2, 0, 1)[None]  # (1, 3, 640, 640)

    except Exception as e:
        return None


def run_inference(session, img_array) -> dict:
    """Run ONNX model and return the highest-confidence detection (if any)."""
    import numpy as np

    input_name = session.get_inputs()[0].name
    outputs = session.run(None, {input_name: img_array})

    # YOLOv8 ONNX output: (1, 5+num_classes, num_anchors) or (1, num_anchors, 5+classes)
    # Shape depends on export settings; handle both transpositions.
    preds = outputs[0]
    if preds.ndim == 3 and preds.shape[1] < preds.shape[2]:
        preds = preds.transpose(0, 2, 1)  # → (1, num_anchors, 5+classes)

    preds = preds[0]  # (num_anchors, 5+classes)
    conf_threshold = 0.40

    # columns: cx, cy, w, h, obj_conf, class_conf...
    obj_conf = preds[:, 4]
    best_idx = int(np.argmax(obj_conf))
    best_conf = float(obj_conf[best_idx])

    if best_conf < conf_threshold:
        return {"found": False, "confidence": best_conf, "bbox": [], "model_version": MODEL_VERSION}

    cx, cy, bw, bh = preds[best_idx, :4]
    x = int((cx - bw / 2) * 640)
    y = int((cy - bh / 2) * 640)
    w = int(bw * 640)
    h = int(bh * 640)

    return {
        "found": True,
        "confidence": round(float(best_conf), 4),
        "bbox": [x, y, w, h],
        "model_version": MODEL_VERSION,
    }


def main():
    parser = argparse.ArgumentParser(description="SAR ship detector for Varuna")
    parser.add_argument("--tile", required=True, help="Path to Sentinel-1 GRD .tiff file")
    parser.add_argument("--lat", type=float, required=True, help="Vessel latitude")
    parser.add_argument("--lon", type=float, required=True, help="Vessel longitude")
    args = parser.parse_args()

    # Graceful degradation: if model file missing, output not-found with explanation
    if not os.path.exists(MODEL_PATH):
        result = {
            "found": False,
            "confidence": 0.0,
            "bbox": [],
            "model_version": MODEL_VERSION,
            "error": f"model not found at {MODEL_PATH} — download from HuggingFace or train on SSDD/HRSID",
        }
        print(json.dumps(result))
        sys.exit(0)

    if not os.path.exists(args.tile):
        result = {
            "found": False,
            "confidence": 0.0,
            "bbox": [],
            "model_version": MODEL_VERSION,
            "error": f"tile not found: {args.tile}",
        }
        print(json.dumps(result))
        sys.exit(0)

    try:
        session = load_model()
        img = load_tile_crop(args.tile, args.lat, args.lon)
        if img is None:
            print(json.dumps({"found": False, "confidence": 0.0, "bbox": [], "model_version": MODEL_VERSION,
                               "error": "could not read/crop tile (missing gdal or corrupt file)"}))
            sys.exit(0)
        result = run_inference(session, img)
        print(json.dumps(result))
    except Exception as e:
        print(json.dumps({"found": False, "confidence": 0.0, "bbox": [], "model_version": MODEL_VERSION,
                           "error": str(e)}))
    sys.exit(0)


if __name__ == "__main__":
    main()
