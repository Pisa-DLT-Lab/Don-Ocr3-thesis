"""
This script performs attribution on a nanoGPT model trained on the Tiny Stories dataset.
The input file is a contains the model output, and the script outputs a file containing the attribution scores.
Author: Matteo Loporchio
"""

import numpy as np
import os
import shutil
import sys
import time
import torch
import model_utils
from pathlib import Path
from torch.utils.data import Dataset
from torch.utils.data import DataLoader
from dattri.algorithm.trak import TRAKAttributor
from dattri.benchmark.datasets.shakespeare_char.data import CustomDataset
from dattri.benchmark.load import load_benchmark
from dattri.benchmark.models.nanoGPT.model import GPT, GPTConfig
from dattri.task import AttributionTask

INPUT_FILE = sys.argv[1] # Contains the previously computed model output
OUTPUT_FILE = sys.argv[2] # Where the scores will be saved
device = sys.argv[3]

TEMP_DIR = "./tmp"
META_PATH = "./nanoGPT/data/shakespeare_char/meta.pkl"
data_path = Path("./nanoGPT/data/shakespeare_char")
checkpoint_path = "./nanoGPT/out-shakespeare-char/ckpt.pt"
block_size = 64
batch_size = 256

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

os.makedirs(TEMP_DIR, exist_ok=True)

print("Loading train and test datasets...")
start_time = time.time()
# Load training dataset.
train_data = np.memmap(os.path.join(data_path, 'train.bin'), dtype=np.uint16, mode='r')
train_dataset = CustomDataset(train_data, block_size)
train_loader = DataLoader(
    train_dataset, 
    batch_size=batch_size, 
    shuffle=False
)
# Load model output as test dataset.
encode, decode = model_utils.load_meta(META_PATH)
model_utils.convert(encode, INPUT_FILE, os.path.join(TEMP_DIR, 'val.bin'))
val_data = np.memmap(os.path.join(TEMP_DIR, 'val.bin'), dtype=np.uint16, mode='r')
val_dataset = CustomDataset(val_data, block_size)
val_loader = DataLoader(
    val_dataset, 
    batch_size=batch_size,
    shuffle=False
)
print(f"Train size: {len(train_data):,}\nTest size: {len(val_data):,}")
elapsed = time.time() - start_time
print(f"Done. Took {elapsed:.3f} seconds.")

print("Loading model...")
start_time = time.time()
checkpoint = torch.load(checkpoint_path, map_location=device)
model_args = checkpoint["model_args"]
print(model_args)
gptconf = GPTConfig(**model_args)
model = GPT(gptconf)
model.to(device)
model.eval()
elapsed = time.time() - start_time
print(f"Done. Took {elapsed:.3f} seconds.")

print(f"Performing training data attribution (device = {device})...")
start_time = time.time()

checkpoints_list = [load_model_from_checkpoint(checkpoint_path, device)]

task = AttributionTask(
    loss_func=loss_func, 
    model=model, 
    checkpoints=checkpoints_list
)

attributor = TRAKAttributor(
    task=task,
    correct_probability_func=correctness_p,
    projector_kwargs={"device": device},
    device=device
)

with torch.no_grad():
    attributor.cache(train_loader)
    score = attributor.attribute(val_loader)
    np.savetxt(OUTPUT_FILE, score.detach().cpu().numpy())

elapsed = time.time() - start_time

print(f"Done. Took {elapsed:.3f} seconds.")
shutil.rmtree(TEMP_DIR)  # Remove temporary directory
