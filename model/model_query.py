"""
Author: Matteo Loporchio
"""

from contextlib import nullcontext
from dattri.benchmark.models.nanoGPT.model import GPT, GPTConfig
import configparser
import torch
import model_utils
import sys

CONFIG_FILE = 'model_config.ini'

# Read and parse the configuration file.
config = configparser.ConfigParser()
config.read(CONFIG_FILE)
meta_path = config['MODEL']['meta_path']
checkpoint_path = config['MODEL']['checkpoint_path']
device = config['MODEL']['device']
block_size = int(config['MODEL']['block_size'])
seed = int(config['MODEL']['seed'])
num_samples = int(config['MODEL']['num_samples'])
max_new_tokens = int(config['MODEL']['max_new_tokens'])
temperature = float(config['MODEL']['temperature'])
top_k = int(config['MODEL']['top_k'])
ctx = nullcontext()
torch.manual_seed(seed)

encode = None
decode = None
model = None

# Load model once at startup
def load_model():
    global encode, decode, model
    encode_f, decode_f = model_utils.load_meta(meta_path)
    encode = encode_f
    decode = decode_f
    checkpoint = torch.load(checkpoint_path, map_location=device)
    gptconf = GPTConfig(**checkpoint['model_args'])
    model = GPT(gptconf)
    state_dict = checkpoint['model']
    unwanted_prefix = '_orig_mod.'
    for k,v in list(state_dict.items()):
        if k.startswith(unwanted_prefix):
            state_dict[k[len(unwanted_prefix):]] = state_dict.pop(k)
    model.load_state_dict(state_dict)
    model.eval()
    model.to(device)

# Query the model
def query_model(query):
    start_ids = encode(query + " ")
    x = (torch.tensor(start_ids, dtype=torch.long, device=device)[None, ...])
    with torch.no_grad():
        with ctx:
            y = model.generate(x, max_new_tokens, temperature=temperature, top_k=top_k)
            return decode(y[0].tolist())

if __name__ == '__main__':
    prompt_text = sys.argv[1]
    output_file = sys.argv[2]
    load_model()
    print("Model loaded and ready to serve requests.")
    print("N. of samples:", num_samples)
    print("Max tokens:", max_new_tokens)
    print("Top k:", top_k)
    with open(output_file, 'w') as f:
        result = query_model(prompt_text)
        f.write(result)
    
