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
import numpy as np

# --- Global State & Configuration ---
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
        # Use IndexFlatIP for cosine similarity with normalized vectors
        # 512 is the embedding dimension for clip-vit-base-patch32
        state["index"] = faiss.IndexFlatIP(512)

    if os.path.exists(META_PATH):
        with open(META_PATH, "rb") as f:
            state["metadata"] = pickle.load(f)
    print(f"Startup complete. Index contains {state['index'].ntotal} vectors.")
    
    yield

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
        response.raise_for_status()
        image = await asyncio.to_thread(Image.open, BytesIO(response.content))
        return image.convert("RGB")
    except httpx.RequestError as e:
        raise HTTPException(status_code=400, detail=f"Could not download image from URL: {e}")
    except Exception:
        raise HTTPException(status_code=400, detail="Invalid or corrupt image file.")

async def get_embedding(image: Image.Image = None, text: str = None) -> torch.Tensor:
    """Calculates a normalized embedding for an image, text, or both."""
    embeddings = []
    
    with torch.no_grad():
        if image:
            inputs = state["processor"](images=image, return_tensors="pt").to(state["device"])
            image_embedding = state["model"].get_image_features(**inputs)
            image_embedding = image_embedding / image_embedding.norm(p=2, dim=-1, keepdim=True)
            embeddings.append(image_embedding)
        
        if text:
            inputs = state["processor"](text=[text], return_tensors="pt").to(state["device"])
            text_embedding = state["model"].get_text_features(**inputs)
            text_embedding = text_embedding / text_embedding.norm(p=2, dim=-1, keepdim=True)
            embeddings.append(text_embedding)

    if not embeddings:
        return None
    
    if len(embeddings) == 1:
        return embeddings[0]
    
    # For multimodal queries, use weighted combination
    # You can experiment with different weights
    image_weight = 0.6
    text_weight = 0.4
    
    if len(embeddings) == 2:
        combined = image_weight * embeddings[0] + text_weight * embeddings[1]
        # Normalize the combined embedding
        combined = combined / combined.norm(p=2, dim=-1, keepdim=True)
        return combined
    
    # Fallback to average (though this shouldn't happen with current logic)
    combined = torch.stack(embeddings).mean(dim=0)
    return combined / combined.norm(p=2, dim=-1, keepdim=True)


# --- API Endpoints ---
class AddItem(BaseModel):
    image_url: str
    description: str = ""

@app.post("/add")
async def add_item(item: AddItem):
    image = await download_and_process_image(item.image_url)
    
    # Store both image-only and multimodal embeddings for better search
    if item.description.strip():
        embedding = await get_embedding(image=image, text=item.description)
    else:
        embedding = await get_embedding(image=image)
    
    if embedding is None:
        raise HTTPException(status_code=400, detail="Failed to generate embedding")
    
    # Convert to numpy and ensure it's the right shape
    embedding_np = embedding.cpu().numpy().astype('float32')
    if embedding_np.ndim == 2:
        embedding_np = embedding_np[0]  # Remove batch dimension
    
    # Add to index
    state["index"].add(embedding_np.reshape(1, -1))
    state["metadata"].append({
        "image_url": item.image_url, 
        "description": item.description
    })

    return {
        "message": f"Item added successfully. Index now contains {state['index'].ntotal} items.",
        "embedding_norm": float(np.linalg.norm(embedding_np))  # Should be ~1.0
    }


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
    
    if embedding is None:
        raise HTTPException(status_code=400, detail="Failed to generate query embedding")
    
    # Convert to numpy
    embedding_np = embedding.cpu().numpy().astype('float32')
    if embedding_np.ndim == 2:
        embedding_np = embedding_np[0]  # Remove batch dimension
    
    # Search with more results to get better ranking
    k = min(10, state["index"].ntotal)  # Don't search for more items than we have
    if k == 0:
        return {"results": [], "scores": []}
    
    D, I = state["index"].search(embedding_np.reshape(1, -1), k)
    
    # Filter out invalid indices and create results
    results = []
    scores = []
    for i, score in zip(I[0], D[0]):
        if i < len(state["metadata"]):  # Valid index
            results.append(state["metadata"][i])
            scores.append(float(score))
    
    # Sort by score (descending - higher is better for IndexFlatIP)
    sorted_pairs = sorted(zip(results, scores), key=lambda x: x[1], reverse=True)
    results, scores = zip(*sorted_pairs) if sorted_pairs else ([], [])
    
    return {
        "results": list(results)[:5],  # Return top 5
        "scores": list(scores)[:5],
        "query_embedding_norm": float(np.linalg.norm(embedding_np)),
        "total_items": state["index"].ntotal
    }

# Add a debug endpoint to check index health
@app.get("/debug/index_stats")
async def index_stats():
    if state["index"] is None:
        return {"error": "Index not initialized"}
    
    return {
        "total_items": state["index"].ntotal,
        "index_dimension": state["index"].d,
        "index_type": type(state["index"]).__name__,
        "metadata_count": len(state["metadata"]),
        "device": state["device"]
    }