# nanoGPT model service

This module implements a service that includes a generative AI model and its corresponding attribution method. The module uses a _nanoGPT_ model (https://github.com/karpathy/nanoGPT), the _dattri_ library (https://github.com/trais-lab/dattri) for the attribution method, and the _Flask_ framework (https://flask.palletsprojects.com/en/stable/) to allow interaction with the model via HTTP requests.

## Requirements

Install the required packages with:

```
pip3 install dattri torch numpy transformers datasets tiktoken wandb tqdm Flask
```

## How to run

To launch the service, type:
```
python3 model_server_service.py
```

By default the model service will be available on port 9090.

To test the model with a simple query, type:
```
curl -X POST http://localhost:9090/attribute -H "Content-Type: application/json" -d '{"text": "Once upon a time there was a small dragon"}'
```
