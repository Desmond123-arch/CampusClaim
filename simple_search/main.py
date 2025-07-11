from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import torch
import faiss
from transformers import CLIPProcessor, CLIPModel
from PIL import Image
from io import BytesIO
import httpx
import pickle
import os
import asyncio
from contextlib import asynccontextmanager

# --- Global State & Configuration ---
# Using a dictionary to hold our "global" state makes it a bit cleaner
# to manage within the lifespan context.
state = {
    "device": "cuda" if torch.cuda.is_available() else "cpu",
    "model": None,
    "processor": None,
    "index": None,
    "metadata": []
}

INDEX_PATH = "index.faiss"
META_PATH = "metadata.pkl"

# --- FastAPI Lifespan Events ---
# This context manager handles startup and shutdown logic.
@asynccontextmanager
async def lifespan(app: FastAPI):
    # --- Startup ---
    print(f"Loading model on device: {state['device']}")
    state["model"] = CLIPModel.from_pretrained("openai/clip-vit-base-patch32").to(state["device"])
    state["processor"] = CLIPProcessor.from_pretrained("openai/clip-vit-base-patch32", use_fast=True)

    print("Loading FAISS index and metadata...")
    if os.path.exists(INDEX_PATH):
        state["index"] = faiss.read_index(INDEX_PATH)
    else:
        # For CLIP, Inner Product (IP) is better as we use cosine similarity.
        # 512 is the embedding dimension for clip-vit-base-patch32.
        state["index"] = faiss.IndexFlatIP(512)

    if os.path.exists(META_PATH):
        with open(META_PATH, "rb") as f:
            state["metadata"] = pickle.load(f)
    print(f"Startup complete. Index contains {state['index'].ntotal} vectors.")
    
    yield # The application runs here

    # --- Shutdown ---
    print("Saving FAISS index and metadata...")
    faiss.write_index(state["index"], INDEX_PATH)
    with open(META_PATH, "wb") as f:
        pickle.dump(state["metadata"], f)
    print("Shutdown complete. Files saved.")


app = FastAPI(lifespan=lifespan)
http_client = httpx.AsyncClient()


# --- Helper Functions ---
async def download_and_process_image(url: str) -> Image.Image:
    """Asynchronously downloads and opens an image."""
    try:
        response = await http_client.get(url)
        response.raise_for_status() # Raise an exception for bad status codes
        # Run synchronous PIL code in a thread pool to avoid blocking
        image = await asyncio.to_thread(Image.open, BytesIO(response.content))
        return image.convert("RGB")
    except httpx.RequestError as e:
        raise HTTPException(status_code=400, detail=f"Could not download image from URL: {e}")
    except Exception:
        raise HTTPException(status_code=400, detail="Invalid or corrupt image file.")

async def get_embedding(image: Image.Image = None, text: str = None) -> torch.Tensor:
    """Calculates a normalized embedding for an image, text, or both."""
    image_embedding = None
    text_embedding = None

    with torch.no_grad():
        if image:
            inputs = state["processor"](images=image, return_tensors="pt").to(state["device"])
            image_embedding = state["model"].get_image_features(**inputs)
            image_embedding /= image_embedding.norm(p=2, dim=-1, keepdim=True)
        
        if text:
            inputs = state["processor"](text=[text], return_tensors="pt").to(state["device"])
            text_embedding = state["model"].get_text_features(**inputs)
            text_embedding /= text_embedding.norm(p=2, dim=-1, keepdim=True)

    if image_embedding is not None and text_embedding is not None:
        embedding = (image_embedding + text_embedding) / 2
        embedding /= embedding.norm(p=2, dim=-1, keepdim=True)
    elif image_embedding is not None:
        embedding = image_embedding
    elif text_embedding is not None:
        embedding = text_embedding
    else:
        return None
        
    return embedding


# --- API Endpoints ---
class AddItem(BaseModel):
    image_url: str
    description: str = ""

@app.post("/add")
async def add_item(item: AddItem):
    image = await download_and_process_image(item.image_url)
    embedding = await get_embedding(image=image, text=item.description)
    
    # Add to in-memory index (very fast)
    state["index"].add(embedding.cpu().numpy())
    state["metadata"].append({ "image_url": item.image_url, "description": item.description })

    # We NO LONGER write to disk here. This is the key performance gain.
    return { "message": f"Item added successfully. Index now contains {state['index'].ntotal} items." }


class SearchQuery(BaseModel):
    image_url: str = None
    text: str = None

@app.post("/search")
async def search(query: SearchQuery):
    if not query.image_url and not query.text:
        raise HTTPException(status_code=400, detail="Must provide either an image_url or text.")

    image = None
    if query.image_url:
        image = await download_and_process_image(query.image_url)

    embedding = await get_embedding(image=image, text=query.text)
    
    D, I = state["index"].search(embedding.cpu().numpy(), k=5)

    results = [state["metadata"][i] for i in I[0]]
    return { "results": results, "scores": D[0].tolist() }