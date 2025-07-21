from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field
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
    ### MODIFIED: We now use two structures for metadata management.
    # 1. A map from the internal FAISS index (integer) to your custom item_id (string)
    "id_map": [],
    # 2. A dictionary storing the actual metadata, keyed by your custom item_id
    "metadata_store": {}
}

# --- MODIFIED: Paths for our new data structures ---
INDEX_PATH = "index.faiss"
ID_MAP_PATH = "id_map.pkl"
METADATA_STORE_PATH = "metadata_store.pkl"


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

    # --- MODIFIED: Load the two new metadata files ---
    if os.path.exists(ID_MAP_PATH):
        with open(ID_MAP_PATH, "rb") as f:
            state["id_map"] = pickle.load(f)
            
    if os.path.exists(METADATA_STORE_PATH):
        with open(METADATA_STORE_PATH, "rb") as f:
            state["metadata_store"] = pickle.load(f)

    print(f"Startup complete. Index contains {state['index'].ntotal} vectors.")
    print(f"ID Map contains {len(state['id_map'])} entries.")
    
    # Sanity check
    if state['index'].ntotal != len(state['id_map']):
        print("WARNING: FAISS index size and ID map size are out of sync!")

    yield

    # --- Shutdown ---
    print("Saving FAISS index and metadata...")
    faiss.write_index(state["index"], INDEX_PATH)
    
    # --- MODIFIED: Save the two new metadata files ---
    with open(ID_MAP_PATH, "wb") as f:
        pickle.dump(state["id_map"], f)
    with open(METADATA_STORE_PATH, "wb") as f:
        pickle.dump(state["metadata_store"], f)
        
    print("Shutdown complete. Files saved.")


app = FastAPI(lifespan=lifespan)
http_client = httpx.AsyncClient()


# --- Helper Functions (No changes needed here) ---
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
    
    # For multimodal queries, you can experiment with weights
    image_weight = 0.6
    text_weight = 0.4
    
    if len(embeddings) == 2:
        combined = image_weight * embeddings[0] + text_weight * embeddings[1]
        combined = combined / combined.norm(p=2, dim=-1, keepdim=True)
        return combined
    
    combined = torch.stack(embeddings).mean(dim=0)
    return combined / combined.norm(p=2, dim=-1, keepdim=True)


# --- API Endpoints ---
# --- MODIFIED: Add 'item_id' to the AddItem model ---
class AddItem(BaseModel):
    item_id: str = Field(..., description="Unique identifier for the item.")
    image_url: str
    description: str = ""

@app.post("/add")
async def add_item(item: AddItem):
    # --- MODIFIED: Check for duplicate item_id ---
    if item.item_id in state["metadata_store"]:
        raise HTTPException(
            status_code=409,  # 409 Conflict is more appropriate
            detail=f"Item with id '{item.item_id}' already exists."
        )
    print(item)
    image = await download_and_process_image(item.image_url)
    
    # Store both image-only and multimodal embeddings for better search
    if item.description.strip():
        embedding = await get_embedding(image=image, text=item.description)
    else:
        embedding = await get_embedding(image=image)
    
    if embedding is None:
        raise HTTPException(status_code=400, detail="Failed to generate embedding")
    
    embedding_np = embedding.cpu().numpy().astype('float32')
    if embedding_np.ndim == 2:
        embedding_np = embedding_np[0]
    
    # --- MODIFIED: Add to index and update our new metadata structures ---
    # 1. Add vector to FAISS index
    state["index"].add(embedding_np.reshape(1, -1))
    
    # 2. Add the new item_id to our id_map. The list index will match the FAISS index.
    state["id_map"].append(item.item_id)
    
    # 3. Add the item's metadata to our metadata_store, keyed by its ID.
    state["metadata_store"][item.item_id] = {
        "image_url": item.image_url, 
        "description": item.description
    }
    
    # Sanity check
    assert state["index"].ntotal == len(state["id_map"]), "CRITICAL: Index and ID map are out of sync!"

    return {
        "message": f"Item '{item.item_id}' added successfully. Index now contains {state['index'].ntotal} items.",
        "embedding_norm": float(np.linalg.norm(embedding_np))
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
    
    embedding_np = embedding.cpu().numpy().astype('float32')
    if embedding_np.ndim == 2:
        embedding_np = embedding_np[0]
    
    k = min(10, state["index"].ntotal)
    if k == 0:
        return {"results": []}
    
    D, I = state["index"].search(embedding_np.reshape(1, -1), k)
    
    # --- MODIFIED: Use the id_map to retrieve results ---
    results = []
    for i, score in zip(I[0], D[0]):
        if i < 0: # FAISS can return -1 for invalid indices
            continue
        
        # 1. Get the custom item_id from our map using the FAISS index
        item_id = state["id_map"][i]
        
        # 2. Get the metadata from our store using the custom item_id
        metadata = state["metadata_store"][item_id]
        
        # 3. Combine them for the final result
        results.append({
            "item_id": item_id,
            "score": float(score),
            "image_url": metadata["image_url"],
            "description": metadata["description"]
        })
    
    # The results are already sorted by FAISS (for L2 distance). For IP (cosine sim), higher is better.
    # We don't need to re-sort here as FAISS returns them in the correct order.
    
    return {
        "results": results[:5], # Return top 5
        "total_items_in_index": state["index"].ntotal
    }

# --- MODIFIED: Update debug endpoint to show new stats ---
@app.get("/debug/index_stats")
async def index_stats():
    if state["index"] is None:
        return {"error": "Index not initialized"}
    
    return {
        "faiss_index_items": state["index"].ntotal,
        "id_map_items": len(state["id_map"]),
        "metadata_store_items": len(state["metadata_store"]),
        "is_synced": state["index"].ntotal == len(state["id_map"]),
        "index_dimension": state["index"].d,
        "index_type": type(state["index"]).__name__,
        "device": state["device"]
    }