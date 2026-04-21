import numpy as np
import pickle

def load_meta(meta_path):
    with open(meta_path, 'rb') as f:
        meta = pickle.load(f)
    # TODO want to make this more general to arbitrary encoder/decoder schemes
    stoi, itos = meta['stoi'], meta['itos']
    encode = lambda s: [stoi[c] for c in s]
    decode = lambda l: ''.join([itos[i] for i in l])
    return (encode, decode)

def convert(encode, input_file, output_file):
    with open(input_file, 'r') as fh:
        data = fh.read()
        data_ids = encode(data)
        result = np.array(data_ids, dtype=np.uint16)
        result.tofile(output_file)