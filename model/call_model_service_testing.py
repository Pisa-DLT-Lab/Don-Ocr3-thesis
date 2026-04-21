#!/usr/bin/env python3
import argparse
import json
import os
import sys
import time
from contextlib import nullcontext
from pathlib import Path

import numpy as np
import torch

BASE_DIR = Path(__file__).resolve().parent
REPO_ROOT = BASE_DIR.parent
sys.path.insert(0, str(BASE_DIR))

# --- MODEL IMPORTS ---
import model_utils
from torch.utils.data import DataLoader
from dattri.algorithm.trak import TRAKAttributor
from dattri.benchmark.datasets.shakespeare_char.data import CustomDataset
from dattri.benchmark.models.nanoGPT.model import GPT, GPTConfig
from dattri.task import AttributionTask

TEMP_DIR = BASE_DIR / "tmp_jobs"
TRAIN_DATASET_HOLDERS_PATH = BASE_DIR / "nanoGPT/data/shakespeare_char/holders.txt"
TRAIN_DATASET_PATH = BASE_DIR / "nanoGPT/data/shakespeare_char/train.bin"
CHECKPOINT_PATH = BASE_DIR / "nanoGPT/out-shakespeare-char/ckpt.pt"
META_PATH = BASE_DIR / "nanoGPT/data/shakespeare_char/meta.pkl"
DEVICE = "cpu"
MAX_NEW_TOKENS = 300
TEMPERATURE = 0.8
TOP_K = 200
BLOCK_SIZE = 64
BATCH_SIZE = 256
SEED = 1337
NORM_FACTOR = 10**18
SCORE_BIT_WIDTH = 96
FILTER_POLICIES = ["TOP_VALUES", "TOP_HOLDERS"]
DEFAULT_TEXT = "To be or not to be that is the question"

TEMP_DIR.mkdir(parents=True, exist_ok=True)

ctx = nullcontext()
torch.manual_seed(SEED)
np.random.seed(SEED)

HOLDERS = np.loadtxt(TRAIN_DATASET_HOLDERS_PATH, dtype=int).tolist()
train_data = np.memmap(TRAIN_DATASET_PATH, dtype=np.uint16, mode="r")
train_dataset = CustomDataset(train_data, BLOCK_SIZE)
train_loader = DataLoader(train_dataset, batch_size=BATCH_SIZE, shuffle=False)
encode, decode = model_utils.load_meta(META_PATH)
model = None
checkpoints_list = None


def softmax(x):
    e = np.exp(x - np.max(x))
    return e / e.sum()


def loss_func(params, data_target_pair):
    x, y = data_target_pair
    x_t = x.unsqueeze(0)
    y_t = y.unsqueeze(0)
    _, loss = torch.func.functional_call(model, params, (x_t, y_t))
    logp = -loss
    return logp - torch.log(1 - torch.exp(logp))


def correctness_p(params, image_label_pair):
    x, y = image_label_pair
    x_t = x.unsqueeze(0)
    y_t = y.unsqueeze(0)
    _, loss = torch.func.functional_call(model, params, (x_t, y_t))
    return torch.exp(-loss)


def load_model():
    global model
    print("Loading model...")
    checkpoint = torch.load(CHECKPOINT_PATH, map_location=DEVICE)
    gptconf = GPTConfig(**checkpoint["model_args"])
    model = GPT(gptconf)
    state_dict = checkpoint["model"]
    unwanted_prefix = "_orig_mod."
    for k, _ in list(state_dict.items()):
        if k.startswith(unwanted_prefix):
            state_dict[k[len(unwanted_prefix) :]] = state_dict.pop(k)
    model.load_state_dict(state_dict)
    model.eval()
    model.to(DEVICE)
    print("Model loaded and ready.")


def load_model_from_checkpoint(ckpt_path, device):
    checkpoint = torch.load(ckpt_path, map_location=device)
    state_dict = checkpoint["model"]
    new_state_dict = {}
    for k, v in state_dict.items():
        if k.startswith("_orig_mod."):
            new_state_dict[k[len("_orig_mod.") :]] = v
        else:
            new_state_dict[k] = v
    return new_state_dict


def generate_text(prompt):
    start_ids = encode(prompt)
    x = torch.tensor(start_ids, dtype=torch.long, device=DEVICE)[None, ...]
    with torch.no_grad():
        with ctx:
            y = model.generate(x, MAX_NEW_TOKENS, temperature=TEMPERATURE, top_k=TOP_K)
            return decode(y[0].tolist())


