import os
import uuid
import time
import threading
import queue
import numpy as np
import torch
from contextlib import nullcontext
from flask import Flask, request, jsonify

# --- MODEL IMPORTS ---
import model_utils
from torch.utils.data import DataLoader
from dattri.algorithm.trak import TRAKAttributor
from dattri.benchmark.datasets.shakespeare_char.data import CustomDataset
from dattri.benchmark.models.nanoGPT.model import GPT, GPTConfig
from dattri.task import AttributionTask

APP_PORT = 53000
APP_HOST = "localhost"
TEMP_DIR = "./tmp_jobs"
TRAIN_DATASET_PATH = "./nanoGPT/data/shakespeare_char/train.bin"
CHECKPOINT_PATH = "./nanoGPT/out-shakespeare-char/ckpt.pt"
META_PATH = "./nanoGPT/data/shakespeare_char/meta.pkl"
DEVICE = "cpu"
NUM_SAMPLES = 1
MAX_NEW_TOKENS = 300
TEMPERATURE = 0.8
TOP_K = 200
BLOCK_SIZE = 64
BATCH_SIZE = 256
SEED = 1337
NORM_FACTOR = 1e18

os.makedirs(TEMP_DIR, exist_ok=True)

app = Flask(__name__)

# SHARED MEMORY AND LOCK
JOBS = {}
jobs_lock = threading.Lock()
JOB_QUEUE = queue.Queue()

# Global model variables
ctx = nullcontext()
torch.manual_seed(SEED)
train_data = np.memmap(TRAIN_DATASET_PATH, dtype=np.uint16, mode='r')
train_dataset = CustomDataset(train_data, BLOCK_SIZE)
train_loader = DataLoader(train_dataset, batch_size=BATCH_SIZE, shuffle=False)
encode_f, decode_f = model_utils.load_meta(META_PATH)
encode = encode_f
decode = decode_f
model = None
checkpoints_list = None

def softmax(x):
    e = np.exp(x - np.max(x))  # stability trick
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
    p = torch.exp(-loss)
    return p

def load_model():
    global model
    print("Loading model...")
    checkpoint = torch.load(CHECKPOINT_PATH, map_location=DEVICE)
    gptconf = GPTConfig(**checkpoint['model_args'])
    model = GPT(gptconf)
    state_dict = checkpoint['model']
    unwanted_prefix = '_orig_mod.'
    for k,v in list(state_dict.items()):
        if k.startswith(unwanted_prefix):
            state_dict[k[len(unwanted_prefix):]] = state_dict.pop(k)
    model.load_state_dict(state_dict)
    model.eval()
    model.to(DEVICE)
    print("Model loaded and ready.")

def load_model_from_checkpoint(ckpt_path, device):
    # TODO: This forces some extra memory usage
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
    x = (torch.tensor(start_ids, dtype=torch.long, device=DEVICE)[None, ...])
    with torch.no_grad():
        with ctx:
            y = model.generate(x, MAX_NEW_TOKENS, temperature=TEMPERATURE, top_k=TOP_K)
            return decode(y[0].tolist())

# --- ATTRIBUTION LOGIC ---
def compute_attribution(job_id, input_file, output_file, method):
    if method == "trak":
        # Prepare test dataset from model output
        model_utils.convert(encode, input_file, os.path.join(TEMP_DIR, f'val_{job_id}.bin'))
        val_data = np.memmap(os.path.join(TEMP_DIR, f'val_{job_id}.bin'), dtype=np.uint16, mode='r')
        val_dataset = CustomDataset(val_data, BLOCK_SIZE)
        val_loader = DataLoader(
            val_dataset, 
            batch_size=BATCH_SIZE,
            shuffle=False
        )
        task = AttributionTask(
            loss_func=loss_func, 
            model=model, 
            checkpoints=checkpoints_list
        )
        attributor = TRAKAttributor(
            task=task,
            correct_probability_func=correctness_p,
            projector_kwargs={"device": DEVICE},
            device=DEVICE
        )
        with torch.no_grad():
            attributor.cache(train_loader)
            score = attributor.attribute(val_loader)
            np.savetxt(output_file, score.detach().cpu().numpy())
    # For testing purposes, we can generate dummy attribution values 
    # instead of running the full TRAK pipeline.
    elif method == "dummy":
        size = len(train_data) // BLOCK_SIZE
        mu, sigma = 3., 1.  # mean and standard deviation
        values = np.random.lognormal(mu, sigma, size).tolist() # Generate dummy attribution values
        np.savetxt(output_file, values)
    else:
        raise ValueError(f"Unknown attribution method: {method}")
    

