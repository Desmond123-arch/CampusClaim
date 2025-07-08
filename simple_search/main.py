from fastapi import FastAPI
from pydantic import BaseModel
import torch
import faiss
from transformers import CLIPProcessor, CLIPModel
from PIL import Image
from io import BytesIO
import requests
import pickle
import os

app = FastAPI()

device = "cuda" if torch.cuda.is_available() else "cpu"
model = CLIPModel.from_pretrained("openai/clip-vit-base-patch32").to(device)
processor = CLIPProcessor.from_pretrained("openai/clip-vit-base-patch32")

INDEX_PATH = "index.faiss"
META_PATH = "metadata.pkl"

if os.path.exists(INDEX_PATH):
    index = faiss.read_index(INDEX_PATH)
else:
    index = faiss.IndexFlatIP(512)

if os.path.exists(META_PATH):
    with open(META_PATH, "rb") as f:
        metadata = pickle.load(f)
else:
    metadata = []

@app.post("/add")
async def add_item(data: dict):
    url = data.get("image_url")
    description = data.get("description", "")

    image = Image.open(BytesIO(requests.get(url).content)).convert("RGB")
    inputs = processor(images=image, return_tensors="pt").to(device)
    with torch.no_grad():
        image_embedding = model.get_image_features(**inputs)
        image_embedding /= image_embedding.norm(p=2, dim=-1, keepdim=True)

    if description:
        text_inputs = processor(text=[description], return_tensors="pt").to(device)
        with torch.no_grad():
            text_embedding = model.get_text_features(**text_inputs)
            text_embedding /= text_embedding.norm(p=2, dim=-1, keepdim=True)
        embedding = (image_embedding + text_embedding) / 2
    else:
        embedding = image_embedding

    embedding = embedding.cpu().numpy()
    index.add(embedding)
    metadata.append({ "image_url": url, "description": description })

    faiss.write_index(index, INDEX_PATH)
    with open(META_PATH, "wb") as f:
        pickle.dump(metadata, f)

    return { "message": "Item added successfully." }

class SearchQuery(BaseModel):
    image_url: str = None
    text: str = None

@app.post("/search")
async def search(query: SearchQuery):
    embed = None

    if query.image_url:
        image = Image.open(BytesIO(requests.get(query.image_url).content)).convert("RGB")
        inputs = processor(images=image, return_tensors="pt").to(device)
        with torch.no_grad():
            image_embedding = model.get_image_features(**inputs)
            image_embedding /= image_embedding.norm(p=2, dim=-1, keepdim=True)
        embed = image_embedding

    if query.text:
        text_inputs = processor(text=[query.text], return_tensors="pt").to(device)
        with torch.no_grad():
            text_embedding = model.get_text_features(**text_inputs)
            text_embedding /= text_embedding.norm(p=2, dim=-1, keepdim=True)
        embed = text_embedding if embed is None else (embed + text_embedding) / 2

    embed /= embed.norm(p=2, dim=-1, keepdim=True)
    D, I = index.search(embed.cpu().numpy(), k=5)

    results = [metadata[i] for i in I[0]]
    return { "results": results, "scores": D[0].tolist() }