def compute_attribution(job_id, input_file, output_file, method):
    if method == "trak":
        model_utils.convert(encode, input_file, TEMP_DIR / f"val_{job_id}.bin")
        val_data = np.memmap(TEMP_DIR / f"val_{job_id}.bin", dtype=np.uint16, mode="r")
        val_dataset = CustomDataset(val_data, BLOCK_SIZE)
        val_loader = DataLoader(val_dataset, batch_size=BATCH_SIZE, shuffle=False)
        task = AttributionTask(
            loss_func=loss_func,
            model=model,
            checkpoints=checkpoints_list,
        )
        attributor = TRAKAttributor(
            task=task,
            correct_probability_func=correctness_p,
            projector_kwargs={"device": DEVICE},
            device=DEVICE,
        )
        with torch.no_grad():
            attributor.cache(train_loader)
            score = attributor.attribute(val_loader)
            np.savetxt(output_file, score.detach().cpu().numpy())
    elif method == "dummy":
        size = len(train_data) // BLOCK_SIZE
        values = np.random.exponential(1, size)
        values = values / values.sum()
        np.savetxt(output_file, values.tolist())
    else:
        raise ValueError(f"Unknown attribution method: {method}")


def advance_dummy_attribution_rng(job_count):
    if job_count <= 0:
        return
    size = len(train_data) // BLOCK_SIZE
    for _ in range(job_count):
        np.random.exponential(1, size)


def process_attribution_scores(raw_data, filter_policy):
    if raw_data.ndim == 2:
        normalized_scores = softmax(np.mean(raw_data, axis=1)).tolist()
    elif raw_data.ndim == 1:
        normalized_scores = raw_data.tolist()
    else:
        raise ValueError("Unexpected shape of attribution scores")

    if len(normalized_scores) != len(HOLDERS):
        raise ValueError("Scores and holders length mismatch")

    combined = list(zip(HOLDERS, normalized_scores))
    if filter_policy == "TOP_VALUES":
        results = combined
    elif filter_policy == "TOP_HOLDERS":
        score_by_holder = {}
        for holder, score in combined:
            score_by_holder[holder] = score_by_holder.get(holder, 0.0) + score
        results = list(score_by_holder.items())
    else:
        raise ValueError(f"Invalid filter policy: {filter_policy}")

    results = sorted(results, key=lambda item: item[0])
    holder_ids, scores = zip(*results)

    int_scores = []
    for score in scores:
        if np.isnan(score) or np.isinf(score):
            score = 0.0
        int_scores.append(int(float(score) * NORM_FACTOR))

    return list(holder_ids), int_scores


def select_top_scores(sorted_list, threshold):
    ordered = sorted(
        sorted_list,
        key=lambda item: (-int(item[1]), int(item[0])),
    )
    selected = ordered[: max(0, min(threshold, len(ordered)))]
    return selected, sorted(selected, key=lambda item: int(item[0]))


def pack_holder_scores(sorted_list):
    packed = []
    seen = set()
    for holder_id, score in sorted_list:
        holder_id = int(holder_id)
        score = int(score)
        if holder_id in seen:
            raise ValueError(f"duplicate holder_id {holder_id}; cannot pack as oracle holder-score vector")
        if holder_id < 0 or holder_id > ((1 << 32) - 1):
            raise ValueError(f"holder_id {holder_id} exceeds uint32")
        if score < 0 or score > ((1 << 96) - 1):
            raise ValueError(f"score for holder_id {holder_id} exceeds uint96")
        seen.add(holder_id)
        packed.append(str((holder_id << SCORE_BIT_WIDTH) | score))
    return packed


def format_score(raw_score):
    raw = int(raw_score)
    whole = raw // NORM_FACTOR
    fraction = str(raw % NORM_FACTOR).zfill(18).rstrip("0")
    if not fraction:
        return str(whole)
    return f"{whole}.{fraction}"


def render_verify_style_text(result):
    rows = result["result"]["filtered_sorted_list"]
    lines = [
        f"   - Array Len: {len(rows)} items",
        "",
        "Decoded Final Scores:",
        "   ------------------------------------",
    ]
    for index, (holder_id, score) in enumerate(rows):
        lines.append(f"   [{index}]: holder={holder_id} score={format_score(score)}")
    lines.append("")
    return "\n".join(lines)