def run_full_process(job_id, prompt):
    print(f"[JOB {job_id}] Step 1: Generating text...")

    input_filename = os.path.join(TEMP_DIR, f"{job_id}.in")
    output_filename = os.path.join(TEMP_DIR, f"{job_id}.out")

    try:
        # A. Generation
        generated_story = generate_text(prompt)
        print(f"[JOB {job_id}] Generated text.")

        # B. Write to file
        with open(input_filename, "w", encoding="utf-8") as f:
            f.write(generated_story)

        # C. Attribution (Subprocess)
        print(f"[JOB {job_id}] Step 2: Computing Attribution...")
        start_time = time.time()
        compute_attribution(job_id, input_filename, output_filename, method="dummy")
        end_time = time.time() - start_time
        print(f"[JOB {job_id}] Attribution computed in {end_time:.2f} seconds.")
        # cmd = [sys.executable, "model_attributor.py", input_filename, output_filename]

        # D. Read and Aggregate results
        print(f"[JOB {job_id}] Step 3: Reading results...")
        raw_data = np.loadtxt(output_filename)

        if raw_data.ndim == 2:
            aggregated_scores = np.mean(raw_data, axis=1)
        elif raw_data.ndim == 1:
            aggregated_scores = raw_data
        else:
            aggregated_scores = np.array([raw_data])

        if aggregated_scores.ndim == 0: 
            aggregated_scores = np.array([aggregated_scores])

        # E. Normalize the result and convert to BigInt strings
        normalized_scores = softmax(aggregated_scores)
        blockchain_ready_scores = []
        for score in normalized_scores:
            # Protection against NaN or Infinity
            if np.isnan(score) or np.isinf(score): 
                score = 0.0
            int_val = int(float(score) * NORM_FACTOR)
            blockchain_ready_scores.append(str(int_val))

        # F. Save Result
        with jobs_lock:
            JOBS[job_id]["result"] = blockchain_ready_scores
            JOBS[job_id]["status"] = "completed"

        print(f"[JOB {job_id}] COMPLETED SUCCESSFULLY!")

    except Exception as e:
        print(f"[JOB {job_id}] CRITICAL ERROR: {str(e)}")
        with jobs_lock:
            JOBS[job_id]["status"] = "error"
            JOBS[job_id]["error"] = str(e)

    finally:
        # Clean up input file, keep output for debug if needed
        if os.path.exists(input_filename): os.remove(input_filename)

# --- BACKGROUND WORKER (SEQUENTIAL EXECUTION) ---
def background_worker():
    """
    This thread runs in background and takes only one job at a time
    from the queue and execute the AI. This grants no parallel executions
    """
    while True:
        job_id, prompt = JOB_QUEUE.get()

        # Update the status to "processing" only when the job starts
        with jobs_lock:
            if job_id in JOBS:
                JOBS[job_id]["status"] = "processing"

        try:
            run_full_process(job_id, prompt)
        except Exception as e:
            print(f"[WORKER] Unexpected error for the job {job_id}: {e}")
        finally:
            # Alert the queue that this task is done, unlocking the next
            JOB_QUEUE.task_done()

# --- ENDPOINTS ---

@app.route('/attribute', methods=['POST'])
def attribute():
    data = request.json
    if not data or 'text' not in data:
        return jsonify({"error": "Missing 'text' field"}), 400

    # --- ROBUST ID LOGIC ---
    # Reads job_id (new standard) OR cid (old standard) OR creates one
    job_id = data.get('job_id') or data.get('cid') or str(uuid.uuid4())

    prompt = data['text']

    with jobs_lock:
        if job_id in JOBS:
            print(f"--> [DEDUPLICATION] Duplicate request for Job {job_id}. Ignoring.")
            return jsonify({
                "message": "Job already exists",
                "job_id": job_id,
                "status": JOBS[job_id]["status"]
            }), 200

        JOBS[job_id] = {"status": "queued", "result": None}

    print(f"--> [NEW] Queuing Job {job_id}")
    # QUEUE THE JOB 
    JOB_QUEUE.put((job_id, prompt))

    return jsonify({"message": "Job Queued", "job_id": job_id}), 202

@app.route('/result/<job_id>', methods=['GET'])
def get_result(job_id):
    job = JOBS.get(job_id)
    if not job:
        return jsonify({"error": "Job not found"}), 404
        
    return jsonify(job), 200

if __name__ == '__main__':
    # Load the model.
    load_model()
    checkpoints_list = [load_model_from_checkpoint(CHECKPOINT_PATH, DEVICE)]

    # Start the background worker before exposing the API
    threading.Thread(target=background_worker, daemon=True).start()

    print(f"Starting server on {APP_HOST}:{APP_PORT}...")
    # debug=False to avoid double loading or thread issues
    app.run(host=APP_HOST, port=APP_PORT, threaded=True, debug=False)