def run_job(job_id, prompt, filter_policy, threshold, method, simulate_job_index):
    input_filename = TEMP_DIR / f"{job_id}.in"
    output_filename = TEMP_DIR / f"{job_id}.out"

    print(f"[JOB {job_id}] Step 1: Generating text...")
    generated_story = generate_text(prompt)
    input_filename.write_text(generated_story, encoding="utf-8")
    print(f"[JOB {job_id}] Generated text.")

    try:
        print(f"[JOB {job_id}] Step 2: Computing attribution with method={method}...")
        start_time = time.time()
        compute_attribution(job_id, input_filename, output_filename, method=method)
        print(f"[JOB {job_id}] Attribution computed in {time.time() - start_time:.2f} seconds.")

        print(f"[JOB {job_id}] Step 3: Reading and aggregating attribution results...")
        raw_data = np.loadtxt(output_filename)
        holder_ids, scores = process_attribution_scores(raw_data, filter_policy)
        sorted_list = [[holder_id, str(score)] for holder_id, score in zip(holder_ids, scores)]
        score_ordered_top_list, filtered_sorted_list = select_top_scores(sorted_list, threshold)

        result = {
            "status": "completed",
            "job_id": str(job_id),
            "simulated_job_index": simulate_job_index,
            "filter_policy": filter_policy,
            "threshold": threshold,
            "method": method,
            "prompt": prompt,
            "generated_text": generated_story,
            "result": {
                "holder_ids": holder_ids,
                "scores": scores,
                "sorted_list": sorted_list,
                "score_ordered_top_list": score_ordered_top_list,
                "filtered_sorted_list": filtered_sorted_list,
            },
        }

        try:
            result["result"]["packed_values"] = pack_holder_scores(sorted_list)
        except ValueError as exc:
            result["result"]["packed_values_error"] = str(exc)

        try:
            result["result"]["filtered_packed_values"] = pack_holder_scores(filtered_sorted_list)
        except ValueError as exc:
            result["result"]["filtered_packed_values_error"] = str(exc)

        try:
            result["result"]["score_ordered_top_packed_values"] = pack_holder_scores(score_ordered_top_list)
        except ValueError as exc:
            result["result"]["score_ordered_top_packed_values_error"] = str(exc)

        print(f"[JOB {job_id}] COMPLETED SUCCESSFULLY.")
        return result
    finally:
        if input_filename.exists():
            input_filename.unlink()


def parse_args():
    parser = argparse.ArgumentParser(
        description="Run the model attribution path once without starting a Flask server."
    )
    parser.add_argument("--job-id", default=str(int(time.time())), help="Job id label for output files.")
    parser.add_argument("--text", default=DEFAULT_TEXT, help="Prompt text to evaluate.")
    parser.add_argument("--text-file", help="Read prompt text from a file instead of --text.")
    parser.add_argument(
        "--filter-policy",
        default="TOP_HOLDERS",
        choices=FILTER_POLICIES,
        help="Same filter policy name used by the oracle request.",
    )
    parser.add_argument("--threshold", type=int, default=100, help="Top-N threshold to apply for comparison.")
    parser.add_argument("--method", default="dummy", choices=["dummy", "trak"], help="Attribution method to run.")
    parser.add_argument(
        "--numpy-seed",
        type=int,
        default=SEED,
        help="NumPy seed for reproducible dummy attribution.",
    )
    parser.add_argument(
        "--simulate-job-index",
        type=int,
        default=0,
        help=(
            "Simulate the N-th sequential dummy job from a freshly seeded model server. "
            "Use 0 for the first job, 1 for the second job, etc."
        ),
    )
    parser.add_argument(
        "--out",
        default=str(REPO_ROOT / "comparison_results/model_service_testing_latest.json"),
        help="JSON output path.",
    )
    parser.add_argument(
        "--txt-out",
        help="Text output path. Defaults to the JSON output path with .txt extension.",
    )
    return parser.parse_args()


def main():
    global checkpoints_list
    args = parse_args()
    if args.simulate_job_index < 0:
        raise ValueError("--simulate-job-index must be non-negative")

    np.random.seed(args.numpy_seed)

    prompt = args.text
    if args.text_file:
        prompt = Path(args.text_file).read_text(encoding="utf-8")

    load_model()
    checkpoints_list = [load_model_from_checkpoint(CHECKPOINT_PATH, DEVICE)]
    if args.method == "dummy":
        advance_dummy_attribution_rng(args.simulate_job_index)

    result = run_job(
        job_id=args.job_id,
        prompt=prompt,
        filter_policy=args.filter_policy,
        threshold=args.threshold,
        method=args.method,
        simulate_job_index=args.simulate_job_index,
    )

    out_path = Path(args.out)
    out_path.parent.mkdir(parents=True, exist_ok=True)
    out_path.write_text(json.dumps(result, indent=2) + "\n", encoding="utf-8")
    print(f"Wrote result JSON to {out_path}")

    txt_out_path = Path(args.txt_out) if args.txt_out else out_path.with_suffix(".txt")
    txt_out_path.parent.mkdir(parents=True, exist_ok=True)
    txt_out_path.write_text(render_verify_style_text(result), encoding="utf-8")
    print(f"Wrote verify-style text to {txt_out_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
